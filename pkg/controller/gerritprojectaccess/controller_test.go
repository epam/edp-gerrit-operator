package gerritprojectaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsV1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	projectAccessInstance := gerritApi.GerritProjectAccess{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metaV1.OwnerReference{},
		},
		Spec: gerritApi.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []gerritApi.Reference{
				{
					Pattern:        "refs/heads/*",
					PermissionName: "label-Code-Review",
					GroupName:      "important-group",
					Min:            -2,
					Max:            2,
					Force:          false,
					Action:         "ALLOW",
				},
			},
			Parent: "lalka",
		},
	}

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: projectAccessInstance.Namespace, Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&projectAccessInstance, &g).Build()

	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)
	clientMock.On("AddAccessRights", projectAccessInstance.Spec.ProjectName,
		prepareAccessInfo(projectAccessInstance.Spec.References)).Return(nil)
	clientMock.On("UpdateAccessRights", projectAccessInstance.Spec.ProjectName,
		prepareAccessInfo(projectAccessInstance.Spec.References)).Return(nil)
	clientMock.On("SetProjectParent", projectAccessInstance.Spec.ProjectName,
		projectAccessInstance.Spec.Parent).Return(nil)
	clientMock.On("DeleteAccessRights", projectAccessInstance.Spec.ProjectName,
		prepareAccessInfo(projectAccessInstance.Spec.References)).Return(nil)

	rcn := Reconcile{
		client:  client,
		log:     &helper.Logger{},
		service: &serviceMock,
	}

	nn := types.NamespacedName{
		Name:      projectAccessInstance.Name,
		Namespace: projectAccessInstance.Namespace}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}

	var updateInstance gerritApi.GerritProjectAccess
	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if updateInstance.Status.Value != helper.StatusOK {
		t.Fatal(updateInstance.Status.Value)
	}

	now := metaV1.Time{Time: time.Now()}
	updateInstance.DeletionTimestamp = &now

	if err := client.Update(context.Background(), &updateInstance); err != nil {
		t.Fatal(err)
	}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}

	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if updateInstance.Status.Value != helper.StatusOK {
		t.Fatal(updateInstance.Status.Value)
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestReconcile_ReconcileFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1Api.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	rcn := Reconcile{
		client: client,
		log:    &helper.Logger{},
	}

	nn := types.NamespacedName{
		Name:      "foo",
		Namespace: "bar"}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}

	projectAccessInstance := gerritApi.GerritProjectAccess{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metaV1.OwnerReference{},
		},
		Spec: gerritApi.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []gerritApi.Reference{
				{
					Pattern:        "refs/heads/*",
					PermissionName: "label-Code-Review",
					GroupName:      "important-group",
					Min:            -2,
					Max:            2,
					Force:          false,
					Action:         "ALLOW",
				},
			},
			Parent: "lalka",
		},
	}

	client = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&projectAccessInstance).Build()

	rcn = Reconcile{
		client: client,
		log:    &helper.Logger{},
	}

	nn = types.NamespacedName{
		Name:      projectAccessInstance.Name,
		Namespace: projectAccessInstance.Namespace}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}
}

func TestReconcile_IsSpecUpdated(t *testing.T) {
	projectAccessInstance := gerritApi.GerritProjectAccess{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metaV1.OwnerReference{},
		},
		Spec: gerritApi.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []gerritApi.Reference{
				{
					Pattern:        "refs/heads/*",
					PermissionName: "label-Code-Review",
					GroupName:      "important-group",
					Min:            -2,
					Max:            2,
					Force:          false,
					Action:         "ALLOW",
				},
			},
			Parent: "lalka",
		},
	}

	changed := isSpecUpdated(event.UpdateEvent{
		ObjectOld: &projectAccessInstance,
		ObjectNew: &projectAccessInstance,
	})

	if changed {
		t.Fatal("isSpecUpdated is wrong")
	}
}

func TestNewReconcile(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	if err != nil {
		t.Fatal(err)
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritGroup{}, &gerritApi.GerritList{}, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}
	_, err = NewReconcile(cl, &sch, logr.Discard())
	assert.NoError(t, err)
}

func TestReconcileGerrit_Reconcile_UpdateStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}

	ctx := context.Background()

	instance := &gerritApi.GerritProjectAccess{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metaV1.OwnerReference{
				{APIVersion: "test"},
			},
		},
	}

	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritProjectAccess{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &gerritApi.GerritProjectAccess{}).Return(cl)
	mc.On("Status").Return(sw)

	logger := helper.Logger{}

	rg := Reconcile{
		client: &mc,
		log:    &logger,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)

	err = logger.LastError()
	assert.Error(t, err)
	assert.ErrorIs(t, err, errTest)

	sw.AssertExpectations(t)
	mc.AssertExpectations(t)
}
