package mergerequest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
)

var (
	rootGerrit   = v1alpha1.Gerrit{ObjectMeta: metav1.ObjectMeta{Name: "gerrit", Namespace: "ns"}}
	mergeRequest = v1alpha1.GerritMergeRequest{ObjectMeta: metav1.ObjectMeta{Name: "mr1",
		Namespace: rootGerrit.Namespace},
		Spec: v1alpha1.GerritMergeRequestSpec{
			SourceBranch: "rev123",
			OwnerName:    rootGerrit.Name,
			ProjectName:  "prjX",
		}}
)

type ControllerTestSuite struct {
	suite.Suite
	gerritService *gmock.Interface
	logger        *helper.Logger
	gitClient     *gmock.GitClient
	gerritClient  *gmock.ClientInterface
	scheme        *runtime.Scheme
}

func (s *ControllerTestSuite) SetupTest() {
	s.scheme = runtime.NewScheme()
	v1alpha1.RegisterTypes(s.scheme)

	s.gerritService = &gmock.Interface{}
	s.logger = &helper.Logger{}
	s.gitClient = &gmock.GitClient{}
	s.gerritClient = &gmock.ClientInterface{}
}

func (s *ControllerTestSuite) TearDownTest() {
	s.gerritService.AssertExpectations(s.T())
	s.gitClient.AssertExpectations(s.T())
	s.gerritClient.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestReconcile() {
	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&rootGerrit, &mergeRequest).Build()
	changeID := "change123"

	s.gitClient.On("Clone", mergeRequest.Spec.ProjectName).Return("path", nil)
	s.gitClient.On("GenerateChangeID").Return(changeID, nil)
	s.gitClient.On("Merge", mergeRequest.Spec.ProjectName, fmt.Sprintf("origin/%s", mergeRequest.Spec.SourceBranch),
		mergeRequest.TargetBranch(), "--no-ff", "-m", fmt.Sprintf("%s\n\nChange-Id: %s", mergeRequest.CommitMessage(), changeID)).
		Return(nil)
	s.gitClient.On("Push", mergeRequest.Spec.ProjectName, "origin", "HEAD:refs/for/master").
		Return("http://gerrit.com/merge/1", nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGitClient: func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			return s.gitClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	err = s.logger.LastError()
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)
}

func (s *ControllerTestSuite) TestReconcileDelete() {
	deleteMergeRequest := mergeRequest.DeepCopy()
	deleteMergeRequest.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	deleteMergeRequest.Status.ChangeID = "change321"

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&rootGerrit, deleteMergeRequest).Build()

	s.gerritClient.On("ChangeGet", deleteMergeRequest.Status.ChangeID).
		Return(&gerritClient.Change{Status: StatusNew}, nil)
	s.gerritClient.On("ChangeAbandon", deleteMergeRequest.Status.ChangeID).Return(nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *v1alpha1.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	err = s.logger.LastError()
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)
}

func (s *ControllerTestSuite) TestReconcileCheckStatus() {
	checkStatusRequest := mergeRequest.DeepCopy()
	checkStatusRequest.Status.ChangeID = "change321"
	checkStatusRequest.Status.Value = StatusNew

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&rootGerrit, checkStatusRequest).Build()

	s.gerritClient.On("ChangeGet", checkStatusRequest.Status.ChangeID).
		Return(&gerritClient.Change{Status: StatusAbandoned}, nil).Once()

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *v1alpha1.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	err = s.logger.LastError()
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), result.RequeueAfter, time.Duration(0))

	var updatedMergeRequest v1alpha1.GerritMergeRequest
	err = rec.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: checkStatusRequest.Name, Namespace: checkStatusRequest.Namespace},
		&updatedMergeRequest)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), updatedMergeRequest.Status.Value,
		StatusAbandoned)
}

func (s *ControllerTestSuite) TestReconcileCheckStatusFailure() {
	checkStatusRequest := mergeRequest.DeepCopy()
	checkStatusRequest.Status.ChangeID = "change321"
	checkStatusRequest.Status.Value = StatusNew

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&rootGerrit, checkStatusRequest).Build()

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *v1alpha1.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	s.gerritClient.On("ChangeGet", checkStatusRequest.Status.ChangeID).
		Return(nil, errors.New("change get fatal")).Once()

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	err = s.logger.LastError()
	assert.Error(s.T(), err)
	assert.EqualError(s.T(), err,
		"unable to get change status: unable to get change id: change get fatal")

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)

	var updatedMergeRequest v1alpha1.GerritMergeRequest
	err = rec.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: checkStatusRequest.Name, Namespace: checkStatusRequest.Namespace},
		&updatedMergeRequest)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), updatedMergeRequest.Status.Value,
		"unable to get change status: unable to get change id: change get fatal")
}

func TestController(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
