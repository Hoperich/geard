package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	. "github.com/openshift/geard/cmd"
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/jobs"
	sshkey "github.com/openshift/geard/pkg/ssh-public-key"
	"github.com/openshift/geard/ssh"
	. "github.com/openshift/geard/ssh/http"
	. "github.com/openshift/geard/ssh/jobs"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

var (
	writeAccess bool
	keyFile     string
)

func registerLocal(parent *cobra.Command) {
	keysForUserCmd := &cobra.Command{
		Use:   "auth-keys-command <username>",
		Short: "(Local) Generate authorized_keys output for sshd.",
		Long:  "Generate authorized keys output for sshd. See sshd_config(5)#AuthorizedKeysCommand",
		Run:   keysForUser,
	}
	parent.AddCommand(keysForUserCmd)
}

func keysForUser(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		Fail(1, "Valid arguments: <login name>\n")
	}

	u, err := user.Lookup(args[0])
	if err != nil {
		Fail(2, "Unable to lookup user")
	}

	if err := ssh.GenerateAuthorizedKeysFor(u); err != nil {
		Fail(1, "Unable to generate authorized_keys file: %s", err.Error())
	}
}

func registerRemote(parent *cobra.Command) {
	addKeysCmd := &cobra.Command{
		Use:   "add-keys <id>...",
		Short: "Set keys for SSH access to a resource",
		Long:  "Upload the provided public keys and enable SSH access to the specified resources (containers, repositories, ...).",
		Run:   addSshKeys,
	}
	addKeysCmd.Flags().BoolVar(&writeAccess, "write", false, "Enable write access for the selected keys and repositories")
	addKeysCmd.Flags().StringVar(&keyFile, "key-file", "", "read input from file specified matching sshd AuthorizedKeysFile format")
	parent.AddCommand(addKeysCmd)
}

func addSshKeys(cmd *cobra.Command, args []string) {
	// validate that arguments for locators are passsed
	if len(args) < 1 {
		Fail(1, "Valid arguments: <id> ...")
	}
	// args... are locators for repositories or containers
	ids, err := NewGenericLocators(ResourceTypeContainer, args...)
	if err != nil {
		Fail(1, "You must pass 1 or more valid names: %s", err.Error())
	}
	for i := range ids {
		switch ids[i].ResourceType() {
		case ResourceTypeContainer, ResourceTypeRepository:
		default:
			Fail(1, "Only repositories or containers may be specified")
		}
	}

	keys, err := readAuthorizedKeysFile(keyFile)
	if err != nil {
		Fail(1, "Unable to read authorized keys file: %s", err.Error())
	}

	Executor{
		On: ids,
		Group: func(on ...Locator) jobs.Job {
			perms := []KeyPermission{}
			for i := range on {
				perm := KeyPermission{}
				//id := on[i].(ResourceLocator).Identifier()
				switch on[i].ResourceType() {
				case ResourceTypeContainer:
					perm.Type = "container"
				case ResourceTypeRepository:
					perm.Type = "repository"
				}
				perms = append(perms, perm)
			}

			return &HttpCreateKeysRequest{
				CreateKeysRequest: CreateKeysRequest{
					&ExtendedCreateKeysData{
						Keys:        keys,
						Permissions: perms,
					},
				},
			}
		},
		Output:    os.Stdout,
		LocalInit: containers.InitializeData,
	}.StreamAndExit()
}

func readAuthorizedKeysFile(keyFile string) ([]KeyData, error) {
	var (
		data []byte
		keys []KeyData
		err  error
	)

	// keyFile - contains the sshd AuthorizedKeysFile location
	// Stdin - contains the AuthorizedKeysFile if keyFile is not specified
	if len(keyFile) != 0 {
		absPath, _ := filepath.Abs(keyFile)
		data, err = ioutil.ReadFile(absPath)
		if err != nil {
			return keys, err
		}
	} else {
		data, _ = ioutil.ReadAll(os.Stdin)
	}

	bytesReader := bytes.NewReader(data)
	scanner := bufio.NewScanner(bytesReader)
	for scanner.Scan() {
		// Parse the AuthorizedKeys line
		pk, _, _, _, ok := sshkey.ParseAuthorizedKey(scanner.Bytes())
		if !ok {
			err = errors.New("Unable to parse authorized key from input source, invalid format")
		}
		value := sshkey.MarshalAuthorizedKey(pk)
		keys = append(keys, KeyData{"authorized_keys", json.RawMessage(value)})
	}

	return keys, err
}
