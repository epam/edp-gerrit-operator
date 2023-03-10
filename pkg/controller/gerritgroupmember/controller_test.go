package gerritgroupmember

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-gerrit-operator/v2/mock"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	name      = "name"
	namespace = "namespace"
)

func TestReconcile_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1.AddToScheme(scheme))

	groupMember := gerritApi.GerritGroupMember{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "mem1",
			Namespace: "ns1",
		},
		Spec: gerritApi.GerritGroupMemberSpec{
			AccountID: "acc1",
			GroupID:   "gr1",
		},
	}

	g := gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: groupMember.Namespace, Name: "ger1"},
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "Gerrit",
		}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&groupMember, &g).Build()

	serviceMock := gmock.Interface{}
	clientMock := gerritClientMocks.ClientInterface{}

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

	var updateInstance gerritApi.GerritGroupMember
	if err := client.Get(context.Background(), nn, &updateInstance); err != nil {
		t.Fatal(err)
	}

	if updateInstance.Status.Value != helper.StatusOK {
		t.Fatal(updateInstance.Status.Value)
	}

	now := metaV1.Time{Time: time.Now()}
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
	t.Parallel()

	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	rcn := Reconcile{
		client: client,
		log:    &helper.Logger{},
	}

	nn := types.NamespacedName{
		Name:      "foo",
		Namespace: "bar",
	}

	_, err := rcn.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})

	assert.NoError(t, err)
}

func TestReconcile_ReconcileFailure2(t *testing.T) {
	scheme := runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(scheme))
	utilRuntime.Must(coreV1.AddToScheme(scheme))

	tests := []struct {
		name            string
		prepareFunc     func(t *testing.T)
		wantRequeueTime time.Duration
	}{
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is not set",
			prepareFunc:     func(t *testing.T) {},
			wantRequeueTime: time.Duration(float64(time.Second) * math.Pow(math.E, 1.0)),
		},
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is empty",
			prepareFunc:     func(t *testing.T) { t.Setenv("GERRIT_GROUP_MEMBER_SYNC_INTERVAL", "") },
			wantRequeueTime: time.Duration(float64(time.Second) * math.Pow(math.E, 1.0)),
		},
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is not a time value",
			prepareFunc:     func(t *testing.T) { t.Setenv("GERRIT_GROUP_MEMBER_SYNC_INTERVAL", "foo") },
			wantRequeueTime: time.Duration(float64(time.Second) * math.Pow(math.E, 1.0)),
		},
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is a negative number",
			prepareFunc:     func(t *testing.T) { t.Setenv("GERRIT_GROUP_MEMBER_SYNC_INTERVAL", "-1h") },
			wantRequeueTime: time.Duration(float64(time.Second) * math.Pow(math.E, 1.0)),
		},
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is a zero number",
			prepareFunc:     func(t *testing.T) { t.Setenv("GERRIT_GROUP_MEMBER_SYNC_INTERVAL", "0h") },
			wantRequeueTime: time.Duration(float64(time.Second) * math.Pow(math.E, 1.0)),
		},
		{
			name:            "GERRIT_GROUP_MEMBER_SYNC_INTERVAL env var is correct",
			prepareFunc:     func(t *testing.T) { t.Setenv("GERRIT_GROUP_MEMBER_SYNC_INTERVAL", "30m") },
			wantRequeueTime: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareFunc(t)

			groupMember := gerritApi.GerritGroupMember{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "mem1",
					Namespace: "ns1",
				},
				Spec: gerritApi.GerritGroupMemberSpec{
					AccountID: "acc1",
					GroupID:   "gr1",
				},
			}

			g := gerritApi.Gerrit{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: groupMember.Namespace, Name: "ger1"},
				TypeMeta: metaV1.TypeMeta{
					APIVersion: "v2.edp.epam.com/v1",
					Kind:       "Gerrit",
				},
			}

			client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&groupMember, &g).Build()

			nn := types.NamespacedName{
				Name:      groupMember.Name,
				Namespace: groupMember.Namespace}
			serviceMock := gmock.Interface{}
			clientMock := gerritClientMocks.ClientInterface{}

			serviceMock.On("GetRestClient", &g).Return(&clientMock, nil)
			clientMock.On("AddUserToGroup", groupMember.Spec.GroupID, groupMember.Spec.AccountID).Return(errors.New("AddUserToGroup fatal"))

			rcn := Reconcile{
				client:  client,
				log:     &helper.Logger{},
				service: &serviceMock,
			}

			result, err := rcn.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})

			assert.NoError(t, err)
			assert.Equal(t, tt.wantRequeueTime, result.RequeueAfter)

			var updateInstance gerritApi.GerritGroupMember

			assert.NoError(t, client.Get(context.Background(), nn, &updateInstance))

			assert.Contains(t, updateInstance.Status.Value, "AddUserToGroup fatal")

			serviceMock.AssertExpectations(t)
			clientMock.AssertExpectations(t)
		})
	}
}

func TestReconcile_IsSpecUpdated(t *testing.T) {
	groupMember := gerritApi.GerritGroupMember{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "mem1",
			Namespace: "ns1",
		},
		Spec: gerritApi.GerritGroupMemberSpec{
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
	require.NoError(t, err)

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritGroup{}, &gerritApi.GerritList{}, &gerritApi.Gerrit{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()
	sch := runtime.Scheme{}

	_, err = NewReconcile(cl, &sch, logr.Discard())
	assert.NoError(t, err)
}

func TestReconcile_UpdateErr(t *testing.T) {
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	ctx := context.Background()

	instance := &gerritApi.GerritGroupMember{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metaV1.OwnerReference{
				{APIVersion: "test"},
			},
		},
		Status: gerritApi.GerritGroupMemberStatus{
			FailureCount: 1,
		},
	}

	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.GerritGroupMember{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	mc.On("Get", nsn, &gerritApi.GerritGroupMember{}).Return(cl)

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
	requeueTime := helper.SetFailureCount(instance)
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, nil, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: requeueTime}, rs)
}
