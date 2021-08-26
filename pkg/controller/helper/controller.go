package helper

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller interface {
	SetupWithManager(mgr ctrl.Manager) error
}

type InitFunc struct {
	ControllerName string
	Func           func(client client.Client, scheme *runtime.Scheme, log logr.Logger) (Controller, error)
}
