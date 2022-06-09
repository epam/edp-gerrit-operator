package gerritproject

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
)

func (r *Reconcile) syncBackendProjects(interval time.Duration) {
	ticker := time.Tick(interval)

	for range ticker {
		if err := r.syncBackendProjectsTick(); err != nil {
			r.log.Error(err, "unable to sync gerrit projects")
		}
	}
}

func (r *Reconcile) syncBackendProjectsTick() error {
	var (
		gerritList        gerritApi.GerritList
		gerritProjectList gerritApi.GerritProjectList
		ctx               = context.Background()
	)

	if err := r.client.List(ctx, &gerritList); err != nil {
		return errors.Wrap(err, "unable to list gerrits")
	}

	if err := r.client.List(ctx, &gerritProjectList); err != nil {
		return errors.Wrap(err, "unable to list gerrit projects")
	}

	for _, gr := range gerritList.Items {
		if err := r.syncGerritInstance(ctx, &gr, gerritProjectList.Items); err != nil {
			return errors.Wrapf(err, "unable to sync gerrit instance: %s", gr.Name)
		}
	}

	return nil
}

func (r *Reconcile) syncGerritInstance(ctx context.Context, gr *gerritApi.Gerrit,
	allK8sGerritProjects []gerritApi.GerritProject) error {
	cl, err := r.service.GetRestClient(gr)
	if err != nil {
		return errors.Wrap(err, "unable to init gerrit client")
	}

	backendProjects, err := cl.ListProjects("CODE")
	if err != nil {
		return errors.Wrap(err, "unable to list projects from gerrit")
	}

	k8sProjects := filterGerritProjectsByGerrit(gr, allK8sGerritProjects)

	for _, backendProject := range backendProjects {
		k8sProject, ok := k8sProjects[backendProject.Name]
		if !ok {
			k8sProject, err = r.createGerritProject(ctx, gr, &backendProject)
			if err != nil {
				return errors.Wrap(err, "unable to create gerrit project")
			}
		}

		if err := r.syncProjectBranches(ctx, cl, k8sProject); err != nil {
			return errors.Wrap(err, "unable to sync gerrit project branches")
		}
	}

	return nil
}

func (r *Reconcile) syncProjectBranches(ctx context.Context, cl gerrit.ClientInterface,
	k8sProject *gerritApi.GerritProject) error {
	branches, err := cl.ListProjectBranches(k8sProject.Spec.Name)
	if err != nil {
		return errors.Wrap(err, "unable to list project branches")
	}

	k8sProject.Status.Branches = make([]string, 0, len(branches))
	for _, br := range branches {
		k8sProject.Status.Branches = append(k8sProject.Status.Branches, br.Ref)
	}

	if err := r.client.Status().Update(ctx, k8sProject); err != nil {
		return errors.Wrap(err, "unable to update gerrit project")
	}

	return nil
}

func (r *Reconcile) createGerritProject(ctx context.Context, gr *gerritApi.Gerrit,
	backendProject *gerritClient.Project) (*gerritApi.GerritProject, error) {

	prj := gerritApi.GerritProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(fmt.Sprintf("%s-%s", gr.Name, backendProject.SlugifyName())),
			Namespace: gr.Namespace,
		},
		Spec: gerritApi.GerritProjectSpec{
			Name:              backendProject.Name,
			Parent:            backendProject.Parent,
			Description:       backendProject.Description,
			SubmitType:        backendProject.SubmitType,
			Owners:            backendProject.Owners,
			RejectEmptyCommit: backendProject.RejectEmptyCommit,
			PermissionsOnly:   backendProject.PermissionsOnly,
			CreateEmptyCommit: backendProject.CreateEmptyCommit,
			Branches:          backendProject.Branches,
			OwnerName:         gr.Name,
		},
	}

	if err := r.client.Create(ctx, &prj); err != nil {
		return nil, errors.Wrap(err, "unable to create gerrit project")
	}

	return &prj, nil
}

func filterGerritProjectsByGerrit(g *gerritApi.Gerrit, projects []gerritApi.GerritProject) map[string]*gerritApi.GerritProject {
	result := make(map[string]*gerritApi.GerritProject)

	for k, p := range projects {
		for _, owner := range p.OwnerReferences {
			if owner.UID == g.UID && owner.Kind == g.Kind {
				result[p.Spec.Name] = &projects[k]
			}
		}
	}

	return result
}
