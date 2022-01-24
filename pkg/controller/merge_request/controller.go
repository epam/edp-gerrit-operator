package mergerequest

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	finalizerName   = "merge_request.gerrit.finalizer.name"
	StatusNew       = "NEW"
	StatusAbandoned = "ABANDONED"
	StatusMerged    = "MERGED"
)

type Reconcile struct {
	k8sClient       client.Client
	service         gerrit.Interface
	log             logr.Logger
	getGitClient    func(ctx context.Context, child gerrit.Child, workDir string) (GitClient, error)
	getGerritClient func(ctx context.Context, child *v1alpha1.GerritMergeRequest) (GerritClient, error)
	gitWorkDir      string
}

type GitClient interface {
	Clone(projectName string) (projectPath string, err error)
	Merge(projectName, sourceBranch, targetBranch string, options ...string) error
	Push(projectName string, remote string, refSpecs ...string) (pushOutput string, retErr error)
	GenerateChangeID() (string, error)
	SetProjectUser(projectName, name, email string) error
}

type GerritClient interface {
	ChangeAbandon(changeID string) error
	ChangeGet(changeID string) (*gerritClient.Change, error)
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
			return gerritService.GetGitClient(ctx, child, workDir)
		}
		r.getGerritClient = func(ctx context.Context, instance *v1alpha1.GerritMergeRequest) (GerritClient, error) {
			return helper.GetGerritClient(ctx, r.k8sClient, instance, instance.OwnerName(), r.service)
		}
	}, nil
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GerritMergeRequest{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.GerritMergeRequest)
	no := e.ObjectNew.(*v1alpha1.GerritMergeRequest)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling GerritMergeRequest has been started")

	var instance v1alpha1.GerritMergeRequest
	if err := r.k8sClient.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			reqLogger.Info("instance not found")
			return
		}
		return reconcile.Result{}, errors.Wrap(err, "unable to get GerritMergeRequest instance")
	}

	if requeue, err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = time.Second * helper.DefaultRequeueTime
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

func (r *Reconcile) tryReconcile(ctx context.Context, instance *v1alpha1.GerritMergeRequest) (bool, error) {
	requeue := false

	if instance.Status.ChangeID == "" {
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
	instance *v1alpha1.GerritMergeRequest) (status *v1alpha1.GerritMergeRequestStatus, retErr error) {
	gitClient, err := r.getGitClient(ctx, instance, r.gitWorkDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init git client")
	}

	projectPath, err := gitClient.Clone(instance.Spec.ProjectName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to clone repo")
	}

	if err := gitClient.SetProjectUser(instance.Spec.ProjectName, instance.Spec.AuthorName,
		instance.Spec.AuthorEmail); err != nil {
		return nil, errors.Wrap(err, "unable to set project author")
	}

	defer func() {
		if err := os.RemoveAll(projectPath); err != nil {
			retErr = err
		}
	}()

	changeID, err := gitClient.GenerateChangeID()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate change id")
	}

	if err := gitClient.Merge(instance.Spec.ProjectName, fmt.Sprintf("origin/%s", instance.Spec.SourceBranch),
		instance.TargetBranch(), "--no-ff", "-m",
		fmt.Sprintf("%s\n\nChange-Id: %s", instance.CommitMessage(), changeID)); err != nil {
		return nil, errors.Wrap(err, "unable to merge branches")
	}

	pushMessage, err := gitClient.Push(instance.Spec.ProjectName, "origin", "HEAD:refs/for/master")
	if err != nil {
		return nil, errors.Wrap(err, "unable to push repo")
	}

	return &v1alpha1.GerritMergeRequestStatus{
		ChangeID:  changeID,
		ChangeURL: extractMrURL(pushMessage),
		Value:     StatusNew,
	}, nil
}

func (r *Reconcile) getChangeStatus(ctx context.Context, instance *v1alpha1.GerritMergeRequest) (string, error) {
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
		`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`).
		FindString(pushMessage)
}

func (r *Reconcile) makeDeletionFunc(ctx context.Context, instance *v1alpha1.GerritMergeRequest) func() error {
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
