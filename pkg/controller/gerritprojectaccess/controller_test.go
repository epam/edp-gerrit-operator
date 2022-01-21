package gerritprojectaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
)

const name = "name"
const namespace = "namespace"

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	projectAccessInstance := v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: v1alpha1.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []v1alpha1.Reference{
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

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: projectAccessInstance.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&projectAccessInstance, &g).Build()

	serviceMock := gmock.Interface{}
	clientMock := gmock.ClientInterface{}

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

	var updateInstance v1alpha1.GerritProjectAccess
	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if updateInstance.Status.Value != helper.StatusOK {
		t.Fatal(updateInstance.Status.Value)
	}

	now := v1.Time{Time: time.Now()}
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
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

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

	projectAccessInstance := v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: v1alpha1.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []v1alpha1.Reference{
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
	projectAccessInstance := v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: v1alpha1.GerritProjectAccessSpec{
			ProjectName: "pro1",
			References: []v1alpha1.Reference{
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
	err := os.Setenv("PLATFORM_TYPE", "test")
	if err != nil {
		assert.NoError(t, err)
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}
	_, err = NewReconcile(cl, &sch, logr.Discard())
	assert.NoError(t, err)

}

func TestReconcileGerrit_Reconcile_UpdateStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}

	ctx := context.Background()

	instance := &v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
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
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritProjectAccess{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &v1alpha1.GerritProjectAccess{}).Return(cl)
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
