package gerritprojectaccess

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	projectAccessInstance := v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "bar",
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

	serviceMock := gerritService.Mock{}
	clientMock := gerritClient.Mock{}

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
}

func TestReconcile_ReconcileFailure(t *testing.T) {
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

	projectAccessInstance := v1alpha1.GerritProjectAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "bar",
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
			Name:            "foo",
			Namespace:       "bar",
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
