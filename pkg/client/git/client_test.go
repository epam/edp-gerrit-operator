package git

import (
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tmpDir      = "/tmp/git_client_test"
	projectsDir = tmpDir + "/local"
)

func TestClient_Clone_Failure(t *testing.T) {
	cl := New("gerrit base url", tmpDir, "admin", "admin")
	_, err := cl.Clone("random")
	assert.Error(t, err)
}

func TestClient_SetProjectUser(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("test-user-repo", t)

	cl := New("base", tmpDir, "admin", "admin")
	err := cl.SetProjectUser("test-user-repo", &User{Name: "foo", Email: "bar"})
	assert.NoError(t, err)
}

func TestClient_SetProjectUserFailure(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("test-user-repo", t)

	cl := New("base", tmpDir, "admin", "admin")
	err := cl.SetProjectUser("test-user-repo1", &User{Name: "foo", Email: "bar"})
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to open repository: repository does not exist")
}

func createFakeProject(name string, t *testing.T) {
	err := os.MkdirAll(projectsDir, 0o777)
	assert.NoError(t, err)

	cloneRepo := path.Join(tmpDir, name)
	repo, err := git.PlainInit(cloneRepo, false)
	assert.NoError(t, err)

	repoConf, err := repo.Config()
	assert.NoError(t, err)

	repoConf.User = struct {
		Name  string
		Email string
	}{Name: "John Doe", Email: "john@doe.org"}
	err = repo.SetConfig(repoConf)

	assert.NoError(t, err)

	fp, err := os.Create(path.Join(cloneRepo, "test.txt"))
	assert.NoError(t, err)

	_, err = fp.WriteString("test")
	assert.NoError(t, err)

	err = fp.Close()
	assert.NoError(t, err)

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = cloneRepo

	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "init commit")
	cmd.Dir = cloneRepo

	bts, err := cmd.CombinedOutput()
	assert.NoError(t, err, string(bts))
}

func TestClient_Clone(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("test-clone", t)

	cl := New(tmpDir, projectsDir, "admin", "admin")

	clonePath, err := cl.Clone("test-clone")
	assert.NoError(t, err)

	err = os.RemoveAll(clonePath)
	assert.NoError(t, err)
}

func TestClient_Merge(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	projectName := "test-merge"

	createFakeProject(projectName, t)

	cmd := exec.Command("git", "branch", "demo")
	cmd.Dir = path.Join(tmpDir, projectName)
	err := cmd.Run()
	assert.NoError(t, err)

	cl := New(tmpDir, tmpDir, "admin", "admin")
	err = cl.Merge(projectName, "demo", "master")
	assert.NoError(t, err)
}

func TestClient_Merge_Failure(t *testing.T) {
	cl := New(tmpDir, tmpDir, "admin", "admin")
	err := cl.Merge("test", "demo", "master")
	assert.Error(t, err)
}

func TestClient_GerritBaseURL(t *testing.T) {
	cl := New("test", "", "", "")
	if cl.GerritBaseURL() != "test" {
		t.Fatal("wrong gerrit base url")
	}
}

func TestClient_GenerateChangeID(t *testing.T) {
	cl := Client{}
	_, err := cl.GenerateChangeID()
	assert.NoError(t, err)
}

func TestClient_SetFileContents(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("demo", t)

	cl := New(tmpDir, tmpDir, "admin", "admin")
	err := cl.SetFileContents("demo", "test.txt", "test")
	assert.NoError(t, err)

	err = cl.SetFileContents("demo-fail", "test.txt", "test")
	assert.Error(t, err)
}

func TestClient_Commit(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("demo", t)

	cl := New(tmpDir, tmpDir, "admin", "admin")
	err := cl.SetFileContents("demo", "kwest.txt", "kwest")
	assert.NoError(t, err)

	err = cl.Commit("demo", "test commit", []string{"kwest.txt"},
		&User{Name: "mike", Email: "mk@gmail.com"})
	assert.NoError(t, err)

	err = cl.Commit("demo-failure", "fail", []string{}, nil)
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to open repository: repository does not exist")

	err = cl.Commit("demo", "test commit", []string{"11kwest.txt"},
		&User{Name: "mike", Email: "mk@gmail.com"})
	assert.EqualError(t, err, "unable to add file: 11kwest.txt: entry not found")
}

func TestClient_CheckoutBranch(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("demo", t)

	cmd := exec.Command("git", "branch", "test-checkout")
	cmd.Dir = path.Join(tmpDir, "demo")
	err := cmd.Run()
	assert.NoError(t, err)

	cl := New(tmpDir, tmpDir, "admin", "admin")
	err = cl.CheckoutBranch("demo", "test-checkout")
	assert.NoError(t, err)

	err = cl.CheckoutBranch("demo", "test-checkout-2")
	assert.EqualError(t, err, "unable to checkout to branch: test-checkout-2: reference not found")

	err = cl.CheckoutBranch("demo-failure", "test-checkout")
	assert.EqualError(t, err, "unable to open repository: repository does not exist")
}

func TestClient_Push(t *testing.T) {
	defer removeAllWithErrCapture(t, tmpDir)

	createFakeProject("demo", t)

	cl := New(tmpDir, tmpDir, "admin", "admin")
	_, err := cl.Push("demo", "origin", "HEAD:refs/for/master")
	assert.EqualError(t, err, "unable to create new remote: unable to get origin remote: remote not found")
}

func removeAllWithErrCapture(t *testing.T, p string) {
	err := os.RemoveAll(p)
	require.NoError(t, err)
}

func TestClient_RemoveFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	projectDir := path.Join(tmp, "test")
	require.NoError(t, os.MkdirAll(projectDir, 0o777))

	tests := []struct {
		name     string
		filePath string
		prepare  func(t *testing.T)
		want     bool
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name:     "file exists",
			filePath: "test.txt",
			prepare: func(t *testing.T) {
				_, err := os.Create(path.Join(projectDir, "test.txt"))
				require.NoError(t, err)
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name:     "file does not exist",
			filePath: "test.txt",
			prepare: func(t *testing.T) {
			},
			want:    false,
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			c := &Client{
				workingDir: tmp,
			}

			got, err := c.RemoveFile("test", tt.filePath)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
