package helper

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
)

func TestTryToDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
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
	v1alpha1.RegisterTypes(scheme)
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

	gerritService := gmock.Interface{}
	gCl := gmock.ClientInterface{}

	gerritService.On("GetRestClient", &g).Return(&gCl, nil)
	if _, err := GetGerritClient(context.Background(), client, &instance, "", &gerritService); err != nil {
		t.Fatal(err)
	}

	gerritService.AssertExpectations(t)
	gCl.AssertExpectations(t)
}

func TestGetGerritClient_Failure_UnableToGetInstanceOwner(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	instance := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t1",
			Namespace: "t2",
			OwnerReferences: []metav1.OwnerReference{
				{},
			},
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
	gerritService := gmock.Interface{}

	_, err := GetGerritClient(context.Background(), client, &instance, "", &gerritService)
	if err == nil {
		t.Fatal("error is not returned")
	}

	if !strings.Contains(err.Error(), "unable to get instance owner") {
		t.Log(err)
		t.Fatal("wrong error returned")
	}

	gerritService.AssertExpectations(t)
}

func TestGetGerritClient_Failure_NoRootGerrits(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	instance := v1alpha1.GerritGroupMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t1",
			Namespace: "t2",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&instance).Build()
	gerritService := gmock.Interface{}

	_, err := GetGerritClient(context.Background(), client, &instance, "", &gerritService)
	if err == nil {
		t.Fatal("error is not returned")
	}

	if !strings.Contains(err.Error(), "no root gerrits found") {
		t.Fatal("wrong error returned")
	}

	gerritService.AssertExpectations(t)
}

func TestGetGerritClient_Failure_UnableToGetRestClient(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
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

	gerritService := gmock.Interface{}

	gerritService.On("GetRestClient", &g).Return(nil, errors.New("mock error"))
	_, err := GetGerritClient(context.Background(), client, &instance, "", &gerritService)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "mock error") {
		t.Fatal("wrong error returned")
	}

	gerritService.AssertExpectations(t)
}

func TestGetGerritInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.RegisterTypes(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))

	g := v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns", Name: "ger1"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&g).Build()
	ctx := context.Background()
	wrongGerritName := "ger321"

	_, err := GetGerritInstance(ctx, client, &wrongGerritName, g.Namespace)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), `gerrits.v2.edp.epam.com "ger321" not found`) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	if _, err := GetGerritInstance(ctx, client, &g.Name, g.Namespace); err != nil {
		t.Fatal(err)
	}
}

func TestGetWatchNamespace(t *testing.T) {
	ns := "test"
	err := os.Setenv(watchNamespaceEnvVar, ns)
	assert.NoError(t, err)
	namespace, err := GetWatchNamespace()
	assert.NoError(t, err)
	assert.Equal(t, ns, namespace)
	err = os.Unsetenv(watchNamespaceEnvVar)
	assert.NoError(t, err)
}

func TestGetWatchNamespaceErr(t *testing.T) {
	namespace, err := GetWatchNamespace()
	assert.Error(t, err)
	assert.Empty(t, namespace)
}

func TestGetDebugMode(t *testing.T) {
	debugMode := "true"
	err := os.Setenv(debugModeEnvVar, debugMode)
	assert.NoError(t, err)
	mode, err := GetDebugMode()
	assert.NoError(t, err)
	assert.True(t, mode)
	err = os.Unsetenv(debugModeEnvVar)
	assert.NoError(t, err)
}

func TestGetDebugMode_EmptyEnv(t *testing.T) {
	mode, err := GetDebugMode()
	assert.NoError(t, err)
	assert.False(t, mode)
}

func TestGetDebugMode_NotBool(t *testing.T) {
	debugMode := "123"
	err := os.Setenv(debugModeEnvVar, debugMode)
	assert.NoError(t, err)
	mode, err := GetDebugMode()
	assert.Error(t, err)
	assert.False(t, mode)
	err = os.Unsetenv(debugModeEnvVar)
	assert.NoError(t, err)
}

func TestGetPlatformTypeEnv(t *testing.T) {
	ns := "test"
	err := os.Setenv(platformType, ns)
	assert.NoError(t, err)
	namespace := GetPlatformTypeEnv()
	assert.NoError(t, err)
	assert.Equal(t, ns, namespace)
	err = os.Unsetenv(platformType)
	assert.NoError(t, err)
}
