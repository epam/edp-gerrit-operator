package mergerequest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/git"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	finalizerName         = "merge_request.gerrit.finalizer.name"
	StatusNew             = "NEW"
	StatusAbandoned       = "ABANDONED"
	StatusMerged          = "MERGED"
	MergeArgNoFastForward = "--no-ff"
	MergeArgCommitMessage = "-m"
)

type Reconcile struct {
	k8sClient       client.Client
	service         gerrit.Interface
	log             logr.Logger
	getGitClient    func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error)
	getGerritClient func(ctx context.Context, child *gerritApi.GerritMergeRequest) (GerritClient, error)
	gitWorkDir      string
}

type GitClient interface {
	Clone(projectName string) (projectPath string, err error)
	Merge(projectName, sourceBranch, targetBranch string, options ...string) error
	Push(projectName string, remote string, refSpecs ...string) (pushOutput string, retErr error)
	GenerateChangeID() (string, error)
	SetProjectUser(projectName string, user *git.User) error
	CheckoutBranch(projectName, branch string) error
	Commit(projectName, message string, files []string, user *git.User) error
	SetFileContents(projectName, filePath, contents string) error
}

type GerritClient interface {
	ChangeAbandon(changeID string) error
	ChangeGet(changeID string) (*gerritClient.Change, error)
}

type MRConfigMapFile struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

func NewReconcile(k8sClient client.Client, log logr.Logger,
	opts ...OptionFunc) helper.Controller {
	r := &Reconcile{
		k8sClient: k8sClient,
		log:       log,
	}

	for i := range opts {
		opts[i](r)
	}

	return r
}

type OptionFunc func(r *Reconcile)

func PrepareWorkDirectoryOption(gitWorkDirectory string) (OptionFunc, error) {
	if err := os.RemoveAll(gitWorkDirectory); err != nil {
		return nil, errors.Wrap(err, "unable to clean git work dir")
	}

	return func(r *Reconcile) {
		r.gitWorkDir = gitWorkDirectory
	}, nil
}

func PrepareGerritServiceOption(k8sClient client.Client, platformType string, scheme *runtime.Scheme) (OptionFunc, error) {
	ps, err := platform.NewService(platformType, scheme)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	gerritService := gerrit.NewComponentService(ps, k8sClient, scheme)

	return func(r *Reconcile) {
		r.service = gerritService
		r.getGitClient = func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error) {
			gitClient, err := gerritService.GetGitClient(ctx, child, workDir)
			if err != nil {
				return nil, fmt.Errorf("failed to create git client: %w", err)
			}

			return gitClient, nil
		}
		r.getGerritClient = func(ctx context.Context, instance *gerritApi.GerritMergeRequest) (GerritClient, error) {
			gerritClientInst, err := helper.GetGerritClient(ctx, r.k8sClient, instance, instance.OwnerName(), r.service)
			if err != nil {
				return nil, fmt.Errorf("failed to create gerrit client: %w", err)
			}

			return gerritClientInst, nil
		}
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&gerritApi.GerritMergeRequest{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup GerritMergeRequest controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*gerritApi.GerritMergeRequest)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*gerritApi.GerritMergeRequest)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritMergeRequest has been started")

	var instance gerritApi.GerritMergeRequest
	if err := r.k8sClient.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			reqLogger.Info("instance not found")
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get GerritMergeRequest instance")
	}

	if requeue, err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter =
			time.Second * helper.DefaultRequeueTime

		reqLogger.Error(err, "an error has occurred while handling GerritMergeRequest", "name",
			request.Name)
	} else if requeue {
		result.RequeueAfter = time.Second * helper.DefaultRequeueTime
	}

	if err := r.k8sClient.Status().Update(ctx, &instance); err != nil {
		resError = err
	}

	reqLogger.Info("Reconciling done")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *gerritApi.GerritMergeRequest) (bool, error) {
	requeue := false

	if instance.Status.ChangeID == "" {
		if instance.Spec.SourceBranch == "" && instance.Spec.ChangesConfigMap == "" {
			return false, errors.New("sourceBranch or changesConfigMap must be specified")
		}

		status, err := r.createChange(ctx, instance)
		if err != nil {
			return false, errors.Wrap(err, "unable to create change")
		}

		instance.Status = *status
		requeue = true
	} else {
		status, err := r.getChangeStatus(ctx, instance)
		if err != nil {
			return false, errors.Wrap(err, "unable to get change status")
		}

		instance.Status.Value = status
		requeue = status == StatusNew
	}

	if err := helper.TryToDelete(ctx, r.k8sClient, instance, finalizerName,
		r.makeDeletionFunc(ctx, instance)); err != nil {
		return false, errors.Wrap(err, "unable to delete resource")
	}

	return requeue, nil
}

