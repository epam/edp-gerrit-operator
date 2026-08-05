package gerrit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonmock "github.com/epam/edp-common/pkg/mock"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
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

func createClient(instance *gerritApi.Gerrit) client.Client {
	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{})

	return fake.NewClientBuilder().WithStatusSubresource(&gerritApi.Gerrit{}).WithObjects(instance).WithScheme(s).Build()
}

func createGerritByStatus(status string) *gerritApi.Gerrit {
	return &gerritApi.Gerrit{
		Spec: gerritApi.GerritSpec{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: gerritApi.GerritStatus{
			Status: status,
		},
	}
}

func TestReconcileGerrit_Reconcile_UpdateInstallStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusInstall)
	cl := createClient(instance)

	errTest := errors.New("test")

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	rg := ReconcileGerrit{
		client: &mc,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateInstallStatus(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusInstall)

	cl := createClient(instance)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	rg := ReconcileGerrit{
		client: &mc,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateEmptyStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus("")

	cl := createClient(instance)

	errTest := errors.New("test")

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client: &mc,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)
	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_EmptyClient(t *testing.T) {
	mc := mocks.Client{}
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithStatusSubresource(&gerritApi.Gerrit{}).WithObjects().WithScheme(s).Build()

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client: &mc,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, isMsgFound := loggerSink.InfoMessages()["instance not found"]

	assert.True(t, isMsgFound)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_DeployErr(t *testing.T) {
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)

	cl := createClient(instance)

	errTest := errors.New("test")
	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(false, errTest)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  cl,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()[fmt.Sprintf("Failed to check Deployment for %v/%v object!",
		instance.Namespace, instance.Name)]

	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_DeployNotReady(t *testing.T) {
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)

	cl := createClient(instance)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(false, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  cl,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)
	msg := fmt.Sprintf("Deployment for %v/%v object is not ready for configuration yet", instance.Namespace,
		instance.Name)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()[msg]

	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateCreatedStatus(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)
	cl := createClient(instance)

	errTest := errors.New("test")

	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_ConfigureErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	errTest := errors.New("test")

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, true, errTest)

	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_ConfigureDPatched(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, true, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()["Restarting deployment after configuration change"]

	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_IsDeploymentReadyErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	errTest := errors.New("test")

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil).Once()
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("IsDeploymentReady", instance).Return(true, errTest)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	msg := fmt.Sprintf("Failed to check Deployment config for %v/%v Gerrit!", instance.Namespace, instance.Name)
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()[msg]

	assert.True(t, ok)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_IsDeploymentReadyFalse(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil).Once()
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("IsDeploymentReady", instance).Return(false, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	msg := fmt.Sprintf("Deployment config for %v/%v Gerrit is not ready for configuration yet",
		instance.Namespace, instance.Name)
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()[msg]

	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_ExposeConfigurationErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	errTest := errors.New("test")

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil).Once()
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, errTest)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusExposeStartErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusExposeStart)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusExposeFinishErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusExposeFinish)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusIntegrationStartErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusIntegrationStart)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateAvailableStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusIntegrationStart)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)

	sw.On("Update").Return(nil).Once()
	mc.On("Status").Return(sw)

	sw.On("Update").Return(errTest).Once()
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest).Once()

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), req)
	msg := fmt.Sprintf("Failed update availability status for Gerrit object with name %s", instance.Name)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, ok = loggerSink.InfoMessages()[msg]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_Valid(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusIntegrationStart)
	cl := createClient(instance)

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)

	sw.On("Update").Return(nil)
	mc.On("Status").Return(sw)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestNewReconcileGerrit(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	require.NoError(t, err)

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritGroup{}, &gerritApi.GerritList{}, &gerritApi.Gerrit{})

	cl := fake.NewClientBuilder().WithStatusSubresource(&gerritApi.Gerrit{}).WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}

	_, err = NewReconcileGerrit(cl, &sch, logr.Discard())
	assert.NoError(t, err)
}

