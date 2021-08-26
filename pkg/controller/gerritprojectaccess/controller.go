package gerritprojectaccess

import (
	"context"
	"reflect"
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

const finalizerName = "gerritprojectaccess.gerrit.finalizer.name"

type Reconcile struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func NewReconcile(client client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
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
		For(&v1alpha1.GerritProjectAccess{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.GerritProjectAccess)
	no := e.ObjectNew.(*v1alpha1.GerritProjectAccess)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritProjectAccess has been started")

	var instance v1alpha1.GerritProjectAccess
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get GerritProjectAccess instance")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			reqLogger.Error(err, "unable to update instance status")
		}
	}()

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		reqLogger.Error(err, "unable to reconcile GerritProjectAccess")
		instance.Status.Value = err.Error()
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	instance.Status.Value = helper.StatusOK
	instance.Status.Created = true

	return
}

func prepareAccessInfo(references []v1alpha1.Reference) []gerritClient.AccessInfo {
	ai := make([]gerritClient.AccessInfo, 0, len(references))
	for _, ref := range references {
		ai = append(ai, gerritClient.AccessInfo{
			Action:          ref.Action,
			Force:           ref.Force,
			GroupName:       ref.GroupName,
			Max:             ref.Max,
			Min:             ref.Min,
			RefPattern:      ref.Pattern,
			PermissionLabel: ref.PermissionLabel,
			PermissionName:  ref.PermissionName,
		})
	}

	return ai
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v1alpha1.GerritProjectAccess) error {
	cl, err := helper.GetGerritClient(ctx, r.client, instance, instance.Spec.OwnerName, r.service)
	if err != nil {
		return errors.Wrap(err, "unable to init gerrit client")
	}

	if len(instance.Spec.References) > 0 && !instance.Status.Created {
		if err := cl.AddAccessRights(instance.Spec.ProjectName, prepareAccessInfo(instance.Spec.References)); err != nil {
			return errors.Wrap(err, "unable to add access rights")
		}
	} else if len(instance.Spec.References) > 0 {
		if err := cl.UpdateAccessRights(instance.Spec.ProjectName, prepareAccessInfo(instance.Spec.References)); err != nil {
			return errors.Wrap(err, "unable to update access rights")
		}
	}

	if instance.Spec.ProjectName != "" {
		if err := cl.SetProjectParent(instance.Spec.ProjectName, instance.Spec.Parent); err != nil {
			return errors.Wrap(err, "unable to set project parent")
		}
	}

	if err := helper.TryToDelete(ctx, r.client, instance, finalizerName,
		r.makeDeletionFunc(cl, instance.Spec.ProjectName, instance.Spec.References)); err != nil {
		return errors.Wrap(err, "error during TryToDelete")
	}

	return nil
}

func (r *Reconcile) makeDeletionFunc(gc gerritClient.ClientInterface, projectName string,
	refs []v1alpha1.Reference) func() error {
	return func() error {
		if err := gc.DeleteAccessRights(projectName, prepareAccessInfo(refs)); err != nil {
			return errors.Wrap(err, "unable to delete access rights")
		}

		return nil
	}
}
