package gerritreplicationconfig

import (
	"bytes"
	"context"
	coreerrors "errors"
	"fmt"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epmd-edp/gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/controller/gerrit"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/controller/helper"
	serviceHelper "github.com/epmd-edp/gerrit-operator/v2/pkg/helper"
	gerritService "github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform"
	platformHelper "github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/helper"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"regexp"
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
	"text/template"
	"time"
)

var log = logf.Log.WithName("controller_gerritreplicationconfig")

// Add creates a new GerritReplicationConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	pt := helper.GetPlatformTypeEnv()
	platformService, _ := platform.NewService(pt, mgr.GetScheme())
	client := mgr.GetClient()
	scheme := mgr.GetScheme()
	componentService := gerritService.NewComponentService(platformService, client, scheme)

	return &ReconcileGerritReplicationConfig{
		client:           client,
		scheme:           scheme,
		platform:         platformService,
		componentService: componentService,
	}
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
			oldObject := e.ObjectOld.(*v1alpha1.GerritReplicationConfig)
			newObject := e.ObjectNew.(*v1alpha1.GerritReplicationConfig)
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
	client           client.Client
	scheme           *runtime.Scheme
	platform         platform.PlatformService
	componentService gerritService.Interface
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

	if gerritInstance.Status.Status == gerrit.StatusReady && (instance.Status.Status == "" || instance.Status.Status == spec.StatusFailed) {
		reqLogger.Info(fmt.Sprintf("Replication configuration of %v/%v object with name has been started",
			gerritInstance.Namespace, gerritInstance.Name))
		reqLogger.Info(fmt.Sprintf("Configuration of %v/%v object with name has been started", instance.Namespace, instance.Name))
		err := r.updateStatus(instance, spec.StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		err = r.configureReplication(instance, gerritInstance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	if instance.Status.Status == spec.StatusConfiguring {
		log.Info(fmt.Sprintf("Configuration of %v/%v object has been finished", instance.Namespace, instance.Name))
		err = r.updateStatus(instance, spec.StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		log.Info("Failed update avalability status for Gerrit Replication Config object with name %s", instance.Name)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	reqLogger.Info(fmt.Sprintf("Reconciling Gerrit Replication Config component %v/%v has been finished", request.Namespace, request.Name))
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
	GerritTemplatesPath := platformHelper.LocalTemplatesRelativePath
	executableFilePath, err := serviceHelper.GetExecutableFilePath()
	if err != nil {
		return err
	}

	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		GerritTemplatesPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultTemplatesDirectory)
	}

	podList, err := r.platform.GetPods(gerrit.Namespace, v1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", gerrit.Name)})
	if err != nil || len(podList.Items) != 1 {
		return err
	}

	gerritUrl, err := r.componentService.GetGerritSSHUrl(gerrit)
	if err != nil {
		return err
	}

	sshPortService, err := r.componentService.GetServicePort(gerrit)
	if err != nil {
		return err
	}

	gerritAdminSshKeys, err := r.platform.GetSecret(gerrit.Namespace, gerrit.Name+"-admin")
	if err != nil {
		return err
	}

	gerritVCSSshKey, err := r.platform.GetSecret(gerrit.Namespace, spec.GerritDefaultVCSKeyName)
	if err != nil {
		return err
	}

	err = r.saveSshReplicationKey(gerrit.Namespace, podList.Items[0].Name, string(gerritVCSSshKey["ssh-privatekey"]))
	if err != nil {
		return err
	}

	client := gerritClient.Client{}
	err = client.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys["id_rsa"], gerritUrl, sshPortService)

	err = r.createReplicationConfig(gerrit.Namespace, podList.Items[0].Name)
	if err != nil {
		return err
	}

	err = r.updateReplicationConfig(gerrit.Namespace, podList.Items[0].Name, *config, GerritTemplatesPath)
	if err != nil {
		return err
	}

	err = r.updateSshConfig(gerrit.Namespace, podList.Items[0].Name, *config, GerritTemplatesPath,
		fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName))
	if err != nil {
		return err
	}

	err = r.reloadReplicationPlugin(client)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createReplicationConfig(namespace, podName string) error {
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createSshConfig(namespace, podName string) error {
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath+"/config")})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) saveSshReplicationKey(namespace, podName string, key string) error {
	path := fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", key, path, path)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateReplicationConfig(namespace, podName string,
	grc v1alpha1.GerritReplicationConfig, templatePath string) error {
	config, err := resolveReplicationTemplate(grc, templatePath, "replication-conf.tmpl")
	if err != nil {
		return err
	}

	_, _, err = r.platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" >> %v", config.String(), spec.DefaultGerritReplicationConfigPath)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateSshConfig(namespace, podName string,
	grc v1alpha1.GerritReplicationConfig, templatePath string, keyPath string) error {
	err := r.createSshConfig(namespace, podName)
	if err != nil {
		return err
	}

	config, err := resolveSshTemplate(grc, templatePath, "ssh-config.tmpl", keyPath)
	if err != nil {
		return err
	}

	_, _, err = r.platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" >> %v", config.String(), spec.DefaultGerritSSHConfigPath+"/config")})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) reloadReplicationPlugin(client gerritClient.Client) error {
	err := client.ReloadPlugin("replication")
	if err != nil {
		return err
	}

	return nil
}

func resolveReplicationTemplate(grc v1alpha1.GerritReplicationConfig, path, templateName string) (*bytes.Buffer, error) {
	var config bytes.Buffer

	tmpl, err := template.New(templateName).ParseFiles(filepath.FromSlash(fmt.Sprintf("%v/%v", path, templateName)))
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(&config, grc)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func resolveSshTemplate(grc v1alpha1.GerritReplicationConfig, path, templateName, keyPath string) (*bytes.Buffer, error) {
	var config bytes.Buffer
	re := regexp.MustCompile(`\@([^\[\]]*)\:`)
	host := re.FindStringSubmatch(grc.Spec.SSHUrl)

	data := struct {
		Hostname string
		KeyPath  string
	}{host[1], keyPath}

	tmpl, err := template.New(templateName).ParseFiles(filepath.FromSlash(fmt.Sprintf("%v/%v", path, templateName)))
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(&config, data)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (r ReconcileGerritReplicationConfig) updateAvailableStatus(instance *v1alpha1.GerritReplicationConfig, value bool) error {
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
