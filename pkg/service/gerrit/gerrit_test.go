package gerrit

import (
	pmock "github.com/epam/edp-gerrit-operator/v2/mock/platform"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestIsErrUserNotFound(t *testing.T) {
	tnf := ErrUserNotFound("err not found")
	err := errors.Wrap(tnf, "error")
	if !IsErrUserNotFound(err) {
		t.Fatal("wrong error type")
	}
}

func TestNewComponentService(t *testing.T) {
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	assert.Equal(t, CS, NewComponentService(ps, kc, ks))
}

func TestComponentService_IsDeploymentReady(t *testing.T) {
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{
		PlatformService: ps,
		client:          kc,
		k8sScheme:       ks,
	}

	inst := &v1alpha1.Gerrit{}

	ps.On("IsDeploymentReady", inst).Return(true, nil)

	ready, err := CS.IsDeploymentReady(inst)
	assert.True(t, ready)
	assert.NoError(t, err)
}
