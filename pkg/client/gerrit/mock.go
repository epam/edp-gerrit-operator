package gerrit

import (
	"github.com/stretchr/testify/mock"
	"gopkg.in/resty.v1"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) GetResty() *resty.Client {
	panic("not implemented")
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
	panic("not implemented")
}

func (m *Mock) UpdateGroup(groupID, description string, visibleToAll bool) error {
	panic("not implemented")
}
