package gerritreplicationconfig

import (
	"context"
	coreerrors "errors"
	"fmt"
	"gerrit-operator/pkg/apis/v2/v1alpha1"
	"gerrit-operator/pkg/controller/gerrit"
	"gerrit-operator/pkg/controller/helper"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

const (
	// StatusReplicationConfiguration = replication configuration
	StatusReplicationConfiguration = "configuration"

	// StatusReplicationFinished = replication configuration
	StatusReplicationFinished = "configured"

	// StatusFailed = failed
	StatusFailed = "failed"
)

var log = logf.Log.WithName("controller_gerritreplicationconfig")

// Add creates a new GerritReplicationConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGerritReplicationConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gerritreplicationconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*v1alpha1.Gerrit)
			newObject := e.ObjectNew.(*v1alpha1.Gerrit)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	// Watch for changes to primary resource GerritReplicationConfig
	err = c.Watch(&source.Kind{Type: &v1alpha1.GerritReplicationConfig{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGerritReplicationConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGerritReplicationConfig{}

// ReconcileGerritReplicationConfig reconciles a GerritReplicationConfig object
type ReconcileGerritReplicationConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a GerritReplicationConfig object and makes changes based on the state read
// and what is in the GerritReplicationConfig.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGerritReplicationConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GerritReplicationConfig")

	// Fetch the GerritReplicationConfig instance
	instance := &v1alpha1.GerritReplicationConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !r.isInstanceOwnerSet(instance) {
		ownerReference := findCROwnerName(*instance)

		gerritInstance, err := r.getGerritInstance(ownerReference, instance.Namespace)
		if err != nil {
			return reconcile.Result{}, err
		}

		instance := r.setOwnerReference(gerritInstance, instance)

		err = r.client.Update(context.TODO(), &instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	gerritInstance, err := r.getInstanceOwner(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if gerritInstance.Status.Status == gerrit.StatusReady && (instance.Status.Status == "" || instance.Status.Status == StatusFailed) {
		reqLogger.Info(fmt.Sprintf("Replication configuration of %v/%v object with name has been started",
			gerritInstance.Namespace, gerritInstance.Name))
		reqLogger.Info(fmt.Sprintf("Configuration of %v/%v object with name has been started", instance.Namespace, instance.Name))
		err := r.updateStatus(instance, StatusReplicationConfiguration)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		err = r.configureReplication(instance, gerritInstance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGerritReplicationConfig) updateStatus(instance *v1alpha1.GerritReplicationConfig, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return err
		}
	}

	log.V(1).Info(fmt.Sprintf("Status for Gerrit Replication Config %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r *ReconcileGerritReplicationConfig) isInstanceOwnerSet(config *v1alpha1.GerritReplicationConfig) bool {
	log.V(1).Info(fmt.Sprintf("Start getting %v/%v owner", config.Kind, config.Name))
	ows := config.GetOwnerReferences()
	if len(ows) == 0 {
		return false
	}

	return true
}

func (r *ReconcileGerritReplicationConfig) getInstanceOwner(config *v1alpha1.GerritReplicationConfig) (*v1alpha1.Gerrit, error) {
	log.V(1).Info(fmt.Sprintf("Start getting %v/%v owner", config.Kind, config.Name))
	ows := config.GetOwnerReferences()
	gerritOwner := getGerritOwner(ows)
	if gerritOwner == nil {
		return nil, coreerrors.New("gerrit replication config cr does not have gerrit cr owner references")
	}

	nsn := types.NamespacedName{
		Namespace: config.Namespace,
		Name:      gerritOwner.Name,
	}

	ownerCr := &v1alpha1.Gerrit{}
	err := r.client.Get(context.TODO(), nsn, ownerCr)
	return ownerCr, err
}

func getGerritOwner(references []v1.OwnerReference) *v1.OwnerReference {
	for _, el := range references {
		if el.Kind == "Gerrit" {
			return &el
		}
	}
	return nil
}

func findCROwnerName(instance v1alpha1.GerritReplicationConfig) *string {
	if len(instance.Spec.OwnerName) == 0 {
		return nil
	}
	own := strings.ToLower(instance.Spec.OwnerName)
	return &own
}

func (r *ReconcileGerritReplicationConfig) getGerritInstance(ownerName *string, namespace string) (*v1alpha1.Gerrit, error) {
	var gerritInstance v1alpha1.Gerrit
	options := client.ListOptions{Namespace: namespace}
	list := &v1alpha1.GerritList{}
	if ownerName == nil {
		err := r.client.List(context.TODO(), &options, list)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		gerritInstance = list.Items[0]
	} else {
		gerritInstance = v1alpha1.Gerrit{}
		err := r.client.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      *ownerName,
		}, &gerritInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
	}

	return &gerritInstance, nil
}

func (r *ReconcileGerritReplicationConfig) setOwnerReference(gerritInstance *v1alpha1.Gerrit,
	instance *v1alpha1.GerritReplicationConfig) v1alpha1.GerritReplicationConfig {
	var listOwnReference []v1.OwnerReference

	ownRef := v1.OwnerReference{
		APIVersion:         gerritInstance.APIVersion,
		Kind:               gerritInstance.Kind,
		Name:               gerritInstance.Name,
		UID:                gerritInstance.UID,
		BlockOwnerDeletion: helper.NewTrue(),
		Controller:         helper.NewTrue(),
	}

	listOwnReference = append(listOwnReference, ownRef)

	instance.SetOwnerReferences(listOwnReference)

	return *instance
}

func (r *ReconcileGerritReplicationConfig) configureReplication(config *v1alpha1.GerritReplicationConfig, gerrit *v1alpha1.Gerrit) error {
	return nil
}
