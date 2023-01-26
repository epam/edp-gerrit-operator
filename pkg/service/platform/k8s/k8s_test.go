package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
)

func TestK8SService_getKeycloakRootUrl(t *testing.T) {
	g := gerritApi.Gerrit{}
	sch := runtime.NewScheme()
	utilruntime.Must(gerritApi.AddToScheme(sch))
	assert.NoError(t, keycloakApi.AddToScheme(sch))

	rlm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "main"}}
	fk := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&g, &rlm).Build()

	s := K8SService{client: fk}
	_, err := s.getKeycloakRootUrl(&g)
	assert.EqualError(t, err, "realm [main] does not have owner refs")
}
