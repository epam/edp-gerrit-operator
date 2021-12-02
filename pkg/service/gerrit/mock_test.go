package gerrit

import (
	"errors"
	"fmt"
	"testing"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
)

func TestMock_IsDeploymentReady(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	m := Mock{}
	if _, err := m.IsDeploymentReady(&v1alpha1.Gerrit{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_Configure(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	m := Mock{}
	if _, _, err := m.Configure(&v1alpha1.Gerrit{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_ExposeConfiguration(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	m := Mock{}
	if _, err := m.ExposeConfiguration(&v1alpha1.Gerrit{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetGerritSSHUrl(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	m := Mock{}
	if _, err := m.GetGerritSSHUrl(&v1alpha1.Gerrit{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetRestClient(t *testing.T) {
	gi := v1alpha1.Gerrit{}

	m := Mock{}
	m.On("GetRestClient", &gi).Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetRestClient(&gi); err == nil {
		t.Fatal("no error returned")
	}

	clInt := gerritClient.Mock{}
	m.On("GetRestClient", &gi).Return(&clInt, nil)
	if _, err := m.GetRestClient(&gi); err != nil {
		t.Fatal(err)
	}
}
