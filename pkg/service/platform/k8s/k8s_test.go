package k8s

import (
	"testing"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
)

func TestK8SService_getKeycloakRootUrl(t *testing.T) {
	g := v1alpha1.Gerrit{}
	sch := runtime.NewScheme()
	v1alpha1.RegisterTypes(sch)
	assert.NoError(t, keycloakApi.AddToScheme(sch))
	rlm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "main"}}
	fk := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&g, &rlm).Build()

	s := K8SService{client: fk}
	_, err := s.getKeycloakRootUrl(&g)
	assert.EqualError(t, err, "realm [main] does not have owner refs")
}
