package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, `rke2-airgap-lifecycle-manager — install

Provisions a new RKE2 cluster and deploys the platform.

Usage:
  sudo ./rke2-airgap-lifecycle-manager

Config:
  Requires ~/.lifecycle.conf:
    ARTIFACTORY_KEY=...
    DOCKER_USER=...
    DOCKER_TOKEN=...

`)
	os.Exit(1)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		usage()
	}
	runInstall()
}
