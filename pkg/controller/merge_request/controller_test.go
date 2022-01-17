package mergerequest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

func TestReconcile_Reconcile(t *testing.T) {
	sch := runtime.NewScheme()
	v1alpha1.RegisterTypes(sch)

	fakeClient := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&rootGerrit, &mergeRequest).Build()
	gService := gmock.Interface{}
	gService.AssertExpectations(t)
	logger := helper.Logger{}
	gitClient := gmock.GitClient{}
	gitClient.AssertExpectations(t)
	changeID := "change123"

	gitClient.On("Clone", mergeRequest.Spec.ProjectName).Return("path", nil)
	gitClient.On("GenerateChangeID").Return(changeID, nil)
	gitClient.On("Merge", mergeRequest.Spec.ProjectName, fmt.Sprintf("origin/%s", mergeRequest.Spec.SourceBranch),
		mergeRequest.TargetBranch(), "--no-ff", "-m", fmt.Sprintf("%s\n\nChange-Id: %s", mergeRequest.CommitMessage(), changeID)).
		Return(nil)
	gitClient.On("Push", mergeRequest.Spec.ProjectName, "origin", "HEAD:refs/for/master").
		Return("http://gerrit.com/merge/1", nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   &gService,
		log:       &logger,
		getGitClient: func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			return &gitClient, nil
		},
	}

	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(t, err)

	err = logger.LastError()
	assert.NoError(t, err)
}

func TestReconcile_Reconcile_Delete(t *testing.T) {
	sch := runtime.NewScheme()
	v1alpha1.RegisterTypes(sch)

	deleteMergeRequest := mergeRequest.DeepCopy()
	deleteMergeRequest.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	deleteMergeRequest.Status.ChangeID = "change321"

	fakeClient := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&rootGerrit, deleteMergeRequest).Build()
	gService := gmock.Interface{}
	gService.AssertExpectations(t)
	logger := helper.Logger{}
	gClient := gmock.ClientInterface{}
	gClient.AssertExpectations(t)

	gClient.On("ChangeGet", deleteMergeRequest.Status.ChangeID).
		Return(&gerritClient.Change{Status: "NEW"}, nil)
	gClient.On("ChangeAbandon", deleteMergeRequest.Status.ChangeID).Return(nil)

	rec := Reconcile{
		k8sClient: fakeClient,
		service:   &gService,
		log:       &logger,
		getGerritClient: func(ctx context.Context, child *v1alpha1.GerritMergeRequest) (GerritClient, error) {
			return &gClient, nil
		},
	}

	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      mergeRequest.Name,
		Namespace: mergeRequest.Namespace,
	}})
	assert.NoError(t, err)

	err = logger.LastError()
	assert.NoError(t, err)
}
