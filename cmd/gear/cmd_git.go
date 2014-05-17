// +build !skip_git

package main

import (
	"github.com/openshift/geard/cmd"
	gitcmd "github.com/openshift/geard/git/cmd"
)

func init() {
	command := &gitcmd.Command{&defaultTransport.TransportFlag}
	cmd.AddCommandExtension(command.RegisterCreateRepo, false)
	cmd.AddCommandExtension(gitcmd.RegisterInitRepo, true)
}
