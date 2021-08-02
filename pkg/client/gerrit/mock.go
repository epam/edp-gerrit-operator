package gerrit

import (
	"github.com/stretchr/testify/mock"
	"gopkg.in/resty.v1"
)

type Mock struct {
	mock.Mock
	restyClient *resty.Client
}

func (m *Mock) GetResty() *resty.Client {
	return m.restyClient
}

func (m *Mock) SetProjectParent(projectName, parentName string) error {
	return m.Called(projectName, parentName).Error(0)
}

func (m *Mock) DeleteAccessRights(projectName string, permissions []AccessInfo) error {
	return m.Called(projectName, permissions).Error(0)
}

func (m *Mock) UpdateAccessRights(projectName string, permissions []AccessInfo) error {
	return m.Called(projectName, permissions).Error(0)
}
func (m *Mock) AddAccessRights(projectName string, permissions []AccessInfo) error {
	return m.Called(projectName, permissions).Error(0)
}

func (m *Mock) CreateGroup(name, description string, visibleToAll bool) (*Group, error) {
	called := m.Called(name, description, visibleToAll)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*Group), nil
}

func (m *Mock) UpdateGroup(groupID, description string, visibleToAll bool) error {
	return m.Called(groupID, description, visibleToAll).Error(0)
}

func (m *Mock) AddUserToGroup(groupName, username string) error {
	return m.Called(groupName, username).Error(0)
}

func (m *Mock) DeleteUserFromGroup(groupName, username string) error {
	return m.Called(groupName, username).Error(0)
}
