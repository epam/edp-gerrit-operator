package service

import (
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

type OpenshiftService struct {
	K8SService

	authClient     authV1Client.AuthorizationV1Client
	templateClient templateV1Client.TemplateV1Client
	projectClient  projectV1Client.ProjectV1Client
	securityClient securityV1Client.SecurityV1Client
	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
}

func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {

	err := service.K8SService.Init(config, scheme)
	if err != nil {
		return logErrorAndReturn(err)
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.templateClient = *templateClient
	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.projectClient = *projectClient
	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.securityClient = *securityClient
	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.appClient = *appClient
	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}
	service.routeClient = *routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}
	service.authClient = *authClient

	return nil
}
