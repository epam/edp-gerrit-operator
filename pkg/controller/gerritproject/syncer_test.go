package gerritproject

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
)

func TestSyncBackendProjectsTick(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "ns", Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			OwnerReferences: []metaV1.OwnerReference{
				{
					Kind: g.Kind,
					UID:  g.UID,
				},
			}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&g, &prj).Build()
	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		log:     &logger,
		service: &serviceMock,
	}

	clientMock.On("ListProjects", "CODE").Return([]gerritClient.Project{
		{
			Name: "alphabet/google",
		},
	}, nil)
	clientMock.On("ListProjectBranches", "alphabet/google").Return([]gerritClient.Branch{
		{
			Ref: "test",
		},
	}, nil)

	if err := rcn.syncBackendProjectsTick(); err != nil {
		t.Fatal(err)
	}

	var k8sGerritProject gerritApi.GerritProject
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "ger1-alphabet-google", Namespace: g.Namespace},
		&k8sGerritProject); err != nil {
		t.Fatal(err)
	}

	if k8sGerritProject.Spec.Name != "alphabet/google" {
		t.Fatalf("wrong gerrit project name: %s", k8sGerritProject.Spec.Name)
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestSyncBackendProjectsTick_BranchesFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "ns", Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			OwnerReferences: []metaV1.OwnerReference{
				{
					Kind: g.Kind,
					UID:  g.UID,
				},
			}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&g, &prj).Build()
	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		log:     &logger,
		service: &serviceMock,
	}

	clientMock.On("ListProjects", "CODE").Return([]gerritClient.Project{
		{
			Name: "alphabet",
		},
	}, nil)
	clientMock.On("ListProjectBranches", "alphabet").Return(nil, errors.New("list branches fatal"))

	err := rcn.syncBackendProjectsTick()
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "list branches fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestSyncBackendProjectsTick_Failure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "ns", Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&g).Build()
	serviceMock := gmock.Interface{}

	serviceMock.On("GetRestClient", &g).
		Return(nil, errors.New("gerrit client fatal")).Once()

	rcn := Reconcile{
		client:  cl,
		service: &serviceMock,
	}

	err := rcn.syncBackendProjectsTick()
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "gerrit client fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	clientMock := gerritClientMocks.ClientInterface{}
	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)

	clientMock.On("ListProjects", "CODE").
		Return(nil, errors.New("list projects fatal"))

	err = rcn.syncBackendProjectsTick()
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "list projects fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestSyncBackendProjects(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "ns", Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&g).Build()
	serviceMock := gmock.Interface{}

	serviceMock.On("GetRestClient", &g).
		Return(nil, errors.New("gerrit client fatal"))

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		service: &serviceMock,
		log:     &logger,
	}

	go rcn.syncBackendProjects(time.Millisecond)
	time.Sleep(time.Millisecond * 10)

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "gerrit client fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
}
