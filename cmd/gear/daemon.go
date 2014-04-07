package main

import (
	. "github.com/openshift/geard/cmd"
	"github.com/openshift/geard/containers"
	"github.com/openshift/geard/encrypted"
	"github.com/openshift/geard/port"
	"github.com/openshift/geard/systemd"

	"github.com/spf13/cobra"
	"log"
	nethttp "net/http"
	"path/filepath"
)

var (
	listenAddr string
)

func init() {
	AddInitializer(systemd.Start, WhenDaemon)
	AddInitializer(containers.InitializeData, WhenDaemon)
}

func daemon(cmd *cobra.Command, args []string) {
	api := conf.Handler()
	nethttp.Handle("/", api)

	if keyPath != "" {
		config, err := encrypted.NewTokenConfiguration(filepath.Join(keyPath, "server"), filepath.Join(keyPath, "client.pub"))
		if err != nil {
			Fail(1, "Unable to load token configuration: %s", err.Error())
		}
		nethttp.Handle("/token/", nethttp.StripPrefix("/token", config.Handler(api)))
	}

	if err := Initialize(WhenDaemon); err != nil {
		log.Fatal(err)
	}

	port.StartPortAllocator(4000, 60000)
	conf.Dispatcher.Start()

	log.Printf("Listening for HTTP on %s ...", listenAddr)
	log.Fatal(nethttp.ListenAndServe(listenAddr, nil))
}
