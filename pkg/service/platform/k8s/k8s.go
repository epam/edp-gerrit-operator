package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingClient "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
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
	CoreClient       coreV1Client.CoreV1Interface
	appsV1Client     *appsV1Client.AppsV1Client
	networkingClient networkingClient.NetworkingV1Interface
	client           k8sClient.Client
}

// GetExternalEndpoint returns host and scheme of an Ingress.
func (s *K8SService) GetExternalEndpoint(namespace, name string) (string, string, error) {
	ctx := context.Background()

	ingress, err := s.networkingClient.Ingresses(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", "", fmt.Errorf("ingress %v in namespace %v not found", name, namespace)
		}

		return "", "", fmt.Errorf("failed to Get Ingress %q: %w", name, err)
	}

	host := ingress.Spec.Rules[0].Host
	scheme := platformHelper.RouteHTTPSScheme

	return host, scheme, nil
}

func (s *K8SService) IsDeploymentReady(gerrit *gerritApi.Gerrit) (bool, error) {
	ctx := context.Background()

	deployment, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(ctx, gerrit.Name, metaV1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to Get Deployment %q: %w", gerrit.Name, err)
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (s *K8SService) PatchDeploymentEnv(gerrit *gerritApi.Gerrit, env []coreV1Api.EnvVar) error {
	ctx := context.Background()

	d, err := s.appsV1Client.Deployments(gerrit.Namespace).Get(ctx, gerrit.Name, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to Get Deployment %q: %w", gerrit.Name, err)
	}

	if len(env) == 0 {
		return nil
	}

	container, err := platformHelper.SelectContainer(d.Spec.Template.Spec.Containers, gerrit.Name)
	if err != nil {
		return fmt.Errorf("not containers found for gerrit resource %q: %w", gerrit.Name, err)
	}

	container.Env = platformHelper.UpdateEnv(container.Env, env)

	d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed encode to json gerrit deployment k8s object %q: %w", gerrit.Name, err)
	}

	_, err = s.appsV1Client.Deployments(d.Namespace).Patch(ctx, d.Name, types.StrategicMergePatchType, jsonDc, metaV1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to Patch Deployment %q: %w", d.Name, err)
	}

	return nil
}

// Init process with K8SService instance initialization actions.
func (s *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	s.Scheme = scheme

	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s coreV1Client: %w", err)
	}

	s.CoreClient = coreClient

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s AppsV1Client: %w", err)
	}

	s.appsV1Client = appsClient

	netClient, err := networkingClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init extensions V1 client for K8S")
	}

	s.networkingClient = netClient

	cl, err := k8sClient.New(config, k8sClient.Options{
		Scheme: s.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create k8s client")
	}

	s.client = cl

	return nil
}

func (s *K8SService) GetDeploymentSSHPort(instance *gerritApi.Gerrit) (int32, error) {
	ctx := context.Background()

	d, err := s.appsV1Client.Deployments(instance.Namespace).Get(ctx, instance.Name, metaV1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to GET Deployment %q: %w", instance.Name, err)
	}

	for _, env := range d.Spec.Template.Spec.Containers[0].Env {
		if env.Name == spec.SSHListnerEnvName {
			re := regexp.MustCompile(`\d+`)
			if re.MatchString(env.Value) {
				ports := re.FindStringSubmatch(env.Value)
				if len(ports) != 1 {
					return 0, nil
				}

				port := ports[0]

				portNumber, err := strconv.ParseInt(port, Base, BitSize)
				if err != nil {
					return 0, fmt.Errorf("failed to parse port value %q: %w", port, err)
				}

				return int32(portNumber), nil
			}
		}
	}

	return 0, nil
}

// GenerateKeycloakSettings generates a set of environment var.
func (s *K8SService) GenerateKeycloakSettings(instance *gerritApi.Gerrit) ([]coreV1Api.EnvVar, error) {
	identityServiceSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix)

	realm, err := s.getKeycloakRealm(instance)
	if err != nil {
		return nil, err
	}

	keycloakUrl, err := s.getKeycloakRootUrl(instance)
	if err != nil {
		return nil, err
	}

	return []coreV1Api.EnvVar{
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
	}, nil
}

