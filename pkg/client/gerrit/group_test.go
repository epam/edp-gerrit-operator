package gerrit

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gid = "123"
const username = "user"
const groupName = "gr1"

func TestErrAlreadyExists_Error(t *testing.T) {
	t.Parallel()

	err := AlreadyExistsError(username)

	assert.Equal(t, username, err.Error())
}

func TestIsErrAlreadyExists(t *testing.T) {
	t.Parallel()

	t.Run("should return true for AlreadyExistsError", func(t *testing.T) {
		t.Parallel()

		err := AlreadyExistsError(username)
		wrapped := errors.Wrap(err, "some test error")

		assert.True(t, IsErrAlreadyExists(err))
		assert.True(t, IsErrAlreadyExists(wrapped))
	})

	t.Run("should return false for other errors", func(t *testing.T) {
		t.Parallel()

		err := errors.New("random error")
		notExistsErr := os.ErrNotExist

		assert.False(t, IsErrAlreadyExists(nil))
		assert.False(t, IsErrAlreadyExists(err))
		assert.False(t, IsErrAlreadyExists(notExistsErr))
	})
}

func TestErrDoesNotExist_Error(t *testing.T) {
	t.Parallel()

	err := DoesNotExistError(username)

	assert.Equal(t, username, err.Error())
}

func TestIsErrDoesNotExist(t *testing.T) {
	t.Parallel()

	t.Run("should return true for IsErrDoesNotExist", func(t *testing.T) {
		t.Parallel()

		err := DoesNotExistError(username)
		wrapped := errors.Wrap(err, "some test error")

		assert.True(t, IsErrDoesNotExist(err))
		assert.True(t, IsErrDoesNotExist(wrapped))
	})

	t.Run("should return false for other errors", func(t *testing.T) {
		t.Parallel()

		err := errors.New("random error")
		notExistsErr := os.ErrNotExist

		assert.False(t, IsErrDoesNotExist(nil))
		assert.False(t, IsErrDoesNotExist(err))
		assert.False(t, IsErrDoesNotExist(notExistsErr))
	})
}

func TestClient_AddUserToGroup(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("PUT", "/groups/foo/members/bar", httpmock.NewStringResponder(200, ""))

	err := cl.AddUserToGroup("foo", "bar")
	assert.NoError(t, err)
}

func TestClient_DeleteUserFromGroup(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("DELETE", "//%2Fgroups%2F"+groupName+"%2Fmembers%2F"+
		username+"/groups/"+groupName+"/members/"+username, httpmock.NewStringResponder(204, ""))

	err := cl.DeleteUserFromGroup(groupName, username)

	assert.NoError(t, err)
}

func TestClient_DeleteUserFromGroup_DeleteErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}

	err := cl.DeleteUserFromGroup(groupName, username)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unable to get Gerrit groups"))
}

func TestClient_DeleteUserFromGroup_RespErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}

	httpmock.RegisterResponder("DELETE", "//%2Fgroups%2F"+groupName+"%2Fmembers%2F"+
		username+"/groups/"+groupName+"/members/"+username, httpmock.NewStringResponder(404, ""))

	err := cl.DeleteUserFromGroup(groupName, username)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "wrong response code"))
}

func TestClient_UpdateGroup(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Foptions/groups/"+gid+"/options",
		httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Fdescription/groups/"+gid+"/description",
		httpmock.NewStringResponder(200, ""))

	err := cl.UpdateGroup(gid, desc, true)

	assert.NoError(t, err)
}

func TestClient_UpdateGroup_FirstPutErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	err := cl.UpdateGroup(gid, desc, true)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unable to update group"))
}

func TestClient_UpdateGroup_FirstPutResp(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Fdescription/groups/"+gid+"/description",
		httpmock.NewStringResponder(404, ""))

	err := cl.UpdateGroup(gid, desc, true)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "404"))
}

func TestClient_UpdateGroup_SecondPutErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Fdescription/groups/"+gid+"/description",
		httpmock.NewStringResponder(200, ""))

	err := cl.UpdateGroup(gid, desc, true)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unable to update group"))
}

func TestClient_UpdateGroup_SecondPutRespErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Foptions/groups/"+gid+"/options",
		httpmock.NewStringResponder(404, ""))
	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"%2Fdescription/groups/"+gid+"/description",
		httpmock.NewStringResponder(200, ""))

	err := cl.UpdateGroup(gid, desc, true)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "404"))
}

func TestClient_CreateGroup(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"
	group := Group{ID: "123",
		GroupID: 12,
		Members: []GroupMember{
			{
				Email:    "t@t.cow",
				Username: username,
			},
		}}

	rawGroups, err := json.Marshal(group)
	require.NoError(t, err)

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"/groups/"+gid,
		httpmock.NewStringResponder(200, "12345"+string(rawGroups)))

	gr, err := cl.CreateGroup(gid, desc, true)

	assert.Equal(t, group, *gr)
	assert.NoError(t, err)
}

func TestClient_CreateGroup_PutErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	_, err := cl.CreateGroup(gid, desc, true)

	assert.Error(t, err)
}

func TestClient_CreateGroup_UnmarshallErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"/groups/"+gid,
		httpmock.NewStringResponder(200, "123456"))

	_, err := cl.CreateGroup(gid, desc, true)

	assert.Error(t, err)
}

func TestClient_CreateGroup_RespErr409(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"/groups/"+gid,
		httpmock.NewStringResponder(409, ""))

	_, err := cl.CreateGroup(gid, desc, true)

	assert.Equal(t, AlreadyExistsError("already exists"), err)
}

func TestClient_CreateGroup_RespErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	desc := "desc"

	httpmock.RegisterResponder("PUT", "//%2Fgroups%2F"+gid+"/groups/"+gid,
		httpmock.NewStringResponder(404, ""))

	_, err := cl.CreateGroup(gid, desc, true)

	assert.Equal(t, errors.Errorf("status: %s, body: %s", "404", "").Error(), err.Error())
}
