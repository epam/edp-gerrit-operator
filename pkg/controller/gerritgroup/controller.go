package gerritgroup

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
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
)

type Reconcile struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func NewReconcile(client client.Client, scheme *runtime.Scheme, log logr.Logger) (*Reconcile, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &Reconcile{
		client:  client,
		service: gerrit.NewComponentService(ps, client, scheme),
		log:     log.WithName("gerrit"),
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GerritGroup{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.GerritGroup)
	no := e.ObjectNew.(*v1alpha1.GerritGroup)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (res reconcile.Result, err error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GerritGroup")

	var instance v1alpha1.GerritGroup
	if err = r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
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
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	instance.Status.Value = helper.StatusOK

	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v1alpha1.GerritGroup) error {
	if !helper.IsInstanceOwnerSet(instance) {
		ownerReference := helper.FindCROwnerName(instance.Spec.OwnerName)

		gerritInstance, err := helper.GetGerritInstance(ctx, r.client, ownerReference, instance.Namespace)
		if err != nil {
			return errors.Wrap(err, "unable to get gerrit instance")
		}

		helper.SetOwnerReference(instance, gerritInstance.TypeMeta, gerritInstance.ObjectMeta)

		if err := r.client.Update(ctx, instance); err != nil {
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
	if err != nil && !gerritClient.IsErrAlreadyExists(err) {
		return errors.Wrap(err, "unable to create group")
	}

	if err != nil && gerritClient.IsErrAlreadyExists(err) {
		if instance.Status.ID != "" {
			if err := cl.UpdateGroup(instance.Status.ID, instance.Spec.Description, instance.Spec.VisibleToAll); err != nil {
				return errors.Wrap(err, "unable to update gerrit group")
			}
		}

		return nil
	}

	instance.Status.ID = gr.ID
	instance.Status.GroupID = strconv.Itoa(gr.GroupID)

	return nil
}
