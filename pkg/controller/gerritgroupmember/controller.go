package gerritgroupmember

import (
	"context"
	"reflect"

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

const (
	finalizerName = "gerritgroupmember.gerrit.finalizer.name"
)

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

	return ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritGroupMember{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*gerritApi.GerritGroupMember)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*gerritApi.GerritGroupMember)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritGroupMember has been started")

	var instance gerritApi.GerritGroupMember
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get GerritGroupMember instance")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			reqLogger.Error(err, "unable to update instance status")
		}
	}()

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		reqLogger.Error(err, "unable to reconcile GerritGroupMember")
		instance.Status.Value = err.Error()
		requeueTime := helper.SetFailureCount(&instance)

		return reconcile.Result{RequeueAfter: requeueTime}, nil
	}

	helper.SetSuccessStatus(&instance)

	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *gerritApi.GerritGroupMember) error {
	cl, err := helper.GetGerritClient(ctx, r.client, instance, instance.Spec.OwnerName, r.service)
	if err != nil {
		return errors.Wrap(err, "unable to init gerrit client")
	}

	if err := cl.AddUserToGroup(instance.Spec.GroupID, instance.Spec.AccountID); err != nil {
		return errors.Wrap(err, "unable to add user to group")
	}

	if err := helper.TryToDelete(ctx, r.client, instance, finalizerName, r.makeDeletionFunc(cl, instance)); err != nil {
		return errors.Wrap(err, "unable to delete CR")
	}

	return nil
}

func (r *Reconcile) makeDeletionFunc(cl gerritClient.ClientInterface, instance *gerritApi.GerritGroupMember) func() error {
	return func() error {
		if err := cl.DeleteUserFromGroup(instance.Spec.GroupID, instance.Spec.AccountID); err != nil {
			return errors.Wrap(err, "unable to delete user from group")
		}

		return nil
	}
}
