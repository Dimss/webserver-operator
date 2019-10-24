package controller

import (
	"github.com/webserver-operator/pkg/controller/webserver"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, webserver.Add)
}
