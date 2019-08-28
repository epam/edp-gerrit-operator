package helper

import (
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("helper")

func NewTrue() *bool {
	value := true
	return &value
}
