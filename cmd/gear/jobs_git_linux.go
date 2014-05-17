// +build linux && !skip_git

package main

import (
	githttp "github.com/openshift/geard/git/http"
	gitjobs "github.com/openshift/geard/git/jobs"
	"github.com/openshift/geard/http"
	"github.com/openshift/geard/jobs"
)

func init() {
	jobs.AddJobExtension(gitjobs.NewGitExtension())
	http.AddHttpExtension(&githttp.HttpExtension{})
}
