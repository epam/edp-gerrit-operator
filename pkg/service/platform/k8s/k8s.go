package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	edpCompApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	edpCompClient "github.com/epmd-edp/edp-component-operator/pkg/client"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/client"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	platformHelper "github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/helper"
	jenkinsV1Api "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsScriptV1Client "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsscript/client"
	JenkinsSAV1Client "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsserviceaccount/client"
	keycloakApi "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsV1Client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"regexp"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
)

var log = logf.Log.WithName("platform")

// K8SService implements platform.Service interface (k8s platform integration)
type K8SService struct {
	Scheme                      *runtime.Scheme
	CoreClient                  *coreV1Client.CoreV1Client
	appsV1Client                *appsV1Client.AppsV1Client
	EdpClient                   *client.EdpV1Client
	JenkinsServiceAccountClient *JenkinsSAV1Client.EdpV1Client
	JenkinsScriptClient         *jenkinsScriptV1Client.EdpV1Client
	extensionsV1Client          extensionsV1Client.ExtensionsV1beta1Client
	k8sClient                   k8sclient.Client
	edpCompClient               edpCompClient.EDPComponentV1Client
}

func (s *K8SService) GetExternalEndpoint(namespace string, name string) (string, string, error) {
	i, err := s.extensionsV1Client.Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", "", errors.New(fmt.Sprintf("Ingress %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return "", "", err
	}

	return i.Spec.Rules[0].Host, platformHelper.RouteHTTPSScheme, nil
}

func (s *K8SService) CreateExternalEndpoint(g *v1alpha1.Gerrit) error {
	//CreateExternalEndpoint - Obsolete interface method. To be removed when refactored
	return nil
}

func (s *K8SService) CreateSecurityContext(g *v1alpha1.Gerrit, sa *coreV1Api.ServiceAccount) error {
	//CreateSecurityContext - Obsolete interface method. To be removed when refactored
	return nil
}

func (s *K8SService) CreateDeployment(g *v1alpha1.Gerrit) error {
	//CreateDeployment - Obsolete interface method. To be removed when refactored
	return nil
}

func (s *K8SService) IsDeploymentReady(gerrit *v1alpha1.Gerrit) (bool, error) {
	deployment, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(gerrit.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (s *K8SService) PatchDeploymentEnv(gerrit v1alpha1.Gerrit, env []coreV1Api.EnvVar) error {
	d, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(gerrit.Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	if len(env) == 0 {
		return nil
	}

	container, err := platformHelper.SelectContainer(d.Spec.Template.Spec.Containers, gerrit.Name)
	if err != nil {
		return err
	}

	container.Env = platformHelper.UpdateEnv(container.Env, env)

	d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(d)
	if err != nil {
		return err
	}

	_, err = s.appsV1Client.Deployments(d.Namespace).Patch(d.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
	}
	return nil
}

// Init process with K8SService instance initialization actions
func (s *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	s.Scheme = scheme

	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	s.CoreClient = coreClient

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	s.appsV1Client = appsClient

	extensionsClient, err := extensionsV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init extensions V1 client for K8S")
	}
	s.extensionsV1Client = *extensionsClient

	edpClient, err := client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}
	s.EdpClient = edpClient

	jenkinsSAClient, err := JenkinsSAV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "Jenkins Service Account client initialization failed!")
	}
	s.JenkinsServiceAccountClient = jenkinsSAClient

	jenkinsScriptClient, err := jenkinsScriptV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	s.JenkinsScriptClient = jenkinsScriptClient

	cl, err := k8sclient.New(config, k8sclient.Options{
		Scheme: s.Scheme,
	})
	s.k8sClient = cl

	compCl, err := edpCompClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to init edp component client")
	}
	s.edpCompClient = *compCl

	return nil
}

func (service K8SService) GetDeploymentSSHPort(instance *v1alpha1.Gerrit) (int32, error) {
	d, err := service.appsV1Client.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	for _, env := range d.Spec.Template.Spec.Containers[0].Env {
		if env.Name == spec.SSHListnerEnvName {
			re := regexp.MustCompile(`[0-9]+`)
			if re.MatchString(env.Value) {
				ports := re.FindStringSubmatch(env.Value)
				if len(ports) != 1 {
					return 0, nil
				}
				portNumber, err := strconv.ParseInt(ports[0], 10, 32)
				if err != nil {
					return 0, err
				}
				return int32(portNumber), nil
			}
		}
	}

	return 0, nil
}

