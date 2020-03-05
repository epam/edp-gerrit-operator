package openshift

import (
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	platformHelper "github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/helper"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/k8s"
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	securityV1Api "github.com/openshift/api/security/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

// GetRoute returns Route object from Openshift
func (service OpenshiftService) GetRoute(namespace string, name string) (*routeV1Api.Route, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Route %v in namespace %v not found", name, namespace)
		return nil, "", nil
	} else if err != nil {
		return nil, "", err
	}

	var routeScheme = "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}
	return route, routeScheme, nil
}

// CreateExternalEndpoint creates a new Endpoint resource for a Gerrit EDP Component
func (s *OpenshiftService) CreateExternalEndpoint(gerrit *v1alpha1.Gerrit) error {
	gerritRouteObject := newGerritRoute(gerrit.Name, gerrit.Namespace)

	if err := controllerutil.SetControllerReference(gerrit, gerritRouteObject, s.Scheme); err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	if _, err := s.routeClient.Routes(gerritRouteObject.Namespace).Get(gerritRouteObject.Name, metav1.GetOptions{}); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Printf("Creating a new Route %s/%s for Gerrit %s", gerritRouteObject.Namespace, gerritRouteObject.Name, gerrit.Name)
			if _, err = s.routeClient.Routes(gerritRouteObject.Namespace).Create(gerritRouteObject); err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("Route %s/%s has been created", gerritRouteObject.Namespace, gerritRouteObject.Name)
		} else {
			return helpers.LogErrorAndReturn(err)
		}
	}
	return nil
}

// CreateSecurityContext creates a new SecurityContextConstraints resource for a Gerrit EDP Component
func (s *OpenshiftService) CreateSecurityContext(gerrit *v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error {
	project, err := s.projectClient.Projects().Get(gerrit.Namespace, metav1.GetOptions{})
	if err != nil {
		return helpers.LogErrorAndReturn(errors.Wrapf(err, "Unable to retrieve project %s", gerrit.Namespace))
	}

	gerritSccObject, err := s.newGerritSecurityContextConstraints(gerrit, project.Name)
	if err != nil {
		return err
	}

	currentSCC, err := s.securityClient.SecurityContextConstraints().Get(gerritSccObject.Name, metav1.GetOptions{})
	if err == nil {
		// Object successfuly retrived

		if !reflect.DeepEqual(currentSCC.Users, gerritSccObject.Users) {
			// .Users slices are not equal

			if _, err := s.securityClient.SecurityContextConstraints().Update(gerritSccObject); err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("Security Context Constraint %s has been updated", gerritSccObject.Name)
		}
	} else if k8serrors.IsNotFound(err) {
		// Object hasn't been found

		log.Printf("Creating a new Security Context Constraint %s for Gerrit %s", gerritSccObject.Name, gerrit.Name)
		if _, err = s.securityClient.SecurityContextConstraints().Create(gerritSccObject); err != nil {
			return helpers.LogErrorAndReturn(err)
		}
		log.Printf("Security Context Constraint %s has been created", gerritSccObject.Name)
	} else {
		// Some other issue has occured during object retrival
		return helpers.LogErrorAndReturn(err)
	}

	// Success: SecurityContextConstraints resource already exists or the new one has been successfuly created
	return nil
}

// CreateDeployConf creates a new DeploymentConfig resource for a Gerrit EDP Component
func (s *OpenshiftService) CreateDeployConf(gerrit *v1alpha1.Gerrit) error {
	gerritRoute, protocol, err := s.GetRoute(gerrit.Namespace, gerrit.Name)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	externalUrl := fmt.Sprintf("%s://%s", protocol, gerritRoute.Spec.Host)

	gerritDCObject := newGerritDeploymentConfig(gerrit, externalUrl)

	if err := controllerutil.SetControllerReference(gerrit, gerritDCObject, s.Scheme); err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	if _, err := s.appClient.DeploymentConfigs(gerritDCObject.Namespace).Get(gerritDCObject.Name, metav1.GetOptions{}); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Printf("Creating a new DeploymentConfig %s/%s for Gerrit %s", gerritDCObject.Namespace, gerritDCObject.Name, gerrit.Name)
			_, err = s.appClient.DeploymentConfigs(gerritDCObject.Namespace).Create(gerritDCObject)
			if err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("DeploymentConfig %s/%s has been created", gerritDCObject.Namespace, gerritDCObject.Name)
		} else {
			return helpers.LogErrorAndReturn(err)
		}
	}
	return nil
}