func (s *K8SService) getKeycloakRealm(instance *gerritApi.Gerrit) (*keycloakApi.KeycloakRealm, error) {
	ctx := context.Background()

	if instance.Spec.KeycloakSpec.Realm != "" {
		realmList := keycloakApi.KeycloakRealmList{}
		listOpts := k8sClient.ListOptions{Namespace: instance.Namespace}

		k8sClient.MatchingLabels(map[string]string{
			"targetRealm": instance.Spec.KeycloakSpec.Realm,
		}).ApplyToList(&listOpts)

		if err := s.client.List(ctx, &realmList, &listOpts); err != nil {
			return nil, errors.Wrap(err, "unable to get reams by label")
		}

		if len(realmList.Items) > 0 {
			return &realmList.Items[0], nil
		}

		if err := s.client.List(ctx, &realmList,
			&k8sClient.ListOptions{Namespace: instance.Namespace}); err != nil {
			return nil, errors.Wrap(err, "unable to get all reams")
		}

		for i := 0; i < len(realmList.Items); i++ {
			if realmList.Items[i].Spec.RealmName == instance.Spec.KeycloakSpec.Realm {
				return &realmList.Items[i], nil
			}
		}
	}

	name := "main"
	realm := &keycloakApi.KeycloakRealm{}

	err := s.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}, realm)
	if err != nil {
		return nil, fmt.Errorf("failed to GET KeycloakRealm resorce %q: %w", name, err)
	}

	return realm, nil
}

func (s *K8SService) getKeycloakRootUrl(instance *gerritApi.Gerrit) (*string, error) {
	ctx := context.Background()

	realm, err := s.getKeycloakRealm(instance)
	if err != nil {
		return nil, err
	}

	if len(realm.OwnerReferences) == 0 {
		return nil, errors.Errorf("realm [%s] does not have owner refs", realm.Name)
	}

	keycloak := &keycloakApi.Keycloak{}
	name := realm.OwnerReferences[0].Name //TODO: check if owner references is not empty before access

	err = s.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}, keycloak)
	if err != nil {
		return nil, fmt.Errorf("failed to GET Keycloak resorce %q: %w", name, err)
	}

	keycloakUrl := keycloak.Spec.Url

	return &keycloakUrl, nil
}

// GetSecret return data field of Secret.
func (s *K8SService) GetSecretData(namespace, name string) (map[string][]byte, error) {
	log.Info("getting secret data", nameKey, name)

	ctx := context.Background()

	secret, err := s.CoreClient.Secrets(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))

		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to GET secret resorce %q: %w", name, err)
	}

	return secret.Data, nil
}

// GetPods returns Pod list according to the filter.
func (s *K8SService) GetPods(namespace string, filter *metaV1.ListOptions) (*coreV1Api.PodList, error) {
	ctx := context.Background()

	podList, err := s.CoreClient.Pods(namespace).List(ctx, *filter)
	if err != nil {
		return &coreV1Api.PodList{}, fmt.Errorf("failed to GET list of Pods: %w", err)
	}

	return podList, nil
}

// ExecInPod executes command in pod.
func (s *K8SService) ExecInPod(namespace, podName string, command []string) (stdout, stderr io.Reader, err error) {
	ctx := context.Background()

	pod, err := s.CoreClient.Pods(namespace).Get(ctx, podName, metaV1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to GET Pod %q: %w", podName, err)
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
		}, k8sScheme.ParameterCodec)

	restConfig, err := newRestConfig()
	if err != nil {
		return nil, nil, err
	}

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the %q server: %w", req.URL(), err)
	}

	stdoutBuffer, stderrBuffer := new(bytes.Buffer), new(bytes.Buffer)

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdoutBuffer,
		Stderr: stderrBuffer,
		Tty:    false,
	})
	if err != nil {
		return nil, stderrBuffer, fmt.Errorf("failed to execute shell-style stream: %w", err)
	}

	stdout = stdoutBuffer
	stderr = stderrBuffer

	return
}

