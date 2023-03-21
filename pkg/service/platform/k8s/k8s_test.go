package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	fakeClient "k8s.io/client-go/kubernetes/fake"
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

func TestK8SService_GetExternalEndpoint(t *testing.T) {
	t.Parallel()

	type args struct {
		namespace string
		name      string
	}

	type want struct {
		host   string
		scheme string
		err    require.ErrorAssertionFunc
	}

	tests := []struct {
		name    string
		ingress *networkingV1.Ingress
		args    args
		want    want
	}{
		{
			name: "should return external endpoint",
			ingress: &networkingV1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: networkingV1.IngressSpec{
					Rules: []networkingV1.IngressRule{
						{
							Host: "test.com",
						},
					},
				},
			},
			args: args{
				namespace: "test",
				name:      "test",
			},
			want: want{
				host:   "test.com",
				scheme: "https",
				err:    require.NoError,
			},
		},
		{
			name:    "should return error if ingress not found",
			ingress: &networkingV1.Ingress{},
			args: args{
				namespace: "test",
				name:      "test",
			},
			want: want{
				host:   "",
				scheme: "",
				err: func(t require.TestingT, err error, _ ...interface{}) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "ingress test in namespace test not found")
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &K8SService{
				networkingClient: fakeClient.NewSimpleClientset(tt.ingress).
					NetworkingV1(),
			}

			gotHost, gotScheme, err := service.GetExternalEndpoint(tt.args.namespace, tt.args.name)
			tt.want.err(t, err)

			assert.Equal(t, tt.want.host, gotHost)
			assert.Equal(t, tt.want.scheme, gotScheme)
		})
	}
}
