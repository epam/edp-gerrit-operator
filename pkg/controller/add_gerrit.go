package controller

import (
	"gerrit-operator/pkg/controller/gerrit"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gerrit.Add)
}
