// +build linux

package jobs

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/jobs"
	"github.com/openshift/geard/systemd"
	"github.com/openshift/geard/utils"
	"github.com/openshift/go-systemd/dbus"
)

const (
	buildImage     = "pmorie/sti-builder-go"
	gearBinaryPath = "/usr/bin/gear"
)

func (j *BuildImageRequest) Execute(resp jobs.Response) {
	w := resp.SuccessWithWrite(jobs.ResponseAccepted, true, false)

	fmt.Fprintf(w, "Processing build-image request:\n")
	// TODO: download source, add bind-mount

	unitName := containers.JobIdentifier(j.Name).UnitNameForBuild()
	unitDescription := fmt.Sprintf("Builder for %s", j.Tag)

	stdout, err := systemd.ProcessLogsForUnit(unitName)
	if err != nil {
		stdout = utils.EmptyReader
		log.Printf("job_build_image: Unable to fetch build logs: %s, %+v", err.Error(), err)
	}
	defer stdout.Close()

	conn, errc := systemd.NewConnection()
	if errc != nil {
		log.Print("job_build_image:", errc)
		fmt.Fprintf(w, "Unable to watch start status", errc)
		return
	}

	if err := conn.Subscribe(); err != nil {
		log.Print("job_build_image:", err)
		fmt.Fprintf(w, "Unable to watch start status", errc)
		return
	}
	defer conn.Unsubscribe()

	// make subscription global for efficiency
	changes, errch := conn.SubscribeUnitsCustom(1*time.Second, 2,
		func(s1 *dbus.UnitStatus, s2 *dbus.UnitStatus) bool {
			return true
		},
		func(unit string) bool {
			return unit != unitName
		})

	fmt.Fprintf(w, "Running sti build unit: %s\n", unitName)
	log.Printf("build_image: Running build %s", unitName)

	var startCmd []string

	if _, err := os.Stat(gearBinaryPath); err != nil {
		log.Println("gear executable is not installed on system; using sti builder image")
		startCmd = []string{
			"/usr/bin/docker", "run",
			"-rm",
			"-v", "/run/docker.sock:/run/docker.sock",
			"-t", buildImage,
			"sti", "build", j.Source, j.BaseImage, j.Tag,
			"-U", "unix:///run/docker.sock",
		}
	} else {
		startCmd = []string{
			gearBinaryPath, "build", j.Source, j.BaseImage, j.Tag,
		}
	}

	if j.RuntimeImage != "" {
		startCmd = append(startCmd, "--runtime-image")
		startCmd = append(startCmd, j.RuntimeImage)
	}

	if j.Clean {
		startCmd = append(startCmd, "--clean")
	}

	if j.Verbose {
		startCmd = append(startCmd, "--debug")
	}

	log.Printf("build_image: Will execute %v", startCmd)
	status, err := systemd.Connection().StartTransientUnit(
		unitName,
		"fail",
		dbus.PropExecStart(startCmd, true),
		dbus.PropDescription(unitDescription),
		dbus.PropRemainAfterExit(true),
		dbus.PropSlice("container-small.slice"),
	)

	if err != nil {
		errType := reflect.TypeOf(err)
		fmt.Fprintf(w, "Unable to start build container for this image due to (%s): %s\n", errType, err.Error())
		return
	} else if status != "done" {
		fmt.Fprintf(w, "Build did not complete successfully: %s\n", status)
	} else {
		fmt.Fprintf(w, "Sti build is running\n")
	}

	go io.Copy(w, stdout)

wait:
	for {
		select {
		case c := <-changes:
			if changed, ok := c[unitName]; ok {
				if changed.SubState != "running" {
					fmt.Fprintf(w, "Build completed\n")
					break wait
				}
			}
		case err := <-errch:
			fmt.Fprintf(w, "Error %+v\n", err)
		case <-time.After(25 * time.Second):
			log.Print("job_build_image:", "timeout")
			break wait
		}
	}

	stdout.Close()
}
