package gerrit

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gerrit-operator/pkg/service/helpers"
	"gerrit-operator/pkg/service/platform"
	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	adminUsername                   = "admin"
	gerritAdminPrivateKeyFileName   = "gerrit_admin_rsa"
	gerritProjectCreatorKeyFileName = "gerrit_project-creator_rsa"
)

// Interface expresses behaviour of the Gerrit EDP Component
type Interface interface {
	IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error)
	Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
}

// ComponentService implements gerrit.Interface
type ComponentService struct {
	// Providing Gerrit EDP component implementation through the interface (platform abstract)
	platformService platform.PlatformService
	k8sClient       client.Client
}

// NewComponentService returns a new instance of a gerrit.Service type
func NewComponentService(ps platform.PlatformService, kc client.Client) Interface {
	return ComponentService{platformService: ps, k8sClient: kc}
}

// IsDeploymentConfigReady check if DC for Gerrit is ready
func (s ComponentService) IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error) {
	gerritIsReady := false
	GerritDc, err := s.platformService.GetDeploymentConfig(instance)
	if err != nil {
		return gerritIsReady, helpers.LogErrorAndReturn(err)
	}
	if GerritDc.Status.AvailableReplicas == 1 {
		gerritIsReady = true
	}
	return gerritIsReady, nil
}

// Install has a minimal set of logic, required to install "vanilla" Gerrit EDP Component
func (s ComponentService) Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	sa, err := s.platformService.CreateServiceAccount(instance)
	if err != nil {
		return nil, err
	}

	if err := s.platformService.CreateSecurityContext(instance, sa); err != nil {
		return nil, err
	}

	if err := s.platformService.CreateService(instance); err != nil {
		return nil, err
	}

	if err := s.platformService.CreateExternalEndpoint(instance); err != nil {
		return nil, err
	}

	if err := s.platformService.CreateVolume(instance); err != nil {
		return nil, err
	}

	if err := s.platformService.CreateDeployConf(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// Configure contains logic related to self configuration of the Gerrit EDP Component
func (s ComponentService) Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	adminSecret := map[string][]byte{
		"user":     []byte(adminUsername),
		"password": []byte(uniuri.New()),
	}
	if err := s.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret); err != nil {
		return nil, err
	}

	gerritAdminPrivateKey, err := helpers.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	gerritAdminPublicKey, err := helpers.GeneratePublicKey(gerritAdminPrivateKey)
	if err != nil {
		return nil, err
	}

	if err := helpers.SaveKeyToFile(helpers.EncodePrivateKey(gerritAdminPrivateKey),
		gerritAdminPrivateKeyFileName, os.Getenv("HOME")+"/"); err != nil {
		return nil, err
	}

	if err := helpers.SaveKeyToFile(gerritAdminPublicKey, gerritAdminPrivateKeyFileName+".pub", os.Getenv("HOME")+"/"); err != nil {
		return nil, err
	}

	gerritAdminSshKeys := map[string][]byte{
		"id_rsa":     helpers.EncodePrivateKey(gerritAdminPrivateKey),
		"id_rsa.pub": gerritAdminPublicKey,
	}

	if err := s.platformService.CreateSecret(instance, instance.Name+"-admin", gerritAdminSshKeys); err != nil {
		return nil, err
	}

	gerritProjectCreatorPrivateKey, err := helpers.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	gerritProjectCreatorPublicKey, err := helpers.GeneratePublicKey(gerritProjectCreatorPrivateKey)
	if err != nil {
		return nil, err
	}

	if err := helpers.SaveKeyToFile(helpers.EncodePrivateKey(gerritProjectCreatorPrivateKey),
		gerritProjectCreatorKeyFileName, os.Getenv("HOME")+"/"); err != nil {
		return nil, err
	}

	if err := helpers.SaveKeyToFile(gerritProjectCreatorPublicKey, gerritProjectCreatorKeyFileName+".pub", os.Getenv("HOME")+"/"); err != nil {
		return nil, err
	}

	gerritProjectCreatorSshKeys := map[string][]byte{
		"id_rsa":     helpers.EncodePrivateKey(gerritAdminPrivateKey),
		"id_rsa.pub": gerritAdminPublicKey,
	}

	if err := s.platformService.CreateSecret(instance, instance.Name+"-project-creator", gerritProjectCreatorSshKeys); err != nil {
		return nil, err
	}

	podList, err := s.platformService.GetPods(instance.Namespace, metav1.ListOptions{LabelSelector: "deploymentconfig=" + instance.Name})
	if err != nil || len(podList.Items) != 1 {
		return nil, errors.Wrapf(err, "[ERROR] Unable to determine Gerrit pod name: %v/%v", err, len(podList.Items))
	}

	_, _, err = s.platformService.ExecInPod(instance.Namespace, podList.Items[0].Name,
		[]string{"/bin/sh", "-c", "mkdir -p /tmp/test"})

	return instance, nil
}

// ExposeConfiguration describes integration points of the Gerrit EDP Component for the other Operators and Components
func (s ComponentService) ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return instance, nil
}

// Integrate applies actions required for the integration with the other EDP Components
func (s ComponentService) Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return instance, nil
}
