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
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/epam/edp-common/pkg/mock"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func createClient(instance *gerritApi.Gerrit) client.Client {
	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{})

	return fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()
}

func createGerritByStatus(status string) *gerritApi.Gerrit {
	return &gerritApi.Gerrit{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Gerrit",
			APIVersion: "apps/v1",
		},
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
		log:    logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.Equal(t, errTest, err)
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
		log:    logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.Equal(t, nil, err)
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

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_EmptyClient(t *testing.T) {
	mc := mocks.Client{}
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	_, isMsgFound := log.InfoMessages["instance not found"]

	assert.True(t, isMsgFound)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileGerrit_Reconcile_DeployErr(t *testing.T) {
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)

	cl := createClient(instance)

	errTest := errors.New("test")
	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(false, errTest)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  cl,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	_, ok := log.InfoMessages[fmt.Sprintf("Failed to check Deployment for %v/%v object!",
		instance.Namespace, instance.Name)]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_DeployNotReady(t *testing.T) {
	ctx := context.Background()

	instance := createGerritByStatus(StatusCreated)

	cl := createClient(instance)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(false, nil)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  cl,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	msg := fmt.Sprintf("Deployment for %v/%v object is not ready for configuration yet", instance.Namespace,
		instance.Name)
	_, ok := log.InfoMessages[msg]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, rs)
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

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, errTest, log.LastError())
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
		log:     logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.Equal(t, errTest, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
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

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	_, ok := log.InfoMessages["Restarting deployment after configuration change"]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
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

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	msg := fmt.Sprintf("Failed to check Deployment config for %v/%v Gerrit!", instance.Namespace, instance.Name)
	rs, err := rg.Reconcile(ctx, req)
	_, ok := log.InfoMessages[msg]
	assert.True(t, ok)
	assert.Equal(t, errTest, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
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

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	msg := fmt.Sprintf("Deployment config for %v/%v Gerrit is not ready for configuration yet",
		instance.Namespace, instance.Name)
	rs, err := rg.Reconcile(ctx, req)
	_, ok := log.InfoMessages[msg]
	assert.True(t, ok)
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, rs)
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, errTest)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, errTest, log.LastError())
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.Equal(t, errTest, err)
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_IntegrateErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusExposeFinish)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &gerritApi.Gerrit{}).Return(cl)
	sw.On("Update").Return(nil)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)

	serviceMock := gmock.Interface{}
	serviceMock.On("IsDeploymentReady", instance).Return(true, nil)
	serviceMock.On("Configure", instance).Return(instance, false, nil)
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)
	serviceMock.On("Integrate", instance).Return(instance, errTest)

	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     logr.Discard(),
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.Equal(t, "Integration failed: "+errTest.Error(), err.Error())
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)
	serviceMock.On("Integrate", instance).Return(instance, nil)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, errTest, log.LastError())
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)
	serviceMock.On("Integrate", instance).Return(instance, nil)

	log := &common.Logger{}
	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	msg := fmt.Sprintf("Failed update avalability status for Gerrit object with name %s", instance.Name)
	_, ok := log.InfoMessages[msg]
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
	serviceMock.On("ExposeConfiguration", instance).Return(instance, nil)
	serviceMock.On("Integrate", instance).Return(instance, nil)

	rg := ReconcileGerrit{
		client:  &mc,
		service: &serviceMock,
		log:     logr.Discard(),
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

	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}

	_, err = NewReconcileGerrit(cl, &sch, logr.Discard())
	assert.NoError(t, err)
}
