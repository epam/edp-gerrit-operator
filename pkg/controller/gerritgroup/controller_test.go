package gerritgroup

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	common "github.com/epam/edp-common/pkg/mock"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"

func createGerritGroupByOwner(owner []metav1.OwnerReference) *v1alpha1.GerritGroup {
	return &v1alpha1.GerritGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: owner,
		},
	}
}

func createGerrit() *v1alpha1.Gerrit {
	return &v1alpha1.Gerrit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gerrit",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.GerritSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func TestReconcileGerrit_Reconcile_GerErr(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := Reconcile{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	_, ok := log.InfoMessages["instance not found"]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_tryToReconcileErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritGroupByOwner([]metav1.OwnerReference{{APIVersion: "test"}})

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	log := &common.Logger{}
	rg := Reconcile{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, "unable to get instance owner: gerrit replication config cr does not have gerrit cr "+
		"owner references", log.LastError().Error())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	instance := createGerritGroupByOwner([]metav1.OwnerReference{{APIVersion: "test"}})

	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)

	log := &common.Logger{}
	rg := Reconcile{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_ListErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritGroupByOwner(nil)
	var list v1alpha1.GerritList
	listOpts := client.ListOptions{Namespace: namespace}

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	mc.On("List", &list).Return(cl.List(ctx, &list, &listOpts))

	log := &common.Logger{}
	rg := Reconcile{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_ListEmpty(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritGroupByOwner(nil)
	var list v1alpha1.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	mc.On("List", &list).Return(cl)

	log := &common.Logger{}
	rg := Reconcile{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, "unable to get gerrit instance: no root gerrits found", log.LastError().Error())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_GetRestClientErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}
	gClientMock := gmock.ClientInterface{}

	instance := createGerritGroupByOwner(nil)
	var list v1alpha1.GerritList
	gerritInstance := createGerrit()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	errTest := errors.New("test")

	gServiceMock.On("GetRestClient", gerritInstance).Return(&gClientMock, errTest)

	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	mc.On("List", &list).Return(cl)

	log := &common.Logger{}
	rg := Reconcile{
		client:  &mc,
		service: &gServiceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, "unable to get rest client: test", log.LastError().Error())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_CreateGroupErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}
	gClientMock := gmock.ClientInterface{}

	instance := createGerritGroupByOwner(nil)
	var list v1alpha1.GerritList
	gerritInstance := createGerrit()
	Group := &gerrit.Group{}

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	errTest := errors.New("test")

	gClientMock.On("CreateGroup", instance.Spec.Name, instance.Spec.Description,
		instance.Spec.VisibleToAll).Return(Group, errTest)
	gServiceMock.On("GetRestClient", gerritInstance).Return(&gClientMock, nil)
	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)
	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)
	mc.On("List", &list).Return(cl)

	log := &common.Logger{}
	rg := Reconcile{
		client:  &mc,
		service: &gServiceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, "unable to create group: test", log.LastError().Error())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_CreateGroup(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	gServiceMock := gmock.Interface{}
	gClientMock := gmock.ClientInterface{}
	instance := createGerritGroupByOwner(nil)
	var list v1alpha1.GerritList

	gerritInstance := &v1alpha1.Gerrit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gerrit",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	Group := gerrit.Group{}
	gClientMock.On("CreateGroup", instance.Spec.Name, instance.Spec.Description,
		instance.Spec.VisibleToAll).Return(&Group, nil)
	gServiceMock.On("GetRestClient", gerritInstance).Return(&gClientMock, nil)
	mc.On("Get", nsn, &v1alpha1.GerritGroup{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	mc.On("List", &list).Return(cl)

	log := &common.Logger{}
	rg := Reconcile{
		client:  &mc,
		service: &gServiceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func Test_isSpecUpdatedFalse(t *testing.T) {
	oldI := &v1alpha1.GerritGroup{}
	newI := &v1alpha1.GerritGroup{}
	e := event.UpdateEvent{
		ObjectOld: oldI,
		ObjectNew: newI,
	}
	assert.False(t, isSpecUpdated(e))
}

func TestNewReconcile(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}
	_, err = NewReconcile(cl, &sch, logr.Discard())
	assert.NoError(t, err)
}
