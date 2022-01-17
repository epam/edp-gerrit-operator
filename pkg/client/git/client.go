package git

import (
	"crypto/sha1"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"time"

	"github.com/go-git/go-git/v5/config"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
)

type Client struct {
	username, password string
	workingDir         string
	gerritBaseURL      string
}

func (c Client) GerritBaseURL() string {
	return c.gerritBaseURL
}

func New(gerritBaseURL, workingDir, username, password string) *Client {
	return &Client{
		workingDir:    workingDir,
		username:      username,
		password:      password,
		gerritBaseURL: gerritBaseURL,
	}
}

func (c *Client) projectPath(projectName string) string {
	return path.Join(c.workingDir, projectName)
}

func (c *Client) Clone(projectName string) (projectPath string, err error) {
	projectPath = c.projectPath(projectName)
	_, err = git.PlainClone(
		projectPath, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: c.username,
				Password: c.password,
			},
			URL: fmt.Sprintf("%s/%s", c.gerritBaseURL, projectName),
		})

	if err != nil {
		return "", errors.Wrap(err, "unable to clone repository")
	}

	return
}

func (c *Client) Merge(projectName, sourceBranch, targetBranch string, options ...string) error {
	projectDir := c.projectPath(projectName)

	cmd := exec.Command("git", "checkout", targetBranch)
	cmd.Dir = projectDir

	bts, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(bts))
	}

	mergeOpts := []string{"merge", sourceBranch}
	mergeOpts = append(mergeOpts, options...)

	cmd = exec.Command("git", mergeOpts...)
	cmd.Dir = projectDir

	bts, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(bts))
	}

	return nil
}

func (c *Client) Push(projectName string, remote string, refSpecs ...string) (pushOutput string, retErr error) {
	projectPath := c.projectPath(projectName)

	r, err := git.PlainOpen(projectPath)
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	unsecureRemoteName := fmt.Sprintf("unsecure_%s", remote)
	_, err = c.createRemoteWithCredential(r, remote, unsecureRemoteName)
	if err != nil {
		return "", errors.Wrap(err, "unable to create new remote")
	}

	defer func() {
		if err = r.DeleteRemote(unsecureRemoteName); err != nil {
			retErr = errors.Wrap(err, "unable to delete tmp remote")
		}
	}()

	pushArgs := []string{"push", unsecureRemoteName}
	pushArgs = append(pushArgs, refSpecs...)

	cmd := exec.Command("git", pushArgs...)
	cmd.Dir = projectPath

	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, string(bts))
	}

	return string(bts), nil
}

func (c *Client) createRemoteWithCredential(repo *git.Repository, baseRemoteName, newRemoteName string) (*git.Remote, error) {
	origin, err := repo.Remote(baseRemoteName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get origin remote")
	}

	if len(origin.Config().URLs) == 0 {
		return nil, errors.New("remote does not have valid urls")
	}

	originURL, err := url.Parse(origin.Config().URLs[0])
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse origin url")
	}

	originURL.User = url.UserPassword(c.username, c.password)

	newRemote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: newRemoteName, URLs: []string{originURL.String()}})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create remote")
	}

	return newRemote, nil
}

func (c *Client) GenerateChangeID() (string, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(time.Now().Format(time.RFC3339))); err != nil {
		return "", errors.Wrap(err, "unable to write hash")
	}
	return fmt.Sprintf("I%x", h.Sum(nil)), nil
}
