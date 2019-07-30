package service

import (
	"fmt"
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gerrit-operator/pkg/client"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type K8SService struct {
	scheme     *runtime.Scheme
	coreClient coreV1Client.CoreV1Client
	edpClient  client.EdpV1Client
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	edpClient, err := client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}

	service.edpClient = *edpClient
	service.coreClient = *coreClient
	service.scheme = scheme
	return nil
}

func (service K8SService) CreateService(gerrit v1alpha1.Gerrit) error {
	portMap := map[string][]coreV1Api.ServicePort{
		gerrit.Name: {
			{
				TargetPort: intstr.IntOrString{StrVal: "ui"},
				Port:       Port,
			},
			{
				TargetPort: intstr.IntOrString{StrVal: "ssh"},
				Port:       SSHPort,
			},
		},
		gerrit.Name + "-db": {
			{
				TargetPort: intstr.IntOrString{StrVal: gerrit.Name + "-db"},
				Port:       DBPort,
			},
		},
	}

	for _, serviceName := range []string{gerrit.Name, gerrit.Name + "-db"} {
		labels := generateLabels(serviceName)

		gerritServiceObject, err := newGerritInternalBalancingService(serviceName, gerrit.Namespace, labels, portMap[serviceName])

		if err := controllerutil.SetControllerReference(&gerrit, gerritServiceObject, service.scheme); err != nil {
			return logErrorAndReturn(err)
		}

		gerritService, err := service.coreClient.Services(gerrit.Namespace).Get(serviceName, metav1.GetOptions{})

		if err != nil && k8serr.IsNotFound(err) {
			log.Printf("Creating a new service %s/%s for gerrit %s", gerritServiceObject.Namespace, gerritServiceObject.Name, gerrit.Name)

			gerritService, err = service.coreClient.Services(gerritServiceObject.Namespace).Create(gerritServiceObject)

			if err != nil {
				return logErrorAndReturn(err)
			}

			log.Printf("service %s/%s has been created", gerritService.Namespace, gerritService.Name)
		} else if err != nil {
			return logErrorAndReturn(err)
		}
	}

	return nil
}

func (service K8SService) CreateSecret(gerrit v1alpha1.Gerrit, name string, data map[string][]byte) error {
	labels := generateLabels(gerrit.Name)

	gerritSecretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gerrit.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&gerrit, gerritSecretObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	gerritSecret, err := service.coreClient.Secrets(gerritSecretObject.Namespace).Get(gerritSecretObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Creating a new Secret %s/%s for Admin Console", gerritSecretObject.Namespace, gerritSecretObject.Name)

		gerritSecret, err = service.coreClient.Secrets(gerritSecretObject.Namespace).Create(gerritSecretObject)

		if err != nil {
			return logErrorAndReturn(err)
		}
		log.Printf("Secret %s/%s has been created", gerritSecret.Namespace, gerritSecret.Name)

	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service K8SService) CreateVolume(gerrit v1alpha1.Gerrit) error {

	labels := generateLabels(gerrit.Name)

	for _, volume := range gerrit.Spec.Volumes {

		gerritVolumeObject := &coreV1Api.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gerrit.Name + "-" + volume.Name,
				Namespace: gerrit.Namespace,
				Labels:    labels,
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

		if err := controllerutil.SetControllerReference(&gerrit, gerritVolumeObject, service.scheme); err != nil {
			return logErrorAndReturn(err)
		}

		gerritVolume, err := service.coreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Get(gerritVolumeObject.Name, metav1.GetOptions{})

		if err != nil && k8serr.IsNotFound(err) {
			log.Printf("Creating a new PersistantVolumeClaim %s/%s for %s", gerritVolumeObject.Namespace, gerritVolumeObject.Name, gerrit.Name)

			gerritVolume, err = service.coreClient.PersistentVolumeClaims(gerritVolumeObject.Namespace).Create(gerritVolumeObject)

			if err != nil {
				return logErrorAndReturn(err)
			}

			log.Printf("PersistantVolumeClaim %s/%s has been created", gerritVolume.Namespace, gerritVolume.Name)
		} else if err != nil {
			return logErrorAndReturn(err)
		}
	}
	return nil
}

func (service K8SService) CreateServiceAccount(gerrit v1alpha1.Gerrit) (*coreV1Api.ServiceAccount, error) {

	labels := generateLabels(gerrit.Name)

	gerritServiceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerrit.Name,
			Namespace: gerrit.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&gerrit, gerritServiceAccountObject, service.scheme); err != nil {
		return nil, logErrorAndReturn(err)
	}

	gerritServiceAccount, err := service.coreClient.ServiceAccounts(gerritServiceAccountObject.Namespace).Get(gerritServiceAccountObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Creating a new ServiceAccount %s/%s for Gerrit %s", gerritServiceAccountObject.Namespace, gerritServiceAccountObject.Name, gerrit.Name)

		gerritServiceAccount, err = service.coreClient.ServiceAccounts(gerritServiceAccountObject.Namespace).Create(gerritServiceAccountObject)

		if err != nil {
			return nil, logErrorAndReturn(err)
		}

		log.Printf("ServiceAccount %s/%s has been created", gerritServiceAccount.Namespace, gerritServiceAccount.Name)
	} else if err != nil {
		return nil, logErrorAndReturn(err)
	}

	return gerritServiceAccount, nil
}

func (service K8SService) CreateExternalEndpoint(sonar v1alpha1.Gerrit) error {
	fmt.Printf("No implementation for K8s yet.")
	return nil
}

func (service K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	sonarSecret, err := service.coreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Secret %v in namespace %v not found", name, namespace)
		return nil, nil
	} else if err != nil {
		return nil, logErrorAndReturn(err)
	}
	return sonarSecret.Data, nil
}

func newGerritInternalBalancingService(serviceName string, namespace string, labels map[string]string, ports []coreV1Api.ServicePort) (*coreV1Api.Service, error) {
	return &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports:    ports,
		},
	}, nil
}

func logErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func generateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}
