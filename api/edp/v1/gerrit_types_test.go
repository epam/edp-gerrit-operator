package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
)

func TestGerritSpec_GetBasePath(t *testing.T) {
	gs := GerritSpec{}
	assert.Equal(t, gs.GetBasePath(), spec.GerritRestApiUrlPath)

	gs.BasePath = "gerrit"
	assert.Equal(t, gs.GetBasePath(), "gerrit/a/")
}
