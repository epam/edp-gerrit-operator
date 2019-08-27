package controller

import (
	"gerrit-operator/pkg/controller/gerritreplicationconfig"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gerritreplicationconfig.Add)
}
