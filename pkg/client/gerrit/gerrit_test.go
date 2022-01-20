package gerrit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"

	mock "github.com/epam/edp-gerrit-operator/v2/mock/ssh"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/ssh"
)

const uuid = "Bxfh1wAg_qyZQNdy5VKc7gNZgoLFm67YHbWhFvvk"

func CreateMockResty() *resty.Client {
	restyClient := resty.New()
	httpmock.DeactivateAndReset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	return restyClient
}

func TestClient_Resty(t *testing.T) {
	rs := &resty.Client{}
	cl := Client{
		instance:  nil,
		resty:     rs,
		sshClient: &ssh.SSHClient{},
	}
	assert.Equal(t, rs, cl.Resty())
}

func TestClient_InitNewSshClient(t *testing.T) {
	cl := Client{}
	pk, err := rsa.GenerateKey(rand.Reader, 128)
	assert.NoError(t, err)
	privkeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	pkey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	err = cl.InitNewSshClient("user", pkey, "testhost", int32(80))
	assert.NoError(t, err)
}

func TestClient_InitNewSshClient_Err(t *testing.T) {
	cl := Client{}
	err := cl.InitNewSshClient("user", []byte{}, "testhost", int32(80))
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "err while initializing new ssh client"))
}

func TestClient_InitNewRestClient(t *testing.T) {
	cl := Client{}
	err := cl.InitNewRestClient(&v1alpha1.Gerrit{}, "", "", "")
	assert.NoError(t, err)
}

func TestClient_CheckCredentials(t *testing.T) {
	restyClient := CreateMockResty()

	cl := Client{
		resty: restyClient,
	}
	httpmock.RegisterResponder("GET", "//%2Fconfig%2Fserver%2Fsummary/config/server/summary", httpmock.NewStringResponder(200, ""))

	credentials, err := cl.CheckCredentials()
	assert.Equal(t, 200, credentials)
	assert.NoError(t, err)
}

func TestClient_CheckGroup(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	restyClient := CreateMockResty()

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t" + uuid)
	sshCl.On("RunCommand", cmd).Return(out, nil)

	cl := Client{
		sshClient: &sshCl,
		resty:     restyClient,
	}
	httpmock.RegisterResponder("GET", "//%2Fgroups%2F"+uuid+"/groups/"+uuid,
		httpmock.NewStringResponder(200, ""))

	status, err := cl.CheckGroup("test")
	assert.Equal(t, 200, *status)
	assert.NoError(t, err)
}

func TestClient_CheckGroup_EmptyUUID(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	restyClient := CreateMockResty()

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t")
	sshCl.On("RunCommand", cmd).Return(out, nil)

	cl := Client{
		sshClient: &sshCl,
		resty:     restyClient,
	}
	httpmock.RegisterResponder("GET", "//%2Fgroups%2F"+uuid+"/groups/"+uuid,
		httpmock.NewStringResponder(200, ""))

	status, err := cl.CheckGroup("test")
	assert.Equal(t, http.StatusNotFound, *status)
	assert.NoError(t, err)
}

func TestClient_getGroupUuid(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t" + uuid)
	sshCl.On("RunCommand", cmd).Return(out, nil)

	id, err := cl.getGroupUuid("test")

	assert.NoError(t, err)
	assert.Equal(t, uuid, id)
}

func TestClient_getGroupUuid_DontMatch(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t12")
	sshCl.On("RunCommand", cmd).Return(out, nil)

	id, err := cl.getGroupUuid("test")

	assert.NoError(t, err)
	assert.Equal(t, "", id)
}

func TestClient_getGroupUuid_RunCommandErr(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	errTest := errors.New("test")
	out := []byte("test\t12")
	sshCl.On("RunCommand", cmd).Return(out, errTest)

	id, err := cl.getGroupUuid("test")

	assert.Error(t, err)
	assert.Equal(t, "", id)
}

func TestClient_GetUser(t *testing.T) {
	restyClient := CreateMockResty()

	cl := Client{
		resty: restyClient,
	}

	user := "test"
	httpmock.RegisterResponder("GET", "//%2Faccounts%2F"+user+"/accounts/"+user,
		httpmock.NewStringResponder(200, ""))

	status, err := cl.GetUser(user)
	assert.Equal(t, 200, *status)
	assert.NoError(t, err)
}

func TestClient_GetUserErr(t *testing.T) {
	restyClient := CreateMockResty()
	cl := Client{
		resty: restyClient,
	}
	user := "test"
	_, err := cl.GetUser(user)
	assert.Error(t, err)
}

func TestClient_ChangePassword(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	password := "1234"
	username := "name"
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-account --http-password \"%v\" \"%v\"", password, username),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	sshCl.On("RunCommand", cmd).Return(nil, nil)

	err := cl.ChangePassword(username, password)
	assert.NoError(t, err)
}

func TestClient_ChangePassword_Err(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	password := "1234"
	username := "name"
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-account --http-password \"%v\" \"%v\"", password, username),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	errTest := errors.New("test")
	sshCl.On("RunCommand", cmd).Return(nil, errTest)

	err := cl.ChangePassword(username, password)

	assert.Error(t, err)
}

