package helper

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTryToDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	instance := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t1",
			Namespace: "t2",
		},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&instance, &g).Build()

	if err := TryToDelete(context.Background(), client, &instance, "fin1", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	instance.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	if err := TryToDelete(context.Background(), client, &instance, "fin1", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	err := TryToDelete(context.Background(), client, &instance, "fin1", func() error {
		return errors.New("try del fatal")
	})
	if err == nil {
		t.Fatal("fatal not returned")
	}
	if errors.Cause(err).Error() != "try del fatal" {
		t.Fatal("wrong error returned")
	}
}

func TestGetGerritClient(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	instance := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t1",
			Namespace: "t2",
		},
	}

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace, Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&instance, &g).Build()

	gSVC := gerritService.Mock{}
	gCl := gerritClient.Mock{}

	gSVC.On("GetRestClient", &g).Return(&gCl, nil)
	if _, err := GetGerritClient(context.Background(), client, &instance, "", &gSVC); err != nil {
		t.Fatal(err)
	}
}
