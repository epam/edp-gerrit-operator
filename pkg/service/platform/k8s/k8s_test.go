package k8s_test

import (
	"github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/platform/k8s"
	"github.com/stretchr/testify/assert"
	coreV1Api "k8s.io/api/core/v1"
	"testing"
)

const (
	namespace = "test"
	name      = "gerrit"
)

func Test_CreateExternalEndpoint(t *testing.T) {
	ks := k8s.K8SService{}
	g := &v1alpha1.Gerrit{}

	actual := ks.CreateExternalEndpoint(g)

	assert.Equal(t, nil, actual)
}

func Test_CreateSecurityContext(t *testing.T) {
	ks := k8s.K8SService{}
	g := &v1alpha1.Gerrit{}
	sa := &coreV1Api.ServiceAccount{}

	actual := ks.CreateSecurityContext(g, sa)

	assert.Equal(t, nil, actual)
}

func Test_CreateDeployment(t *testing.T) {
	ks := k8s.K8SService{}
	g := &v1alpha1.Gerrit{}

	actual := ks.CreateDeployment(g)

	assert.Equal(t, nil, actual)
}

func Test_GetExternalEndpoint(t *testing.T) {

}
