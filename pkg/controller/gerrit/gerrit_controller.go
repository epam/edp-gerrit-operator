package gerrit

import (
	"context"
	"fmt"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/controller/helper"
	"time"

	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform"

	errorsf "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

var log = logf.Log.WithName("controller_gerrit")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Gerrit Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	client := mgr.GetClient()
	pt := helper.GetPlatformTypeEnv()
	ps, _ := platform.NewService(pt, scheme)
	return &ReconcileGerrit{
		client:  client,
		scheme:  scheme,
		service: gerrit.NewComponentService(ps, client, scheme),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gerrit-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Gerrit
	err = c.Watch(&source.Kind{Type: &v1alpha1.Gerrit{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileGerrit{}

// ReconcileGerrit reconciles a Gerrit object
type ReconcileGerrit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	service gerrit.Interface
}

// Reconcile reads that state of the cluster for a Gerrit object and makes changes based on the state read
// and what is in the Gerrit.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGerrit) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Gerrit")

	// Fetch the Gerrit instance
	instance := &v1alpha1.Gerrit{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		reqLogger.Info(fmt.Sprintf("%v/%v Gerrit installation started", instance.Namespace, instance.Name))
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.Install(instance)
	if err != nil {
		reqLogger.Info(fmt.Sprintf("%s/%s Gerrit installation failed!",instance.Name, instance.Namespace))
		r.resourceActionFailed(instance, err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	if instance.Status.Status == StatusInstall {
		reqLogger.Info(fmt.Sprintf("%v/%v Gerrit has been installed", instance.Namespace, instance.Name))
		r.updateStatus(instance, StatusCreated)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dIsReady, err := r.service.IsDeploymentReady(instance); err != nil {
		msg := fmt.Sprintf("Failed to check Deployment for %v/%v object!", instance.Namespace, instance.Name)
		reqLogger.Info(msg)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	} else if !dIsReady {
		reqLogger.Info(fmt.Sprintf("Deployment for %v/%v object is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		msg := fmt.Sprintf("Configuration for %v/%v object has started", instance.Namespace, instance.Name)
		reqLogger.Info(msg)
		err := r.updateStatus(instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, dPatched, err := r.service.Configure(instance)
	if err != nil {
		msg := fmt.Sprintf("%v/%v Gerrit configuration has failed", instance.Namespace, instance.Name)
		reqLogger.Info(msg)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	if dPatched {
		reqLogger.Info("Restarting deployment after configuration change")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dIsReady, err := r.service.IsDeploymentReady(instance); err != nil {
		reqLogger.Info(fmt.Sprintf("Failed to check Deployment config for %v/%v Gerrit!", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	} else if !dIsReady {
		reqLogger.Info(fmt.Sprintf("Deployment config for %v/%v Gerrit is not ready for configuration yet", instance.Namespace, instance.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		msg := fmt.Sprintf("%v/%v Gerrit configuration has finished", instance.Namespace, instance.Name)
		reqLogger.Info(msg)
		err = r.updateStatus(instance, StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusConfigured {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusExposeStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.ExposeConfiguration(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusExposeStart {
		reqLogger.Info("Exposing configuration has finished")
		err = r.updateStatus(instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil 
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		reqLogger.Info("Integration has started")
		err = r.updateStatus(instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.service.Integrate(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrapf(err, "Integration failed")
	}

	if instance.Status.Status == StatusIntegrationStart {
		msg := fmt.Sprintf("Configuration of %v/%v object has been finished", instance.Namespace, instance.Name)
		reqLogger.Info(msg)
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		msg := fmt.Sprintf("Failed update avalability status for Gerrit object with name %s", instance.Name)
		reqLogger.Info(msg)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	reqLogger.Info(fmt.Sprintf("Reconciling Gerrit component %v/%v has been finished", request.Namespace, request.Name))
	return reconcile.Result{}, nil
}

func (r *ReconcileGerrit) updateStatus(instance *v1alpha1.Gerrit, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return err
		}
	}
	log.Info(fmt.Sprintf("Status for Gerrit %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r *ReconcileGerrit) resourceActionFailed(instance *v1alpha1.Gerrit, err error) error {
	if r.updateStatus(instance, StatusFailed) != nil {
		return err
	}
	return err
}

func (r ReconcileGerrit) updateAvailableStatus(instance *v1alpha1.Gerrit, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
