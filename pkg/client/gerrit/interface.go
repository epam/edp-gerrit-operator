package gerrit

import (
	"gopkg.in/resty.v1"
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
}
