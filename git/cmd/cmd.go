package cmd

import (
	. "github.com/openshift/geard/cmd"
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/git"
	"github.com/openshift/geard/git/http"
	gitjobs "github.com/openshift/geard/git/jobs"
	"github.com/openshift/geard/jobs"
	_ "github.com/openshift/geard/ssh"
	"github.com/openshift/geard/systemd"
	"github.com/spf13/cobra"
	"os"
)

func repoCreate(c *cobra.Command, args []string) {
	if len(args) < 1 {
		Fail(1, "Valid arguments: <id> [<clone repo url>]\n")
	}

	id, err := NewGenericLocator(ResourceTypeRepository, args[0])
	if err != nil {
		Fail(1, "You must pass one valid repository name: %s\n", err.Error())
	}

	if id.ResourceType() != ResourceTypeRepository {
		Fail(1, "You must pass one valid repository name: %s\n", err.Error())
	}

	Executor{
		On: Locators{id},
		Serial: func(on Locator) jobs.Job {
			var req http.HttpCreateRepositoryRequest
			req = http.HttpCreateRepositoryRequest{}
			req.Id = git.RepoIdentifier(on.(ResourceLocator).Identifier())

			return &req
		},
		Output:    os.Stdout,
		LocalInit: LocalInitializers(systemd.Start, containers.InitializeData),
	}.StreamAndExit()
}

func initRepository(cmd *cobra.Command, args []string) {
	if len(args) < 1 || len(args) > 2 {
		Fail(1, "Valid arguments: <repo_id> [<repo_url>]\n")
	}

	repoId, err := containers.NewIdentifier(args[0])
	if err != nil {
		Fail(1, "Argument 1 must be a valid repository identifier: %s\n", err.Error())
	}

	repoUrl := ""
	if len(args) == 2 {
		repoUrl = args[1]
	}

	if err := systemd.Start(); err != nil {
		Fail(2, err.Error())
	}
	if err := gitjobs.InitializeRepository(git.RepoIdentifier(repoId), repoUrl); err != nil {
		Fail(2, "Unable to initialize repository %s\n", err.Error())
	}
}
