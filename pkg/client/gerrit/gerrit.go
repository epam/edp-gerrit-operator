package gerrit

import (
	"fmt"
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gerrit-operator/pkg/client/ssh"
	"gerrit-operator/pkg/service/platform"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	"io/ioutil"
	"path/filepath"
	"log"
	"os"
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
		return instance, errors.Wrapf(err, "[ERROR] Failed to create add-initial-admin-user.sh script inside gerrit pod")
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
		Path:   fmt.Sprintf("gerrit create-group --description '%v' --visible-to-all '%v'", groupDescription, groupName),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	_, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		log.Printf("[ERROR] Create %v group failed: %v", groupName, err)
	}
	return err
}

func (gc *Client) InitNewSshClient(userName string, privateKey []byte, host string, port int32) error {
	var err error
	gc.sshClient, err = ssh.SshInit(userName, privateKey, host, port)
	return err
}
