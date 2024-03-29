package gerritreplicationconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonmock "github.com/epam/edp-common/pkg/mock"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	gerritController "github.com/epam/edp-gerrit-operator/v2/controllers/gerrit"
	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	pmocks "github.com/epam/edp-gerrit-operator/v2/mock/platform"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	name      = "name"
	namespace = "namespace"
)

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func createGerritReplicationConfig(status string) *gerritApi.GerritReplicationConfig {
	return &gerritApi.GerritReplicationConfig{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: gerritApi.GerritReplicationConfigStatus{
			Status: status,
		},
	}
}

func createGerritByStatus(status string) *gerritApi.Gerrit {
	return &gerritApi.Gerrit{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Gerrit",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: gerritApi.GerritStatus{
			Status: status,
		},
	}
}

func TestReconcileGerritReplicationConfig_Reconcile_GetUnregisteredErr(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritGroup{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_GetErr(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()["instance not found"]
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_GetGerritInstanceErr(t *testing.T) {
	ctx := context.Background()

	instance := createGerritReplicationConfig("")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_UpdateAfterSetOwnerErr(t *testing.T) {
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")

	gerritInstance := createGerritByStatus("")

	var list gerritApi.GerritList

	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	mc.On("Update").Return(errTest)
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("List", &list).Return(cl)

	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_GetInstanceOwnerErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")

	gerritInstance := createGerritByStatus("")

	var list gerritApi.GerritList

	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	sw.On("Update").Return(nil)

	mc.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(errTest).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_Valid(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig(gerritController.StatusReady)

	gerritInstance := createGerritByStatus("")

	var list gerritApi.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.GerritList{}, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	sw.On("Update").Return(nil)

	mc.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_StatusConfiguringUpdateErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig(spec.StatusConfiguring)
	gerritInstance := createGerritByStatus("")

	var list gerritApi.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{},
		&gerritApi.Gerrit{}, &gerritApi.GerritList{})

	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()
	errTest := errors.New("test")

	sw.On("Update").Return(errTest).Once()

	mc.On("Update").Return(nil).Once()
	mc.On("Update").Return(errTest).Once()
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}

	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_StatusUpdate(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig(spec.StatusConfiguring)
	gerritInstance := createGerritByStatus("")

	var list gerritApi.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.Gerrit{},
		&gerritApi.GerritList{})

	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()
	errTest := errors.New("test")

	sw.On("Update").Return(nil)

	mc.On("Update").Return(nil).Once()
	mc.On("Update").Return(errTest).Once()
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_UpdateStatusReadyErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	var list gerritApi.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.Gerrit{},
		&gerritApi.GerritList{})

	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()
	errTest := errors.New("test")

	sw.On("Update").Return(errTest).Once()

	mc.On("Update").Return(nil).Once()
	mc.On("Update").Return(errTest).Once()
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	log := commonmock.NewLogr()
	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		componentService: &gServiceMock,
		log:              log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerritReplicationConfig_Reconcile_configureReplicationErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()
	gServiceMock := gmock.Interface{}
	platformMock := pmocks.PlatformService{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	var list gerritApi.GerritList

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritReplicationConfig{}, &gerritApi.Gerrit{},
		&gerritApi.GerritList{})

	cl := fake.NewClientBuilder().WithObjects(instance, gerritInstance).WithScheme(s).Build()

	errTest := errors.New("test")

	sw.On("Update").Return(nil)

	mc.On("Update").Return(nil).Once()
	mc.On("Update").Return(errTest).Once()
	mc.On("Get", nsn, &gerritApi.GerritReplicationConfig{}).Return(cl)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl).Once()
	mc.On("Status").Return(sw)
	mc.On("List", &list).Return(cl)

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(&coreV1Api.PodList{}, errTest)

	rg := ReconcileGerritReplicationConfig{
		client:           &mc,
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, reconcile.Result{}, rs)
}

func Test_configureReplication_GetGerritSSHUrlErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_GetServicePortErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(0), errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_FirstGetSecretErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(0), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(nil, errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_SecondGetSecretErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(0), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(nil, nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, spec.GerritDefaultVCSKeyName).Return(nil, errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_saveSshReplicationKeyErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(0), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(nil, nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, spec.GerritDefaultVCSKeyName).Return(nil, nil)

	path := fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", "", path, path),
	}

	platformMock.On("ExecInPod", namespace, "", tr).Return(nil, nil, errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_InitNewSshClientErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(0), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(nil, nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, spec.GerritDefaultVCSKeyName).Return(nil, nil)

	path := fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", "", path, path),
	}

	platformMock.On("ExecInPod", namespace, "", tr).Return(nil, nil, nil)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err := rg.configureReplication(instance, gerritInstance)
	assert.Error(t, err)
}

func Test_configureReplication_createReplicationConfigErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	pk, err := rsa.GenerateKey(rand.Reader, 128)
	require.NoError(t, err)

	privkeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)

	keys := map[string][]byte{
		"id_rsa": privkeyPem,
	}

	errTest := errors.New("test")

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("testurl", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(80), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(keys, nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, spec.GerritDefaultVCSKeyName).Return(nil, nil)

	path := fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", "", path, path),
	}

	platformMock.On("ExecInPod", namespace, "", tr).Return(nil, nil, nil)

	tr2 := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath),
	}
	platformMock.On("ExecInPod", namespace, "", tr2).Return(nil, nil, errTest)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err = rg.configureReplication(instance, gerritInstance)
	assert.ErrorIs(t, err, errTest)
}

