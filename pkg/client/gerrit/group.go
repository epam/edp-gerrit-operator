package gerrit

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const (
	contentType = "Content-Type"
)

type ErrAlreadyExists string

func (e ErrAlreadyExists) Error() string {
	return string(e)
}

func IsErrAlreadyExists(err error) bool {
	switch errors.Cause(err).(type) {
	case ErrAlreadyExists:
		return true
	}

	return false
}

type ErrDoesNotExist string

func (e ErrDoesNotExist) Error() string {
	return string(e)
}

func IsErrDoesNotExist(err error) bool {
	_, ok := errors.Cause(err).(ErrDoesNotExist)
	return ok
}

type Group struct {
	ID      string        `json:"id"`
	GroupID int           `json:"group_id"`
	Members []GroupMember `json:"members"`
}

type GroupMember struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Deleted unused func
//func (gc *Client) getUserGroups() (map[string][]string, error) {}

func (gc *Client) DeleteUserFromGroup(groupName, username string) error {
	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		Delete(fmt.Sprintf("groups/%s/members/%s", groupName, username))
	if err != nil {
		return errors.Wrapf(err, "Unable to get Gerrit groups")
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("wrong response code: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (gc *Client) AddUserToGroup(groupName, username string) error {
	resp, err := gc.resty.R().Put(fmt.Sprintf("groups/%s/members/%s", groupName, username))
	return parseRestyResponse(resp, err)
}

func (gc *Client) UpdateGroup(groupID, description string, visibleToAll bool) error {
	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		SetHeader(contentType, applicationJson).
		SetBody(map[string]interface{}{
			"description": description,
		}).
		Put(fmt.Sprintf("groups/%s/description", groupID))

	if err != nil {
		return errors.Wrap(err, "unable to update group")
	}

	if resp.IsError() {
		return errors.Errorf("status: %s, body: %s", resp.Status(), resp.String())
	}

	resp, err = gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		SetHeader(contentType, applicationJson).
		SetBody(map[string]interface{}{
			"visible_to_all": visibleToAll,
		}).
		Put(fmt.Sprintf("groups/%s/options", groupID))

	if err != nil {
		return errors.Wrap(err, "unable to update group")
	}

	if resp.IsError() {
		return errors.Errorf("status: %s, body: %s", resp.Status(), resp.String())
	}

	return nil
}

func (gc *Client) CreateGroup(name, description string, visibleToAll bool) (*Group, error) {
	resp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		SetHeader(contentType, applicationJson).
		SetBody(map[string]interface{}{
			"description":    description,
			"name":           name,
			"visible_to_all": visibleToAll,
		}).
		Put(fmt.Sprintf("groups/%s", name))

	if err != nil {
		return nil, errors.Wrap(err, "unable to create group")
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusConflict {
			return nil, ErrAlreadyExists("already exists")
		}

		return nil, errors.Errorf("status: %s, body: %s", resp.Status(), resp.String())
	}

	var gr Group
	if err := decodeGerritResponse(resp.String(), &gr); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal group response")
	}

	return &gr, nil
}
