package service

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	appsV1Api "github.com/openshift/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type PlatformService interface {
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

func logErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func generateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}
