// +build linux

package main

import (
	// Commands only available on linux systems
	cleancmd "github.com/openshift/geard/cleanup/cmd"
	"github.com/openshift/geard/cmd"
	initcmd "github.com/openshift/geard/containers/systemd/init"
)

func init() {
	cmd.AddCommandExtension(cleancmd.RegisterCleanup, true)
	cmd.AddCommandExtension(initcmd.RegisterInit, true)
}
