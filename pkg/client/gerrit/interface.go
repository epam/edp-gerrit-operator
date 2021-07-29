package gerrit

import "gopkg.in/resty.v1"

type ClientInterface interface {
	GetResty() *resty.Client
	SetProjectParent(projectName, parentName string) error
	DeleteAccessRights(projectName string, permissions []AccessInfo) error
	UpdateAccessRights(projectName string, permissions []AccessInfo) error
	AddAccessRights(projectName string, permissions []AccessInfo) error
	CreateGroup(name, description string, visibleToAll bool) (*Group, error)
	UpdateGroup(groupID, description string, visibleToAll bool) error
}
