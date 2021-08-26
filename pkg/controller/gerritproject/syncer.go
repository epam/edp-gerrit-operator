package gerritproject

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		gerritList        v1alpha1.GerritList
		gerritProjectList v1alpha1.GerritProjectList
	)

	if err := r.client.List(context.Background(), &gerritList); err != nil {
		return errors.Wrap(err, "unable to list gerrits")
	}

	if err := r.client.List(context.Background(), &gerritProjectList); err != nil {
		return errors.Wrap(err, "unable to list gerrit projects")
	}

	for _, gr := range gerritList.Items {
		if err := r.syncGerritInstance(&gr, gerritProjectList.Items); err != nil {
			return errors.Wrapf(err, "unable to sync gerrit instance: %s", gr.Name)
		}
	}

	return nil
}

func (r *Reconcile) syncGerritInstance(gr *v1alpha1.Gerrit, allK8sGerritProjects []v1alpha1.GerritProject) error {
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
		if _, ok := k8sProjects[backendProject.Name]; !ok {
			if err := r.createGerritProject(gr, &backendProject); err != nil {
				return errors.Wrap(err, "unable to create gerrit project")
			}
		}
	}

	return nil
}

func (r *Reconcile) createGerritProject(gr *v1alpha1.Gerrit, backendProject *gerritClient.Project) error {
	if err := r.client.Create(context.Background(), &v1alpha1.GerritProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(fmt.Sprintf("%s-%s", gr.Name, backendProject.Name)),
			Namespace: gr.Namespace,
		},
		Spec: v1alpha1.GerritProjectSpec{
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
	}); err != nil {
		return errors.Wrap(err, "unable to create gerrit project")
	}

	return nil
}

func filterGerritProjectsByGerrit(g *v1alpha1.Gerrit, projects []v1alpha1.GerritProject) map[string]v1alpha1.GerritProject {
	result := make(map[string]v1alpha1.GerritProject)

	for _, p := range projects {
		for _, owner := range p.OwnerReferences {
			if owner.UID == g.UID && owner.Kind == g.Kind {
				result[p.Spec.Name] = p
			}
		}
	}

	return result
}
