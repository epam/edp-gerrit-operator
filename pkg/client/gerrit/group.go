package gerrit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
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

type Group struct {
	ID      string        `json:"id"`
	GroupID int           `json:"group_id"`
	Members []GroupMember `json:"members"`
}

type GroupMember struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (gc *Client) RemoveUsersFromGroup(users []v1alpha1.GerritUsers, processedUsers map[string]struct{}) error {
	usersGroups, err := gc.getUserGroups()
	if err != nil {
		return errors.Wrap(err, "unable to get users groups")
	}

	currentUsers := make(map[string]v1alpha1.GerritUsers)
	for _, u := range users {
		currentUsers[u.Username] = u
	}

	for username := range processedUsers {
		if err := gc.checkProcessedUserGroups(usersGroups, currentUsers, username); err != nil {
			return errors.Wrap(err, "unable to check processed user groups")
		}
	}

	return nil
}

func (gc *Client) checkProcessedUserGroups(usersGroups map[string][]string, currentUsers map[string]v1alpha1.GerritUsers,
	username string) error {
	currentUser, ok := currentUsers[username]

	if !ok {
		for _, ug := range usersGroups[username] {
			if err := gc.DeleteUserFromGroup(ug, username); err != nil {
				return errors.Wrap(err, "unable to delete user from group")
			}
		}

		return nil
	}

	for _, gr := range usersGroups[currentUser.Username] {
		groupDelete := true
		for _, currentUserGroup := range currentUser.Groups {
			if gr == currentUserGroup {
				groupDelete = false
				break
			}
		}

		if groupDelete {
			if err := gc.DeleteUserFromGroup(gr, username); err != nil {
				return errors.Wrap(err, "unable to delete user from group")
			}
		}
	}

	return nil
}

func (gc *Client) getUserGroups() (map[string][]string, error) {
	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		Get("groups/?o=MEMBERS")
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit groups")
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("wrong response code: %d, body: %s", resp.StatusCode(), resp.String())
	}

	body := resp.String()[5:]
	var groups map[string]Group
	if err := json.Unmarshal([]byte(body), &groups); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal group response")
	}

	usersGroups := make(map[string][]string)
	for groupName, gr := range groups {
		for _, u := range gr.Members {
			usersGroups[u.Email] = append(usersGroups[u.Email], groupName)
		}
	}

	return usersGroups, nil
}

func (gc *Client) DeleteUserFromGroup(groupName, username string) error {
	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		Delete(fmt.Sprintf("groups/%s/members/%s", groupName, username))
	if err != nil {
		return errors.Wrapf(err, "Unable to get Gerrit groups")
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("wrong response code: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (gc *Client) UpdateGroup(groupID, description string, visibleToAll bool) error {
	resp, err := gc.resty.R().
		SetHeader("accept", "application/json").
		SetHeader("content-type", "application/json").
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
		SetHeader("accept", "application/json").
		SetHeader("content-type", "application/json").
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
		SetHeader("accept", "application/json").
		SetHeader("content-type", "application/json").
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
		if resp.StatusCode() == 409 {
			return nil, ErrAlreadyExists("already exists")
		}

		return nil, errors.Errorf("status: %s, body: %s", resp.Status(), resp.String())
	}

	body := resp.String()[5:]
	var gr Group
	if err := json.Unmarshal([]byte(body), &gr); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal group response")
	}

	return &gr, nil
}