// CreateSecret creates a new Secret Resource for a Gerrit EDP Component.
func (s *K8SService) CreateSecret(gerrit *gerritApi.Gerrit, secretName string, data map[string][]byte, labels map[string]string) error {
	ctx := context.Background()
	vLog := log.WithValues(nameKey, secretName)
	vLog.Info("creating secret")

	_, err := s.CoreClient.Secrets(gerrit.Namespace).Get(ctx, secretName, metaV1.GetOptions{})
	if err == nil {
		return nil
	}

	if !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to GET secret resorce %q: %w", secretName, err)
	}

	log.Info("Creating a new Secret for Gerrit", nameKey, secretName)

	gerritSecretObject := newGerritSecret(secretName, gerrit.Name, gerrit.Namespace, data)

	maps.Copy(gerritSecretObject.Labels, labels)

	err = controllerutil.SetControllerReference(gerrit, gerritSecretObject, s.Scheme)
	if err != nil {
		return fmt.Errorf("failed to set owner %q (Gerrit resource) as a controller OwnerReference on %q (Gerrit Secret resource): %w", gerrit.Name, gerritSecretObject.Name, err)
	}

	_, err = s.CoreClient.Secrets(gerritSecretObject.Namespace).Create(ctx, gerritSecretObject, metaV1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create gerrit Secret resource %q: %w", gerritSecretObject.Name, err)
	}

	log.Info("Secret has been created", nameKey, gerritSecretObject.Name)

	return nil
}

// GetSecret returns data section of an existing Secret resource of a Gerrit EDP Component.
func (s *K8SService) GetSecret(namespace, name string) (map[string][]byte, error) {
	ctx := context.Background()

	secret, err := s.CoreClient.Secrets(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err == nil {
		return secret.Data, nil
	}

	if k8sErrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))

		return nil, nil
	}

	// got unexpected error
	return nil, fmt.Errorf("failed to GET %q secret: %w", name, err)
}

// GetService returns existing Service resource of a Gerrit EDP Component.
func (s *K8SService) GetService(namespace, name string) (*coreV1Api.Service, error) {
	ctx := context.Background()

	service, err := s.CoreClient.Services(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err == nil {
		return service, nil
	}

	if k8sErrors.IsNotFound(err) {
		log.Info("Service %v in namespace %v not found", name, namespace)

		return nil, nil
	}

	// got unexpected error
	return nil, fmt.Errorf("failed to GET %q service: %w", name, err)
}

// UpdateService updates target port of a Gerrit EDP Component.
func (s *K8SService) UpdateService(svc *coreV1Api.Service, nodePort int32) error {
	ctx := context.Background()
	ports := svc.Spec.Ports

	updatedPorts, err := updatePort(ports, "ssh", nodePort)
	if err != nil {
		return err
	}

	svc.Spec.Ports = updatedPorts

	_, err = s.CoreClient.Services(svc.Namespace).Update(ctx, svc, metaV1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("faile to update %q service: %w", svc.Name, err)
	}

	return nil
}

func newGerritSecret(name, gerritName, namespace string, data map[string][]byte) *coreV1Api.Secret {
	return &coreV1Api.Secret{
		ObjectMeta: metaV1.ObjectMeta{
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
		return nil, fmt.Errorf("failed to retrive config: %w", err)
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
	ctx := context.Background()
	labels := platformHelper.GenerateLabels(instance.Name)
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	if err := controllerutil.SetControllerReference(instance, configMapObject, s.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Config Map %v object", configMapObject.Name)
	}

	_, err := s.CoreClient.ConfigMaps(instance.Namespace).Get(ctx, configMapObject.Name, metaV1.GetOptions{})
	if err == nil {
		return nil
	}

	if !k8sErrors.IsNotFound(err) {
		return errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMapObject.Name)
	}

	cm, err := s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(ctx, configMapObject, metaV1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Couldn't create Config Map %v object", cm.Name)
	}

	log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))

	return nil
}
