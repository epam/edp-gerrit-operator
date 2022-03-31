package gerrit

import (
	"gopkg.in/resty.v1"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
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
	InitNewRestClient(instance *v1alpha1.Gerrit, url string, user string, password string) error
	CheckCredentials() (int, error)
	InitAdminUser(instance v1alpha1.Gerrit, platform platform.PlatformService, GerritScriptsPath string, podName string, gerritAdminPublicKey string) (v1alpha1.Gerrit, error)
	InitNewSshClient(userName string, privateKey []byte, host string, port int32) error
	CheckGroup(groupName string) (*int, error)
	InitAllProjects(instance v1alpha1.Gerrit, platform platform.PlatformService, GerritScriptsPath string,
		podName string, gerritAdminPublicKey string) error
	CreateUser(username string, password string, fullname string, publicKey string) error
	ChangePassword(username string, password string) error
	AddUserToGroups(userName string, groupNames []string) error
}
