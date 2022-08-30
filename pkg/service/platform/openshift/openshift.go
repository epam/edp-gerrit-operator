package openshift

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"

	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/k8s"
)

// OpenshiftService implements platform.Service interface (OpenShift platform integration).
type OpenshiftService struct {
	k8s.K8SService

	authClient     *authV1Client.AuthorizationV1Client
	templateClient *templateV1Client.TemplateV1Client
	projectClient  *projectV1Client.ProjectV1Client
	securityClient *securityV1Client.SecurityV1Client
	appClient      *appsV1client.AppsV1Client
	routeClient    *routeV1Client.RouteV1Client
}

const (
	deploymentTypeEnvName           = "DEPLOYMENT_TYPE"
	deploymentConfigsDeploymentType = "deploymentConfigs"
)

// Init process with OpenshiftService instance initialization actions.
func (s *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	if err := s.K8SService.Init(config, scheme); err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}
	s.templateClient = templateClient

	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}
	s.projectClient = projectClient

	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	s.securityClient = securityClient
	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	s.appClient = appClient
	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}
	s.routeClient = routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}
	s.authClient = authClient

	return nil
}

func (s OpenshiftService) GetExternalEndpoint(namespace string, name string) (string, string, error) {
	route, err := s.routeClient.Routes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Route %v in namespace %v not found", name, namespace)
		return "", "", err
	} else if err != nil {
		return "", "", err
	}

	var routeScheme = platformHelper.RouteHTTPScheme
	if route.Spec.TLS.Termination != "" {
		routeScheme = platformHelper.RouteHTTPSScheme
	}
	return route.Spec.Host, routeScheme, nil
}

func (s OpenshiftService) GetDeploymentSSHPort(instance *gerritApi.Gerrit) (int32, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}

		for _, env := range dc.Spec.Template.Spec.Containers[0].Env {
			if env.Name == spec.SSHListnerEnvName {
				p, err := getPort(env.Value)
				if err != nil {
					return 0, err
				}
				return p, nil
			}
		}

		return 0, nil
	}
	return s.K8SService.GetDeploymentSSHPort(instance)
}

func (s OpenshiftService) IsDeploymentReady(instance *gerritApi.Gerrit) (bool, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1, nil
	}
	return s.K8SService.IsDeploymentReady(instance)
}

func (s *OpenshiftService) PatchDeploymentEnv(gerrit gerritApi.Gerrit, env []coreV1Api.EnvVar) error {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(gerrit.Namespace).Get(context.TODO(), gerrit.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(env) == 0 {
			return nil
		}

		container, err := platformHelper.SelectContainer(dc.Spec.Template.Spec.Containers, gerrit.Name)
		if err != nil {
			return err
		}

		container.Env = platformHelper.UpdateEnv(container.Env, env)

		dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

		jsonDc, err := json.Marshal(dc)
		if err != nil {
			return err
		}

		_, err = s.appClient.DeploymentConfigs(dc.Namespace).Patch(context.TODO(), dc.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
	}
	return s.K8SService.PatchDeploymentEnv(gerrit, env)
}

func getPort(value string) (int32, error) {
	re := regexp.MustCompile(`[0-9]+`)
	if re.MatchString(value) {
		ports := re.FindStringSubmatch(value)
		if len(ports) != 1 {
			return 0, nil
		}
		portNumber, err := strconv.ParseInt(ports[0], k8s.Base, k8s.BitSize)
		if err != nil {
			return 0, err
		}
		return int32(portNumber), nil
	}

	return 0, nil
}
