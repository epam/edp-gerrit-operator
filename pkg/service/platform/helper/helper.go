package helper

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
)

const (
	DefaultConfigFilesAbsolutePath = "/usr/local/"

	LocalConfigsRelativePath = "configs"

	DefaultTemplatesDirectory = "templates"

	DefaultScriptsDirectory = "scripts"

	LocalTemplatesRelativePath = DefaultConfigFilesAbsolutePath + LocalConfigsRelativePath + "/" + DefaultTemplatesDirectory

	LocalScriptsRelativePath = DefaultConfigFilesAbsolutePath + LocalConfigsRelativePath + "/" + DefaultScriptsDirectory

	JenkinsPluginConfigFileName = "config-gerrit-plugin.tmpl"

	RouteHTTPSScheme = "https"

	RouteHTTPScheme = "http"

	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

type JenkinsPluginData struct {
	ServerName   string
	ExternalUrl  string
	SshPort      int32
	UserName     string
	HttpPassword string
}

func InitNewJenkinsPluginInfo() JenkinsPluginData {
	return JenkinsPluginData{}
}

func ParseDefaultTemplate(data JenkinsPluginData) (bytes.Buffer, error) {
	var ScriptContext bytes.Buffer

	executableFilePath, err := GetExecutableFilePath()
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "Unable to get executable file path")
	}

	templatesDirectoryPath := LocalTemplatesRelativePath
	if !RunningInCluster() {
		templatesDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, LocalConfigsRelativePath, DefaultTemplatesDirectory)
	}

	templateAbsolutePath := fmt.Sprintf("%v/%v", templatesDirectoryPath, JenkinsPluginConfigFileName)
	if !fileExists(templateAbsolutePath) {
		errMsg := fmt.Sprintf("Template file not found in path specified! Path: %s", templateAbsolutePath)
		return bytes.Buffer{}, errors.New(errMsg)
	}

	t := template.Must(template.New(JenkinsPluginConfigFileName).ParseFiles(templateAbsolutePath))

	err = t.Execute(&ScriptContext, data)
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "Couldn't parse template %v", JenkinsPluginConfigFileName)
	}

	return ScriptContext, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func GetExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get path for the executable that started process: %w", err)
	}

	return filepath.Dir(executableFilePath), nil
}

func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return !os.IsNotExist(err)
}

// GenerateLabels returns initialized map using name parameter.
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func SelectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	for i := 0; i < len(containers); i++ {
		if containers[i].Name == name {
			return containers[i], nil
		}
	}

	return coreV1Api.Container{}, errors.New("no matching container in spec found")
}

func UpdateEnv(existing, env []coreV1Api.EnvVar) []coreV1Api.EnvVar {
	out := make([]coreV1Api.EnvVar, 0)
	covered := make([]string, 0)

	for _, e := range existing {
		newer, ok := findEnv(env, e.Name)
		if ok {
			covered = append(covered, e.Name)
			out = append(out, newer)

			continue
		}

		out = append(out, e)
	}

	for _, e := range env {
		if helpers.IsStringInSlice(e.Name, covered) {
			continue
		}

		covered = append(covered, e.Name)
		out = append(out, e)
	}

	return out
}

func findEnv(env []coreV1Api.EnvVar, name string) (coreV1Api.EnvVar, bool) {
	for _, e := range env {
		if e.Name == name {
			return e, true
		}
	}

	return coreV1Api.EnvVar{}, false
}
