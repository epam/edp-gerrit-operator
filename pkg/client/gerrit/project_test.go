package gerrit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
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

	if err := cl.CreateProject(&Project{Name: "test"}); err != nil {
		t.Fatal(err)
	}
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

	if err := cl.UpdateProject(&Project{Name: "test"}); err != nil {
		t.Fatal(err)
	}
}

func TestClient_DeleteProject(t *testing.T) {
	httpmock.Reset()
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/delete-project~delete", httpmock.NewStringResponder(200, ""))

	if err := cl.DeleteProject("test"); err != nil {
		t.Fatal(err)
	}
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

	if _, err := cl.GetProject("test"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_UpdateProject_Failure(t *testing.T) {
	httpmock.Reset()
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}
	err := cl.UpdateProject(&Project{})
	if err == nil {
		t.Fatal("no error returned")
	}
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
	if err == nil {
		t.Fatal("no error returned")
	}
	if !strings.Contains(err.Error(), "no responder found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test", httpmock.NewStringResponder(404, ""))
	_, err = cl.GetProject("test")
	if err == nil {
		t.Fatal("no error returned")
	}
	if !IsErrDoesNotExist(err) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test",
		httpmock.NewStringResponder(500, "500 fatal"))
	_, err = cl.GetProject("test")
	if err == nil {
		t.Fatal("no error returned")
	}
	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/test",
		httpmock.NewStringResponder(200, `}}}}}wrong json`))

	_, err = cl.GetProject("test")
	if err == nil {
		t.Fatal("no error returned")
	}
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

	if _, err := cl.ListProjects("CODE"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_ListProjects_Failure(t *testing.T) {
	httpmock.Reset()
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.ListProjects("CODE")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "Unable to get Gerrit project") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/?type=CODE&d=1&t=1",
		httpmock.NewStringResponder(500, "500 fatal"))

	_, err = cl.ListProjects("CODE")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/?type=CODE&d=1&t=1",
		httpmock.NewStringResponder(200, `}}}}}zazazaza`))

	_, err = cl.ListProjects("CODE")
	if err == nil {
		t.Fatal("no error returned")
	}

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
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ListProjectBranches_Failure(t *testing.T) {
	httpmock.Reset()
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	_, err := cl.ListProjectBranches("prh1")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "Unable to get Gerrit project branches") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/prh1/branches/",
		httpmock.NewStringResponder(500, "500 fatal"))

	_, err = cl.ListProjectBranches("prh1")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "500 fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "/projects/prh1/branches/",
		httpmock.NewStringResponder(200, `}}}}}zazazaza`))

	_, err = cl.ListProjectBranches("prh1")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "unmarshal project response") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
