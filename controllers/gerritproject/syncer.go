package gerritproject

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
)

const syncRetries = 3

func (r *Reconcile) syncBackendProjects(interval time.Duration) {
	ticker := time.Tick(interval)

	for range ticker {
		for i := 0; i < syncRetries; i++ {
			if err := r.syncBackendProjectsTick(); err != nil {
				r.log.Error(err, "unable to sync gerrit projects")
				continue
			}

			break
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

	for i := 0; i < len(gerritList.Items); i++ {
		if err := r.syncGerritInstance(ctx, &gerritList.Items[i], gerritProjectList.Items); err != nil {
			return errors.Wrapf(err, "unable to sync gerrit instance: %s", gerritList.Items[i].Name)
		}
	}

	return nil
}

func (r *Reconcile) syncGerritInstance(ctx context.Context, gr *gerritApi.Gerrit,
	allK8sGerritProjects []gerritApi.GerritProject,
) error {
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

func (r *Reconcile) syncProjectBranches(ctx context.Context, cl gerritClient.ClientInterface,
	k8sProject *gerritApi.GerritProject,
) error {
	branches, err := cl.ListProjectBranches(k8sProject.Spec.Name)
	if err != nil {
		return errors.Wrap(err, "unable to list project branches")
	}

	// reload gerrit project to prevent conflict with gerrit project operator
	var prj gerritApi.GerritProject
	if err := r.client.Get(ctx, types.NamespacedName{
		Name:      k8sProject.Name,
		Namespace: k8sProject.Namespace,
	}, &prj); err != nil {
		return errors.Wrap(err, "unable to get gerrit project")
	}

	prj.Status.Branches = make([]string, 0, len(branches))
	for _, br := range branches {
		prj.Status.Branches = append(prj.Status.Branches, br.Ref)
	}

	if err := r.client.Status().Update(ctx, &prj); err != nil {
		return errors.Wrap(err, "unable to update gerrit project")
	}

	return nil
}

func (r *Reconcile) createGerritProject(ctx context.Context, gr *gerritApi.Gerrit,
	backendProject *gerritClient.Project,
) (*gerritApi.GerritProject, error) {
	prj := gerritApi.GerritProject{
		ObjectMeta: metaV1.ObjectMeta{
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

	for i := 0; i < len(projects); i++ {
		for _, owner := range projects[i].OwnerReferences {
			if owner.UID == g.UID && owner.Kind == g.Kind {
				result[projects[i].Spec.Name] = &projects[i]
			}
		}
	}

	return result
}
