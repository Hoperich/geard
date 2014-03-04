package cmd

import (
	"fmt"
	"github.com/smarterclayton/cobra"
	"github.com/smarterclayton/geard/http"
	"github.com/smarterclayton/geard/jobs"
	"github.com/smarterclayton/geard/pkg/logstreamer"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
)

type check interface {
	Check() error
}

// A simple executor that groups each remote / local system and simultaneous streams
// output to the client.  Exits with 0 if all succeeded or the first error code.
func run(cmd *cobra.Command, localInit func(), init func(...Locator) jobs.Job, on ...Locator) {
	exitch := make(chan int, len(on))
	stdout := log.New(cmd.Out(), "", log.Ldate|log.Ltime)
	wg := &sync.WaitGroup{}
	local, remote := Locators(on).Group()

	if len(local) > 0 {
		localInit()
		go func() {
			wg.Add(1)
			w := logstreamer.NewLogstreamer(stdout, "local ", false)
			defer w.Close()
			defer wg.Done()

			job := init(local...)
			if check, ok := job.(check); ok {
				if err := check.Check(); err != nil {
					fmt.Fprintf(w, "Not valid: %s", err.Error())
					exitch <- 1
					return
				}
			}
			response := &CliJobResponse{stdout: w, stderr: w}
			job.Execute(response)
			if response.exitCode != 0 {
				if response.message == "" {
					response.message = "Command failed"
				}
				fmt.Fprintf(w, response.message)
			}
			exitch <- response.exitCode
		}()
	}

	for i := range remote {
		go func() {
			wg.Add(1)
			ids := remote[i]
			host := ids[0].Identity()
			locator := ids[0].(http.RemoteLocator)
			w := logstreamer.NewLogstreamer(stdout, host+" ", false)
			defer w.Close()
			defer wg.Done()

			dispatcher := http.NewHttpDispatcher(locator, log.New(w, "", 0))

			job := init(ids...)
			if check, ok := job.(check); ok {
				if err := check.Check(); err != nil {
					fmt.Fprintf(w, "Not valid: %s", err.Error())
					exitch <- 1
					return
				}
			}

			code := 0
			if remotable, ok := job.(http.RemoteExecutable); ok {
				response := &CliJobResponse{stdout: w, stderr: w}
				if err := dispatcher.Dispatch(remotable, response); err != nil {
					fmt.Fprintf(w, "Unable to retrieve response: %s", err.Error())
				} else if response.exitCode != 0 {
					code = response.exitCode
					if response.message == "" {
						response.message = "Command failed"
					}
					fmt.Fprintf(w, response.message)
				}
			} else {
				fmt.Fprintf(w, "Unable to run this action (%+v) against a remote server", reflect.TypeOf(job))
				code = 1
			}
			exitch <- code
		}()
	}

	var code int
	select {
	case code = <-exitch:
	}
	wg.Wait()
	os.Exit(code)
}

// A simple executor that runs commands on different servers in parallel but invokes
// one job per identifier.
func runEach(cmd *cobra.Command, localInit func(), init func(Locator) jobs.Job, on ...Locator) {
	exitch := make(chan int, len(on))
	stdout := log.New(cmd.Out(), "", log.Ldate|log.Ltime)
	wg := &sync.WaitGroup{}
	local, remote := Locators(on).Group()

	if len(local) > 0 {
		localInit()
		go func() {
			wg.Add(1)
			w := logstreamer.NewLogstreamer(stdout, "local ", false)
			defer w.Close()
			defer wg.Done()

			code := 0
			for i := range local {
				job := init(local[i])
				if check, ok := job.(check); ok {
					if err := check.Check(); err != nil {
						fmt.Fprintf(w, "Not valid: %s", err.Error())
						code = 1
						continue
					}
				}
				response := &CliJobResponse{stdout: w, stderr: w}
				job.Execute(response)
				if response.exitCode != 0 {
					if response.message == "" {
						response.message = "Command failed"
					}
					fmt.Fprintf(w, response.message)
					code = response.exitCode
				}
			}
			exitch <- code
		}()
	}

	for i := range remote {
		go func() {
			wg.Add(1)
			ids := remote[i]
			host := ids[0].Identity()
			locator := ids[0].(http.RemoteLocator)
			w := logstreamer.NewLogstreamer(stdout, host+" ", false)
			defer w.Close()
			defer wg.Done()

			dispatcher := http.NewHttpDispatcher(locator, log.New(w, "", 0))

			code := 0
			for j := range ids {
				job := init(ids[j])
				if check, ok := job.(check); ok {
					if err := check.Check(); err != nil {
						fmt.Fprintf(w, "Not valid: %s", err.Error())
						code = 1
						continue
					}
				}
				if remotable, ok := job.(http.RemoteExecutable); ok {
					response := &CliJobResponse{stdout: w, stderr: w}
					if err := dispatcher.Dispatch(remotable, response); err != nil {
						fmt.Fprintf(w, "Unable to retrieve response: %s", err.Error())
					} else if response.exitCode != 0 {
						code = response.exitCode
						if response.message == "" {
							response.message = "Command failed"
						}
						fmt.Fprintf(w, response.message)
					}
				} else {
					fmt.Fprintf(w, "Unable to run this action (%+v) against a remote server", reflect.TypeOf(job))
					if code == 0 {
						code = 1
					}
				}
			}
			exitch <- code
		}()
	}

	var code int
	select {
	case code = <-exitch:
	}
	wg.Wait()
	os.Exit(code)
}

func fail(code int, format string, other ...interface{}) {
	fmt.Fprintf(os.Stderr, format, other...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(os.Stderr)
	}
	os.Exit(code)
}