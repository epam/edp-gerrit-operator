package openshift

import (
	"context"
	"encoding/json"
	"fmt"
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
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
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
		helpers.LogError(err)
		return fmt.Errorf("failed to init k8s client: %w", err)
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create TemplateV1Client: %w", err)
	}

	s.templateClient = templateClient

	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create ProjectV1Client: %w", err)
	}

	s.projectClient = projectClient

	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create SecurityV1Client: %w", err)
	}

	s.securityClient = securityClient

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create AppsV1Client: %w", err)
	}

	s.appClient = appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create RouteV1Client: %w", err)
	}

	s.routeClient = routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		helpers.LogError(err)
		return fmt.Errorf("failed to create AuthorizationV1Client: %w", err)
	}

	s.authClient = authClient

	return nil
}

func (s *OpenshiftService) GetExternalEndpoint(namespace, name string) (host, scheme string, err error) {
	ctx := context.Background()

	route, err := s.routeClient.Routes(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		log.Printf("Route %v in namespace %v not found", name, namespace)
		return "", "", fmt.Errorf("didn't found route %q in namespace %q: %w", name, namespace, err)
	} else if err != nil {
		return "", "", fmt.Errorf("failed to Get OpenShift route %q: %w", name, err)
	}

	host = route.Spec.Host
	scheme = platformHelper.RouteHTTPScheme

	if route.Spec.TLS.Termination != "" {
		scheme = platformHelper.RouteHTTPSScheme
	}

	return
}

func (s *OpenshiftService) GetDeploymentSSHPort(instance *gerritApi.Gerrit) (int32, error) {
	ctx := context.Background()

	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(instance.Namespace).Get(ctx, instance.Name, metaV1.GetOptions{})
		if err != nil {
			return 0, fmt.Errorf("failed to GET OpenShift Deployment %q: %w", instance.Name, err)
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

	p, err := s.K8SService.GetDeploymentSSHPort(instance)
	if err != nil {
		return 0, fmt.Errorf("failed to Get gerrit ssh port in k8s deployment: %w", err)
	}

	return p, nil
}

func (s *OpenshiftService) IsDeploymentReady(instance *gerritApi.Gerrit) (bool, error) {
	ctx := context.Background()

	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(instance.Namespace).Get(ctx, instance.Name, metaV1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to Get Gerrit Deployment Config %q: %w", instance.Name, err)
		}

		return dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1, nil
	}

	ready, err := s.K8SService.IsDeploymentReady(instance)
	if err != nil {
		return false, fmt.Errorf("failed to check  if deployment %q is ready: %w", instance.Name, err)
	}

	return ready, nil
}

func (s *OpenshiftService) PatchDeploymentEnv(gerrit *gerritApi.Gerrit, env []coreV1Api.EnvVar) error {
	ctx := context.Background()

	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := s.appClient.DeploymentConfigs(gerrit.Namespace).Get(ctx, gerrit.Name, metaV1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to Get Gerrit Deployment Config %q: %w", gerrit.Name, err)
		}

		if len(env) == 0 {
			return nil
		}

		container, err := platformHelper.SelectContainer(dc.Spec.Template.Spec.Containers, gerrit.Name)
		if err != nil {
			return fmt.Errorf("didn't found container %q: %w", gerrit.Name, err)
		}

		container.Env = platformHelper.UpdateEnv(container.Env, env)

		dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

		jsonDc, err := json.Marshal(dc)
		if err != nil {
			return err
		}

		_, err = s.appClient.DeploymentConfigs(dc.Namespace).Patch(ctx, dc.Name, types.StrategicMergePatchType, jsonDc, metaV1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to patch OpenShift Deployment Config %q: %w", dc.Name, err)
		}

		return nil
	}

	err := s.K8SService.PatchDeploymentEnv(gerrit, env)
	if err != nil {
		return fmt.Errorf("fail to update k8s deployment env: %w", err)
	}

	return nil
}

func getPort(value string) (int32, error) {
	re := regexp.MustCompile(`\d+`)
	if re.MatchString(value) {
		ports := re.FindStringSubmatch(value)
		if len(ports) != 1 {
			return 0, nil
		}

		port := ports[0]

		portNumber, err := strconv.ParseInt(port, k8s.Base, k8s.BitSize)
		if err != nil {
			return 0, fmt.Errorf("failed to parse port value %q: %w", port, err)
		}

		return int32(portNumber), nil
	}

	return 0, nil
}
