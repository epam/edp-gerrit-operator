package helper

import (
	"bytes"
	"fmt"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/helper"
	gerritHelper "github.com/epmd-edp/gerrit-operator/v2/pkg/helper"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"text/template"
)

const (
	DefaultConfigFilesAbsolutePath = "/usr/local/"

	//LocalConfigsRelativePath - default directory for configs
	LocalConfigsRelativePath = "configs"

	//DefaultScriptsDirectory
	DefaultTemplatesDirectory = "templates"

	//DefaultTemplatesDirectory
	DefaultScriptsDirectory = "scripts"

	//LocalTemplatesRelativePath - default directory for templates
	LocalTemplatesRelativePath = DefaultConfigFilesAbsolutePath + LocalConfigsRelativePath + "/" + DefaultTemplatesDirectory

	//LocalScriptsRelativePath - scripts
	LocalScriptsRelativePath = DefaultConfigFilesAbsolutePath + LocalConfigsRelativePath + "/" + DefaultScriptsDirectory

	//JenkinsPluginConfigFileName
	JenkinsPluginConfigFileName = "config-gerrit-plugin.tmpl"
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
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "Unable to get executable file path")
	}

	templatesDirectoryPath := LocalTemplatesRelativePath
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		templatesDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, LocalConfigsRelativePath, DefaultTemplatesDirectory)
	}

	templateAbsolutePath := fmt.Sprintf("%v/%v", templatesDirectoryPath, JenkinsPluginConfigFileName)
	if !gerritHelper.FileExists(templateAbsolutePath) {
		errMsg := fmt.Sprintf("Template file not found in path specificed! Path: %s", templateAbsolutePath)
		return bytes.Buffer{}, errors.New(errMsg)
	}
	t := template.Must(template.New(JenkinsPluginConfigFileName).ParseFiles(templateAbsolutePath))
	err = t.Execute(&ScriptContext, data)
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "Couldn't parse template %v", JenkinsPluginConfigFileName)
	}

	return ScriptContext, nil
}

// GenerateLabels returns initialized map using name parameter
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}
