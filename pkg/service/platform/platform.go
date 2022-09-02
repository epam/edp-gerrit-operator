package platform

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/k8s"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/openshift"
)

const (
	Kubernetes = "kubernetes"
	OpenShift  = "openshift"
	Test       = "test"
)

// PlatformService defines common behaviour of the services for the supported platforms.
type PlatformService interface {
	GetPods(namespace string, filter *metav1.ListOptions) (*coreV1Api.PodList, error)
	GetExternalEndpoint(namespace string, name string) (string, string, error)
	ExecInPod(namespace string, podName string, command []string) (io.Reader, io.Reader, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateSecret(gerrit *gerritApi.Gerrit, name string, data map[string][]byte) error
	GetSecret(namespace string, name string) (map[string][]byte, error)
	IsDeploymentReady(instance *gerritApi.Gerrit) (bool, error)
	PatchDeploymentEnv(gerrit *gerritApi.Gerrit, env []coreV1Api.EnvVar) error
	GetDeploymentSSHPort(gerrit *gerritApi.Gerrit) (int32, error)
	GetService(namespace string, name string) (*coreV1Api.Service, error)
	UpdateService(svc *coreV1Api.Service, port int32) error
	GenerateKeycloakSettings(instance *gerritApi.Gerrit) ([]coreV1Api.EnvVar, error)
	CreateJenkinsServiceAccount(namespace string, secretName string, serviceAccountType string) error
	CreateJenkinsScript(namespace string, configMap string) error
	CreateConfigMap(instance *gerritApi.Gerrit, configMapName string, configMapData map[string]string) error
	CreateEDPComponentIfNotExist(gerrit *gerritApi.Gerrit, url string, icon string) error
}

// NewService creates a new instance of the platform.Service type using scheme parameter provided.
func NewService(platformType string, scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(helpers.LogErrorAndReturn(err), "Failed to get rest configs for platform")
	}

	platformType = strings.ToLower(platformType)

	switch platformType {
	case OpenShift:
		platform := &openshift.OpenshiftService{}
		if err = platform.Init(restConfig, scheme); err != nil {
			return nil, errors.Wrap(helpers.LogErrorAndReturn(err), "Failed to init for Openshift platform")
		}

		return platform, nil
	case Kubernetes:
		platform := &k8s.K8SService{}

		err = platform.Init(restConfig, scheme)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Kubernetes platform")
		}

		return platform, nil
	case Test:
		return nil, nil // for tests only. remove it when will fix platform.Init()
	default:
		return nil, fmt.Errorf("unknown platform type '%s'", platformType)
	}
}
