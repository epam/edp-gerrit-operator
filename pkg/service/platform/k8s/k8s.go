package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingClient "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
)

const (
	nameKey = "name"
	Base    = 10
	BitSize = 32
)

var log = ctrl.Log.WithName("platform")

// K8SService implements platform.Service interface (k8s platform integration).
type K8SService struct {
	Scheme           *runtime.Scheme
	CoreClient       *coreV1Client.CoreV1Client
	appsV1Client     *appsV1Client.AppsV1Client
	networkingClient networkingClient.NetworkingV1Interface
	client           k8sclient.Client
}

func (s *K8SService) GetExternalEndpoint(namespace string, name string) (string, string, error) {
	i, err := s.networkingClient.Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", "", errors.New(fmt.Sprintf("Ingress %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return "", "", err
	}

	return i.Spec.Rules[0].Host, platformHelper.RouteHTTPSScheme, nil
}

func (s *K8SService) IsDeploymentReady(gerrit *gerritApi.Gerrit) (bool, error) {
	deployment, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(context.TODO(), gerrit.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (s *K8SService) PatchDeploymentEnv(gerrit gerritApi.Gerrit, env []coreV1Api.EnvVar) error {
	d, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(context.TODO(), gerrit.Name, metav1.GetOptions{})

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

	_, err = s.appsV1Client.Deployments(d.Namespace).Patch(context.TODO(), d.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Init process with K8SService instance initialization actions.
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

	netClient, err := networkingClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init extensions V1 client for K8S")
	}
	s.networkingClient = netClient

	cl, err := k8sclient.New(config, k8sclient.Options{
		Scheme: s.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create k8s client")
	}
	s.client = cl

	return nil
}

func (service K8SService) GetDeploymentSSHPort(instance *gerritApi.Gerrit) (int32, error) {
	d, err := service.appsV1Client.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
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
				portNumber, err := strconv.ParseInt(ports[0], Base, BitSize)
				if err != nil {
					return 0, err
				}
				return int32(portNumber), nil
			}
		}
	}

	return 0, nil
}

// GenerateKeycloakSettings generates a set of environment var.
func (s *K8SService) GenerateKeycloakSettings(instance *gerritApi.Gerrit) (*[]coreV1Api.EnvVar, error) {
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
					Key: "clientSecret",
				},
			},
		},
	}

	return &envVar, nil
}

func (s K8SService) getKeycloakRealm(instance *gerritApi.Gerrit) (*keycloakApi.KeycloakRealm, error) {
	if instance.Spec.KeycloakSpec.Realm != "" {
		var realmList keycloakApi.KeycloakRealmList
		listOpts := k8sclient.ListOptions{Namespace: instance.Namespace}
		k8sclient.MatchingLabels(map[string]string{
			"targetRealm": instance.Spec.KeycloakSpec.Realm}).ApplyToList(&listOpts)

		if err := s.client.List(context.Background(), &realmList, &listOpts); err != nil {
			return nil, errors.Wrap(err, "unable to get reams by label")
		}

		if len(realmList.Items) > 0 {
			return &realmList.Items[0], nil
		}

		if err := s.client.List(context.Background(), &realmList,
			&k8sclient.ListOptions{Namespace: instance.Namespace}); err != nil {
			return nil, errors.Wrap(err, "unable to get all reams")
		}
		for _, r := range realmList.Items {
			if r.Spec.RealmName == instance.Spec.KeycloakSpec.Realm {
				return &r, nil
			}
		}
	}

	realm := &keycloakApi.KeycloakRealm{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      "main",
		Namespace: instance.Namespace,
	}, realm)
	if err != nil {
		return nil, err
	}

	return realm, nil
}

func (s K8SService) getKeycloakRootUrl(instance *gerritApi.Gerrit) (*string, error) {
	realm, err := s.getKeycloakRealm(instance)
	if err != nil {
		return nil, err
	}

	if len(realm.OwnerReferences) == 0 {
		return nil, errors.Errorf("realm [%s] does not have owner refs", realm.Name)
	}

	keycloak := &keycloakApi.Keycloak{}
	err = s.client.Get(context.TODO(), types.NamespacedName{
		Name:      realm.OwnerReferences[0].Name, //TODO: check if owner references is not empty before access
		Namespace: instance.Namespace,
	}, keycloak)
	if err != nil {
		return nil, err
	}

	keycloakUrl := keycloak.Spec.Url

	return &keycloakUrl, nil
}

// GetSecret return data field of Secret.
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	log.Info("getting secret data", nameKey, name)
	secret, err := service.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// GetPods returns Pod list according to the filter.
func (s *K8SService) GetPods(namespace string, filter metav1.ListOptions) (*coreV1Api.PodList, error) {
	PodList, err := s.CoreClient.Pods(namespace).List(context.TODO(), filter)
	if err != nil {
		return &coreV1Api.PodList{}, err
	}

	return PodList, nil
}

