package helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1Api "k8s.io/api/core/v1"
)

const name = "name"

func TestFileExists(t *testing.T) {
	if fileExists("is not") {
		t.Fatal("file is not exists")
	}

	fp, err := os.Create("/tmp/test")
	require.NoError(t, err)

	err = fp.Close()
	require.NoError(t, err)

	if !fileExists("/tmp/test") {
		t.Fatal("file exists")
	}

	err = os.Remove("/tmp/test")
	assert.NoError(t, err)
}

func TestGetExecutableFilePath(t *testing.T) {
	if _, err := GetExecutableFilePath(); err != nil {
		t.Fatal(err)
	}
}

func TestParseDefaultTemplate(t *testing.T) {
	data := JenkinsPluginData{}
	_, err := ParseDefaultTemplate(data)
	assert.Error(t, err)
}

func TestGenerateLabels(t *testing.T) {
	labels := GenerateLabels(name)
	assert.Equal(t, map[string]string{"app": name}, labels)
}

func Test_findEnv_False(t *testing.T) {
	var env []coreV1Api.EnvVar
	envVar, b := findEnv(env, name)
	assert.False(t, b)
	assert.Equal(t, coreV1Api.EnvVar{}, envVar)
}

func Test_findEnv_True(t *testing.T) {
	env := []coreV1Api.EnvVar{
		{Name: name},
	}
	envVar, b := findEnv(env, name)
	assert.True(t, b)
	assert.Equal(t, coreV1Api.EnvVar{Name: name}, envVar)
}

func TestSelectContainerErr(t *testing.T) {
	var containers []coreV1Api.Container

	containerName := "name"

	_, err := SelectContainer(containers, containerName)
	assert.Error(t, err)
}

func TestSelectContainer(t *testing.T) {
	containers := []coreV1Api.Container{
		{Name: name},
	}
	name := "name"
	c, err := SelectContainer(containers, name)
	assert.NoError(t, err)
	assert.Equal(t, coreV1Api.Container{Name: name}, c)
}

func Test_UpdateEnv(t *testing.T) {
	env1 := []coreV1Api.EnvVar{
		{Name: name + "1"},
		{Name: name + "2"},
	}
	env2 := []coreV1Api.EnvVar{
		{Name: name + "2"},
	}
	sum := UpdateEnv(env1, env2)
	assert.Equal(t, env1, sum)
}

func TestInitNewJenkinsPluginInfo(t *testing.T) {
	assert.Equal(t, JenkinsPluginData{}, InitNewJenkinsPluginInfo())
}
