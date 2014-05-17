// +build !skip_router

package main

import (
	"github.com/openshift/geard/cmd"
	routercmd "github.com/openshift/geard/router/cmd"
)

func init() {
	cmd.AddCommandExtension(routercmd.RegisterRouter, true)
}
