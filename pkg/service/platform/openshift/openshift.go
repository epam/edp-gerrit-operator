package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/k8s"
	securityV1Api "github.com/openshift/api/security/v1"
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
	"log"
	"os"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

// OpenshiftService implements platform.Service interface (OpenShift platform integration)
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

// Init process with OpenshiftService instance initialization actions
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

func (service OpenshiftService) GetExternalEndpoint(namespace string, name string) (string, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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

func (service OpenshiftService) GetDeploymentSSHPort(instance *v1alpha1.Gerrit) (int32, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
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
	return service.K8SService.GetDeploymentSSHPort(instance)
}

func (service OpenshiftService) IsDeploymentReady(instance *v1alpha1.Gerrit) (bool, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1, nil
	}
	return service.K8SService.IsDeploymentReady(instance)
}

// newGerritSCC returns a new instance of securityV1Api.SecurityContextConstraints type
func (s *OpenshiftService) newGerritSecurityContextConstraints(gerrit *v1alpha1.Gerrit, projectName string) (*securityV1Api.SecurityContextConstraints, error) {
	priority := int32(1)
	gerritSccObject := &securityV1Api.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", gerrit.Name, projectName),
			Namespace: gerrit.Namespace,
			Labels:    platformHelper.GenerateLabels(gerrit.Name),
		},
		Volumes: []securityV1Api.FSType{
			securityV1Api.FSTypeSecret,
			securityV1Api.FSTypeDownwardAPI,
			securityV1Api.FSTypeEmptyDir,
			securityV1Api.FSTypePersistentVolumeClaim,
			securityV1Api.FSProjected,
			securityV1Api.FSTypeConfigMap,
		},
		AllowHostDirVolumePlugin: false,
		AllowHostIPC:             true,
		AllowHostNetwork:         false,
		AllowHostPID:             false,
		AllowHostPorts:           false,
		AllowPrivilegedContainer: false,
		AllowedCapabilities:      []coreV1Api.Capability{},
		AllowedFlexVolumes:       []securityV1Api.AllowedFlexVolume{},
		DefaultAddCapabilities:   []coreV1Api.Capability{},
		FSGroup: securityV1Api.FSGroupStrategyOptions{
			Type:   securityV1Api.FSGroupStrategyRunAsAny,
			Ranges: []securityV1Api.IDRange{},
		},
		Groups:                 []string{},
		Priority:               &priority,
		ReadOnlyRootFilesystem: false,
		RunAsUser: securityV1Api.RunAsUserStrategyOptions{
			Type:        securityV1Api.RunAsUserStrategyRunAsAny,
			UID:         nil,
			UIDRangeMin: nil,
			UIDRangeMax: nil,
		},
		SELinuxContext: securityV1Api.SELinuxContextStrategyOptions{
			Type:           securityV1Api.SELinuxStrategyMustRunAs,
			SELinuxOptions: nil,
		},
		SupplementalGroups: securityV1Api.SupplementalGroupsStrategyOptions{
			Type:   securityV1Api.SupplementalGroupsStrategyRunAsAny,
			Ranges: nil,
		},
		Users: []string{
			"system:serviceaccount:" + gerrit.Namespace + ":" + gerrit.Name,
		},
	}

	if err := controllerutil.SetControllerReference(gerrit, gerritSccObject, s.Scheme); err != nil {
		return nil, helpers.LogErrorAndReturn(err)
	}

	return gerritSccObject, nil
}

func (s *OpenshiftService) PatchDeploymentEnv(gerrit v1alpha1.Gerrit, env []coreV1Api.EnvVar) error {
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
		portNumber, err := strconv.ParseInt(ports[0], 10, 32)
		if err != nil {
			return 0, err
		}
		return int32(portNumber), nil
	}

	return 0, nil
}
