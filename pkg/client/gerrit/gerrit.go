package gerrit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	ctrl "sigs.k8s.io/controller-runtime"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/ssh"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	acceptHeader    = "Accept"
	applicationJson = "application/json"
	path            = "/bin/sh"
	minBodyLength   = 5
	containerFlag   = "-c"
)

var log = ctrl.Log.WithName("client_gerrit")

type Client struct {
	instance  *gerritApi.Gerrit //TODO: remove this
	resty     *resty.Client
	sshClient ssh.SSHClientInterface
}

func NewClient(instance *gerritApi.Gerrit, resty *resty.Client, sshClient ssh.SSHClientInterface) Client {
	return Client{
		instance: instance,
		resty: resty.SetHeaders(map[string]string{
			acceptHeader: applicationJson,
		}),
		sshClient: sshClient,
	}
}

func (gc *Client) Resty() *resty.Client {
	return gc.resty
}

// InitNewRestClient performs initialization of Gerrit connection.
func (gc *Client) InitNewRestClient(instance *gerritApi.Gerrit, url string, user string, password string) error {
	gc.resty = resty.SetHostURL(url).SetBasicAuth(user, password).SetDisableWarn(true)
	gc.instance = instance
	return nil
}

func (gc *Client) InitNewSshClient(userName string, privateKey []byte, host string, port int32) error {
	client, err := ssh.SshInit(userName, privateKey, host, port, log)
	if err != nil {
		return errors.Wrap(err, "err while initializing new ssh client")
	}
	gc.sshClient = &client
	return nil
}

// CheckCredentials checks whether provided creds are correct.
func (gc Client) CheckCredentials() (int, error) {
	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		Get("config/server/summary")
	if err != nil {
		return 0, errors.Wrapf(err, "Unable to verify Gerrit credentials")
	}

	return resp.StatusCode(), nil
}

// CheckGroup checks gerrit group.
func (gc Client) CheckGroup(groupName string) (*int, error) {
	vLog := log.WithValues("group name", groupName)
	vLog.Info("checking group...")
	statusNotFound := http.StatusNotFound
	uuid, err := gc.getGroupUuid(groupName)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit group uuid")
	}
	if uuid == "" {
		vLog.Info("group wasn't found")
		return &statusNotFound, nil
	}

	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		Get(fmt.Sprintf("groups/%v", uuid))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit groups")
	}

	status := resp.StatusCode()

	return &status, nil
}

// GetUser checks gerrit user.
func (gc Client) GetUser(username string) (*int, error) {
	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		Get(fmt.Sprintf("accounts/%v", username))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit user")
	}

	status := resp.StatusCode()

	return &status, nil
}

func (gc Client) InitAdminUser(instance gerritApi.Gerrit, platform platform.PlatformService, GerritScriptsPath string, podName string, gerritAdminPublicKey string) (gerritApi.Gerrit, error) {
	addInitialAdminUserScript, err := os.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/add-initial-admin-user.sh", GerritScriptsPath)))
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to read add-initial-admin-user.sh script")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag, "mkdir -p /tmp/scripts && touch /tmp/scripts/add-initial-admin-user.sh && chmod +x /tmp/scripts/add-initial-admin-user.sh"})
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to create add-initial-admin-user.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag, fmt.Sprintf("echo \"%v\" > /tmp/scripts/add-initial-admin-user.sh", string(addInitialAdminUserScript))})
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to add content to add-initial-admin-user.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag, fmt.Sprintf("sh /tmp/scripts/add-initial-admin-user.sh \"%v\"", gerritAdminPublicKey)})
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to execute add-initial-admin-user.sh script inside gerrit pod")
	}

	return instance, nil
}

func (gc *Client) ChangePassword(username string, password string) error {
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit set-account --http-password \"%v\" \"%v\"", password, username),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	out, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "Changing %v password failed. %v", username, bytes.NewBuffer(out).String())
	}
	return nil
}

