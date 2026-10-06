# rke2-airgap-lifecycle-manager

A Go CLI for RKE2 lifecycle automation in air-gapped environments (Ubuntu): version detection, hardware/software preflight checks, ISO lifecycle handling, post-upgrade health monitoring.
Status: work in progress. Published so far: TLS/preflight validation, config loading, Kubernetes helpers. Upgrade and install flows are in development and not yet in this repository.

---

## Why this tool exists

Air-gap RKE2 clusters require a multi-step, error-prone process and cannot handle errors, so tool verifies the hardware and software before there would be any manupulations provided with the server.

---

## Features

- Detects current versions: platform OS build, Helm operator, RKE2, and airgap ISO
- Queries an artifact repository and OCI registry to find the latest available builds
- Validates TLS certificate and key pair: expiry, FQDN/SAN match, public-key consistency
- Manages the full install sequence: RKE2, kubectl, Helm, local-path-provisioner, system tools
- Handles ISO lifecycle: disk-space pre-flight, cleanup of superseded ISOs, unmount of stale loop mounts, download with progress, integrity check
- Triggers upgrade via the platform's internal state-machine API
- Rebuilds the in-cluster Helm chart index after a new ISO is mounted (airgap upgrade)
- Patches the `HelmRepository` CR URL to point at the new chart index
- Surfaces TLS cert expiry and pod health on every run
- Detects cluster hardware profile and gates optional add-ons accordingly

---

## Repository structure

```
cmd/
  install/        main entrypoint — interactive install flow
  upgrade/        main entrypoint — upgrade flow (online + airgap)
internal/
  config/         credential loading from ~/.lifecycle.conf
  version/        current version detection (OS, Helm, RKE2, ISO)
  latest/         latest version discovery (Artifactory + Docker Hub OCI)
  airgap/         ISO lifecycle: download, mount, unmount, cleanup
  kube/           Kubernetes interactions (nodes, pods, secrets, CRs)
  certcheck/      TLS certificate validation
docs/
  notes.md        engineering notes and manual command reference
```

---

## Installation

```bash
# Build
go build -o rke2-airgap-manager ./cmd/install   # or ./cmd/upgrade

# Run (elevated privileges required for ISO mount, installer execution,
# PVC writes, and system-level checks)
sudo ./rke2-airgap-manager
```

---

## Configuration

Create `~/.lifecycle.conf` before running:

```
ARTIFACTORY_KEY=...
DOCKER_USER=...
DOCKER_TOKEN=...
```

The tool refuses to start if any of the three values are missing.

---

## Install flow

```
Collect config interactively
  └─ FQDN + TLS certs  OR  bare IP (self-signed)
  └─ network interface, node name, feature flags, hardware profile

Validate TLS cert/key pair

Generate platformResponses.json
  └─ profile-aware storage sizing
  └─ wizard completion timestamps

Teardown existing RKE2 (idempotent)

Install RKE2  →  kubectl  →  Helm  →  local-path-provisioner  →  tools

Download installer  [+  airgap ISO if air-gap mode]

[rke2 ingress mode]  →  apply ingress bridge Service
```

---

## Upgrade flow

```
Detect current versions  (OS build, Helm operator, RKE2, ISO)
Query latest versions    (Artifactory folders + Docker Hub OCI tags)

Present diff, ask operator which components to upgrade

[online path]
  check - download - apply - verify

[airgap path]
  Disk-space pre-flight  →  delete stale ISOs if needed
  Unmount old ISO loop mounts
  Download new ISO  →  run installer -f <iso> -b
  Access code  →  JWT  →  trigger reconciliation
  Delete superseded ISOs
  helm repo index inside setup pod  →  patch HelmRepository CR URL
  helm upgrade <operator> from ISO chart bundle

Monitor pod health
```

---

## Notes

- All sensitive URLs, registry paths, namespaces, and credential values have been removed and replaced with `<placeholders>`.
- This repository contains only generic logic suitable for demonstration and portfolio purposes.
- The tool is designed for air-gap environments and assumes restricted or zero external network access.