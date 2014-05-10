package http

import (
	"encoding/json"
	"errors"
	"fmt"
	cjobs "github.com/openshift/geard/containers/jobs"
	"github.com/openshift/geard/jobs"
	"github.com/openshift/geard/transport"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const DefaultHttpPort = "43273"

type RemoteLocator interface {
	ToURL() *url.URL
}

type RemoteJob interface {
	HttpMethod() string
	HttpPath() string
}
type RemoteExecutable interface {
	RemoteJob
	MarshalRequestIdentifier() jobs.RequestIdentifier
	MarshalUrlQuery(*url.Values)
	MarshalHttpRequestBody(io.Writer) error
	UnmarshalHttpResponse(headers http.Header, r io.Reader, mode ResponseContentMode) (interface{}, error)
}

type HttpTransport struct {
	client *http.Client
}

func NewHttpTransport() *HttpTransport {
	return &HttpTransport{&http.Client{}}
}

func (h *HttpTransport) LocatorFor(value string) (transport.Locator, error) {
	return transport.NewHostLocator(value)
}

func (h *HttpTransport) RemoteJobFor(locator transport.Locator, j interface{}) (job jobs.Job, err error) {
	baseUrl, errl := urlForLocator(locator)
	if errl != nil {
		err = errors.New("The provided host is not valid '" + locator.String() + "': " + errl.Error())
		return
	}
	httpJob, errh := HttpJobFor(j)
	if errh == jobs.ErrNoJobForRequest {
		err = transport.ErrNotTransportable
		return
	}
	if errh != nil {
		err = errh
		return
	}

	job = jobs.JobFunction(func(res jobs.Response) {
		if err := h.ExecuteRemote(baseUrl, httpJob, res); err != nil {
			res.Failure(err)
		}
	})
	return
}

func urlForLocator(locator transport.Locator) (*url.URL, error) {
	base := locator.String()
	if strings.Contains(base, ":") {
		host, port, err := net.SplitHostPort(base)
		if err != nil {
			return nil, err
		}
		if port == "" {
			base = net.JoinHostPort(host, DefaultHttpPort)
		}
	} else {
		base = net.JoinHostPort(base, DefaultHttpPort)
	}
	return &url.URL{Scheme: "http", Host: base}, nil
}

func HttpJobFor(job interface{}) (exc RemoteExecutable, err error) {
	switch j := job.(type) {
	case *cjobs.InstallContainerRequest:
		exc = &HttpInstallContainerRequest{InstallContainerRequest: *j}
	case *cjobs.StartedContainerStateRequest:
		exc = &HttpStartContainerRequest{StartedContainerStateRequest: *j}
	case *cjobs.StoppedContainerStateRequest:
		exc = &HttpStopContainerRequest{StoppedContainerStateRequest: *j}
	case *cjobs.RestartContainerRequest:
		exc = &HttpRestartContainerRequest{RestartContainerRequest: *j}
	case *cjobs.PutEnvironmentRequest:
		exc = &HttpPutEnvironmentRequest{PutEnvironmentRequest: *j}
	case *cjobs.PatchEnvironmentRequest:
		exc = &HttpPatchEnvironmentRequest{PatchEnvironmentRequest: *j}
	case *cjobs.ContainerStatusRequest:
		exc = &HttpContainerStatusRequest{ContainerStatusRequest: *j}
	case *cjobs.ContentRequest:
		exc = &HttpContentRequest{ContentRequest: *j}
	case *cjobs.DeleteContainerRequest:
		exc = &HttpDeleteContainerRequest{DeleteContainerRequest: *j}
	case *cjobs.LinkContainersRequest:
		exc = &HttpLinkContainersRequest{LinkContainersRequest: *j}
	case *cjobs.ListContainersRequest:
		exc = &HttpListContainersRequest{ListContainersRequest: *j}
	default:
		for _, ext := range extensions {
			req, errr := ext.HttpJobFor(job)
			if errr == jobs.ErrNoJobForRequest {
				continue
			}
			if errr != nil {
				return nil, errr
			}
			return req, nil
		}
		err = jobs.ErrNoJobForRequest
	}
	return
}

func (h *HttpTransport) ExecuteRemote(baseUrl *url.URL, job RemoteExecutable, res jobs.Response) error {
	reader, writer := io.Pipe()
	httpreq, errn := http.NewRequest(job.HttpMethod(), baseUrl.String(), reader)
	if errn != nil {
		return errn
	}

	id := job.MarshalRequestIdentifier()
	if len(id) == 0 {
		id = jobs.NewRequestIdentifier()
	}

	query := &url.Values{}
	job.MarshalUrlQuery(query)

	req := httpreq
	req.Header.Set("X-Request-Id", id.String())
	req.Header.Set("If-Match", "api="+ApiVersion())
	req.Header.Set("Content-Type", "application/json")
	//TODO: introduce API version per job
	//TODO: content request signing for GETs
	req.URL.Path = job.HttpPath()
	req.URL.RawQuery = query.Encode()
	go func() {
		if err := job.MarshalHttpRequestBody(writer); err != nil {
			log.Printf("http_remote: Error when writing to http: %v", err)
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	isJson := resp.Header.Get("Content-Type") == "application/json"

	switch code := resp.StatusCode; {
	case code == 202:
		if isJson {
			return errors.New("Decoding of streaming JSON has not been implemented")
		}
		data, err := job.UnmarshalHttpResponse(resp.Header, nil, ResponseTable)
		if err != nil {
			return err
		}
		if pending, ok := data.(map[string]interface{}); ok {
			for k := range pending {
				res.WritePendingSuccess(k, pending[k])
			}
		}
		w := res.SuccessWithWrite(jobs.ResponseOk, false, false)
		if _, err := io.Copy(w, resp.Body); err != nil {
			return err
		}
	case code == 204:
		data, err := job.UnmarshalHttpResponse(resp.Header, nil, ResponseTable)
		if err != nil {
			return err
		}
		if pending, ok := data.(map[string]interface{}); ok {
			for k := range pending {
				res.WritePendingSuccess(k, pending[k])
			}
		}
		res.Success(jobs.ResponseOk)
	case code >= 200 && code < 300:
		if !isJson {
			return errors.New(fmt.Sprintf("remote: Response with %d status code had content type %s (should be application/json)", code, resp.Header.Get("Content-Type")))
		}
		data, err := job.UnmarshalHttpResponse(nil, resp.Body, ResponseJson)
		if err != nil {
			return err
		}
		res.SuccessWithData(jobs.ResponseOk, data)
	default:
		if isJson {
			decoder := json.NewDecoder(resp.Body)
			data := httpFailureResponse{}
			if err := decoder.Decode(&data); err != nil {
				return err
			}
			res.Failure(jobs.SimpleError{jobs.ResponseError, data.Message})
			return nil
		}
		io.Copy(os.Stderr, resp.Body)
		res.Failure(jobs.SimpleError{jobs.ResponseError, "Unable to decode response."})
	}
	return nil
}