// GetDeploymentConfig returns DeploymentConfig object from Openshift
func (service OpenshiftService) GetDeploymentConfig(instance v1alpha1.Gerrit) (*appsV1Api.DeploymentConfig, error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return nil, helpers.LogErrorAndReturn(err)
	}

	return deploymentConfig, nil
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

func (s *OpenshiftService) PatchDeployConfEnv(gerrit v1alpha1.Gerrit, dc *appsV1Api.DeploymentConfig, env []coreV1Api.EnvVar) error {

	if len(env) == 0 {
		return nil
	}

	container, err := selectContainer(dc.Spec.Template.Spec.Containers, gerrit.Name)
	if err != nil {
		return err
	}

	container.Env = updateEnv(container.Env, env)

	dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	_, err = s.appClient.DeploymentConfigs(dc.Namespace).Patch(dc.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
	}
	return nil
}

func newGerritRoute(name, namespace string) *routeV1Api.Route {
	return &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    platformHelper.GenerateLabels(name),
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: name,
				Kind: "Service",
			},
			Port: &routeV1Api.RoutePort{
				TargetPort: intstr.IntOrString{IntVal: 8080, StrVal: "ui"},
			},
		},
	}
}

func newGerritDeploymentConfig(gerrit *v1alpha1.Gerrit, externalUrl string) *appsV1Api.DeploymentConfig {
	labels := platformHelper.GenerateLabels(gerrit.Name)
	return &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerrit.Name,
			Namespace: gerrit.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRecreate,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					ImagePullSecrets: gerrit.Spec.ImagePullSecrets,
					Containers: []coreV1Api.Container{
						{
							Name:            gerrit.Name,
							Image:           fmt.Sprintf("%s:%s", gerrit.Spec.Image, gerrit.Spec.Version),
							ImagePullPolicy: coreV1Api.PullIfNotPresent,
							Env: []coreV1Api.EnvVar{
								{
									Name:  "HTTPD_LISTENURL",
									Value: "proxy-https://*:8080",
								},
								{
									Name:  "WEBURL",
									Value: externalUrl,
								},
								{
									Name: "GERRIT_INIT_ARGS",
									Value: "--install-plugin=commit-message-length-validator " +
										"--install-plugin=download-commands --install-plugin=hooks " +
										"--install-plugin=reviewnotes --install-plugin=singleusergroup " +
										"--install-plugin=replication",
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									Name:          gerrit.Name,
									ContainerPort: spec.Port,
								},
							},
							LivenessProbe:          generateProbe(spec.LivenessProbeDelay),
							ReadinessProbe:         generateProbe(spec.ReadinessProbeDelay),
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(spec.MemoryRequest),
								},
							},
							VolumeMounts: []coreV1Api.VolumeMount{
								{
									MountPath: "/var/gerrit/review_site",
									Name:      "gerrit-data",
								},
							},
						},
					},
					ServiceAccountName: gerrit.Name,
					Volumes: []coreV1Api.Volume{
						{
							Name: "gerrit-data",
							VolumeSource: coreV1Api.VolumeSource{
								PersistentVolumeClaim: &coreV1Api.PersistentVolumeClaimVolumeSource{
									ClaimName: gerrit.Name + "-data",
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateProbe(delay int32) *coreV1Api.Probe {
	return &coreV1Api.Probe{
		FailureThreshold:    5,
		InitialDelaySeconds: delay,
		PeriodSeconds:       20,
		SuccessThreshold:    1,
		Handler: coreV1Api.Handler{
			HTTPGet: &coreV1Api.HTTPGetAction{
				Port: intstr.IntOrString{
					IntVal: spec.Port,
				},
				Path: "/",
			},
		},
		TimeoutSeconds: 5,
	}
}

func selectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}

	return coreV1Api.Container{}, errors.New("No matching container in spec found!")
}

func updateEnv(existing []coreV1Api.EnvVar, env []coreV1Api.EnvVar) []coreV1Api.EnvVar {
	var out []coreV1Api.EnvVar
	var covered []string

	for _, e := range existing {
		newer, ok := findEnv(env, e.Name)
		if ok {
			covered = append(covered, e.Name)
			out = append(out, newer)
			continue
		}
		out = append(out, e)
	}
	for _, e := range env {
		if helpers.IsStringInSlice(e.Name, covered) {
			continue
		}
		covered = append(covered, e.Name)
		out = append(out, e)
	}
	return out
}

func findEnv(env []coreV1Api.EnvVar, name string) (coreV1Api.EnvVar, bool) {
	for _, e := range env {
		if e.Name == name {
			return e, true
		}
	}
	return coreV1Api.EnvVar{}, false
}
