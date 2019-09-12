package platform

import (
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/openshift"
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

// PlatformService defines common behaviour of the services for the supported platforms
type PlatformService interface {
	GetPods(namespace string, filter metav1.ListOptions) (*coreV1Api.PodList, error)
	GetRoute(namespace string, name string) (*routeV1Api.Route, string, error)
	ExecInPod(namespace string, podName string, command []string) (string, string, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateExternalEndpoint(gerrit *v1alpha1.Gerrit) error
	CreateSecurityContext(gerrit *v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error
	CreateService(gerrit *v1alpha1.Gerrit) error
	CreateSecret(gerrit *v1alpha1.Gerrit, name string, data map[string][]byte) error
	CreateVolume(gerrit *v1alpha1.Gerrit) error
	CreateServiceAccount(gerrit *v1alpha1.Gerrit) (*coreV1Api.ServiceAccount, error)
	GetSecret(namespace string, name string) (map[string][]byte, error)
	CreateDeployConf(gerrit *v1alpha1.Gerrit) error
	GetDeploymentConfig(instance v1alpha1.Gerrit) (*appsV1Api.DeploymentConfig, error)
	GetService(namespace string, name string) (*coreV1Api.Service, error)
	PatchDeployConfEnv(gerrit v1alpha1.Gerrit, dc *appsV1Api.DeploymentConfig, env []coreV1Api.EnvVar) error
	UpdateService(svc coreV1Api.Service, port int32) error
	GenerateKeycloakSettings(instance *v1alpha1.Gerrit) []coreV1Api.EnvVar
	CreateConfigMapFromData(instance *v1alpha1.Gerrit, configMapName string, configMapData map[string]string, labels map[string]string, ownerReference metav1.Object) error
}

// NewService creates a new instance of the platform.Service type using scheme parameter provided
func NewService(scheme *runtime.Scheme) PlatformService {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		helpers.LogErrorAndReturn(err)
		return nil
	}

	platform := &openshift.OpenshiftService{}
	if err = platform.Init(restConfig, scheme); err != nil {
		helpers.LogErrorAndReturn(err)
		return nil
	}
	return platform
}
