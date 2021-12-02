package gerrit

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type AccessInfo struct {
	RefPattern      string `json:"refPattern"`
	PermissionName  string `json:"permissionName"`
	PermissionLabel string `json:"permissionLabel,omitempty"`
	GroupName       string `json:"groupName"`
	Action          string `json:"action"`
	Force           bool   `json:"force"`
	Min             int    `json:"min"`
	Max             int    `json:"max"`
}

type groupPermissions struct {
	Min      int    `json:"min"`
	Max      int    `json:"max"`
	Force    bool   `json:"force"`
	Action   string `json:"action"`
	Modified bool   `json:"modified"`
	Added    bool   `json:"added"`
}

type permission struct {
	Label string                      `json:"label"`
	Rules map[string]groupPermissions `json:"rules"`
}

type reference struct {
	Permissions map[string]permission `json:"permissions"`
}

func (gc Client) AddAccessRights(projectName string, permissions []AccessInfo) error {
	accessInfo := generateSetAccessRequest(permissions, true, false)
	addRequest := map[string]map[string]reference{"add": accessInfo}

	rsp, err := gc.resty.R().SetBody(addRequest).SetHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/projects/%s/access", projectName))

	return parseRestyResponse(rsp, err)
}

func (gc Client) UpdateAccessRights(projectName string, permissions []AccessInfo) error {
	accessInfo := generateSetAccessRequest(permissions, false, true)
	addRequest := map[string]map[string]reference{"add": accessInfo, "remove": accessInfo}

	rsp, err := gc.resty.R().SetBody(addRequest).SetHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/projects/%s/access", projectName))

	return parseRestyResponse(rsp, err)
}

func (gc Client) DeleteAccessRights(projectName string, permissions []AccessInfo) error {
	accessInfo := generateSetAccessRequest(permissions, false, false)
	addRequest := map[string]map[string]reference{"remove": accessInfo}

	rsp, err := gc.resty.R().SetBody(addRequest).SetHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/projects/%s/access", projectName))

	return parseRestyResponse(rsp, err)
}

func generateSetAccessRequest(permissions []AccessInfo, added bool, modified bool) map[string]reference {
	refs := make(map[string]reference)

	for _, perm := range permissions {
		_, ok := refs[perm.RefPattern]
		if !ok {
			ref := reference{Permissions: make(map[string]permission)}
			refs[perm.RefPattern] = ref
		}

		_, ok = refs[perm.RefPattern].Permissions[perm.PermissionName]
		if !ok {
			permName := permission{Rules: make(map[string]groupPermissions), Label: perm.PermissionLabel}
			refs[perm.RefPattern].Permissions[perm.PermissionName] = permName
		}

		_, ok = refs[perm.RefPattern].Permissions[perm.PermissionName].Rules[perm.GroupName]
		if !ok {
			groupPerm := groupPermissions{
				Action:   perm.Action,
				Force:    perm.Force,
				Max:      perm.Max,
				Min:      perm.Min,
				Added:    added,
				Modified: modified,
			}

			refs[perm.RefPattern].Permissions[perm.PermissionName].Rules[perm.GroupName] = groupPerm
		}
	}

	return refs
}

func (gc Client) SetProjectParent(projectName, parentName string) error {
	rsp, err := gc.resty.R().SetBody(map[string]string{
		"parent": parentName,
	}).SetHeader("Content-Type", "application/json").
		Put(fmt.Sprintf("/projects/%s/parent", projectName))

	return parseRestyResponse(rsp, err)
}

func parseRestyResponse(rsp *resty.Response, err error) error {
	if err != nil {
		return errors.Wrap(err, "error during post request")
	}

	if rsp.IsError() {
		return errors.Errorf("status: %s, body: %s", rsp.Status(), rsp.String())
	}

	return nil
}
