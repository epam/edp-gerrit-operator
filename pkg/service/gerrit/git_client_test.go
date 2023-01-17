package gerrit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/edp/v1"
	mock "github.com/epam/edp-gerrit-operator/v2/mock/platform"
)

type testChild struct{}

func (testChild) GetNamespace() string {
	return "ns"
}

func (testChild) OwnerName() string {
	return "gerrit"
}

func TestComponentService_GetGitClient_Failure(t *testing.T) {
	sch := runtime.NewScheme()
	plt := mock.PlatformService{}

	err := coreV1.AddToScheme(sch)
	assert.NoError(t, err)
	utilRuntime.Must(gerritApi.AddToScheme(sch))

	s := ComponentService{
		PlatformService: &plt,
		k8sScheme:       sch,
		client:          fake.NewClientBuilder().WithScheme(sch).Build(),
	}

	testCh := testChild{}

	_, err = s.GetGitClient(context.Background(), testCh, "")
	assert.Error(t, err)
	assert.EqualError(t, err,
		"unable to get parent gerrit: gerrits.v2.edp.epam.com \"gerrit\" not found")

	rootGerrit := gerritApi.Gerrit{ObjectMeta: metaV1.ObjectMeta{Name: testCh.OwnerName(),
		Namespace: testCh.GetNamespace()}}
	s.client = fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&rootGerrit).Build()
	plt.On("GetSecretData", testCh.GetNamespace(), fmt.Sprintf("%v-admin-password", rootGerrit.Name)).
		Return(nil, errors.New("secret fatal")).Once()

	_, err = s.GetGitClient(context.Background(), testCh, "")
	assert.Error(t, err)
	assert.EqualError(t, err,
		"Failed to get Gerrit admin password from secret for ns/gerrit: Failed to get Secret gerrit-admin-password for ns/gerrit: secret fatal")

	plt.AssertExpectations(t)
}

func TestComponentService_GetGitClient(t *testing.T) {
	sch := runtime.NewScheme()
	plt := mock.PlatformService{}

	err := coreV1.AddToScheme(sch)
	assert.NoError(t, err)
	utilRuntime.Must(gerritApi.AddToScheme(sch))

	testCh := testChild{}
	rootGerrit := gerritApi.Gerrit{ObjectMeta: metaV1.ObjectMeta{Name: testCh.OwnerName(),
		Namespace: testCh.GetNamespace()}}

	s := ComponentService{
		PlatformService: &plt,
		k8sScheme:       sch,
		client:          fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&rootGerrit).Build(),
		runningInClusterFunc: func() bool {
			return true
		},
	}

	plt.On("GetSecretData", testCh.GetNamespace(), fmt.Sprintf("%v-admin-password", rootGerrit.Name)).
		Return(map[string][]byte{"password": []byte("secret")}, nil)

	_, err = s.GetGitClient(context.Background(), testCh, "")
	assert.NoError(t, err)

	plt.AssertExpectations(t)
}