// GenerateKeycloakSettings generates a set of environment var
func (s *K8SService) GenerateKeycloakSettings(instance *v1alpha1.Gerrit) (*[]coreV1Api.EnvVar, error) {
	identityServiceSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix)
	realm, err := s.getKeycloakRealm(instance)
	if err != nil {
		return nil, err
	}

	keycloakUrl, err := s.getKeycloakRootUrl(instance)
	if err != nil {
		return nil, err
	}

	envVar := []coreV1Api.EnvVar{
		{
			Name:  "AUTH_TYPE",
			Value: "OAUTH",
		},
		{
			Name:  "OAUTH_KEYCLOAK_CLIENT_ID",
			Value: instance.Name,
		},
		{
			Name:  "OAUTH_KEYCLOAK_REALM",
			Value: realm.Spec.RealmName,
		},
		{
			Name:  "OAUTH_KEYCLOAK_ROOT_URL",
			Value: *keycloakUrl,
		},
		{
			Name: "OAUTH_KEYCLOAK_CLIENT_SECRET",
			ValueFrom: &coreV1Api.EnvVarSource{
				SecretKeyRef: &coreV1Api.SecretKeySelector{
					LocalObjectReference: coreV1Api.LocalObjectReference{
						Name: identityServiceSecretName,
					},
					Key: "client_secret",
				},
			},
		},
	}

	return &envVar, nil
}

func (s K8SService) getKeycloakRealm(instance *v1alpha1.Gerrit) (*keycloakApi.KeycloakRealm, error) {
	realm := &keycloakApi.KeycloakRealm{}
	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      "main",
		Namespace: instance.Namespace,
	}, realm)
	if err != nil {
		return nil, err
	}

	return realm, nil
}

func (s K8SService) getKeycloakRootUrl(instance *v1alpha1.Gerrit) (*string, error) {
	realm, err := s.getKeycloakRealm(instance)
	if err != nil {
		return nil, err
	}

	keycloak := &keycloakApi.Keycloak{}
	err = s.k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      realm.OwnerReferences[0].Name,
		Namespace: instance.Namespace,
	}, keycloak)
	if err != nil {
		return nil, err
	}

	keycloakUrl := keycloak.Spec.Url

	return &keycloakUrl, nil
}

// GetSecret return data field of Secret
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// GetPods returns Pod list according to the filter
func (s *K8SService) GetPods(namespace string, filter metav1.ListOptions) (*coreV1Api.PodList, error) {
	PodList, err := s.CoreClient.Pods(namespace).List(filter)
	if err != nil {
		return &coreV1Api.PodList{}, err
	}

	return PodList, nil
}

// ExecInPod executes command in pod
func (s *K8SService) ExecInPod(namespace string, podName string, command []string) (string, string, error) {
	pod, err := s.CoreClient.Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	req := s.CoreClient.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&coreV1Api.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	restConfig, err := newRestConfig()
	if err != nil {
		return "", "", err
	}

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

// CreateService creates a new Service Resource for a Gerrit EDP Component
func (s *K8SService) CreateService(gerrit *v1alpha1.Gerrit) error {
	for _, serviceName := range []string{gerrit.Name} {
		if err := s.createGerritService(serviceName, gerrit); err != nil {
			return err
		}
	}
	return nil
}

// CreateSecret creates a new Secret Resource for a Gerrit EDP Component
func (s *K8SService) CreateSecret(gerrit *v1alpha1.Gerrit, secretName string, data map[string][]byte) error {
	gerritSecretObject := newGerritSecret(secretName, gerrit.Name, gerrit.Namespace, data)

	if err := controllerutil.SetControllerReference(gerrit, gerritSecretObject, s.Scheme); err != nil {
		return err
	}

	if _, err := s.CoreClient.Secrets(gerritSecretObject.Namespace).Get(gerritSecretObject.Name, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Secret %s/%s for Gerrit", gerritSecretObject.Namespace, gerritSecretObject.Name)
			log.V(1).Info(msg)
			if _, err = s.CoreClient.Secrets(gerritSecretObject.Namespace).Create(gerritSecretObject); err != nil {
				return err
			}
			msg = fmt.Sprintf("Secret %s/%s has been created", gerritSecretObject.Namespace, gerritSecretObject.Name)
			log.Info(msg)
		} else {
			return err
		}
	}
	return nil
}

// CreateVolume creates a new PersitentVolume resource for a Gerrit EDP Component
func (s *K8SService) CreateVolume(gerrit *v1alpha1.Gerrit) error {
	for _, volume := range gerrit.Spec.Volumes {
		if err := s.createGerritPersistentVolume(gerrit, volume); err != nil {
			return err
		}
	}
	return nil
}

