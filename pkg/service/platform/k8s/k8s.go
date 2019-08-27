package k8s

import (
	"bytes"
	"gerrit-operator/pkg/apis/v2/v1alpha1"
	"gerrit-operator/pkg/client"
	"gerrit-operator/pkg/service/gerrit/spec"
	"gerrit-operator/pkg/service/helpers"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// K8SService implements platform.Service interface (k8s platform integration)
type K8SService struct {
	Scheme     *runtime.Scheme
	CoreClient *coreV1Client.CoreV1Client
	EdpClient  *client.EdpV1Client
}

// Init process with K8SService instance initialization actions
func (s *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	s.Scheme = scheme

	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return helpers.LogErrorAndReturn(err)
	}
	s.CoreClient = coreClient

	edpClient, err := client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}
	s.EdpClient = edpClient

	return nil
}

// GetSecret return data field of Secret
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Secret %v in namespace %v not found", name, namespace)
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "Unable to get secret data")
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

	restConfig := newRestConfig()

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
		return "", "", err
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
		return helpers.LogErrorAndReturn(err)
	}

	if _, err := s.CoreClient.Secrets(gerritSecretObject.Namespace).Get(gerritSecretObject.Name, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			log.Printf("Creating a new Secret %s/%s for Gerrit", gerritSecretObject.Namespace, gerritSecretObject.Name)
			if _, err = s.CoreClient.Secrets(gerritSecretObject.Namespace).Create(gerritSecretObject); err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("Secret %s/%s has been created", gerritSecretObject.Namespace, gerritSecretObject.Name)
		} else {
			return helpers.LogErrorAndReturn(err)
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
	gerritServiceAccountObject := newGerritServiceAccount(gerrit.Name, gerrit.Namespace)

	if err := controllerutil.SetControllerReference(gerrit, gerritServiceAccountObject, s.Scheme); err != nil {
		return nil, helpers.LogErrorAndReturn(err)
	}

	if account, e = s.CoreClient.ServiceAccounts(gerritServiceAccountObject.Namespace).Get(gerritServiceAccountObject.Name, metav1.GetOptions{}); e != nil {
		if k8serr.IsNotFound(e) {
			log.Printf("Creating a new ServiceAccount %s/%s for Gerrit %s", gerritServiceAccountObject.Namespace, gerritServiceAccountObject.Name, gerrit.Name)
			if account, e = s.CoreClient.ServiceAccounts(gerritServiceAccountObject.Namespace).Create(gerritServiceAccountObject); e != nil {
				return nil, helpers.LogErrorAndReturn(e)
			}
			log.Printf("ServiceAccount %s/%s has been created", gerritServiceAccountObject.Namespace, gerritServiceAccountObject.Name)
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
			log.Printf("Secret %v in namespace %v not found", name, namespace)
			return nil, nil
		}
		return nil, helpers.LogErrorAndReturn(err)
	}
	return secret.Data, nil
}

// GetService returns existing Service resource of a Gerrit EDP Component
func (s *K8SService) GetService(namespace string, name string) (*coreV1Api.Service, error) {
	service, err := s.CoreClient.Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Printf("Service %v in namespace %v not found", name, namespace)
			return nil, nil
		}
		return nil, helpers.LogErrorAndReturn(err)
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
		return helpers.LogErrorAndReturn(err)
	}

	if _, err := s.CoreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Get(gerritVolumeObject.Name, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			log.Printf("Creating a new PersistantVolumeClaim %s/%s for %s", gerritVolumeObject.Namespace, gerritVolumeObject.Name, gerrit.Name)
			if _, err = s.CoreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Create(gerritVolumeObject); err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("PersistantVolumeClaim %s/%s has been created", gerritVolumeObject.Namespace, gerritVolumeObject.Name)
		} else {
			return helpers.LogErrorAndReturn(err)
		}
	}
	return nil
}

func (s *K8SService) createGerritService(serviceName string, gerrit *v1alpha1.Gerrit) error {
	portMap := newGerritPortMap(gerrit.Name)
	gerritServiceObject := newGerritInternalBalancingService(serviceName, gerrit.Namespace, portMap[serviceName])

	if err := controllerutil.SetControllerReference(gerrit, gerritServiceObject, s.Scheme); err != nil {
		return helpers.LogErrorAndReturn(err)
	}

	if _, err := s.CoreClient.Services(gerrit.Namespace).Get(serviceName, metav1.GetOptions{}); err != nil {
		if k8serr.IsNotFound(err) {
			log.Printf("Creating a new service %s/%s for gerrit %s", gerritServiceObject.Namespace, gerritServiceObject.Name, gerrit.Name)
			if _, err = s.CoreClient.Services(gerritServiceObject.Namespace).Create(gerritServiceObject); err != nil {
				return helpers.LogErrorAndReturn(err)
			}
			log.Printf("Service %s/%s has been created", gerritServiceObject.Namespace, gerritServiceObject.Name)
		} else {
			return helpers.LogErrorAndReturn(err)
		}
	}
	return nil
}

func newGerritServiceAccount(name, namespace string) *coreV1Api.ServiceAccount {
	return &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    helpers.GenerateLabels(name),
		},
	}
}

func newGerritPersistentVolumeClaim(volume v1alpha1.GerritVolumes, gerritName, namespace string) *coreV1Api.PersistentVolumeClaim {
	return &coreV1Api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerritName + "-" + volume.Name,
			Namespace: namespace,
			Labels:    helpers.GenerateLabels(gerritName),
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

func newGerritPortMap(gerritName string) map[string][]coreV1Api.ServicePort {
	return map[string][]coreV1Api.ServicePort{
		gerritName: {
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

func newGerritInternalBalancingService(serviceName, namespace string, ports []coreV1Api.ServicePort) *coreV1Api.Service {
	labels := helpers.GenerateLabels(serviceName)
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
			Labels:    helpers.GenerateLabels(gerritName),
		},
		Data: data,
		Type: "Opaque",
	}
}

func newRestConfig() *rest.Config {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		helpers.LogErrorAndReturn(err)
		return nil
	}

	return restConfig
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