// ExecInPod executes command in pod.
func (s *K8SService) ExecInPod(namespace string, podName string, command []string) (string, string, error) {
	pod, err := s.CoreClient.Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
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

// CreateSecret creates a new Secret Resource for a Gerrit EDP Component.
func (s *K8SService) CreateSecret(gerrit *gerritApi.Gerrit, secretName string, data map[string][]byte) error {
	vLog := log.WithValues(nameKey, secretName)
	vLog.Info("creating secret")

	if _, err := s.CoreClient.Secrets(gerrit.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			log.Info("Creating a new Secret for Gerrit", nameKey, secretName)

			gerritSecretObject := newGerritSecret(secretName, gerrit.Name, gerrit.Namespace, data)

			if err := controllerutil.SetControllerReference(gerrit, gerritSecretObject, s.Scheme); err != nil {
				return err
			}

			if _, err = s.CoreClient.Secrets(gerritSecretObject.Namespace).Create(context.TODO(), gerritSecretObject, metav1.CreateOptions{}); err != nil {
				return err
			}
			log.Info("Secret has been created", nameKey, gerritSecretObject.Name)
		}
		return err
	}
	return nil
}

// GetSecret returns data section of an existing Secret resource of a Gerrit EDP Component.
func (s *K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	secret, err := s.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
			return nil, nil
		}
		return nil, err
	}
	return secret.Data, nil
}

// GetService returns existing Service resource of a Gerrit EDP Component.
func (s *K8SService) GetService(namespace string, name string) (*coreV1Api.Service, error) {
	service, err := s.CoreClient.Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info("Service %v in namespace %v not found", name, namespace)
			return nil, nil
		}
		return nil, err
	}
	return service, nil
}

// UpdateService updates target port of a Gerrit EDP Component.
func (s *K8SService) UpdateService(svc coreV1Api.Service, nodePort int32) error {
	ports := svc.Spec.Ports
	updatedPorts, err := updatePort(ports, "ssh", nodePort)
	if err != nil {
		return err
	}
	svc.Spec.Ports = updatedPorts

	_, err = s.CoreClient.Services(svc.Namespace).Update(context.TODO(), &svc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
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

func (s *K8SService) CreateConfigMap(instance *gerritApi.Gerrit, configMapName string, configMapData map[string]string) error {
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

	_, err := s.CoreClient.ConfigMaps(instance.Namespace).Get(context.TODO(), configMapObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			cm, err := s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(context.TODO(), configMapObject, metav1.CreateOptions{})
			if err != nil {
				return errors.Wrapf(err, "Couldn't create Config Map %v object", cm.Name)
			}
			log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))
		}
		return errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMapObject.Name)
	}
	return nil
}

func (s K8SService) CreateJenkinsServiceAccount(namespace, secretName, serviceAccountType string) error {
	vLog := log.WithValues(nameKey, secretName, "service account type", serviceAccountType)
	vLog.Info("creating jenkins service account")
	if _, err := s.getJenkinsServiceAccount(secretName, namespace); err != nil {
		if k8serr.IsNotFound(err) {
			jsa := &jenkinsV1Api.JenkinsServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Spec: jenkinsV1Api.JenkinsServiceAccountSpec{
					Type:        serviceAccountType,
					Credentials: secretName,
				},
			}

			if err := s.client.Create(context.TODO(), jsa); err != nil {
				return err
			}
			vLog.Info("jenkins service account has been created.")
			return nil
		}
		return err
	}
	vLog.Info("jenkins service account already exists.")
	return nil
}

func (s K8SService) getJenkinsServiceAccount(name, namespace string) (*jenkinsV1Api.JenkinsServiceAccount, error) {
	jsa := &jenkinsV1Api.JenkinsServiceAccount{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, jsa)
	if err != nil {
		return nil, err
	}
	return jsa, nil
}

func (s K8SService) CreateJenkinsScript(namespace string, configMap string) error {
	if _, err := s.getJenkinsScript(configMap, namespace); err != nil {
		if k8serr.IsNotFound(err) {
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
			return s.client.Create(context.TODO(), js)
		}
		return err
	}
	return nil
}

func (s K8SService) getJenkinsScript(name, namespace string) (*jenkinsV1Api.JenkinsScript, error) {
	js := &jenkinsV1Api.JenkinsScript{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, js)
	if err != nil {
		return nil, err
	}
	return js, nil
}

func (s K8SService) CreateEDPComponentIfNotExist(gerrit gerritApi.Gerrit, url string, icon string) error {
	vLog := log.WithValues(nameKey, gerrit.Name)
	vLog.Info("creating EDP component")
	if _, err := s.getEDPComponent(gerrit.Name, gerrit.Namespace); err != nil {
		if k8serr.IsNotFound(err) {
			return s.createEDPComponent(gerrit, url, icon)
		}
		return errors.Wrapf(err, "failed to get edp component: %v", gerrit.Name)
	}
	vLog.Info("edp component already exists")
	return nil
}

func (s K8SService) getEDPComponent(name, namespace string) (*edpCompApi.EDPComponent, error) {
	c := &edpCompApi.EDPComponent{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s K8SService) createEDPComponent(gerrit gerritApi.Gerrit, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerrit.Name,
			Namespace: gerrit.Namespace,
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

	if err := s.client.Create(context.TODO(), obj); err != nil {
		return err
	}
	log.Info("edp component has been created.", nameKey, gerrit.Name)
	return nil
}
