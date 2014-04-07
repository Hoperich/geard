package ssh

import (
	"encoding/json"
	"github.com/openshift/geard/config"
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/utils"
	"os"
	"path/filepath"
)

func init() {
	AddPermissionHandler("", &containerPermission{})
}

type containerPermission struct{}

func (c containerPermission) CreatePermission(locator KeyLocator, value json.RawMessage) error {
	var idString string
	if err := json.Unmarshal(value, &idString); err != nil {
		return err
	}

	id, err := containers.NewIdentifier(idString)
	if err != nil {
		return err
	}

	if _, err := os.Stat(id.UnitPathFor()); err != nil {
		return err
	}
	if err := os.Symlink(locator.PathToKey(), SshAccessPathFor(id, locator.NameForKey())); err != nil && !os.IsExist(err) {
		return err
	}
	if _, err := os.Stat(id.AuthKeysPathFor()); err == nil {
		if err := os.Remove(id.AuthKeysPathFor()); err != nil {
			return err
		}
	}
	return nil
}

func SshAccessBasePath(i containers.Identifier) string {
	return utils.IsolateContentPathWithPerm(filepath.Join(config.ContainerBasePath(), "access", "containers", "ssh"), string(i), "", 0775)
}

func SshAccessPathFor(i containers.Identifier, name string) string {
	return utils.IsolateContentPathWithPerm(filepath.Join(config.ContainerBasePath(), "access", "containers", "ssh"), string(i), name, 0775)
}
