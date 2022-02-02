package gerritgroupmember

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	groupMember := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mem1",
			Namespace: "ns1",
		},
		Spec: v1alpha1.GerritGroupMemberSpec{
			AccountID: "acc1",
			GroupID:   "gr1",
		},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: groupMember.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&groupMember, &g).Build()

	serviceMock := gmock.Interface{}
	clientMock := gmock.ClientInterface{}

	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)
	clientMock.On("AddUserToGroup", groupMember.Spec.GroupID, groupMember.Spec.AccountID).Return(nil)
	clientMock.On("DeleteUserFromGroup", groupMember.Spec.GroupID, groupMember.Spec.AccountID).Return(nil)

	rcn := Reconcile{
		client:  client,
		log:     &helper.Logger{},
		service: &serviceMock,
	}

	nn := types.NamespacedName{
		Name:      groupMember.Name,
		Namespace: groupMember.Namespace}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}

	var updateInstance v1alpha1.GerritGroupMember
	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if updateInstance.Status.Value != helper.StatusOK {
		t.Fatal(updateInstance.Status.Value)
	}

	now := metav1.Time{Time: time.Now()}
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

func TestReconcile_ReconcileFailure1(t *testing.T) {
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
}

func TestReconcile_ReconcileFailure2(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	groupMember := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mem1",
			Namespace: "ns1",
		},
		Spec: v1alpha1.GerritGroupMemberSpec{
			AccountID: "acc1",
			GroupID:   "gr1",
		},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: groupMember.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&groupMember, &g).Build()

	serviceMock := gmock.Interface{}
	clientMock := gmock.ClientInterface{}

	serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)
	clientMock.On("AddUserToGroup", groupMember.Spec.GroupID, groupMember.Spec.AccountID).Return(errors.New("AddUserToGroup fatal"))

	rcn := Reconcile{
		client:  client,
		log:     &helper.Logger{},
		service: &serviceMock,
	}

	nn := types.NamespacedName{
		Name:      groupMember.Name,
		Namespace: groupMember.Namespace}

	if _, err := rcn.Reconcile(context.Background(),
		reconcile.Request{
			NamespacedName: nn}); err != nil {
		t.Fatal(err)
	}

	var updateInstance v1alpha1.GerritGroupMember
	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(updateInstance.Status.Value, "AddUserToGroup fatal") {
		t.Log(updateInstance.Status.Value)
		t.Fatal("wrong instance status")
	}

	serviceMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}

func TestReconcile_IsSpecUpdated(t *testing.T) {
	groupMember := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mem1",
			Namespace: "ns1",
		},
		Spec: v1alpha1.GerritGroupMemberSpec{
			AccountID: "acc1",
			GroupID:   "gr1",
		},
	}

	changed := isSpecUpdated(event.UpdateEvent{
		ObjectOld: &groupMember,
		ObjectNew: &groupMember,
	})

	if changed {
		t.Fatal("isSpecUpdated is wrong")
	}
}

func TestNewReconcile(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	assert.NoError(t, err)
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}
	_, err = NewReconcile(cl, &sch, logr.Discard())
	assert.NoError(t, err)

}

func TestReconcile_UpdateErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := &v1alpha1.GerritGroupMember{
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
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroupMember{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.GerritGroupMember{}).Return(cl)

	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	rg := Reconcile{
		client: &mc,
		log:    logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)

}
