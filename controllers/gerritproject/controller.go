package gerritproject

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

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	"github.com/epam/edp-gerrit-operator/v2/controllers/helper"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	finalizerName       = "gerritproject.gerrit.finalizer.name"
	syncIntervalEnv     = "GERRIT_PROJECT_SYNC_INTERVAL"
	defaultSyncInterval = 300 * time.Second // 5 minutes
	requeueTime         = 10 * time.Second
)

type Reconcile struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func NewReconcile(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger) (*Reconcile, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &Reconcile{
		client:  k8sClient,
		service: gerrit.NewComponentService(ps, k8sClient, scheme),
		log:     log.WithName("gerrit-project"),
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, syncInterval time.Duration) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	go r.syncBackendProjects(syncInterval)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritProject{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup GerritProject controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*gerritApi.GerritProject)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*gerritApi.GerritProject)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritprojects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritprojects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritprojects/finalizers,verbs=update

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritProject has been started")

	var instance gerritApi.GerritProject
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			reqLogger.Info("instance not found")
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get GerritProject instance")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			reqLogger.Error(err, "unable to update instance status")
		}
	}()

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		reqLogger.Error(err, "unable to reconcile GerritProject")
		instance.Status.Value = err.Error()

		return reconcile.Result{RequeueAfter: requeueTime}, nil
	}

	instance.Status.Value = helper.StatusOK

	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *gerritApi.GerritProject) error {
	cl, err := helper.GetGerritClient(ctx, r.client, instance, instance.Spec.OwnerName, r.service)
	if err != nil {
		return errors.Wrap(err, "unable to init gerrit client")
	}

	_, err = cl.GetProject(instance.Spec.Name)
	if err != nil && !gerritClient.IsErrDoesNotExist(err) {
		return errors.Wrap(err, "unable to get project")
	}

	prj := gerritClient.Project{
		Name:              instance.Spec.Name,
		Description:       instance.Spec.Description,
		Parent:            instance.Spec.Parent,
		Branches:          instance.Spec.Branches,
		CreateEmptyCommit: instance.Spec.CreateEmptyCommit,
		Owners:            instance.Spec.Owners,
		PermissionsOnly:   instance.Spec.PermissionsOnly,
		RejectEmptyCommit: instance.Spec.RejectEmptyCommit,
		SubmitType:        instance.Spec.SubmitType,
	}

	if gerritClient.IsErrDoesNotExist(err) {
		if err := cl.CreateProject(&prj); err != nil {
			return errors.Wrap(err, "unable to create gerrit project")
		}
	} else {
		if err := cl.UpdateProject(&prj); err != nil {
			return errors.Wrap(err, "unable to update project")
		}
	}

	if err := helper.TryToDelete(ctx, r.client, instance, finalizerName,
		r.makeDeletionFunc(cl, instance.Spec.Name)); err != nil {
		return errors.Wrap(err, "error during TryToDelete")
	}

	return nil
}

func (*Reconcile) makeDeletionFunc(gc gerritClient.ClientInterface, projectName string) func() error {
	return func() error {
		if err := gc.DeleteProject(projectName); err != nil {
			return errors.Wrap(err, "unable to delete project")
		}

		return nil
	}
}

func SyncInterval() time.Duration {
	value, ok := os.LookupEnv(syncIntervalEnv)
	if !ok {
		return defaultSyncInterval
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return defaultSyncInterval
	}

	return interval
}
