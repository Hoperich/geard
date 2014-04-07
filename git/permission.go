package git

import (
	"encoding/json"
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/ssh"
	"os"
)

func init() {
	ssh.AddPermissionHandler("repository", &repositoryPermission{})
}

type repositoryPermission struct{}

type permission struct {
	Id    string
	Write bool
}

func (r repositoryPermission) CreatePermission(locator ssh.KeyLocator, value json.RawMessage) error {
	p := permission{}
	if err := json.Unmarshal(value, &p); err != nil {
		return err
	}

	id, err := containers.NewIdentifier(p.Id)
	if err != nil {
		return err
	}
	repoId := RepoIdentifier(id)

	if _, err := os.Stat(repoId.RepositoryPathFor()); err != nil {
		return err
	}
	accessPath := repoId.GitAccessPathFor(locator.NameForKey(), p.Write)

	if err := os.Symlink(locator.PathToKey(), accessPath); err != nil && !os.IsExist(err) {
		return err
	}
	negAccessPath := repoId.GitAccessPathFor(locator.NameForKey(), !p.Write)
	if err := os.Remove(negAccessPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(repoId.AuthKeysPathFor()); err == nil {
		if err := os.Remove(repoId.AuthKeysPathFor()); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
