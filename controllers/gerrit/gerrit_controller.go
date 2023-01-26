package gerrit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	"github.com/epam/edp-gerrit-operator/v2/controllers/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	// StatusInstall = installing.
	StatusInstall = "installing"

	// StatusFailed = failed.
	StatusFailed = "failed"

	// StatusCreated = created.
	StatusCreated = "created"

	// StatusConfiguring = configuring.
	StatusConfiguring = "configuring"

	// StatusConfigured = configured.
	StatusConfigured = "configured"

	// StatusExposeStart = exposing config.
	StatusExposeStart = "exposing config"

	// StatusExposeFinish = config exposed.
	StatusExposeFinish = "config exposed"

	// StatusIntegrationStart = integration started.
	StatusIntegrationStart = "integration started"

	// StatusReady = ready.
	StatusReady = "ready"

	// RequeueTime10 = 10.
	RequeueTime10 = 10 * time.Second

	// requeueTime30 = 30.
	requeueTime30 = 30 * time.Second

	// requeueTime60 = 60.
	requeueTime60 = 60 * time.Second

	status = "status"

	updatingStatusErr = "error while updating status"
)

func NewReconcileGerrit(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &ReconcileGerrit{
		client:  k8sClient,
		service: gerrit.NewComponentService(ps, k8sClient, scheme),
		log:     log.WithName("gerrit"),
	}, nil
}

type ReconcileGerrit struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func (r *ReconcileGerrit) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.Gerrit{}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup Gerrit controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerrits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerrits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerrits/finalizers,verbs=update

func (r *ReconcileGerrit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Gerrit")

	instance := &gerritApi.Gerrit{}

	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed Get Gerrit CR %q: %w", request.NamespacedName, err)
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info(fmt.Sprintf("%s/%s Gerrit installation started", instance.Namespace, instance.Name))

		err = r.updateStatus(ctx, instance, StatusInstall)
		if err != nil {
			log.Error(err, updatingStatusErr, status, instance.Status.Status)
			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info(fmt.Sprintf("%s/%s Gerrit has been installed", instance.Namespace, instance.Name))

		err = r.updateStatus(ctx, instance, StatusCreated)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: RequeueTime10}, nil
	}

	dIsReady, err := r.service.IsDeploymentReady(instance)
	if err != nil {
		msg := fmt.Sprintf("Failed to check Deployment for %s/%s object!", instance.Namespace, instance.Name)
		log.Info(msg)

		return reconcile.Result{RequeueAfter: RequeueTime10}, nil
	}

	if !dIsReady {
		log.Info(fmt.Sprintf("Deployment for %s/%s object is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: requeueTime30}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		msg := fmt.Sprintf("Configuration for %s/%s object has started", instance.Namespace, instance.Name)
		log.Info(msg)

		err = r.updateStatus(ctx, instance, StatusConfiguring)
		if err != nil {
			log.Error(err, updatingStatusErr, status, instance.Status.Status)
			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	var finalRequeueAfterTimeout time.Duration

	instance, dPatched, err := r.service.Configure(instance)
	if err != nil {
		log.Error(err, "Gerrit configuration has been failed.")

		if !gerrit.IsErrUserNotFound(err) {
			return reconcile.Result{RequeueAfter: RequeueTime10}, fmt.Errorf("failed to configure Gerrit: %w", err)
		}

		finalRequeueAfterTimeout = requeueTime60
	}

	if dPatched {
		log.Info("Restarting deployment after configuration change")
		return reconcile.Result{RequeueAfter: RequeueTime10}, nil
	}

	dIsReady, err = r.service.IsDeploymentReady(instance)
	if err != nil {
		msg := fmt.Sprintf("Failed to check Deployment config for %s/%s Gerrit!", instance.Namespace, instance.Name)

		log.Info(msg)

		return reconcile.Result{RequeueAfter: RequeueTime10}, fmt.Errorf("%s: %w", msg, err)
	}

	if !dIsReady {
		log.Info(fmt.Sprintf("Deployment config for %v/%v Gerrit is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: requeueTime30}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		msg := fmt.Sprintf("%s/%s Gerrit configuration has finished", instance.Namespace, instance.Name)
		log.Info(msg)

		err = r.updateStatus(ctx, instance, StatusConfigured)
		if err != nil {
			log.Error(err, updatingStatusErr, status, instance.Status.Status)
			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")

		err = r.updateStatus(ctx, instance, StatusExposeStart)
		if err != nil {
			log.Error(err, updatingStatusErr, status, instance.Status.Status)

			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	exposedInstance, err := r.service.ExposeConfiguration(ctx, instance)
	if err != nil {
		log.Error(err, "error while exposing configuration", "name", instance.Name)
		return reconcile.Result{RequeueAfter: RequeueTime10}, nil
	}

	if exposedInstance.Status.Status == StatusExposeStart {
		log.Info("Exposing configuration has finished")

		err = r.updateStatus(ctx, exposedInstance, StatusExposeFinish)
		if err != nil {
			log.Error(err, updatingStatusErr, status, StatusExposeStart)
			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	if exposedInstance.Status.Status == StatusExposeFinish {
		log.Info("Integration has started")

		err = r.updateStatus(ctx, exposedInstance, StatusIntegrationStart)
		if err != nil {
			log.Error(err, updatingStatusErr, status, exposedInstance.Status.Status)
			return reconcile.Result{RequeueAfter: RequeueTime10}, err
		}
	}

	exposedInstance, err = r.service.Integrate(ctx, exposedInstance)
	if err != nil {
		return reconcile.Result{RequeueAfter: RequeueTime10}, errors.Wrapf(err, "Integration failed")
	}

	if exposedInstance.Status.Status == StatusIntegrationStart {
		msg := fmt.Sprintf("Configuration of %s/%s object has been finished", exposedInstance.Namespace, exposedInstance.Name)
		log.Info(msg)

		err = r.updateStatus(ctx, exposedInstance, StatusReady)
		if err != nil {
			log.Error(err, updatingStatusErr, status, exposedInstance.Status.Status)
			return reconcile.Result{RequeueAfter: RequeueTime10}, nil
		}
	}

	err = r.updateAvailableStatus(ctx, exposedInstance, true)
	if err != nil {
		msg := fmt.Sprintf("Failed update availability status for Gerrit object with name %s", exposedInstance.Name)
		log.Info(msg)

		return reconcile.Result{RequeueAfter: requeueTime30}, nil
	}

	log.Info(fmt.Sprintf("Reconciling Gerrit component %s/%s has been finished", request.Namespace, request.Name))

	return reconcile.Result{RequeueAfter: finalRequeueAfterTimeout}, nil
}

func (r *ReconcileGerrit) updateStatus(ctx context.Context, instance *gerritApi.Gerrit, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = metav1.Now()

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err = r.client.Update(ctx, instance)
		if err != nil {
			return fmt.Errorf("failed to update Gerrit resource: %w", err)
		}
	}

	r.log.Info(fmt.Sprintf("Status for Gerrit %s has been updated to '%s' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))

	return nil
}

func (r ReconcileGerrit) updateAvailableStatus(ctx context.Context, instance *gerritApi.Gerrit, value bool) error {
	if instance.Status.Available == value {
		return nil
	}

	instance.Status.Available = value
	instance.Status.LastTimeUpdated = metav1.Now()

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err = r.client.Update(ctx, instance)
		if err != nil {
			return fmt.Errorf("failed to update Gerrit resource: %w", err)
		}
	}

	return nil
}
