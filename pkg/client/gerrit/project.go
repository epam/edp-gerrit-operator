package gerrit

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
)

type Project struct {
	Name              string `json:"name"`
	Parent            string `json:"parent,omitempty"`
	Description       string `json:"description,omitempty"`
	PermissionsOnly   bool   `json:"permissions_only"`
	CreateEmptyCommit bool   `json:"create_empty_commit"`
	SubmitType        string `json:"submit_type,omitempty"`
	Branches          string `json:"branches,omitempty"`
	Owners            string `json:"owners,omitempty"`
	RejectEmptyCommit string `json:"reject_empty_commit,omitempty"`
}

func (p *Project) SlugifyName() string {
	return slug.Make(p.Name)
}

type Branch struct {
	Ref       string    `json:"ref"`
	Revision  string    `json:"revision"`
	CanDelete bool      `json:"can_delete"`
	WebLinks  []WebLink `json:"web_link"`
}

type WebLink struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	ImageURL string `json:"image_url"`
	Target   string `json:"target"`
}

func (gc *Client) CreateProject(prj *Project) error {
	rsp, err := gc.resty.R().SetBody(prj).SetHeader(contentType, applicationJson).
		Put(fmt.Sprintf("/projects/%s", url.QueryEscape(prj.Name)))

	return parseRestyResponse(rsp, err)
}

func (gc *Client) GetProject(name string) (*Project, error) {
	rsp, err := gc.resty.R().
		SetHeader(acceptHeader, applicationJson).
		Get(fmt.Sprintf("/projects/%s", url.QueryEscape(name)))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit project")
	}

	if rsp.StatusCode() != http.StatusOK {
		if rsp.StatusCode() == http.StatusNotFound {
			return nil, DoesNotExistError("does not exists")
		}

		return nil, errors.Errorf("wrong response code: %d, body: %s", rsp.StatusCode(), rsp.String())
	}

	var prj Project
	if err := decodeGerritResponse(rsp.String(), &prj); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal project response")
	}

	return &prj, nil
}

func (gc *Client) UpdateProject(prj *Project) error {
	rsp, err := gc.resty.R().SetHeader(contentType, applicationJson).
		SetBody(map[string]string{
			"description":    prj.Description,
			"commit_message": "Update the project description",
		}).Put(fmt.Sprintf("/projects/%s/description", url.QueryEscape(prj.Name)))

	err = parseRestyResponse(rsp, err)
	if err != nil {
		return errors.Wrap(err, "unable to update project description")
	}

	rsp, err = gc.resty.R().SetHeader(contentType, applicationJson).
		SetBody(map[string]string{
			"parent":         prj.Parent,
			"commit_message": "Update the project parent",
		}).Put(fmt.Sprintf("/projects/%s/parent", url.QueryEscape(prj.Name)))

	return parseRestyResponse(rsp, err)
}

func (gc *Client) DeleteProject(name string) error {
	rsp, err := gc.resty.R().SetHeader(contentType, applicationJson).
		SetBody(map[string]bool{
			"force":    false,
			"preserve": false,
		}).Post(fmt.Sprintf("/projects/%s/delete-project~delete", url.QueryEscape(name)))

	return parseRestyResponse(rsp, err)
}

func (gc *Client) ListProjects(_type string) ([]Project, error) {
	rsp, err := gc.resty.R().SetHeader(acceptHeader, applicationJson).
		Get(fmt.Sprintf("/projects/?type=%s&d=1&t=1", _type))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit project")
	}

	if rsp.IsError() {
		return nil, errors.Errorf("wrong response code: %d, body: %s", rsp.StatusCode(), rsp.String())
	}

	var preProjects map[string]Project
	if err := decodeGerritResponse(rsp.String(), &preProjects); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal project response, body: %s", rsp.String())
	}

	delete(preProjects, "All-Projects")
	delete(preProjects, "All-Projects")
	delete(preProjects, "All-Users")

	projects := make([]Project, 0, len(preProjects))

	for k, v := range preProjects {
		v.Name = k
		projects = append(projects, v)
	}

	return projects, nil
}

func (gc *Client) ListProjectBranches(projectName string) ([]Branch, error) {
	rsp, err := gc.resty.R().SetHeader(acceptHeader, applicationJson).
		Get(fmt.Sprintf("/projects/%s/branches/", url.QueryEscape(projectName)))
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Gerrit project branches")
	}

	if rsp.IsError() {
		return nil, errors.Errorf("wrong response code: %d, body: %s", rsp.StatusCode(), rsp.String())
	}

	var branches []Branch
	if err := decodeGerritResponse(rsp.String(), &branches); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal project response, body: %s", rsp.String())
	}

	return branches, nil
}
