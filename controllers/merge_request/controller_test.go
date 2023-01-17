package mergerequest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonmock "github.com/epam/edp-common/pkg/mock"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/edp/v1"
	"github.com/epam/edp-gerrit-operator/v2/controllers/helper"
	gmock "github.com/epam/edp-gerrit-operator/v2/mock/gerrit"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/git"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
)

type ControllerTestSuite struct {
	suite.Suite
	gerritService *gmock.Interface
	logger        logr.Logger
	gitClient     *gmock.GitClient
	gerritClient  *gerritClientMocks.ClientInterface
	scheme        *runtime.Scheme
	rootGerrit    *gerritApi.Gerrit
	mergeRequest  *gerritApi.GerritMergeRequest
}

func (s *ControllerTestSuite) SetupTest() {
	s.scheme = runtime.NewScheme()
	utilRuntime.Must(gerritApi.AddToScheme(s.scheme))

	s.gerritService = &gmock.Interface{}
	s.logger = commonmock.NewLogr()
	s.gitClient = &gmock.GitClient{}
	s.gerritClient = &gerritClientMocks.ClientInterface{}

	s.rootGerrit = &gerritApi.Gerrit{ObjectMeta: metaV1.ObjectMeta{Name: "gerrit", Namespace: "ns"}}
	s.mergeRequest = &gerritApi.GerritMergeRequest{ObjectMeta: metaV1.ObjectMeta{Name: "mr1",
		Namespace: s.rootGerrit.Namespace},
		Spec: gerritApi.GerritMergeRequestSpec{
			SourceBranch:        "rev123",
			OwnerName:           s.rootGerrit.Name,
			ProjectName:         "prjX",
			AuthorEmail:         "john.doe@example.com",
			AuthorName:          "John Doe",
			AdditionalArguments: []string{"-q"},
		}}
}

func (s *ControllerTestSuite) TearDownTest() {
	s.gerritService.AssertExpectations(s.T())
	s.gitClient.AssertExpectations(s.T())
	s.gerritClient.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestReconcileSetAuthorFailure() {
	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit, s.mergeRequest).Build()

	s.gitClient.On("Clone", s.mergeRequest.Spec.ProjectName).Return("path", nil)
	s.gitClient.On("GenerateChangeID").Return("change-id-1", nil)
	s.gitClient.On("SetProjectUser", s.mergeRequest.Spec.ProjectName,
		&git.User{Name: s.mergeRequest.Spec.AuthorName, Email: s.mergeRequest.Spec.AuthorEmail}).
		Return(errors.New("set author fatal"))

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGitClient: func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			return s.gitClient, nil
		},
	}

	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.Error(s.T(), loggerSink.LastError())
	assert.EqualError(s.T(), loggerSink.LastError(),
		"unable to create change: unable to perform merge: unable to set project author: set author fatal")
}

func (s *ControllerTestSuite) TestReconcile() {
	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit, s.mergeRequest).Build()
	changeID := "change123"

	s.gitClient.On("Clone", s.mergeRequest.Spec.ProjectName).Return("path", nil)
	s.gitClient.On("GenerateChangeID").Return(changeID, nil)
	s.gitClient.On("Merge", s.mergeRequest.Spec.ProjectName,
		fmt.Sprintf("origin/%s", s.mergeRequest.Spec.SourceBranch),
		s.mergeRequest.TargetBranch(), MergeArgNoFastForward, MergeArgCommitMessage, fmt.Sprintf("%s\n\nChange-Id: %s",
			s.mergeRequest.CommitMessage(), changeID), "-q").
		Return(nil)
	s.gitClient.On("Push", s.mergeRequest.Spec.ProjectName, "origin", "HEAD:refs/for/master").
		Return("http://gerrit.com/merge/1", nil)
	s.gitClient.On("SetProjectUser", s.mergeRequest.Spec.ProjectName,
		&git.User{Name: s.mergeRequest.Spec.AuthorName, Email: s.mergeRequest.Spec.AuthorEmail}).
		Return(nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGitClient: func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			return s.gitClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(s.T(), loggerSink.LastError())

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)
}

