package jobs

import (
	"github.com/openshift/geard/jobs"
)

type PurgeContainersRequest struct{}

func (p *PurgeContainersRequest) Execute(res jobs.Response) {
	Clean()
}
