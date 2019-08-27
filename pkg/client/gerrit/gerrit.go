package gerrit

import (
	"bytes"
	"fmt"
	"gerrit-operator/pkg/apis/v2/v1alpha1"
	"gerrit-operator/pkg/client/ssh"
	"gerrit-operator/pkg/service/gerrit/spec"
	"gerrit-operator/pkg/service/platform"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Client struct {
	instance  *v1alpha1.Gerrit
	resty     resty.Client
	sshClient ssh.SSHClient
}

// InitNewRestClient performs initialization of Gerrit connection
func (gc *Client) InitNewRestClient(instance *v1alpha1.Gerrit, url string, user string, password string) error {
	gc.resty = *resty.SetHostURL(url).SetBasicAuth(user, password)
	gc.instance = instance
	return nil
}

// CheckCredentials checks whether provided creds are correct
func (gc Client) CheckCredentials() (int, error) {
	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		Get("config/server/summary")
	if err != nil {
		return 401, errors.Wrapf(err, "[ERROR] Unable to verify Gerrit credentials")
	}

	return resp.StatusCode(), nil
}

// CheckCredentials checks gerrit group
func (gc Client) CheckGroup(groupName string) (*int, error) {
	statusNotFound := http.StatusNotFound
	uuid, err := gc.getGroupUuid(groupName)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit group uuid")
	}
	if uuid == "" {
		return &statusNotFound, nil
	}

	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		Get(fmt.Sprintf("groups/%v", uuid))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit groups")
	}

	status := resp.StatusCode()

	return &status, nil
}

//GetUser checks gerrit user
func (gc Client) GetUser(username string) (*int, error) {
	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		Get(fmt.Sprintf("accounts/%v", username))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit user")
	}

	status := resp.StatusCode()

	return &status, nil
}

func (gc Client) InitAdminUser(instance v1alpha1.Gerrit, platform platform.PlatformService, GerritScriptsPath string, podName string, gerritAdminPublicKey string) (v1alpha1.Gerrit, error) {
	addInitialAdminUserScript, err := ioutil.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/add-initial-admin-user.sh", GerritScriptsPath)))
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to read add-initial-admin-user.sh script")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", "mkdir -p /tmp/scripts && touch /tmp/scripts/add-initial-admin-user.sh && chmod +x /tmp/scripts/add-initial-admin-user.sh"})
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create add-initial-admin-user.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", fmt.Sprintf("echo \"%v\" > /tmp/scripts/add-initial-admin-user.sh", string(addInitialAdminUserScript))})
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to add content to add-initial-admin-user.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", fmt.Sprintf("sh /tmp/scripts/add-initial-admin-user.sh \"%v\"", gerritAdminPublicKey)})
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to execute add-initial-admin-user.sh script inside gerrit pod")
	}

	return instance, nil
}

func (gc *Client) CreateGroup(groupName string, groupDescription string) error {
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit create-group --description \"%v\" --visible-to-all \"%v\"", groupDescription, groupName),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	_, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "Group %v creation failed: %v", groupName)
	}
	return nil
}

func (gc *Client) ChangePassword(username string, password string) error {
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-account --http-password \"%v\" \"%v\"", password, username),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	_, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "Changing %v password failed", username)
	}
	return nil
}

func (gc *Client) CreateUser(username string, password string, fullname string, publicKey string) error {
	userStatus, err := gc.GetUser(username)
	if err != nil {
		return errors.Wrapf(err, "Getting %v user failed", username)
	}

	if *userStatus == 404 {
		cmd := &ssh.SSHCommand{
			Path: fmt.Sprintf("gerrit create-account --full-name \"%v\" --http-password \"%v\" --ssh-key \"%v\" \"%v\"",
				fullname, password, publicKey, username),
			Env:    []string{},
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		_, err = gc.sshClient.RunCommand(cmd)
		if err != nil {
			return errors.Wrapf(err, "Creating %v user failed", username)
		}
		return nil
	}
	return nil
}

func (gc *Client) getGroupUuid(groupName string) (string, error) {
	var re = regexp.MustCompile(fmt.Sprintf(`%v\t[A-Za-z0-9_]{40}`, groupName))
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprint("gerrit ls-groups -v"),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		return "", errors.Wrap(err, "Receiving Gerrit groups password failed")
	}

	groups := bytes.NewBuffer(out).String()
	group := re.FindStringSubmatch(groups)
	if group == nil {
		return "", err
	}
	uuid := strings.Split(group[0], "\t")[1]

	return uuid, nil
}

func (gc *Client) InitAllProjects(instance v1alpha1.Gerrit, platform platform.PlatformService, GerritScriptsPath string,
	podName string, gerritAdminPublicKey string) error {
	initAllProjectsScript, err := ioutil.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/init-all-projects.sh", GerritScriptsPath)))
	if err != nil {
		return errors.Wrapf(err, "Failed to read init-all-projects.sh script")
	}

	gerritConfig, err := ioutil.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/../gerrit.config", GerritScriptsPath)))
	if err != nil {
		return errors.Wrapf(err, "Failed to read init-all-projects.sh script")
	}

	ciToolsGroupUuid, err := gc.getGroupUuid(spec.GerritCIToolsGroupName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get %v group ID", spec.GerritCIToolsGroupName)
	}

	projectBootstrappersGroupUuid, err := gc.getGroupUuid(spec.GerritProjectBootstrappersGroupName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get %v group ID", spec.GerritCIToolsGroupName)
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", "mkdir -p /tmp/scripts && touch /tmp/scripts/init-all-projects.sh && chmod +x /tmp/scripts/init-all-projects.sh"})
	if err != nil {
		return errors.Wrapf(err, "Failed to create init-all-projects.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", fmt.Sprintf("echo \"%v\" > /tmp/scripts/init-all-projects.sh", string(initAllProjectsScript))})
	if err != nil {
		return errors.Wrapf(err, "Failed to create init-all-projects.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{"/bin/sh", "-c", fmt.Sprintf("sh /tmp/scripts/init-all-projects.sh \"%v\" \"%v\" \"%v\"",
			gerritConfig, ciToolsGroupUuid, projectBootstrappersGroupUuid)})
	if err != nil {
		return errors.Wrapf(err, "Failed to execute init-all-projects.sh script inside gerrit pod")
	}

	return nil
}

func (gc *Client) InitNewSshClient(userName string, privateKey []byte, host string, port int32) error {
	var err error
	gc.sshClient, err = ssh.SshInit(userName, privateKey, host, port)
	return err
}
