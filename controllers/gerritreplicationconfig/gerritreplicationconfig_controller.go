package gerritreplicationconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	gerritController "github.com/epam/edp-gerrit-operator/v2/controllers/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/controllers/helper"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
)

const (
	config        = "/config"
	bin           = "/bin/sh"
	requeueTime   = 10 * time.Second
	requeueTime30 = 30 * time.Second
	containerFlag = "-c"
)

type configurationArguments struct {
	sshPortService     int32
	gerritPodName      string
	gerritUrl          string
	gerritAdminSshKeys map[string][]byte
	gerritVCSSshKey    map[string][]byte
}

func NewReconcileGerritReplicationConfig(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform service: %w", err)
	}

	return &ReconcileGerritReplicationConfig{
		client:           k8sClient,
		scheme:           scheme,
		platform:         ps,
		componentService: gerritService.NewComponentService(ps, k8sClient, scheme),
		log:              log.WithName("gerrit-replication-config"),
	}, nil
}

type ReconcileGerritReplicationConfig struct {
	client           client.Client
	scheme           *runtime.Scheme
	platform         platform.PlatformService
	componentService gerritService.Interface
	log              logr.Logger
}

func (r *ReconcileGerritReplicationConfig) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritReplicationConfig{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup GerritReplicationConfig controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*gerritApi.GerritReplicationConfig)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*gerritApi.GerritReplicationConfig)
	if !ok {
		return false
	}

	return oo.Status == no.Status
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritreplicationconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritreplicationconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gerritreplicationconfigs/finalizers,verbs=update

func (r *ReconcileGerritReplicationConfig) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GerritReplicationConfig")

	instance := &gerritApi.GerritReplicationConfig{}

	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get instance: %w", err)
	}

	if !helper.IsInstanceOwnerSet(instance) {
		ownerReference := helper.FindCROwnerName(instance.Spec.OwnerName)

		var gerritInstance *gerritApi.Gerrit

		gerritInstance, err = helper.GetGerritInstance(ctx, r.client, ownerReference, instance.Namespace)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to get gerrit instance: %w", err)
		}

		helper.SetOwnerReference(instance, gerritInstance.TypeMeta, &gerritInstance.ObjectMeta)

		err = r.client.Update(ctx, instance)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update instance owner refs: %w", err)
		}
	}

	gerritInstance, err := helper.GetInstanceOwner(ctx, r.client, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get instance owner: %w", err)
	}

	if gerritInstance.Status.Status == gerritController.StatusReady && (instance.Status.Status == "" || instance.Status.Status == spec.StatusFailed) {
		log.Info(fmt.Sprintf("Replication configuration of %s/%s object with name has been started",
			gerritInstance.Namespace, gerritInstance.Name))
		log.Info(fmt.Sprintf("Configuration of %s/%s object with name has been started", instance.Namespace, instance.Name))

		err = r.updateStatus(ctx, instance, spec.StatusConfiguring)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)
			return reconcile.Result{RequeueAfter: requeueTime}, nil
		}

		err = r.configureReplication(instance, gerritInstance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	if instance.Status.Status == spec.StatusConfiguring {
		log.Info(fmt.Sprintf("Configuration of %s/%s object has been finished", instance.Namespace, instance.Name))

		err = r.updateStatus(ctx, instance, spec.StatusConfigured)
		if err != nil {
			log.Error(err, "error while updating status", "status", instance.Status.Status)

			return reconcile.Result{RequeueAfter: requeueTime}, nil
		}
	}

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed update availability status for Gerrit Replication Config object with name %s", instance.Name)
		return reconcile.Result{RequeueAfter: requeueTime30}, nil
	}

	log.Info(fmt.Sprintf("Reconciling Gerrit Replication Config component %s/%s has been finished", request.Namespace, request.Name))

	return reconcile.Result{}, nil
}

func (r *ReconcileGerritReplicationConfig) updateStatus(ctx context.Context, instance *gerritApi.GerritReplicationConfig, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = metaV1.Now()

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err = r.client.Update(ctx, instance)
		if err != nil {
			return fmt.Errorf("failed to update GerritReplicationConfig CR %q: %w", instance.Name, err)
		}
	}

	r.log.V(1).Info(fmt.Sprintf("Status for Gerrit Replication Config %s has been updated to '%s' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))

	return nil
}

func (r *ReconcileGerritReplicationConfig) configureReplication(config *gerritApi.GerritReplicationConfig, gerritObj *gerritApi.Gerrit) error {
	gerritTemplatesPath := platformHelper.LocalTemplatesRelativePath

	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return errors.New("failed to check if operator running in cluster")
	}

	if helper.RunningInCluster() {
		gerritTemplatesPath = fmt.Sprintf("%s/../%s/%s",
			executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultTemplatesDirectory)
	}

	configArgs, err := r.getConfigurationArgs(gerritObj)
	if err != nil {
		return fmt.Errorf("failed to get configuration arguments: %w", err)
	}

	if err := r.saveSshReplicationKey(gerritObj.Namespace, configArgs.gerritPodName,
		string(configArgs.gerritVCSSshKey["ssh-privatekey"])); err != nil {
		return err
	}

	k8sClient := gerritClient.Client{}

	if err := k8sClient.InitNewSshClient(spec.GerritDefaultAdminUser, configArgs.gerritAdminSshKeys["id_rsa"],
		configArgs.gerritUrl, configArgs.sshPortService); err != nil {
		return fmt.Errorf("failed to init ssh client for Gerrit admin user: %w", err)
	}

	if err := r.createReplicationConfig(gerritObj.Namespace, configArgs.gerritPodName); err != nil {
		return err
	}

	if err := r.updateReplicationConfig(gerritObj.Namespace, configArgs.gerritPodName, gerritTemplatesPath, config); err != nil {
		return err
	}

	if err := r.updateSshConfig(gerritObj.Namespace, configArgs.gerritPodName, gerritTemplatesPath,
		filepath.Join(spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName), config); err != nil {
		return err
	}

	return r.reloadReplicationPlugin(&k8sClient)
}

