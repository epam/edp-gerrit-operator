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

	if err := r.tryToDelete(ctx, cl, instance); err != nil {
		return errors.Wrap(err, "unable to ")
	}

	return nil
}

func (r *Reconcile) tryToDelete(ctx context.Context, gc gerritClient.ClientInterface,
	instance *v1alpha1.GerritProjectAccess) error {
	if instance.GetDeletionTimestamp().IsZero() {
		if !helper.ContainsString(instance.ObjectMeta.Finalizers, finalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
			if err := r.client.Update(ctx, instance); err != nil {
				return errors.Wrap(err, "unable to update instance finalizer")
			}
		}

		return nil
	}

	if err := gc.DeleteAccessRights(instance.Spec.ProjectName, prepareAccessInfo(instance.Spec.References)); err != nil {
		return errors.Wrap(err, "unable to delete access rights")
	}

	instance.ObjectMeta.Finalizers = helper.RemoveString(instance.ObjectMeta.Finalizers, finalizerName)
	if err := r.client.Update(ctx, instance); err != nil {
		return errors.Wrap(err, "unable to remove finalizer from instance")
	}

	return nil
}
