package gerrit

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gerrit-operator/pkg/service/helpers"
	"gerrit-operator/pkg/service/platform"

	"github.com/dchest/uniuri"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	username := []byte("admin")

	adminSecret := map[string][]byte{
		"user":     username,
		"password": []byte(uniuri.New()),
	}
	if err := s.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret); err != nil {
		return nil, err
	}

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