func Test_configureReplication_updateReplicationConfigErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}
	gServiceMock := gmock.Interface{}

	instance := createGerritReplicationConfig("")
	gerritInstance := createGerritByStatus(gerritController.StatusReady)

	pl := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{
			{},
		},
	}

	pk, err := rsa.GenerateKey(rand.Reader, 128)
	require.NoError(t, err)

	privkeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)

	keys := map[string][]byte{
		"id_rsa": privkeyPem,
	}

	platformMock.On("GetPods", namespace, &metaV1.ListOptions{LabelSelector: fmt.Sprintf("deploymentconfig=%v", name)}).Return(pl, nil)
	gServiceMock.On("GetGerritSSHUrl", gerritInstance).Return("testurl", nil)
	gServiceMock.On("GetServicePort", gerritInstance).Return(int32(80), nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, gerritInstance.Name+"-admin").Return(keys, nil)
	platformMock.On("GetSecret", gerritInstance.Namespace, spec.GerritDefaultVCSKeyName).Return(nil, nil)

	path := fmt.Sprintf("%v/%v", spec.GerritDefaultVCSKeyPath, spec.GerritDefaultVCSKeyName)
	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("echo \"%v\" > %v && chmod 600 %v", "", path, path),
	}

	platformMock.On("ExecInPod", namespace, "", tr).Return(nil, nil, nil)

	tr2 := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || printf '%%s\n  %%s\n' '[gerrit]' 'defaultForceUpdate = true' > %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath, spec.DefaultGerritReplicationConfigPath),
	}
	platformMock.On("ExecInPod", namespace, "", tr2).Return(nil, nil, nil)

	rg := ReconcileGerritReplicationConfig{
		platform:         &platformMock,
		componentService: &gServiceMock,
		log:              logr.Discard(),
	}

	err = rg.configureReplication(instance, gerritInstance)
	assert.Error(t, err)
}

func Test_reloadReplicationPluginErr(t *testing.T) {
	gclient := gerritClientMocks.ClientInterface{}

	errTest := errors.New("test")

	gclient.On("ReloadPlugin", "replication").Return(errTest)

	rg := ReconcileGerritReplicationConfig{
		log: logr.Discard(),
	}
	err := rg.reloadReplicationPlugin(&gclient)

	assert.ErrorIs(t, err, errTest)
}

func Test_reloadReplicationPlugin(t *testing.T) {
	gclient := gerritClientMocks.ClientInterface{}

	gclient.On("ReloadPlugin", "replication").Return(nil)

	rg := ReconcileGerritReplicationConfig{
		log: logr.Discard(),
	}
	err := rg.reloadReplicationPlugin(&gclient)

	assert.NoError(t, err)
}

func TestReconcileGerritReplicationConfig_Reconcile(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	require.NoError(t, err)

	fClient := fake.NewClientBuilder().Build()
	sc := &runtime.Scheme{}

	_, err = NewReconcileGerritReplicationConfig(fClient, sc, logr.Discard())
	assert.NoError(t, err)

	err = os.Unsetenv("PLATFORM_TYPE")
	assert.NoError(t, err)
}

func Test_createSshConfigErr(t *testing.T) {
	platformMock := pmocks.PlatformService{}

	errTest := errors.New("test")

	rg := ReconcileGerritReplicationConfig{
		platform: &platformMock,
		log:      logr.Discard(),
	}

	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath+"/config"),
	}

	platformMock.On("ExecInPod", "", "", tr).Return(nil, nil, errTest)

	err := rg.createSshConfig("", "")

	assert.ErrorIs(t, err, errTest)
}

func Test_createSshConfig(t *testing.T) {
	platformMock := pmocks.PlatformService{}

	rg := ReconcileGerritReplicationConfig{
		platform: &platformMock,
		log:      logr.Discard(),
	}

	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath+"/config"),
	}

	platformMock.On("ExecInPod", "", "", tr).Return(nil, nil, nil)

	err := rg.createSshConfig("", "")

	assert.NoError(t, err)
}

func Test_updateSshConfig(t *testing.T) {
	platformMock := pmocks.PlatformService{}

	rg := ReconcileGerritReplicationConfig{
		platform: &platformMock,
		log:      logr.Discard(),
	}

	tr := []string{
		"/bin/sh", "-c",
		fmt.Sprintf("[[ -f %v ]] || mkdir -p %v && touch %v && chown -R gerrit2:gerrit2 %v",
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath,
			spec.DefaultGerritSSHConfigPath+"/config", spec.DefaultGerritSSHConfigPath+"/config"),
	}

	platformMock.On("ExecInPod", name, "", tr).Return(nil, nil, nil)

	config := &gerritApi.GerritReplicationConfig{
		Spec: gerritApi.GerritReplicationConfigSpec{SSHUrl: "@:"},
	}

	err := rg.updateSshConfig(name, "", "", "", config)

	assert.Error(t, err)
}

func Test_updateAvailableStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	errTest := errors.New("test")

	mc.On("Update").Return(errTest)
	mc.On("Status").Return(sw).Once()
	sw.On("Update").Return(errTest)

	rg := ReconcileGerritReplicationConfig{
		client: &mc,
		log:    logr.Discard(),
	}

	err := rg.updateAvailableStatus(ctx, &gerritApi.GerritReplicationConfig{}, true)
	assert.ErrorIs(t, err, errTest)
}