func (r *Reconcile) createChange(ctx context.Context,
	instance *gerritApi.GerritMergeRequest) (status *gerritApi.GerritMergeRequestStatus, retErr error) {
	// init git client
	gitClient, err := r.getGitClient(ctx, instance, r.gitWorkDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init git client")
	}

	// clone project
	projectPath, err := gitClient.Clone(instance.Spec.ProjectName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to clone repo")
	}

	// clear cloned project
	defer func() {
		if remErr := os.RemoveAll(projectPath); remErr != nil {
			retErr = remErr
		}
	}()

	// generate change id for commit or merge
	changeID, err := gitClient.GenerateChangeID()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate change id")
	}

	// perform merge or commit files from config map
	if instance.Spec.SourceBranch != "" {
		err = mergeBranches(instance, gitClient, changeID)
		if err != nil {
			return nil, errors.Wrap(err, "unable to perform merge")
		}
	} else {
		err = r.commitFiles(ctx, instance, gitClient, changeID)
		if err != nil {
			return nil, errors.Wrap(err, "unable to commit files")
		}
	}

	// push changes for review
	refSpec := fmt.Sprintf("HEAD:refs/for/%s", instance.TargetBranch())

	pushMessage, err := gitClient.Push(instance.Spec.ProjectName, "origin", refSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to push repo")
	}

	return &gerritApi.GerritMergeRequestStatus{
		ChangeID:  changeID,
		ChangeURL: extractMrURL(pushMessage),
		Value:     StatusNew,
	}, nil
}

func (r *Reconcile) commitFiles(ctx context.Context, instance *gerritApi.GerritMergeRequest, gitClient GitClient, changeID string) error {
	var cMap corev1.ConfigMap
	if err := r.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.ChangesConfigMap,
	}, &cMap); err != nil {
		return errors.Wrap(err, "unable to get files config map")
	}

	if err := gitClient.CheckoutBranch(instance.Spec.ProjectName, instance.TargetBranch()); err != nil {
		return errors.Wrap(err, "unable to checkout branch")
	}

	addFiles := make([]string, 0, len(cMap.Data))

	for _, mrContents := range cMap.Data {
		var mrFile MRConfigMapFile
		if err := json.Unmarshal([]byte(mrContents), &mrFile); err != nil {
			return errors.Wrap(err, "unable to decode file")
		}

		if err := gitClient.SetFileContents(instance.Spec.ProjectName, mrFile.Path, mrFile.Contents); err != nil {
			return errors.Wrap(err, "unable to set file contents")
		}

		addFiles = append(addFiles, mrFile.Path)
	}

	gitUser := &git.User{Name: instance.Spec.AuthorName, Email: instance.Spec.AuthorEmail}
	message := commitMessage(instance.CommitMessage(), changeID)

	if err := gitClient.Commit(instance.Spec.ProjectName, message, addFiles, gitUser); err != nil {
		return errors.Wrap(err, "unable to commit changes")
	}

	return nil
}

func mergeBranches(instance *gerritApi.GerritMergeRequest, gitClient GitClient, changeID string) error {
	projectName := instance.Spec.ProjectName
	gitUser := &git.User{Name: instance.Spec.AuthorName, Email: instance.Spec.AuthorEmail}

	if err := gitClient.SetProjectUser(projectName, gitUser); err != nil {
		return errors.Wrap(err, "unable to set project author")
	}

	mergeArguments := []string{
		MergeArgNoFastForward,
		MergeArgCommitMessage,
		commitMessage(instance.CommitMessage(), changeID),
	}
	if len(instance.Spec.AdditionalArguments) > 0 {
		mergeArguments = append(mergeArguments, instance.Spec.AdditionalArguments...)
	}

	sourceBranch := fmt.Sprintf("origin/%s", instance.Spec.SourceBranch)
	if err := gitClient.Merge(projectName, sourceBranch, instance.TargetBranch(), mergeArguments...); err != nil {
		return errors.Wrap(err, "unable to merge branches")
	}

	return nil
}

func commitMessage(commitMessage, changeID string) string {
	return fmt.Sprintf("%s\n\nChange-Id: %s", commitMessage, changeID)
}

func (r *Reconcile) getChangeStatus(ctx context.Context, instance *gerritApi.GerritMergeRequest) (string, error) {
	gClient, err := r.getGerritClient(ctx, instance)
	if err != nil {
		return "", errors.Wrap(err, "unable to get gerrit client")
	}

	change, err := gClient.ChangeGet(instance.Status.ChangeID)
	if err != nil {
		return "", errors.Wrap(err, "unable to get change id")
	}

	return change.Status, nil
}

func extractMrURL(pushMessage string) string {
	return regexp.MustCompile(
		`https?://(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`).
		FindString(pushMessage)
}

func (r *Reconcile) makeDeletionFunc(ctx context.Context, instance *gerritApi.GerritMergeRequest) func() error {
	return func() error {
		gClient, err := r.getGerritClient(ctx, instance)
		if err != nil {
			return errors.Wrap(err, "unable to get gerrit client")
		}

		change, err := gClient.ChangeGet(instance.Status.ChangeID)
		if err != nil {
			return errors.Wrap(err, "unable to get change id")
		}

		if change.Status == StatusNew {
			if err := gClient.ChangeAbandon(instance.Status.ChangeID); err != nil {
				return errors.Wrap(err, "unable to abandon change")
			}
		}

		return nil
	}
}
