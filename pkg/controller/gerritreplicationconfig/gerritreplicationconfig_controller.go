package gerritreplicationconfig

import (
	"bytes"
	"context"
	coreerrors "errors"
	"fmt"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"text/template"
	"time"
)

type ReconcileGerritReplicationConfig struct {
	Client           client.Client
	Scheme           *runtime.Scheme
	Platform         platform.PlatformService
	ComponentService gerritService.Interface
	Log              logr.Logger
}

func (r *ReconcileGerritReplicationConfig) SetupWithManager(mgr ctrl.Manager) error {
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GerritReplicationConfig{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileGerritReplicationConfig) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GerritReplicationConfig")

	instance := &v1alpha1.GerritReplicationConfig{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !r.isInstanceOwnerSet(instance) {
		ownerReference := findCROwnerName(*instance)

		gerritInstance, err := r.getGerritInstance(ctx, ownerReference, instance.Namespace)
		if err != nil {
			return reconcile.Result{}, err
		}

		instance := r.setOwnerReference(gerritInstance, instance)

		err = r.Client.Update(ctx, &instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	gerritInstance, err := r.getInstanceOwner(ctx, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if gerritInstance.Status.Status == gerrit.StatusReady && (instance.Status.Status == "" || instance.Status.Status == spec.StatusFailed) {
		log.Info(fmt.Sprintf("Replication configuration of %v/%v object with name has been started",
			gerritInstance.Namespace, gerritInstance.Name))
		log.Info(fmt.Sprintf("Configuration of %v/%v object with name has been started", instance.Namespace, instance.Name))
		err := r.updateStatus(ctx, instance, spec.StatusConfiguring)
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
		err = r.updateStatus(ctx, instance, spec.StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed update avalability status for Gerrit Replication Config object with name %s", instance.Name)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info(fmt.Sprintf("Reconciling Gerrit Replication Config component %v/%v has been finished", request.Namespace, request.Name))
	return reconcile.Result{}, nil
}

func (r *ReconcileGerritReplicationConfig) updateStatus(ctx context.Context, instance *v1alpha1.GerritReplicationConfig, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.Client.Status().Update(ctx, instance)
	if err != nil {
		err := r.Client.Update(ctx, instance)
		if err != nil {
			return err
		}
	}

	r.Log.V(1).Info(fmt.Sprintf("Status for Gerrit Replication Config %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r *ReconcileGerritReplicationConfig) isInstanceOwnerSet(config *v1alpha1.GerritReplicationConfig) bool {
	r.Log.V(1).Info(fmt.Sprintf("Start getting %v/%v owner", config.Kind, config.Name))
	ows := config.GetOwnerReferences()
	if len(ows) == 0 {
		return false
	}

	return true
}

func (r *ReconcileGerritReplicationConfig) getInstanceOwner(ctx context.Context, config *v1alpha1.GerritReplicationConfig) (*v1alpha1.Gerrit, error) {
	r.Log.V(1).Info(fmt.Sprintf("Start getting %v/%v owner", config.Kind, config.Name))
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
	err := r.Client.Get(ctx, nsn, ownerCr)
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

func (r *ReconcileGerritReplicationConfig) getGerritInstance(ctx context.Context, ownerName *string, namespace string) (*v1alpha1.Gerrit, error) {
	var gerritInstance v1alpha1.Gerrit
	options := client.ListOptions{Namespace: namespace}
	list := &v1alpha1.GerritList{}
	if ownerName == nil {
		err := r.Client.List(ctx, list, &options)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		gerritInstance = list.Items[0]
	} else {
		gerritInstance = v1alpha1.Gerrit{}
		err := r.Client.Get(ctx, client.ObjectKey{
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
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return err
	}

	if helper.RunningInCluster() {
		GerritTemplatesPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultTemplatesDirectory)
	}

	podList, err := r.Platform.GetPods(gerrit.Namespace, v1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", gerrit.Name)})
	if err != nil || len(podList.Items) != 1 {
		return err
	}

	gerritUrl, err := r.ComponentService.GetGerritSSHUrl(gerrit)
	if err != nil {
		return err
	}

	sshPortService, err := r.ComponentService.GetServicePort(gerrit)
	if err != nil {
		return err
	}

	gerritAdminSshKeys, err := r.Platform.GetSecret(gerrit.Namespace, gerrit.Name+"-admin")
	if err != nil {
		return err
	}

	gerritVCSSshKey, err := r.Platform.GetSecret(gerrit.Namespace, spec.GerritDefaultVCSKeyName)
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
	_, _, err := r.Platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createSshConfig(namespace, podName string) error {
	_, _, err := r.Platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
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
	_, _, err := r.Platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
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

	_, _, err = r.Platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
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

	_, _, err = r.Platform.ExecInPod(namespace, podName, []string{"/bin/sh", "-c",
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

func (r ReconcileGerritReplicationConfig) updateAvailableStatus(ctx context.Context, instance *v1alpha1.GerritReplicationConfig, value bool) error {
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