func TestNewReconcileGerrit_UnknownPlatformErr(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "unknown-platform")

	_, err := NewReconcileGerrit(nil, runtime.NewScheme(), logr.Discard())
	assert.Error(t, err)
}

func TestReconcileGerrit_SetupWithManager(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, gerritApi.AddToScheme(s))

	mgr, err := ctrl.NewManager(&rest.Config{Host: "http://127.0.0.1:1"}, ctrl.Options{
		Scheme:  s,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	require.NoError(t, err)

	rg := &ReconcileGerrit{}
	assert.NoError(t, rg.SetupWithManager(mgr))
}

func TestReconcileGerrit_SetupWithManager_KindNotRegisteredErr(t *testing.T) {
	mgr, err := ctrl.NewManager(&rest.Config{Host: "http://127.0.0.1:1"}, ctrl.Options{
		Scheme:  runtime.NewScheme(),
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	require.NoError(t, err)

	rg := &ReconcileGerrit{}
	err = rg.SetupWithManager(mgr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to setup Gerrit controller")
}

func TestReconcileGerrit_Reconcile_GetErr(t *testing.T) {
	mc := mocks.Client{}
	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(errTest)

	rg := ReconcileGerrit{
		client: &mc,
	}
	rs, err := rg.Reconcile(context.Background(), reconcile.Request{NamespacedName: nsn})

	assert.ErrorIs(t, err, errTest)
	assert.Contains(t, err.Error(), "failed Get Gerrit CR")
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_ConfigureUserNotFound(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfigured)
	cl := createClient(instance)

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(nil)
	mc.On("Status").Return(sw)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).
		Return(instance, false, gerritService.UserNotFoundError("user not found"))
	serviceMock.On("ExposeConfiguration", mock.Anything, instance).Return(instance, nil)

	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	rs, err := rg.Reconcile(ctx, reconcile.Request{NamespacedName: nsn})

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 60 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusConfiguredErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfiguring)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), reconcile.Request{NamespacedName: nsn})

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusExposeStartFromConfiguredErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusConfigured)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(errTest)
	mc.On("Status").Return(sw)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)

	log := commonmock.NewLogr()
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
	}
	rs, err := rg.Reconcile(ctrl.LoggerInto(ctx, log), reconcile.Request{NamespacedName: nsn})

	assert.NoError(t, err)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	assert.ErrorIs(t, loggerSink.LastError(), errTest)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_UpdateAvailableStatus_NoChange(t *testing.T) {
	instance := createGerritByStatus(StatusReady)
	instance.Status.Available = true

	rg := ReconcileGerrit{}

	assert.NoError(t, rg.updateAvailableStatus(context.Background(), instance, true))
}

func TestReconcileGerrit_UpdateStatusWithRetry_Conflict(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)
	cl := createClient(instance)

	conflictErr := k8sErrors.NewConflict(
		schema.GroupResource{Group: "v2.edp.epam.com", Resource: "gerrits"}, name, errors.New("conflict"))

	sw.On("Update").Return(conflictErr).Once()
	sw.On("Update").Return(nil).Once()
	mc.On("Status").Return(sw)
	mc.On("Get", nsn, instance).Return(cl)

	rg := ReconcileGerrit{client: &mc}

	err := rg.updateStatusWithRetry(ctx, instance, func() {
		instance.Status.Status = StatusReady
	})

	assert.NoError(t, err)
	sw.AssertNumberOfCalls(t, "Update", 2)
}

func TestReconcileGerrit_UpdateStatusWithRetry_ConflictGetErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)

	conflictErr := k8sErrors.NewConflict(
		schema.GroupResource{Group: "v2.edp.epam.com", Resource: "gerrits"}, name, errors.New("conflict"))
	errTest := errors.New("get failed")

	sw.On("Update").Return(conflictErr)
	mc.On("Status").Return(sw)
	mc.On("Get", nsn, instance).Return(errTest)

	rg := ReconcileGerrit{client: &mc}

	err := rg.updateStatusWithRetry(ctx, instance, func() {
		instance.Status.Status = StatusReady
	})

	assert.ErrorIs(t, err, errTest)
}
