package jobs

import (
	"github.com/smarterclayton/geard/gears"
	"github.com/smarterclayton/geard/systemd"
	"log"
	"os"
	"path/filepath"
)

type DeleteContainerRequest struct {
	GearId gears.Identifier
}

func (j *DeleteContainerRequest) Execute(resp JobResponse) {
	unitName := j.GearId.UnitNameFor()
	unitPath := j.GearId.UnitPathFor()
	socketUnitPath := j.GearId.SocketUnitPathFor()
	unitDefinitionPath := j.GearId.UnitDefinitionPathFor()

	_, err := systemd.Connection().GetUnitProperties(unitName)
	switch {
	case systemd.IsNoSuchUnit(err):
		resp.Success(JobResponseOk)
		return
	case err != nil:
		resp.Failure(ErrDeleteContainerFailed)
		return
	}

	if err := systemd.Connection().StopUnitJob(unitName, "fail"); err != nil {
		log.Printf("delete_container: Unable to queue stop unit job: %v", err)
	}

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		resp.Failure(ErrDeleteContainerFailed)
		return
	}

	if err := os.Remove(socketUnitPath); err != nil && !os.IsNotExist(err) {
		log.Printf("delete_container: Unable to remove socket unit path: %v", err)
	}

	ports, err := gears.GetExistingPorts(j.GearId)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("delete_container: Unable to read existing port definitions: %v", err)
		}
		ports = gears.PortPairs{}
	}

	if err := os.RemoveAll(filepath.Dir(unitDefinitionPath)); err != nil {
		log.Printf("delete_container: Unable to remove definitions for gear: %v", err)
	}

	if err := gears.ReleaseExternalPorts(filepath.Dir(unitDefinitionPath), ports); err != nil {
		log.Printf("delete_container: Unable to release ports: %v", err)
	}

	if _, err := systemd.Connection().DisableUnitFiles([]string{unitPath, socketUnitPath}, false); err != nil {
		log.Printf("delete_container: Some units have not been disabled: %v", err)
	}

	resp.Success(JobResponseOk)
}