// CreateServiceAccount creates a new ServiceAcount resource for a Gerrit EDP Component
func (s *K8SService) CreateServiceAccount(gerrit *v1alpha1.Gerrit) (account *coreV1Api.ServiceAccount, e error) {
	gerritSAObject := newGerritServiceAccount(gerrit.Name, gerrit.Namespace)

	if err := controllerutil.SetControllerReference(gerrit, gerritSAObject, s.Scheme); err != nil {
		return nil, err
	}

	if account, e = s.CoreClient.ServiceAccounts(gerritSAObject.Namespace).Get(gerritSAObject.Name, metav1.GetOptions{}); e != nil {
		if k8serr.IsNotFound(e) {
			msg := fmt.Sprintf("Creating a new ServiceAccount %s for Gerrit %s/%s", gerritSAObject.Name, gerrit.Name, gerrit.Name)
			log.V(1).Info(msg)
			if account, e = s.CoreClient.ServiceAccounts(gerritSAObject.Namespace).Create(gerritSAObject); e != nil {
				return nil, helpers.LogErrorAndReturn(e)
			}
			msg = fmt.Sprintf("ServiceAccount %s/%s has been created", gerritSAObject.Namespace, gerritSAObject.Name)
			log.Info(msg)
		} else if e != nil {
			return nil, helpers.LogErrorAndReturn(e)
		}
	}
	return account, nil
}

// GetSecret returns data section of an existing Secret resource of a Gerrit EDP Component
func (s *K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	secret, err := s.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
			return nil, nil
		}
		return nil, err
	}
	return secret.Data, nil
}

// GetService returns existing Service resource of a Gerrit EDP Component
func (s *K8SService) GetService(namespace string, name string) (*coreV1Api.Service, error) {
	service, err := s.CoreClient.Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info("Service %v in namespace %v not found", name, namespace)
			return nil, nil
		}
		return nil, err
	}
	return service, nil
}

// UpdateService updates target port of a Gerrit EDP Component
func (s *K8SService) UpdateService(svc coreV1Api.Service, nodePort int32) error {
	ports := svc.Spec.Ports
	updatedPorts, err := updatePort(ports, "ssh", nodePort)
	svc.Spec.Ports = updatedPorts

	_, err = s.CoreClient.Services(svc.Namespace).Update(&svc)
	if err != nil {
		return err
	}

	return nil
}

func (s *K8SService) createGerritPersistentVolume(gerrit *v1alpha1.Gerrit, gerritVolume v1alpha1.GerritVolumes) error {
	gerritVolumeObject := newGerritPersistentVolumeClaim(gerritVolume, gerrit.Name, gerrit.Namespace)

	if err := controllerutil.SetControllerReference(gerrit, gerritVolumeObject, s.Scheme); err != nil {
		return err
	}

	if _, err := s.CoreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Get(gerritVolumeObject.Name, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new PersistantVolumeClaim %s for %s/%s", gerritVolumeObject.Name, gerrit.Namespace, gerrit.Name)
			log.V(1).Info(msg)
			if _, err = s.CoreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Create(gerritVolumeObject); err != nil {
				return err
			}
			msg = fmt.Sprintf("PersistantVolumeClaim %s/%s has been created", gerritVolumeObject.Namespace, gerritVolumeObject.Name)
			log.Info(msg)
		} else {
			return err
		}
	}
	return nil
}

func (s *K8SService) createGerritService(serviceName string, gerrit *v1alpha1.Gerrit) error {
	portMap := newGerritPortMap(*gerrit)
	gerritServiceObject := newGerritInternalBalancingService(serviceName, gerrit.Namespace, portMap[serviceName])

	if err := controllerutil.SetControllerReference(gerrit, gerritServiceObject, s.Scheme); err != nil {
		return err
	}

	if _, err := s.CoreClient.Services(gerrit.Namespace).Get(serviceName, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new service %s for gerrit %s/%s", gerritServiceObject.Name, gerrit.Namespace, gerrit.Name)
			log.V(1).Info(msg)
			if _, err = s.CoreClient.Services(gerritServiceObject.Namespace).Create(gerritServiceObject); err != nil {
				return err
			}
			msg = fmt.Sprintf("Service %s/%s has been created", gerritServiceObject.Namespace, gerritServiceObject.Name)
			log.Info(msg)
		} else {
			return err
		}
	}
	return nil
}

func newGerritServiceAccount(name, namespace string) *coreV1Api.ServiceAccount {
	return &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    platformHelper.GenerateLabels(name),
		},
	}
}

func newGerritPersistentVolumeClaim(volume v1alpha1.GerritVolumes, gerritName, namespace string) *coreV1Api.PersistentVolumeClaim {
	return &coreV1Api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerritName + "-" + volume.Name,
			Namespace: namespace,
			Labels:    platformHelper.GenerateLabels(gerritName),
		},
		Spec: coreV1Api.PersistentVolumeClaimSpec{
			AccessModes: []coreV1Api.PersistentVolumeAccessMode{
				coreV1Api.ReadWriteOnce,
			},
			StorageClassName: &volume.StorageClass,
			Resources: coreV1Api.ResourceRequirements{
				Requests: map[coreV1Api.ResourceName]resource.Quantity{
					coreV1Api.ResourceStorage: resource.MustParse(volume.Capacity),
				},
			},
		},
	}
}

