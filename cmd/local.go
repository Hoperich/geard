package cmd

import (
	"github.com/openshift/geard/jobs"
	"github.com/openshift/geard/transport"
)

type localTransport struct {
	remote transport.Transport
}

// Create a transport that will invoke the default job implementation for a given
// request with a local locator, and pass any other requests to the remote transport.
func NewTransport(remote transport.Transport) *localTransport {
	t := &localTransport{remote}
	return t
}

func (h *localTransport) LocatorFor(value string) (transport.Locator, error) {
	if transport.Local.String() != value {
		return h.remote.LocatorFor(value)
	}
	return transport.Local, nil
}

func (h *localTransport) RemoteJobFor(locator transport.Locator, j interface{}) (job jobs.Job, err error) {
	if locator != transport.Local {
		return h.remote.RemoteJobFor(locator, j)
	}

	job, err = jobs.JobFor(j)
	if err != nil {
		return
	}
	return
}
