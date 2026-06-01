package main

import (
	"fmt"
	"os"
)

// usage prints help and exits.
func usage() {
	fmt.Fprintf(os.Stderr, `rke2-airgap-lifecycle-manager

Manages the full lifecycle of an RKE2 cluster in air-gap environments.

Usage:
  sudo ./rke2-airgap-lifecycle-manager <command>

Commands:
  install   Provision a new RKE2 cluster and deploy the platform
  upgrade   Detect and apply available updates (online or airgap)

Config:
  Both commands require ~/.lifecycle.conf with:
    ARTIFACTORY_KEY=...
    DOCKER_USER=...
    DOCKER_TOKEN=...

Examples:
  sudo ./rke2-airgap-lifecycle-manager install
  sudo ./rke2-airgap-lifecycle-manager upgrade

`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "install":
		runInstallCommand()
	case "upgrade":
		runUpgradeCommand()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", os.Args[1])
		usage()
	}
}
