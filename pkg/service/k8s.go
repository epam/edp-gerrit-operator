package service

import (
	"gerrit-operator/pkg/client"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type K8SService struct {
	scheme     *runtime.Scheme
	coreClient coreV1Client.CoreV1Client
	edpClient  client.EdpV1Client
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	edpClient, err := client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}

	service.edpClient = *edpClient
	service.coreClient = *coreClient
	service.scheme = scheme
	return nil
}
