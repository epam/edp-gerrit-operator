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
}