func (gc *Client) ReloadPlugin(plugin string) error {
	cmd := &ssh.SSHCommand{
		Path:   fmt.Sprintf("gerrit plugin reload \"%v\"", plugin),
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	_, err := gc.sshClient.RunCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "Reloading %v plugin failed", plugin)
	}
	return nil
}

func (gc *Client) CreateUser(username string, password string, fullname string, publicKey string) error {
	log.Info("creating user", "name", username)
	userStatus, err := gc.GetUser(username)
	if err != nil {
		return errors.Wrapf(err, "Getting %v user failed", username)
	}

	if *userStatus == http.StatusNotFound {
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

func (gc *Client) AddUserToGroups(userName string, groupNames []string) error {
	for _, group := range groupNames {
		groupStatus, err := gc.CheckGroup(group)
		if err != nil {
			return err
		}

		if *groupStatus == http.StatusNotFound {
			log.Info(fmt.Sprintf("Group %v not found in Gerrit", group))
		} else {
			cmd := &ssh.SSHCommand{
				Path:   fmt.Sprintf("gerrit set-members --add \"%v\" \"%v\"", userName, group),
				Env:    []string{},
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}

			_, err := gc.sshClient.RunCommand(cmd)
			if err != nil {
				return errors.Wrapf(err, "Failed to add user %v to group %v", userName, group)
			}
		}
	}
	return nil
}

func (gc *Client) getGroupUuid(groupName string) (string, error) {
	var re = regexp.MustCompile(fmt.Sprintf(`%v\t[A-Za-z0-9_]{40}`, groupName))
	cmd := &ssh.SSHCommand{
		Path:   "gerrit ls-groups -v",
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

func (gc *Client) InitAllProjects(instance gerritApi.Gerrit, platform platform.PlatformService, GerritScriptsPath string,
	podName string, gerritAdminPublicKey string) error {
	initAllProjectsScript, err := os.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/init-all-projects.sh", GerritScriptsPath)))
	if err != nil {
		return errors.Wrapf(err, "Failed to read init-all-projects.sh script")
	}

	gerritConfig, err := os.ReadFile(filepath.FromSlash(fmt.Sprintf("%v/../gerrit.config", GerritScriptsPath)))
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

	developersGroupUuid, err := gc.getGroupUuid(spec.GerritProjectDevelopersGroupName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get %v group ID", spec.GerritCIToolsGroupName)
	}

	readOnlyGroupUuid, err := gc.getGroupUuid(spec.GerritReadOnlyGroupName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get %s group ID", spec.GerritReadOnlyGroupName)
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag, "mkdir -p /tmp/scripts && touch /tmp/scripts/init-all-projects.sh && chmod +x /tmp/scripts/init-all-projects.sh"})
	if err != nil {
		return errors.Wrapf(err, "Failed to create init-all-projects.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag, fmt.Sprintf("echo \"%v\" > /tmp/scripts/init-all-projects.sh", string(initAllProjectsScript))})
	if err != nil {
		return errors.Wrapf(err, "Failed to create init-all-projects.sh script inside gerrit pod")
	}

	_, _, err = platform.ExecInPod(instance.Namespace, podName,
		[]string{path, containerFlag,
			fmt.Sprintf("sh /tmp/scripts/init-all-projects.sh \"%v\" \"%v\" \"%v\" \"%v\" \"%v\"",
				string(gerritConfig), ciToolsGroupUuid, projectBootstrappersGroupUuid, developersGroupUuid, readOnlyGroupUuid)})
	if err != nil {
		return errors.Wrapf(err, "Failed to execute init-all-projects.sh script inside gerrit pod")
	}

	return nil
}

func decodeGerritResponse(body string, v interface{}) error {
	if len(body) < minBodyLength {
		return errors.New("wrong gerrit body format")
	}
	// gerrit has prefix )]}' in all responses so we need to truncate it
	if err := json.Unmarshal([]byte(body[minBodyLength:]), v); err != nil {
		return errors.Wrap(err, "unable to decode gerrit response")
	}

	return nil
}