func TestClient_ReloadPlugin(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	plugin := "test"
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit plugin reload \"%v\"", plugin),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	sshCl.On("RunCommand", cmd).Return(nil, nil)

	err := cl.ReloadPlugin(plugin)
	assert.NoError(t, err)
}

func TestClient_ReloadPluginErr(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	cl := Client{
		sshClient: &sshCl,
	}

	plugin := "test"
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit plugin reload \"%v\"", plugin),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	errTest := errors.New("test")
	sshCl.On("RunCommand", cmd).Return(nil, errTest)

	err := cl.ReloadPlugin(plugin)
	assert.Error(t, err)
}

func TestClient_CreateUser(t *testing.T) {
	restyClient := CreateMockResty()
	sshCl := mock.SSHClientInterface{}

	cl := Client{
		resty:     restyClient,
		sshClient: &sshCl,
	}

	user := "test"
	password := "1234"
	fullname := "full"
	pub := "pub"

	httpmock.RegisterResponder("GET", "//%2Faccounts%2F"+user+"/accounts/"+user,
		httpmock.NewStringResponder(404, ""))

	cmd := &ssh.SSHCommand{
		Path: fmt.Sprintf("gerrit create-account --full-name \"%v\" --http-password \"%v\" --ssh-key \"%v\" \"%v\"",
			fullname, password, pub, user),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	sshCl.On("RunCommand", cmd).Return(nil, nil)

	err := cl.CreateUser(user, password, fullname, pub)

	assert.NoError(t, err)
}

func TestClient_CreateUser_GetUserErr(t *testing.T) {
	restyClient := CreateMockResty()
	sshCl := mock.SSHClientInterface{}

	cl := Client{
		resty:     restyClient,
		sshClient: &sshCl,
	}

	user := "test"
	password := "1234"
	fullname := "full"
	pub := "pub"

	err := cl.CreateUser(user, password, fullname, pub)

	assert.Error(t, err)
}

func TestClient_CreateUser_RunCommandErr(t *testing.T) {
	restyClient := CreateMockResty()
	sshCl := mock.SSHClientInterface{}

	cl := Client{
		resty:     restyClient,
		sshClient: &sshCl,
	}

	user := "test"
	password := "1234"
	fullname := "full"
	pub := "pub"

	httpmock.RegisterResponder("GET", "//%2Faccounts%2F"+user+"/accounts/"+user,
		httpmock.NewStringResponder(404, ""))

	cmd := &ssh.SSHCommand{
		Path: fmt.Sprintf("gerrit create-account --full-name \"%v\" --http-password \"%v\" --ssh-key \"%v\" \"%v\"",
			fullname, password, pub, user),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	errTest := errors.New("test")
	sshCl.On("RunCommand", cmd).Return(nil, errTest)

	err := cl.CreateUser(user, password, fullname, pub)

	assert.Error(t, err)
}

func TestClient_CreateUser_UserExist(t *testing.T) {
	restyClient := CreateMockResty()
	sshCl := mock.SSHClientInterface{}

	cl := Client{
		resty:     restyClient,
		sshClient: &sshCl,
	}

	user := "test"
	password := "1234"
	fullname := "full"
	pub := "pub"

	httpmock.RegisterResponder("GET", "//%2Faccounts%2F"+user+"/accounts/"+user,
		httpmock.NewStringResponder(200, ""))

	err := cl.CreateUser(user, password, fullname, pub)

	assert.NoError(t, err)
}

func TestClient_AddUserToGroups(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	restyClient := CreateMockResty()

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t" + uuid)
	sshCl.On("RunCommand", cmd).Return(out, nil)

	cl := Client{
		sshClient: &sshCl,
		resty:     restyClient,
	}
	httpmock.RegisterResponder("GET", "//%2Fgroups%2F"+uuid+"/groups/"+uuid,
		httpmock.NewStringResponder(200, ""))

	name := "name"
	groups := []string{"test"}
	cmd = &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-members --add \"%v\" \"%v\"", name, groups[0]),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	sshCl.On("RunCommand", cmd).Return(out, nil)
	err := cl.AddUserToGroups(name, groups)

	assert.NoError(t, err)
}

func TestClient_AddUserToGroups_RunCommandErr(t *testing.T) {
	sshCl := mock.SSHClientInterface{}
	restyClient := CreateMockResty()

	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out := []byte("test\t" + uuid)
	sshCl.On("RunCommand", cmd).Return(out, nil)

	cl := Client{
		sshClient: &sshCl,
		resty:     restyClient,
	}
	httpmock.RegisterResponder("GET", "//%2Fgroups%2F"+uuid+"/groups/"+uuid,
		httpmock.NewStringResponder(200, ""))

	name := "name"
	groups := []string{"test"}
	cmd = &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-members --add \"%v\" \"%v\"", name, groups[0]),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	errTest := errors.New("test")
	sshCl.On("RunCommand", cmd).Return(out, errTest)
	err := cl.AddUserToGroups(name, groups)

	assert.Error(t, err)
}

func TestNewClient(t *testing.T) {
	cl := NewClient(nil, resty.New(), nil)
	accept, ok := cl.resty.Header["Accept"]
	if !ok || len(accept) == 0 {
		t.Fatal("no accept header set")
	}

	assert.Equal(t, accept[0], "application/json")
}
