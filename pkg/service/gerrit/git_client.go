package gerrit

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/git"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
)

type Child interface {
	GetNamespace() string
	OwnerName() string
}

func (s ComponentService) GetGitClient(ctx context.Context, child Child, workDir string) (*git.Client, error) {
	var g v1alpha1.Gerrit
	if err := s.client.Get(ctx, types.NamespacedName{Name: child.OwnerName(),
		Namespace: child.GetNamespace()}, &g); err != nil {
		return nil, errors.Wrap(err, "unable to get parent gerrit")
	}

	gerritAdminPassword, err := s.getGerritAdminPassword(&g)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Gerrit admin password from secret for %s/%s", g.Namespace, g.Name)
	}

	gerritApiUrl, err := s.getGerritRestApiUrl(&g)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Gerrit REST API URL %s/%s", g.Namespace, g.Name)
	}

	return git.New(gerritApiUrl, workDir, spec.GerritDefaultAdminUser, gerritAdminPassword), nil
}
