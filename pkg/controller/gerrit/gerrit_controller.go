package gerrit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	// StatusInstall = installing
	StatusInstall = "installing"

	// StatusFailed = failed
	StatusFailed = "failed"

	// StatusCreated = created
	StatusCreated = "created"

	// StatusConfiguring = configuring
	StatusConfiguring = "configuring"

	// StatusConfigured = configured
	StatusConfigured = "configured"

	// StatusExposeStart = exposing config
	StatusExposeStart = "exposing config"

	// StatusExposeFinish = config exposed
	StatusExposeFinish = "config exposed"

	// StatusIntegrationStart = integration started
	StatusIntegrationStart = "integration started"

	// StatusReady = ready
	StatusReady = "ready"
)

func NewReconcileGerrit(client client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &ReconcileGerrit{
		client:  client,
		service: gerrit.NewComponentService(ps, client, scheme),
		log:     log.WithName("gerrit"),
	}, nil
}

type ReconcileGerrit struct {
	client  client.Client
	service gerrit.Interface
	log     logr.Logger
}

func (r *ReconcileGerrit) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Gerrit{}).
		Complete(r)
}

func (r *ReconcileGerrit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Gerrit")

	instance := &v1alpha1.Gerrit{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info(fmt.Sprintf("%v/%v Gerrit installation started", instance.Namespace, instance.Name))
		err = r.updateStatus(ctx, instance, StatusInstall)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info(fmt.Sprintf("%v/%v Gerrit has been installed", instance.Namespace, instance.Name))
		err = r.updateStatus(ctx, instance, StatusCreated)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dIsReady, err := r.service.IsDeploymentReady(instance); err != nil {
		msg := fmt.Sprintf("Failed to check Deployment for %v/%v object!", instance.Namespace, instance.Name)
		log.Info(msg)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	} else if !dIsReady {
		log.Info(fmt.Sprintf("Deployment for %v/%v object is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		msg := fmt.Sprintf("Configuration for %v/%v object has started", instance.Namespace, instance.Name)
		log.Info(msg)
		err = r.updateStatus(ctx, instance, StatusConfiguring)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, dPatched, err := r.service.Configure(instance)
	var finalRequeueAfterTimeout time.Duration
	if err != nil {
		log.Error(err, "Gerrit configuration has been failed.")
		if gerrit.IsErrUserNotFound(err) {
			finalRequeueAfterTimeout = 60 * time.Second
		} else {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	if dPatched {
		log.Info("Restarting deployment after configuration change")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dIsReady, err := r.service.IsDeploymentReady(instance); err != nil {
		log.Info(fmt.Sprintf("Failed to check Deployment config for %v/%v Gerrit!", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	} else if !dIsReady {
		log.Info(fmt.Sprintf("Deployment config for %v/%v Gerrit is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		msg := fmt.Sprintf("%v/%v Gerrit configuration has finished", instance.Namespace, instance.Name)
		log.Info(msg)
		err = r.updateStatus(ctx, instance, StatusConfigured)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")
		err = r.updateStatus(ctx, instance, StatusExposeStart)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.ExposeConfiguration(instance)
	if err != nil {
		log.Error(err, "error while exposing configuration", "name", instance.Name)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusExposeStart {
		log.Info("Exposing configuration has finished")
		err = r.updateStatus(ctx, instance, StatusExposeFinish)
		if err != nil {
			log.Error(err, "error while updating status", "status", StatusExposeStart)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		log.Info("Integration has started")
		err = r.updateStatus(ctx, instance, StatusIntegrationStart)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.service.Integrate(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errors.Wrapf(err, "Integration failed")
	}

	if instance.Status.Status == StatusIntegrationStart {
		msg := fmt.Sprintf("Configuration of %v/%v object has been finished", instance.Namespace, instance.Name)
		log.Info(msg)
		err = r.updateStatus(ctx, instance, StatusReady)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		msg := fmt.Sprintf("Failed update avalability status for Gerrit object with name %s", instance.Name)
		log.Info(msg)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info(fmt.Sprintf("Reconciling Gerrit component %v/%v has been finished", request.Namespace, request.Name))
	return reconcile.Result{RequeueAfter: finalRequeueAfterTimeout}, nil
}

func (r *ReconcileGerrit) updateStatus(ctx context.Context, instance *v1alpha1.Gerrit, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err := r.client.Update(ctx, instance)
		if err != nil {
			return err
		}
	}
	r.log.Info(fmt.Sprintf("Status for Gerrit %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r ReconcileGerrit) updateAvailableStatus(ctx context.Context, instance *v1alpha1.Gerrit, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			err := r.client.Update(ctx, instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
