package openshift

import (
	"testing"

	routeV1 "github.com/openshift/api/route/v1"
	fakeRoute "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpenshiftService_GetExternalEndpoint(t *testing.T) {
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
		name  string
		route *routeV1.Route
		args  args
		want  want
	}{
		{
			name: "should return external http endpoint",
			route: &routeV1.Route{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: routeV1.RouteSpec{
					Host: "test.com",
					TLS: &routeV1.TLSConfig{
						Termination: "",
					},
				},
			},
			args: args{
				namespace: "test",
				name:      "test",
			},
			want: want{
				host:   "test.com",
				scheme: "http",
				err:    require.NoError,
			},
		},
		{
			name: "should return external https endpoint",
			route: &routeV1.Route{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: routeV1.RouteSpec{
					Host: "test.com",
					TLS: &routeV1.TLSConfig{
						Termination: "not empty",
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
			name:  "should return error if route not found",
			route: &routeV1.Route{},
			args: args{
				namespace: "test",
				name:      "test",
			},
			want: want{
				host:   "",
				scheme: "",
				err: func(t require.TestingT, err error, _ ...interface{}) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "failed to find route \"test\" in namespace \"test\"")
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &OpenshiftService{
				routeClient: fakeRoute.NewSimpleClientset(tt.route).
					RouteV1(),
			}

			gotHost, gotScheme, err := service.GetExternalEndpoint(tt.args.namespace, tt.args.name)
			tt.want.err(t, err)

			require.Equal(t, tt.want.host, gotHost)
			require.Equal(t, tt.want.scheme, gotScheme)
		})
	}
}
