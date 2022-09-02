package gerrit

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/jarcoal/httpmock"
	"gopkg.in/resty.v1"
)

func TestClient_AddAccessRights(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/access", httpmock.NewStringResponder(200, ""))

	if err := cl.AddAccessRights("test", []AccessInfo{
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "important-group",
			Min:            -2,
			Max:            2,
			Force:          false,
			Action:         "ALLOW",
		}}); err != nil {
		t.Fatal(err)
	}
}

func TestClient_UpdateAccessRights(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/access", httpmock.NewStringResponder(200, ""))

	if err := cl.UpdateAccessRights("test", []AccessInfo{
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "important-group",
			Min:            -2,
			Max:            2,
			Force:          false,
			Action:         "ALLOW",
		}}); err != nil {
		t.Fatal(err)
	}
}

func TestClient_DeleteAccessRights(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/access", httpmock.NewStringResponder(200, ""))

	if err := cl.DeleteAccessRights("test", []AccessInfo{
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "important-group",
			Min:            -2,
			Max:            2,
			Force:          false,
			Action:         "ALLOW",
		}}); err != nil {
		t.Fatal(err)
	}
}

func TestClient_DeleteAccessRightsFailure(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/projects/test/access", httpmock.NewStringResponder(500, ""))

	if err := cl.DeleteAccessRights("test", []AccessInfo{
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "important-group",
			Min:            -2,
			Max:            2,
			Force:          false,
			Action:         "ALLOW",
		}}); err == nil {
		t.Fatal("no error returned")
	}
}

func TestClient_SetProjectParent(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	httpmock.RegisterResponder("PUT", "/projects/test/parent", httpmock.NewStringResponder(200, ""))

	cl := Client{
		resty: restyClient,
	}

	if err := cl.SetProjectParent("test", "parent"); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSetAccessRequest(t *testing.T) {
	rq := generateSetAccessRequest([]AccessInfo{
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "important-group",
			Min:            -2,
			Max:            2,
			Force:          false,
			Action:         "ALLOW",
		},
		{
			RefPattern:     "refs/heads/*",
			PermissionName: "label-Code-Review",
			GroupName:      "ok-group",
			Action:         "DENY",
			Max:            2,
			Min:            -2,
		},
		{
			RefPattern:      "refs/heads/*",
			PermissionName:  "label-Verified",
			PermissionLabel: "Verified",
			Min:             -1,
			Max:             1,
			GroupName:       "important-group",
			Force:           false,
			Action:          "ALLOW",
		},
	}, false, false)

	bts, err := json.Marshal(rq)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(bts))
}

func TestParseRestyResponse(t *testing.T) {
	if err := parseRestyResponse(nil, errors.New("fatal")); err == nil {
		t.Fatal("no error")
	}
}
