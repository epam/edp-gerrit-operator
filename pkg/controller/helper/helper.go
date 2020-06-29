package helper

import (
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("helper")

const (
	//platformType
	platformType = "PLATFORM_TYPE"
)

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