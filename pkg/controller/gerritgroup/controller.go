package gerritgroup

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const requeueTime = 10 * time.Second

type Reconcile struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func NewReconcile(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &Reconcile{
		client:  k8sClient,
		service: gerrit.NewComponentService(ps, k8sClient, scheme),
		log:     log.WithName("gerrit"),
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritGroup{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup GerritGroup controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*gerritApi.GerritGroup)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*gerritApi.GerritGroup)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GerritGroup")

	var instance gerritApi.GerritGroup
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get gerrit group")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			log.Error(err, "unable to update instance status")
		}
	}()

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		log.Error(err, "unable to reconcile gerrit group")
		instance.Status.Value = err.Error()

		return reconcile.Result{RequeueAfter: requeueTime}, nil
	}

	instance.Status.Value = helper.StatusOK

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *gerritApi.GerritGroup) error {
	if !helper.IsInstanceOwnerSet(instance) {
		ownerReference := helper.FindCROwnerName(instance.Spec.OwnerName)

		gerritInstance, err := helper.GetGerritInstance(ctx, r.client, ownerReference, instance.Namespace)
		if err != nil {
			return errors.Wrap(err, "unable to get gerrit instance")
		}

		helper.SetOwnerReference(instance, gerritInstance.TypeMeta, &gerritInstance.ObjectMeta)

		err = r.client.Update(ctx, instance)
		if err != nil {
			return errors.Wrap(err, "unable to update instance owner refs")
		}
	}

	gerritInstance, err := helper.GetInstanceOwner(ctx, r.client, instance)
	if err != nil {
		return errors.Wrap(err, "unable to get instance owner")
	}

	cl, err := r.service.GetRestClient(gerritInstance)
	if err != nil {
		return errors.Wrap(err, "unable to get rest client")
	}

	gr, err := cl.CreateGroup(instance.Spec.Name, instance.Spec.Description, instance.Spec.VisibleToAll)
	if err == nil {
		instance.Status.ID = gr.ID
		instance.Status.GroupID = strconv.Itoa(gr.GroupID)

		// group is created, job done, we can exit
		return nil
	}

	if !gerritClient.IsErrAlreadyExists(err) {
		// unexpected error
		return errors.Wrap(err, "unable to create group")
	}

	// in case group already exists,
	// we want to make sure that CRs spec is in sync with group

	// if ID is not set yet, we can move on
	if instance.Status.ID == "" {
		return nil
	}

	err = cl.UpdateGroup(instance.Status.ID, instance.Spec.Description, instance.Spec.VisibleToAll)
	if err != nil {
		return errors.Wrap(err, "unable to update gerrit group")
	}

	return nil
}
