package gerritreplicationconfig

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
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

func NewReconcileGerritReplicationConfig(client client.Client, scheme *runtime.Scheme, log logr.Logger) (helper.Controller, error) {
	ps, err := platform.NewService(helper.GetPlatformTypeEnv(), scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &ReconcileGerritReplicationConfig{
		client:           client,
		scheme:           scheme,
		platform:         ps,
		componentService: gerritService.NewComponentService(ps, client, scheme),
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
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*gerritApi.GerritReplicationConfig)
			newObject := e.ObjectNew.(*gerritApi.GerritReplicationConfig)
			return oldObject.Status == newObject.Status
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritReplicationConfig{}, builder.WithPredicates(p)).
		Complete(r)
}

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
		return reconcile.Result{}, errors.Wrap(err, "unable to get instance")
	}

	if !helper.IsInstanceOwnerSet(instance) {
		ownerReference := helper.FindCROwnerName(instance.Spec.OwnerName)

		gerritInstance, err := helper.GetGerritInstance(ctx, r.client, ownerReference, instance.Namespace)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "unable to get gerrit instance")
		}

		helper.SetOwnerReference(instance, gerritInstance.TypeMeta, gerritInstance.ObjectMeta)

		if err := r.client.Update(ctx, instance); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "unable to update instance owner refs")
		}
	}

	gerritInstance, err := helper.GetInstanceOwner(ctx, r.client, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get instance owner")
	}

	if gerritInstance.Status.Status == gerrit.StatusReady && (instance.Status.Status == "" || instance.Status.Status == spec.StatusFailed) {
		log.Info(fmt.Sprintf("Replication configuration of %s/%s object with name has been started",
			gerritInstance.Namespace, gerritInstance.Name))
		log.Info(fmt.Sprintf("Configuration of %s/%s object with name has been started", instance.Namespace, instance.Name))
		err := r.updateStatus(ctx, instance, spec.StatusConfiguring)
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
	instance.Status.LastTimeUpdated = metav1.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err := r.client.Update(ctx, instance)
		if err != nil {
			return err
		}
	}

	r.log.V(1).Info(fmt.Sprintf("Status for Gerrit Replication Config %s has been updated to '%s' at %v.", instance.Name, status, instance.Status.LastTimeUpdated))
	return nil
}

func (r *ReconcileGerritReplicationConfig) configureReplication(config *gerritApi.GerritReplicationConfig, gerrit *gerritApi.Gerrit) error {
	GerritTemplatesPath := platformHelper.LocalTemplatesRelativePath
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return err
	}

	if helper.RunningInCluster() {
		GerritTemplatesPath = fmt.Sprintf("%s/../%s/%s", executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultTemplatesDirectory)
	}

	podList, err := r.platform.GetPods(gerrit.Namespace, metav1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%s", gerrit.Name)})
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
	if err != nil {
		return err
	}

	err = r.createReplicationConfig(gerrit.Namespace, podList.Items[0].Name)
	if err != nil {
		return err
	}

	err = r.updateReplicationConfig(gerrit.Namespace, podList.Items[0].Name, *config, GerritTemplatesPath)
	if err != nil {
		return err
	}

	err = r.updateSshConfig(gerrit.Namespace, podList.Items[0].Name, *config, GerritTemplatesPath,
		filepath.Join(spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName))
	if err != nil {
		return err
	}

	err = r.reloadReplicationPlugin(&client)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createReplicationConfig(namespace, podName string) error {
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{bin, containerFlag,
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) createSshConfig(namespace, podName string) error {
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{bin, containerFlag,
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+config, spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+config, spec.DefaultGerritSSHConfigPath+config)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) saveSshReplicationKey(namespace, podName string, key string) error {
	path := filepath.Join(spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	_, _, err := r.platform.ExecInPod(namespace, podName, []string{bin, containerFlag,
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", key, path, path)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateReplicationConfig(namespace, podName string,
	grc gerritApi.GerritReplicationConfig, templatePath string) error {
	config, err := resolveReplicationTemplate(grc, templatePath, "replication-conf.tmpl")
	if err != nil {
		return err
	}

	_, _, err = r.platform.ExecInPod(namespace, podName, []string{bin, containerFlag,
		fmt.Sprintf("echo \"%v\" >> %v", config.String(), spec.DefaultGerritReplicationConfigPath)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) updateSshConfig(namespace, podName string,
	grc gerritApi.GerritReplicationConfig, templatePath string, keyPath string) error {
	err := r.createSshConfig(namespace, podName)
	if err != nil {
		return err
	}

	sshTemplate, err := resolveSshTemplate(grc, templatePath, "ssh-sshTemplate.tmpl", keyPath)
	if err != nil {
		return err
	}

	_, _, err = r.platform.ExecInPod(namespace, podName, []string{bin, containerFlag,
		fmt.Sprintf("echo \"%s\" >> %s", sshTemplate.String(), spec.DefaultGerritSSHConfigPath+config)})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileGerritReplicationConfig) reloadReplicationPlugin(client gerritClient.ClientInterface) error {
	err := client.ReloadPlugin("replication")
	if err != nil {
		return err
	}

	return nil
}

func resolveReplicationTemplate(grc gerritApi.GerritReplicationConfig, path, templateName string) (*bytes.Buffer, error) {
	var config bytes.Buffer

	tmpl, err := template.New(templateName).ParseFiles(filepath.FromSlash(filepath.Join(path, templateName)))
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(&config, grc)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func resolveSshTemplate(grc gerritApi.GerritReplicationConfig, path, templateName, keyPath string) (*bytes.Buffer, error) {
	var config bytes.Buffer
	re := regexp.MustCompile(`\@([^\[\]]*)\:`)
	host := re.FindStringSubmatch(grc.Spec.SSHUrl)

	data := struct {
		Hostname string
		KeyPath  string
	}{host[1], keyPath}

	tmpl, err := template.New(templateName).ParseFiles(filepath.FromSlash(filepath.Join(path, templateName)))
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(&config, data)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (r ReconcileGerritReplicationConfig) updateAvailableStatus(ctx context.Context, instance *gerritApi.GerritReplicationConfig, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = metav1.Now()
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
