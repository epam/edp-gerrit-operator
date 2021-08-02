package gerritgroupmember

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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

	serviceMock := gerritService.Mock{}
	clientMock := gerritClient.Mock{}

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
}

func TestReconcile_ReconcileFailure1(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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

	serviceMock := gerritService.Mock{}
	clientMock := gerritClient.Mock{}

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
