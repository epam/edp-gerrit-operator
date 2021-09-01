package gerrit

import (
	"errors"
	"testing"
)

func TestMock_All(t *testing.T) {
	m := Mock{}
	if m.GetResty() != m.restyClient {
		t.Fatal("wrong resty client returned")
	}

	m.On("SetProjectParent", "foo", "bar").Return(nil)
	if err := m.SetProjectParent("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	m.On("DeleteAccessRights", "foo", []AccessInfo{}).Return(nil)
	if err := m.DeleteAccessRights("foo", []AccessInfo{}); err != nil {
		t.Fatal(err)
	}

	m.On("UpdateAccessRights", "foo", []AccessInfo{}).Return(nil)
	if err := m.UpdateAccessRights("foo", []AccessInfo{}); err != nil {
		t.Fatal(err)
	}

	m.On("AddAccessRights", "foo", []AccessInfo{}).Return(nil)
	if err := m.AddAccessRights("foo", []AccessInfo{}); err != nil {
		t.Fatal(err)
	}

	m.On("CreateGroup", "foo", "bar", false).Return(&Group{}, nil).Once()
	if _, err := m.CreateGroup("foo", "bar", false); err != nil {
		t.Fatal(err)
	}

	m.On("CreateGroup", "foo", "bar", false).Return(nil, errors.New("fatal")).Once()
	if _, err := m.CreateGroup("foo", "bar", false); err == nil {
		t.Fatal("no error")
	}

	m.On("UpdateGroup", "foo", "bar", false).Return(nil)
	if err := m.UpdateGroup("foo", "bar", false); err != nil {
		t.Fatal(err)
	}

	m.On("AddUserToGroup", "foo", "bar").Return(nil)
	m.On("DeleteUserFromGroup", "foo", "bar").Return(nil)

	if err := m.AddUserToGroup("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	if err := m.DeleteUserFromGroup("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	m.On("CreateProject", &Project{}).Return(nil)
	if err := m.CreateProject(&Project{}); err != nil {
		t.Fatal(err)
	}

	m.On("GetProject", "test").Return(&Project{}, nil).Once()
	if _, err := m.GetProject("test"); err != nil {
		t.Fatal(err)
	}

	m.On("GetProject", "test").Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetProject("test"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("UpdateProject", &Project{}).Return(nil)
	if err := m.UpdateProject(&Project{}); err != nil {
		t.Fatal(err)
	}

	m.On("DeleteProject", "test").Return(nil)
	if err := m.DeleteProject("test"); err != nil {
		t.Fatal(err)
	}

	m.On("ListProjects", "test").Return([]Project{}, nil).Once()
	if _, err := m.ListProjects("test"); err != nil {
		t.Fatal(err)
	}

	m.On("ListProjects", "test").Return(nil, errors.New("fatal")).Once()
	if _, err := m.ListProjects("test"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("ListProjectBranches", "test").Return([]Branch{}, nil).Once()
	if _, err := m.ListProjectBranches("test"); err != nil {
		t.Fatal(err)
	}

	m.On("ListProjectBranches", "test").Return(nil, errors.New("fatal")).Once()
	if _, err := m.ListProjectBranches("test"); err == nil {
		t.Fatal("no error returned")
	}
}
