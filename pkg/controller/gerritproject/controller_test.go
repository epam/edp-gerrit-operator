package gerritproject

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	coreV1Api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
)

func TestReconcile_Reconcile_CreateProject(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1"},
		Spec:       gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

	clientMock.On("GetProject", prj.Spec.Name).Return(nil, gerritClient.DoesNotExistError("")).Once()
	clientMock.On("CreateProject", &gerritClient.Project{Name: prj.Spec.Name}).Return(nil).Once()
	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		log:     &logger,
		service: &serviceMock,
	}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatalf("%+v", err)
	}

	clientMock.On("GetProject", prj.Spec.Name).Return(nil, gerritClient.DoesNotExistError("")).Once()
	clientMock.On("CreateProject", &gerritClient.Project{Name: prj.Spec.Name}).Return(errors.New("create fatal")).Once()

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "create fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}

	clientMock.On("GetProject", prj.Spec.Name).
		Return(nil, errors.New("unknown get fatal")).Once()

	_, err = rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}})
	require.NoError(t, err)

	err = logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "unknown get fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestReconcile_Reconcile_UpdateProject(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

	clientMock.On("GetProject", prj.Spec.Name).Return(&gerritClient.Project{}, nil)
	clientMock.On("UpdateProject", &gerritClient.Project{Name: prj.Spec.Name}).Return(nil).Once()
	clientMock.On("DeleteProject", prj.Spec.Name).Return(nil).Once()
	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		log:     &logger,
		service: &serviceMock,
	}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatalf("%+v", err)
	}

	clientMock.On("UpdateProject", &gerritClient.Project{Name: prj.Spec.Name}).Return(errors.New("update fatal")).Once()

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "update fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}

	clientMock.On("UpdateProject", &gerritClient.Project{Name: prj.Spec.Name}).Return(nil).Once()
	clientMock.On("DeleteProject", prj.Spec.Name).Return(errors.New("deletion fatal")).Once()

	_, err = rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}})
	require.NoError(t, err)

	err = logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "deletion fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestIsSpecUpdated(t *testing.T) {
	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	if isSpecUpdated(event.UpdateEvent{ObjectOld: &prj, ObjectNew: &prj}) {
		t.Fatal("spec is updated")
	}
}

func TestReconcile_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj).Build()
	logger := helper.Logger{}

	rcn := Reconcile{
		client: cl,
		log:    &logger,
	}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: "nm", Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	if _, ok := logger.InfoMessages["instance not found"]; !ok {
		t.Fatal("not found message is not logged")
	}
}

func TestReconcile_Reconcile_FailureGetClient(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()}},
		Spec: gerritApi.GerritProjectSpec{Name: "sprj1"},
	}

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gmock.Interface{}
	serviceMock.On("GetRestClient", &g).Return(nil, errors.New("no g client"))

	logger := helper.Logger{}

	rcn := Reconcile{
		client:  cl,
		log:     &logger,
		service: &serviceMock,
	}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	err := logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "no g client") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}

	serviceMock.AssertExpectations(t)
}
