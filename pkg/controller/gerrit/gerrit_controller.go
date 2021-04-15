package gerrit

import (
	"context"
	"fmt"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	errorsf "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

type ReconcileGerrit struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	Service gerrit.Interface
	Log     logr.Logger
}

func (r *ReconcileGerrit) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Gerrit{}).
		Complete(r)
}

func (r *ReconcileGerrit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Gerrit")

	instance := &v1alpha1.Gerrit{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info(fmt.Sprintf("%v/%v Gerrit installation started", instance.Namespace, instance.Name))
		err = r.updateStatus(ctx, instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info(fmt.Sprintf("%v/%v Gerrit has been installed", instance.Namespace, instance.Name))
		r.updateStatus(ctx, instance, StatusCreated)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dIsReady, err := r.Service.IsDeploymentReady(instance); err != nil {
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
		err := r.updateStatus(ctx, instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, dPatched, err := r.Service.Configure(instance)
	var finalRequeueAfterTimeout time.Duration
	if err != nil {
		log.Info(fmt.Sprintf("%v/%v Gerrit configuration has failed", instance.Namespace, instance.Name))
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

	if dIsReady, err := r.Service.IsDeploymentReady(instance); err != nil {
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
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")
		err = r.updateStatus(ctx, instance, StatusExposeStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.Service.ExposeConfiguration(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusExposeStart {
		log.Info("Exposing configuration has finished")
		err = r.updateStatus(ctx, instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		log.Info("Integration has started")
		err = r.updateStatus(ctx, instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.Service.Integrate(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrapf(err, "Integration failed")
	}

	if instance.Status.Status == StatusIntegrationStart {
		msg := fmt.Sprintf("Configuration of %v/%v object has been finished", instance.Namespace, instance.Name)
		log.Info(msg)
		err = r.updateStatus(ctx, instance, StatusReady)
		if err != nil {
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
	err := r.Client.Status().Update(ctx, instance)
	if err != nil {
		err := r.Client.Update(ctx, instance)
		if err != nil {
			return err
		}
	}
	r.Log.Info(fmt.Sprintf("Status for Gerrit %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r *ReconcileGerrit) resourceActionFailed(ctx context.Context, instance *v1alpha1.Gerrit, err error) error {
	if r.updateStatus(ctx, instance, StatusFailed) != nil {
		return err
	}
	return err
}

func (r ReconcileGerrit) updateAvailableStatus(ctx context.Context, instance *v1alpha1.Gerrit, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.Client.Status().Update(ctx, instance)
		if err != nil {
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
