package gerritproject

import (
	"context"
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

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	finalizerName       = "gerritproject.gerrit.finalizer.name"
	syncIntervalEnv     = "GERRIT_PROJECT_SYNC_INTERVAL"
	defaultSyncInterval = 300 * time.Second // 5 minutes
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
		log:     log.WithName("gerrit-project"),
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, syncInterval time.Duration) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	go r.syncBackendProjects(syncInterval)

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GerritProject{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.GerritProject)
	no := e.ObjectNew.(*v1alpha1.GerritProject)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritProject has been started")

	var instance v1alpha1.GerritProject
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
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	instance.Status.Value = helper.StatusOK

	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v1alpha1.GerritProject) error {
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

func (r *Reconcile) makeDeletionFunc(gc gerritClient.ClientInterface, projectName string) func() error {
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
