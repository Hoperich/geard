// +build linux

package main

import (
	chttp "github.com/openshift/geard/containers/http"
	cjobs "github.com/openshift/geard/containers/jobs"
	"github.com/openshift/geard/http"
	"github.com/openshift/geard/jobs"
)

func init() {
	jobs.AddJobExtension(cjobs.NewContainerExtension())
	http.AddHttpExtension(&chttp.HttpExtension{})
}
