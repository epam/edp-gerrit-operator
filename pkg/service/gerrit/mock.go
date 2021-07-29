package gerrit

import (
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) IsDeploymentReady(instance *v1alpha1.Gerrit) (bool, error) {
	panic("not implemented")
}

func (m *Mock) Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, bool, error) {
	panic("not implemented")
}

func (m *Mock) ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	panic("not implemented")
}

func (m *Mock) Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	panic("not implemented")
}

func (m *Mock) GetGerritSSHUrl(instance *v1alpha1.Gerrit) (string, error) {
	panic("not implemented")
}

func (m *Mock) GetServicePort(instance *v1alpha1.Gerrit) (int32, error) {
	panic("not implemented")
}

func (m *Mock) GetRestClient(gerritInstance *v1alpha1.Gerrit) (gerrit.ClientInterface, error) {
	called := m.Called(gerritInstance)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(gerrit.ClientInterface), nil
}
