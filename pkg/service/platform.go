package service

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

type PlatformService interface {
	CreateExternalEndpoint(gerrit v1alpha1.Gerrit) error
	CreateSecurityContext(gerrit v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error
	CreateService(gerrit v1alpha1.Gerrit) error
	CreateSecret(gerrit v1alpha1.Gerrit, name string, data map[string][]byte) error
	CreateVolume(gerrit v1alpha1.Gerrit) error
	CreateServiceAccount(gerrit v1alpha1.Gerrit) (*coreV1Api.ServiceAccount, error)
	GetSecret(namespace string, name string) (map[string][]byte, error)
	CreateDeployConf(gerrit v1alpha1.Gerrit) error
}

func NewPlatformService(scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, logErrorAndReturn(err)
	}

	platform := OpenshiftService{}

	err = platform.Init(restConfig, scheme)
	if err != nil {
		return nil, logErrorAndReturn(err)
	}
	return platform, nil
}