func (s *ControllerTestSuite) TestReconcileDelete() {
	deleteMergeRequest := s.mergeRequest.DeepCopy()
	deleteMergeRequest.DeletionTimestamp = &metaV1.Time{Time: time.Now()}
	deleteMergeRequest.Finalizers = []string{"test_fake_finalizer"}
	deleteMergeRequest.Status.ChangeID = "change321"

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit, deleteMergeRequest).Build()

	s.gerritClient.On("ChangeGet", deleteMergeRequest.Status.ChangeID).
		Return(&gerritClient.Change{Status: StatusNew}, nil)
	s.gerritClient.On("ChangeAbandon", deleteMergeRequest.Status.ChangeID).Return(nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *gerritApi.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(s.T(), loggerSink.LastError())

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)
}

func (s *ControllerTestSuite) TestReconcileCheckStatus() {
	checkStatusRequest := s.mergeRequest.DeepCopy()
	checkStatusRequest.Status.ChangeID = "change321"
	checkStatusRequest.Status.Value = StatusNew

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit, checkStatusRequest).Build()

	s.gerritClient.On("ChangeGet", checkStatusRequest.Status.ChangeID).
		Return(&gerritClient.Change{Status: StatusAbandoned}, nil).Once()

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *gerritApi.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(s.T(), loggerSink.LastError())

	assert.Equal(s.T(), result.RequeueAfter, time.Duration(0))

	var updatedMergeRequest gerritApi.GerritMergeRequest
	err = rec.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: checkStatusRequest.Name, Namespace: checkStatusRequest.Namespace},
		&updatedMergeRequest)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), updatedMergeRequest.Status.Value,
		StatusAbandoned)
}

func (s *ControllerTestSuite) TestConfigMap() {
	s.mergeRequest.Spec.SourceBranch = ""
	s.mergeRequest.Spec.ChangesConfigMap = "changes"

	err := coreV1.AddToScheme(s.scheme)
	require.NoError(s.T(), err)

	cm := coreV1.ConfigMap{Data: map[string]string{"test.txt": `{"path": "test.txt", "contents": "test"}`}, ObjectMeta: metaV1.ObjectMeta{
		Name: s.mergeRequest.Spec.ChangesConfigMap, Namespace: s.mergeRequest.Namespace,
	}}

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).
		WithRuntimeObjects(s.rootGerrit, s.mergeRequest, &cm).Build()
	changeID := "change123"

	s.gitClient.On("Clone", s.mergeRequest.Spec.ProjectName).Return("path", nil)
	s.gitClient.On("GenerateChangeID").Return(changeID, nil)
	s.gitClient.On("Push", s.mergeRequest.Spec.ProjectName, "origin", "HEAD:refs/for/master").
		Return("http://gerrit.com/merge/1", nil)
	s.gitClient.On("CheckoutBranch", s.mergeRequest.Spec.ProjectName, s.mergeRequest.TargetBranch()).
		Return(nil)
	s.gitClient.On("SetFileContents", s.mergeRequest.Spec.ProjectName, "test.txt", "test").Return(nil)
	s.gitClient.On("Commit", s.mergeRequest.Spec.ProjectName, commitMessage(s.mergeRequest.CommitMessage(),
		changeID), []string{"test.txt"},
		&git.User{Name: s.mergeRequest.Spec.AuthorName, Email: s.mergeRequest.Spec.AuthorEmail}).Return(nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGitClient: func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			return s.gitClient, nil
		},
	}

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(s.T(), loggerSink.LastError())

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)
}

func (s *ControllerTestSuite) TestReconcileCheckStatusFailure() {
	checkStatusRequest := s.mergeRequest.DeepCopy()
	checkStatusRequest.Status.ChangeID = "change321"
	checkStatusRequest.Status.Value = StatusNew

	fakeClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit, checkStatusRequest).Build()

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   s.gerritService,
		log:       s.logger,
		getGerritClient: func(ctx context.Context, child *gerritApi.GerritMergeRequest) (GerritClient, error) {
			return s.gerritClient, nil
		},
	}

	s.gerritClient.On("ChangeGet", checkStatusRequest.Status.ChangeID).
		Return(nil, errors.New("change get fatal")).Once()

	result, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      s.mergeRequest.Name,
		Namespace: s.mergeRequest.Namespace,
	}})
	assert.NoError(s.T(), err)

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.Error(s.T(), loggerSink.LastError())
	assert.EqualError(s.T(), loggerSink.LastError(),
		"unable to get change status: unable to get change id: change get fatal")

	assert.Equal(s.T(), result.RequeueAfter, time.Second*helper.DefaultRequeueTime)

	var updatedMergeRequest gerritApi.GerritMergeRequest
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
