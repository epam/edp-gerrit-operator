package gerrit

import (
	"fmt"
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gerrit-operator/pkg/client/gerrit"
	"gerrit-operator/pkg/helper"
	"gerrit-operator/pkg/service/gerrit/spec"
	"gerrit-operator/pkg/service/helpers"
	"gerrit-operator/pkg/service/platform"
	"github.com/dchest/uniuri"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("service_gerrit")

// Interface expresses behaviour of the Gerrit EDP Component
type Interface interface {
	IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error)
	Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
}

// ComponentService implements gerrit.Interface
type ComponentService struct {
	// Providing Gerrit EDP component implementation through the interface (platform abstract)
	PlatformService platform.PlatformService
	k8sClient       client.Client
	gerritClient    gerrit.Client
}

// NewComponentService returns a new instance of a gerrit.Service type
func NewComponentService(ps platform.PlatformService, kc client.Client) Interface {
	return ComponentService{PlatformService: ps, k8sClient: kc}
}

// IsDeploymentConfigReady check if DC for Gerrit is ready
func (s ComponentService) IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error) {
	gerritIsReady := false
	GerritDc, err := s.PlatformService.GetDeploymentConfig(instance)
	if err != nil {
		return gerritIsReady, helpers.LogErrorAndReturn(err)
	}
	if GerritDc.Status.AvailableReplicas == 1 {
		gerritIsReady = true
	}
	return gerritIsReady, nil
}

// Install has a minimal set of logic, required to install "vanilla" Gerrit EDP Component
func (s ComponentService) Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	sa, err := s.PlatformService.CreateServiceAccount(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Service Account for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateSecurityContext(instance, sa); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Security Context for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateService(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Service for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateExternalEndpoint(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create External Endpoint for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateVolume(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Volume for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateDeployConf(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Deploy Config for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateSecret(instance, instance.Name+"-admin-password", map[string][]byte{
		"user":     []byte(spec.GerritDefaultAdminUser),
		"password": []byte(uniuri.New()),
	}); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create admin Secret %v for Gerrit", instance.Name+"-admin-password")
	}

	return instance, nil
}

// Configure contains logic related to self configuration of the Gerrit EDP Component
func (s ComponentService) Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Unable to get executable file path")
	}

	GerritScriptsPath := spec.GerritDefaultScriptsPath
	if _, err = k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		GerritScriptsPath = fmt.Sprintf("%v/../%v/scripts", executableFilePath, spec.LocalConfigsRelativePath)
	}

	gerritApiUrl, err := s.getGerritRestApiUrl(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to get Gerrit REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to get Gerrit admin password from secret for %v/%v", instance.Namespace, instance.Name)
	}

	podList, err := s.PlatformService.GetPods(instance.Namespace, metav1.ListOptions{LabelSelector: "deploymentconfig=" + instance.Name})
	if err != nil || len(podList.Items) != 1 {
		return instance, errors.Wrapf(err, "[ERROR] Unable to determine Gerrit pod name: %v", len(podList.Items))
	}

	_, gerritAdminPublicKey, err := s.createSSHKeyPairs(instance, instance.Name+"-admin")
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Gerrit admin SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	_, _, err = s.createSSHKeyPairs(instance, instance.Name+"-project-creator")
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Gerrit project-creator SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	status, err := s.gerritClient.CheckCredentials()
	if status == 401 {
		err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, spec.GerritDefaultAdminPassword)
		if err != nil {
			return instance, errors.Wrapf(err, "[ERROR] Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
		}

		status, err = s.gerritClient.CheckCredentials()
		if status == 401 {
			instance, err := s.gerritClient.InitAdminUser(*instance, s.PlatformService, GerritScriptsPath, podList.Items[0].Name,
				string(gerritAdminPublicKey))
			if err != nil {
				return &instance, errors.Wrapf(err, "[ERROR] Failed to initialize Gerrit Admin User")
			}
		}
	}

	_, _, err = s.PlatformService.ExecInPod(instance.Namespace, podList.Items[0].Name,
		[]string{"/bin/sh", "-c", "chown -R gerrit2:gerrit2 /var/gerrit/review_site"})
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to chown /var/gerrit/review_site folder")
	}

	return instance, nil
}

// ExposeConfiguration describes integration points of the Gerrit EDP Component for the other Operators and Components
func (s ComponentService) ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return instance, nil
}

// Integrate applies actions required for the integration with the other EDP Components
func (s ComponentService) Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return instance, nil
}

func (s ComponentService) getGerritRestApiUrl(instance *v1alpha1.Gerrit) (string, error) {
	gerritApiUrl := fmt.Sprintf("http://%v.%v:%v/%v", instance.Name, instance.Namespace, spec.GerritPort, spec.GerritRestApiUrlPath)
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		gerritRoute, gerritRouteScheme, err := s.PlatformService.GetRoute(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "[ERROR] Failed to get Route for %v/%v", instance.Namespace, instance.Name)
		}
		gerritApiUrl = fmt.Sprintf("%v://%v/%v", gerritRouteScheme, gerritRoute.Spec.Host, spec.GerritRestApiUrlPath)
	}
	return gerritApiUrl, nil
}

func (s ComponentService) getGerritAdminPassword(instance *v1alpha1.Gerrit) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	gerritAdminCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "[ERROR] Failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}
	return string(gerritAdminCredentials["password"]), nil
}

func (s ComponentService) createSSHKeyPairs(instance *v1alpha1.Gerrit, secretName string) ([]byte, []byte, error) {
	privateKey, publicKey, err := helpers.GenerateKeyPairs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "[ERROR] Unable to generate SSH key pairs for Gerrit")
	}

	if err := s.PlatformService.CreateSecret(instance, secretName, map[string][]byte{
		"id_rsa":     privateKey,
		"id_rsa.pub": publicKey,
	}); err != nil {
		return nil, nil, errors.Wrapf(err, "[ERROR] Failed to create Secret with SSH key pairs for Gerrit")
	}

	return privateKey, publicKey, nil
}