func newGerritPortMap(i v1alpha1.Gerrit) map[string][]coreV1Api.ServicePort {
	if i.Spec.SshPort != 0 {
		return map[string][]coreV1Api.ServicePort{
			i.Name: {
				{
					TargetPort: intstr.IntOrString{StrVal: "ui"},
					Port:       spec.Port,
					Name:       "ui",
				},
				{
					TargetPort: intstr.IntOrString{StrVal: spec.SSHPortName},
					Port:       spec.SSHPort,
					Name:       spec.SSHPortName,
					NodePort:   i.Spec.SshPort,
				},
			},
		}
	} else {
		return map[string][]coreV1Api.ServicePort{
			i.Name: {
				{
					TargetPort: intstr.IntOrString{StrVal: "ui"},
					Port:       spec.Port,
					Name:       "ui",
				},
				{
					TargetPort: intstr.IntOrString{StrVal: spec.SSHPortName},
					Port:       spec.SSHPort,
					Name:       spec.SSHPortName,
				},
			},
		}
	}
}

func newGerritInternalBalancingService(serviceName, namespace string, ports []coreV1Api.ServicePort) *coreV1Api.Service {
	labels := platformHelper.GenerateLabels(serviceName)
	return &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports:    ports,
			Type:     "NodePort",
		},
	}
}

func newGerritSecret(name, gerritName, namespace string, data map[string][]byte) *coreV1Api.Secret {
	return &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    platformHelper.GenerateLabels(gerritName),
		},
		Data: data,
		Type: "Opaque",
	}
}

func newRestConfig() (*rest.Config, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func updatePort(ports []coreV1Api.ServicePort, name string, nodePort int32) ([]coreV1Api.ServicePort, error) {
	for i, p := range ports {
		if p.Name == name {
			p.Port = nodePort
			p.TargetPort.IntVal = nodePort
		}
		ports[i] = p
	}

	return ports, nil
}

func (s *K8SService) CreateConfigMap(instance *v1alpha1.Gerrit, configMapName string, configMapData map[string]string) error {
	labels := platformHelper.GenerateLabels(instance.Name)
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	if err := controllerutil.SetControllerReference(instance, configMapObject, s.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Config Map %v object", configMapObject.Name)
	}

	cm, err := s.CoreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			cm, err = s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
			if err != nil {
				return errors.Wrapf(err, "Couldn't create Config Map %v object", cm.Name)
			}
			log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))
		}
		return errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMapObject.Name)
	}
	return nil
}

func (s K8SService) CreateJenkinsServiceAccount(namespace string, secretName string, serviceAccountType string) error {
	jsa := &jenkinsV1Api.JenkinsServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Spec: jenkinsV1Api.JenkinsServiceAccountSpec{
			Type:        serviceAccountType,
			Credentials: secretName,
		},
	}

	_, err := s.JenkinsServiceAccountClient.Get(secretName, namespace, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			_, err = s.JenkinsServiceAccountClient.Create(jsa, namespace)
			if err != nil {
				return err
			}
		}
		return err
	}

	return nil
}

func (s K8SService) CreateJenkinsScript(namespace string, configMap string) error {
	js := &jenkinsV1Api.JenkinsScript{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMap,
			Namespace: namespace,
		},
		Spec: jenkinsV1Api.JenkinsScriptSpec{
			SourceCmName: configMap,
		},
	}

	_, err := s.JenkinsScriptClient.Get(configMap, namespace, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			_, err = s.JenkinsScriptClient.Create(js, namespace)
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil

}

func (s K8SService) CreateEDPComponentIfNotExist(gerrit v1alpha1.Gerrit, url string, icon string) error {
	comp, err := s.edpCompClient.
		EDPComponents(gerrit.Namespace).
		Get(gerrit.Name, metav1.GetOptions{})
	if err == nil {
		log.Info("edp component already exists", "name", comp.Name)
		return nil
	}
	if k8serr.IsNotFound(err) {
		return s.createEDPComponent(gerrit, url, icon)
	}
	return errors.Wrapf(err, "failed to get edp component: %v", gerrit.Name)
}

func (s K8SService) createEDPComponent(gerrit v1alpha1.Gerrit, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: gerrit.Name,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "gerrit",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}
	if err := controllerutil.SetControllerReference(&gerrit, obj, s.Scheme); err != nil {
		return err
	}
	_, err := s.edpCompClient.
		EDPComponents(gerrit.Namespace).
		Create(obj)
	return err
}
