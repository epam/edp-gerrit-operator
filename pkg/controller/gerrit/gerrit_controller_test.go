package gerrit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	common "github.com/epam/edp-common/pkg/mock"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func createClient(instance *v1alpha1.Gerrit) client.Client {
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{})
	return fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()
}

func createGerritByStatus(status string) *v1alpha1.Gerrit {
	return &v1alpha1.Gerrit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gerrit",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.GerritSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: v1alpha1.GerritStatus{
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_EmptyClient(t *testing.T) {
	mc := mocks.Client{}
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)

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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateStatusExposeFinishErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusExposeFinish)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, errTest, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_IntegrateErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusExposeFinish)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)
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
	assert.Equal(t, errTest, log.LastError())
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileGerrit_Reconcile_UpdateAvailableStatusErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := createGerritByStatus(StatusIntegrationStart)
	cl := createClient(instance)

	errTest := errors.New("test")

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)

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

	mc.On("Get", nsn, &v1alpha1.Gerrit{}).Return(cl)

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
	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestNewReconcileGerrit(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", platform.Test)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.GerritGroup{}, &v1alpha1.GerritList{}, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}
	_, err = NewReconcileGerrit(cl, &sch, logr.Discard())
	assert.NoError(t, err)

}
