package gerritproject

import (
	"context"
	"strings"
	"testing"
	"time"

	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_Reconcile_CreateProject(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	prj := v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "prj1"},
		Spec:       v1alpha1.GerritProjectSpec{Name: "sprj1"},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gerritService.Mock{}
	clientMock := gmock.ClientInterface{}

	clientMock.On("GetProject", prj.Spec.Name).Return(nil, gerritClient.ErrDoesNotExist("")).Once()
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
	clientMock.On("GetProject", prj.Spec.Name).Return(nil, gerritClient.ErrDoesNotExist("")).Once()
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
	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	err = logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "unknown get fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}
}

func TestReconcile_Reconcile_UpdateProject(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	prj := v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()}},
		Spec: v1alpha1.GerritProjectSpec{Name: "sprj1"},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gerritService.Mock{}
	clientMock := gmock.ClientInterface{}

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

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{
			Name: prj.Name, Namespace: prj.Namespace}}); err != nil {
		t.Fatal(err)
	}

	err = logger.LastError()
	if err == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(err.Error(), "deletion fatal") {
		t.Fatalf("wrong error returnded: %s", err.Error())
	}
}

func TestIsSpecUpdated(t *testing.T) {
	prj := v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()}},
		Spec: v1alpha1.GerritProjectSpec{Name: "sprj1"},
	}

	if isSpecUpdated(event.UpdateEvent{ObjectOld: &prj, ObjectNew: &prj}) {
		t.Fatal("spec is updated")
	}
}

func TestReconcile_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	prj := v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()}},
		Spec: v1alpha1.GerritProjectSpec{Name: "sprj1"},
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
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	prj := v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "prj1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()}},
		Spec: v1alpha1.GerritProjectSpec{Name: "sprj1"},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prj.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&prj, &g).Build()
	serviceMock := gerritService.Mock{}
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
}
