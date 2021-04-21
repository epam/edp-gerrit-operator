package helper

import (
	"fmt"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

var log = ctrl.Log.WithName("helper")

const (
	platformType           = "PLATFORM_TYPE"
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetDebugMode returns the debug mode value
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, err
	}
	return b, nil
}

// Check whether the operator is running in cluster or locally
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func NewTrue() *bool {
	value := true
	return &value
}

func GetPlatformTypeEnv() string {
	platformType, found := os.LookupEnv(platformType)
	if !found {
		panic("Environment variable PLATFORM_TYPE is not defined")
	}
	return platformType
}

func GetExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executableFilePath), nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
