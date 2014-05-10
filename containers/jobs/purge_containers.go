package jobs

import (
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/jobs"
)

type PurgeContainersRequest struct{}

func (p *PurgeContainersRequest) Execute(res jobs.Response) {
	containers.Clean()
}
