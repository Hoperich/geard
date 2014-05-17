// +build linux && !skip_ssh

package main

import (
	"github.com/openshift/geard/http"
	"github.com/openshift/geard/jobs"
	sshhttp "github.com/openshift/geard/ssh/http"
	sshjobs "github.com/openshift/geard/ssh/jobs"
)

func init() {
	jobs.AddJobExtension(sshjobs.NewSshExtension())
	http.AddHttpExtension(&sshhttp.HttpExtension{})
}
