package platform

import (
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/k8s"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/openshift"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

// PlatformService defines common behaviour of the services for the supported platforms
type PlatformService interface {
	GetPods(namespace string, filter metav1.ListOptions) (*coreV1Api.PodList, error)
	GetExternalEndpoint(namespace string, name string) (string, string, error)
	ExecInPod(namespace string, podName string, command []string) (string, string, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateExternalEndpoint(gerrit *v1alpha1.Gerrit) error
	CreateSecurityContext(gerrit *v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error
	CreateService(gerrit *v1alpha1.Gerrit) error
	CreateSecret(gerrit *v1alpha1.Gerrit, name string, data map[string][]byte) error
	CreateVolume(gerrit *v1alpha1.Gerrit) error
	CreateServiceAccount(gerrit *v1alpha1.Gerrit) (*coreV1Api.ServiceAccount, error)
	GetSecret(namespace string, name string) (map[string][]byte, error)
	CreateDeployment(gerrit *v1alpha1.Gerrit) error
	IsDeploymentReady(instance *v1alpha1.Gerrit) (bool, error)
	PatchDeploymentEnv(gerrit v1alpha1.Gerrit, env []coreV1Api.EnvVar) error
	GetDeploymentSSHPort(gerrit *v1alpha1.Gerrit) (int32, error)
	GetService(namespace string, name string) (*coreV1Api.Service, error)
	UpdateService(svc coreV1Api.Service, port int32) error
	GenerateKeycloakSettings(instance *v1alpha1.Gerrit) (*[]coreV1Api.EnvVar, error)
	CreateJenkinsServiceAccount(namespace string, secretName string, serviceAccountType string) error
	CreateJenkinsScript(namespace string, configMap string) error
	CreateConfigMap(instance *v1alpha1.Gerrit, configMapName string, configMapData map[string]string) error
	CreateEDPComponentIfNotExist(gerrit v1alpha1.Gerrit, url string, icon string) error
}

// NewService creates a new instance of the platform.Service type using scheme parameter provided
func NewService(platformType string, scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		helpers.LogErrorAndReturn(err)
		return nil, errors.Wrap(err, "Failed to get rest configs for platform")
	}

	switch strings.ToLower(platformType) {
	case "openshift":
		platform := &openshift.OpenshiftService{}
		if err = platform.Init(restConfig, scheme); err != nil {
			helpers.LogErrorAndReturn(err)
			return nil, errors.Wrap(err, "Failed to init for Openshift platform")
		}
		return platform, nil
	case "kubernetes":
		platform := &k8s.K8SService{}
		err := platform.Init(restConfig, scheme)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Kubernetes platform")
		}
		return platform, nil

	default:
		return nil, errors.Wrap(err, "Unknown platform type")
	}
}
