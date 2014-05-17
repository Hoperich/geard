// +build !skip_ssh

package main

import (
	"github.com/openshift/geard/cmd"
	sshcmd "github.com/openshift/geard/ssh/cmd"
)

func init() {
	command := &sshcmd.Command{&defaultTransport.TransportFlag}
	cmd.AddCommandExtension(command.RegisterAddKeys, false)
	cmd.AddCommandExtension(sshcmd.RegisterAuthorizedKeys, true)
}
