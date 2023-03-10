package gerritgroupmember

import (
	"context"
	"fmt"
	"os"
	"reflect"
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

const (
	finalizerName   = "gerritgroupmember.gerrit.finalizer.name"
	syncIntervalEnv = "GERRIT_GROUP_MEMBER_SYNC_INTERVAL"
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

	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritGroupMember{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup GerritGroupMember controller: %w", err)
	}

	return nil
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

		expRequeueTime := helper.SetFailureCount(&instance)

		constTimeout, isSet := getSyncInterval(syncIntervalEnv)
		if !isSet {
			reqLogger.Info("Unable to get sync interval from env. Requeue time will be set using the exponential formula")
			reqLogger.Info("Requeue time", "time", expRequeueTime.String())

			return reconcile.Result{RequeueAfter: expRequeueTime}, nil
		}

		reqLogger.Info("Requeue time", "time", constTimeout.String())

		return reconcile.Result{RequeueAfter: constTimeout}, nil
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

func (*Reconcile) makeDeletionFunc(cl gerritClient.ClientInterface, instance *gerritApi.GerritGroupMember) func() error {
	return func() error {
		if err := cl.DeleteUserFromGroup(instance.Spec.GroupID, instance.Spec.AccountID); err != nil {
			return errors.Wrap(err, "unable to delete user from group")
		}

		return nil
	}
}

func getSyncInterval(envVarName string) (time.Duration, bool) {
	value, ok := os.LookupEnv(envVarName)
	if !ok {
		return 0, false
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return 0, false
	}

	if interval <= 0 {
		return 0, false
	}

	return interval, true
}