func (r *ReconcileGerritReplicationConfig) getConfigurationArgs(gerritObj *gerritApi.Gerrit,
) (*configurationArguments, error) {
	podList, err := r.platform.GetPods(gerritObj.Namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%s", gerritObj.Name)})
	if err != nil {
		return nil, fmt.Errorf("failed to get Gerrit pods: %w", err)
	}

	if len(podList.Items) != 1 {
		return nil, errors.New("found multiple pods of Gerrit instance. It seems that some of old pods did not shutdown yet")
	}

	var args configurationArguments

	args.gerritPodName = podList.Items[0].Name

	args.gerritUrl, err = r.componentService.GetGerritSSHUrl(gerritObj)
	if err != nil {
		return nil, fmt.Errorf("failed to Get ssh url for gerrit instance: %w", err)
	}

	args.sshPortService, err = r.componentService.GetServicePort(gerritObj)
	if err != nil {
		return nil, fmt.Errorf("failed to Get ssh port for gerrit: %w", err)
	}

	args.gerritAdminSshKeys, err = r.platform.GetSecret(gerritObj.Namespace, gerritObj.Name+"-admin")
	if err != nil {
		return nil, fmt.Errorf("failed to Get a ssh key for admin user : %w", err)
	}

	args.gerritVCSSshKey, err = r.platform.GetSecret(gerritObj.Namespace, spec.GerritDefaultVCSKeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to Get a ssh key for autouser: %w", err)
	}

	return &args, nil
}

func (r *ReconcileGerritReplicationConfig) createReplicationConfig(namespace, podName string) error {
	command := []string{
		bin, containerFlag,
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath),
	}

	_, _, err := r.platform.ExecInPod(namespace, podName, command)
	if err != nil {
		return fmt.Errorf("failed executing command to create replication config: %w", err)
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createSshConfig(namespace, podName string) error {
	command := []string{
		bin, containerFlag,
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+config, spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+config, spec.DefaultGerritSSHConfigPath+config),
	}

	_, _, err := r.platform.ExecInPod(namespace, podName, command)
	if err != nil {
		return fmt.Errorf("failed executing command to create ssh config: %w", err)
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) saveSshReplicationKey(namespace, podName, key string) error {
	path := filepath.Join(spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	command := []string{bin, containerFlag, fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", key, path, path)}

	_, _, err := r.platform.ExecInPod(namespace, podName, command)
	if err != nil {
		return fmt.Errorf("failed executing command save ssh key: %w", err)
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateReplicationConfig(
	namespace, podName, templatePath string,
	grc *gerritApi.GerritReplicationConfig,
) error {
	config, err := resolveReplicationTemplate(grc, templatePath, "replication-conf.tmpl")
	if err != nil {
		return err
	}

	command := []string{bin, containerFlag, fmt.Sprintf("echo \"%v\" >> %v", config.String(), spec.DefaultGerritReplicationConfigPath)}

	_, _, err = r.platform.ExecInPod(namespace, podName, command)
	if err != nil {
		return fmt.Errorf("failed executing command to update Gerrit replication config: %w", err)
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateSshConfig(
	namespace, podName, templatePath, keyPath string,
	grc *gerritApi.GerritReplicationConfig,
) error {
	if err := r.createSshConfig(namespace, podName); err != nil {
		return err
	}

	sshTemplate, err := resolveSshTemplate(grc, templatePath, "ssh-sshTemplate.tmpl", keyPath)
	if err != nil {
		return err
	}

	command := []string{
		bin,
		containerFlag,
		fmt.Sprintf("echo %q >> %s", sshTemplate.String(), spec.DefaultGerritSSHConfigPath+config),
	}

	_, _, err = r.platform.ExecInPod(namespace, podName, command)
	if err != nil {
		return fmt.Errorf("failed to exec command in pod: %w", err)
	}

	return nil
}

func (*ReconcileGerritReplicationConfig) reloadReplicationPlugin(k8sClient gerritClient.ClientInterface) error {
	pluginName := "replication"

	err := k8sClient.ReloadPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("failed to reload Gerrit %q plugin: %w", pluginName, err)
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateAvailableStatus(ctx context.Context, instance *gerritApi.GerritReplicationConfig, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = metaV1.Now()

		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			err = r.client.Update(ctx, instance)
			if err != nil {
				return fmt.Errorf("failed to update GerritReplicationConfig CR %q: %w", instance.Name, err)
			}
		}
	}

	return nil
}

func resolveReplicationTemplate(grc *gerritApi.GerritReplicationConfig, path, templateName string) (*bytes.Buffer, error) {
	var config bytes.Buffer

	templatePath := filepath.FromSlash(filepath.Join(path, templateName))

	tmpl, err := template.New(templateName).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file %q: %w", templatePath, err)
	}

	err = tmpl.Execute(&config, grc)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &config, nil
}

func resolveSshTemplate(grc *gerritApi.GerritReplicationConfig, path, templateName, keyPath string) (*bytes.Buffer, error) {
	var config bytes.Buffer

	re := regexp.MustCompile(`@([^\[\]]*):`)
	host := re.FindStringSubmatch(grc.Spec.SSHUrl)

	data := struct {
		Hostname string
		KeyPath  string
	}{host[1], keyPath}
	templatePath := filepath.FromSlash(filepath.Join(path, templateName))

	tmpl, err := template.New(templateName).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file %q: %w", templatePath, err)
	}

	err = tmpl.Execute(&config, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &config, nil
}
