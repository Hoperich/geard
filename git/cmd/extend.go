// +build !disable_git

package cmd

import (
	. "github.com/openshift/geard/cmd"
	"github.com/openshift/geard/git"
	githttp "github.com/openshift/geard/git/http"
	"github.com/openshift/geard/http"
	"github.com/spf13/cobra"
)

func init() {
	AddInitializer(git.InitializeData, WhenDaemon)

	http.AddHttpExtension(githttp.Routes)

	AddCommandExtension(func(parent *cobra.Command) {
		createCmd := &cobra.Command{
			Use:   "create-repo",
			Short: "Create a new git repository",
			Run:   repoCreate,
		}
		parent.AddCommand(createCmd)
	}, false)
	AddCommandExtension(func(parent *cobra.Command) {
		initRepoCmd := &cobra.Command{
			Use:   "init-repo",
			Short: `(Local) Setup the environment for a git repository`,
			Long:  ``,
			Run:   initRepository,
		}
		parent.AddCommand(initRepoCmd)
	}, true)
}
