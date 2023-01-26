package gerrit

import (
	"gopkg.in/resty.v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

type ClientInterface interface {
	Resty() *resty.Client
	SetProjectParent(projectName, parentName string) error
	DeleteAccessRights(projectName string, permissions []AccessInfo) error
	UpdateAccessRights(projectName string, permissions []AccessInfo) error
	AddAccessRights(projectName string, permissions []AccessInfo) error
	CreateGroup(name, description string, visibleToAll bool) (*Group, error)
	UpdateGroup(groupID, description string, visibleToAll bool) error
	AddUserToGroup(groupName, username string) error
	DeleteUserFromGroup(groupName, username string) error
	CreateProject(prj *Project) error
	GetProject(name string) (*Project, error)
	UpdateProject(prj *Project) error
	DeleteProject(name string) error
	ListProjects(_type string) ([]Project, error)
	ListProjectBranches(projectName string) ([]Branch, error)
	ReloadPlugin(plugin string) error
	ChangeAbandon(changeID string) error
	ChangeGet(changeID string) (*Change, error)
	InitNewRestClient(instance *gerritApi.Gerrit, url string, user string, password string) error
	CheckCredentials() (int, error)
	InitAdminUser(instance *gerritApi.Gerrit, platform platform.PlatformService, GerritScriptsPath string, podName string, gerritAdminPublicKey string) (*gerritApi.Gerrit, error)
	InitNewSshClient(userName string, privateKey []byte, host string, port int32) error
	CheckGroup(groupName string) (*int, error)
	InitAllProjects(instance *gerritApi.Gerrit, platform platform.PlatformService, GerritScriptsPath string,
		podName string, gerritAdminPublicKey string) error
	CreateUser(username string, password string, fullName string, publicKey string) error
	ChangePassword(username string, password string) error
	AddUserToGroups(userName string, groupNames []string) error
}
