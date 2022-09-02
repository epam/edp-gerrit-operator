package gerrit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/resty.v1"
)

func TestClient_CreateProject(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("PUT", "/projects/test", httpmock.NewStringResponder(200, ""))

	err := cl.CreateProject(&Project{Name: "test"})
	assert.NoError(t, err)
}

func TestClient_UpdateProject(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("PUT", "/projects/test/description", httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("PUT", "/projects/test/parent", httpmock.NewStringResponder(200, ""))

	err := cl.UpdateProject(&Project{Name: "test"})
	assert.NoError(t, err)
}

func TestClient_DeleteProject(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/delete-project~delete", httpmock.NewStringResponder(200, ""))

	err := cl.DeleteProject("test")
	assert.NoError(t, err)
}

func TestClient_GetProject(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("GET", "/projects/test",
		httpmock.NewStringResponder(200, `}}}}}{"foo": "bar"}`))

	_, err := cl.GetProject("test")
	assert.NoError(t, err)
}

func TestClient_UpdateProject_Failure(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	err := cl.UpdateProject(&Project{})
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "error during post request") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestClient_GetProject_Failure(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.GetProject("test")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "no responder found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test", httpmock.NewStringResponder(404, ""))

	_, err = cl.GetProject("test")
	require.NotNil(t, err)

	if !IsErrDoesNotExist(err) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test",
		httpmock.NewStringResponder(500, "500 fatal"))

	_, err = cl.GetProject("test")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test",
		httpmock.NewStringResponder(200, `}}}}}wrong json`))

	_, err = cl.GetProject("test")
	assert.NotNil(t, err)

	if !strings.Contains(err.Error(), "unable to unmarshal project response") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestClient_ListProjects(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("GET", "/projects/?type=CODE&d=1&t=1",
		httpmock.NewStringResponder(200, `}}}}}{"prf": {"name": "prf"}}`))

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.ListProjects("CODE")
	assert.NoError(t, err)
}

func TestClient_ListProjects_Failure(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.ListProjects("CODE")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "Unable to get Gerrit project") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/?type=CODE&d=1&t=1",
		httpmock.NewStringResponder(500, "500 fatal"))

	_, err = cl.ListProjects("CODE")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/?type=CODE&d=1&t=1",
		httpmock.NewStringResponder(200, `}}}}}zazazaza`))

	_, err = cl.ListProjects("CODE")
	assert.NotNil(t, err)

	if !strings.Contains(err.Error(), "unmarshal project response") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestClient_ListProjectBranches(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("GET", fmt.Sprintf("/projects/%s/branches/", "prh1"),
		httpmock.NewStringResponder(200, `}}}}}[{"ref": "test"}]`))

	_, err := cl.ListProjectBranches("prh1")
	assert.NoError(t, err)
}

func TestClient_ListProjectBranches_Failure(t *testing.T) {
	httpmock.Reset()

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.ListProjectBranches("prh1")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "Unable to get Gerrit project branches") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/prh1/branches/",
		httpmock.NewStringResponder(500, "500 fatal"))

	_, err = cl.ListProjectBranches("prh1")
	require.NotNil(t, err)

	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/prh1/branches/",
		httpmock.NewStringResponder(200, `}}}}}zazazaza`))

	_, err = cl.ListProjectBranches("prh1")
	assert.NotNil(t, err)

	if !strings.Contains(err.Error(), "unmarshal project response") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
