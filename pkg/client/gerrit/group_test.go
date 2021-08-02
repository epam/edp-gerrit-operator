package gerrit

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"gopkg.in/resty.v1"
)

func TestClient_AddUserToGroup(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("PUT", "/groups/foo/members/bar", httpmock.NewStringResponder(200, ""))
	if err := cl.AddUserToGroup("foo", "bar"); err != nil {
		t.Fatal(err)
	}
}
