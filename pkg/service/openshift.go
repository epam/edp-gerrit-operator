package service

import (
	"fmt"
	"gerrit-operator/pkg/apis/edp/v1alpha1"
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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (service OpenshiftService) CreateExternalEndpoint(gerrit v1alpha1.Gerrit) error {

	labels := generateLabels(gerrit.Name)

	gerritRouteObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerrit.Name,
			Namespace: gerrit.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: gerrit.Name,
				Kind: "Service",
			},
		},
	}

	if err := controllerutil.SetControllerReference(&gerrit, gerritRouteObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	gerritRoute, err := service.routeClient.Routes(gerritRouteObject.Namespace).Get(gerritRouteObject.Name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new Route %s/%s for Gerrit %s", gerritRouteObject.Namespace, gerritRouteObject.Name, gerrit.Name)
		gerritRoute, err = service.routeClient.Routes(gerritRouteObject.Namespace).Create(gerritRouteObject)

		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Route %s/%s has been created", gerritRoute.Namespace, gerritRoute.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service OpenshiftService) CreateSecurityContext(gerrit v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error {

	labels := generateLabels(gerrit.Name)
	priority := int32(1)

	project, err := service.projectClient.Projects().Get(gerrit.Namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return logErrorAndReturn(errors.New(fmt.Sprintf("Unable to retrieve project %s", gerrit.Namespace)))
	}

	displayName := project.GetObjectMeta().GetAnnotations()["openshift.io/display-name"]
	if displayName == "" {
		return logErrorAndReturn(errors.New(fmt.Sprintf("Project display name does not set")))
	}

	gerritSccObject := &securityV1Api.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", gerrit.Name, displayName),
			Namespace: gerrit.Namespace,
			Labels:    labels,
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

	if err := controllerutil.SetControllerReference(&gerrit, gerritSccObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	gerritSCC, err := service.securityClient.SecurityContextConstraints().Get(gerritSccObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new Security Context Constraint %s for Gerrit %s", gerritSccObject.Name, gerrit.Name)

		gerritSCC, err = service.securityClient.SecurityContextConstraints().Create(gerritSccObject)

		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Security Context Constraint %s has been created", gerritSCC.Name)
	} else if err != nil {
		return logErrorAndReturn(err)

	} else {
		// TODO(Serhii Shydlovskyi): Reflect reports that present users and currently stored in object are different for some reason.
		if !reflect.DeepEqual(gerritSCC.Users, gerritSccObject.Users) {

			gerritSCC, err = service.securityClient.SecurityContextConstraints().Update(gerritSccObject)

			if err != nil {
				return logErrorAndReturn(err)
			}

			log.Printf("Security Context Constraint %s has been updated", gerritSCC.Name)
		}
	}

	return nil
}

func (service OpenshiftService) CreateDeployConf(gerrit v1alpha1.Gerrit) error {
	labels := generateLabels(gerrit.Name)

	gerritDcObject := newGerritDeploymentConfig(gerrit.Name, gerrit.Namespace, gerrit.Spec.Version, labels)
	if err := controllerutil.SetControllerReference(&gerrit, gerritDcObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	gerritDc, err := service.appClient.DeploymentConfigs(gerritDcObject.Namespace).Get(gerritDcObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new DeploymentConfig %s/%s for Gerrit %s", gerritDcObject.Namespace, gerritDcObject.Name, gerrit.Name)

		gerritDc, err = service.appClient.DeploymentConfigs(gerritDcObject.Namespace).Create(gerritDcObject)
		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("DeploymentConfig %s/%s has been created", gerritDc.Namespace, gerritDc.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func newGerritDeploymentConfig(name string, namespace string, version string, labels map[string]string) *appsV1Api.DeploymentConfig {
	return &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
				Type: appsV1Api.DeploymentStrategyTypeRolling,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            name,
							Image:           Image + ":" + version,
							ImagePullPolicy: coreV1Api.PullIfNotPresent,
							Ports: []coreV1Api.ContainerPort{
								{
									Name:          name,
									ContainerPort: Port,
								},
							},
							LivenessProbe:          generateProbe(LivenessProbeDelay),
							ReadinessProbe:         generateProbe(ReadinessProbeDelay),
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(MemoryRequest),
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
					ServiceAccountName: name,
					Volumes: []coreV1Api.Volume{
						{
							Name: "gerrit-data",
							VolumeSource: coreV1Api.VolumeSource{
								PersistentVolumeClaim: &coreV1Api.PersistentVolumeClaimVolumeSource{
									ClaimName: name + "-data",
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
					IntVal: Port,
				},
				Path: "/",
			},
		},
		TimeoutSeconds: 5,
	}
}
