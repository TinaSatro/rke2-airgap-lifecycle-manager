# rke2-airgap-lifecycle-manager

A Go CLI for RKE2 lifecycle automation in air-gapped environments (Ubuntu).
Planned scope: version detection, hardware/software preflight checks, ISO lifecycle handling, post-upgrade health monitoring.

Status: work in progress. See Implemented and In development below. Upgrade and install flows are not yet in this repository.

---

## Why this tool exists

Air-gap RKE2 upgrades are multi-step and manual, and a failure midway can leave a server half-upgraded. The tool is designed to verify hardware, software and TLS before changing anything, and to stop if checks fail.

---

## Implemented

- TLS certificate validation
- Config loading
- Kubernetes helpers (nodes, pods, secrets)

---

## In development

- Version detection, ISO download, upgrade and install flows

---

## Repository structure

```
cmd/
  install/        main entrypoint — interactive install flow (in development)
  upgrade/        main entrypoint — upgrade flow (planned)
internal/
  config/         credential loading from ~/.lifecycle.conf
  version/        current version detection (planned)
  latest/         latest version discovery (planned)
  airgap/         ISO lifecycle: download, mount, unmount, cleanup (planned)
  kube/           Kubernetes interactions (nodes, pods, secrets)
  certcheck/      TLS certificate validation
```

---

## Installation

```bash
# Build (the install entrypoint is currently a stub; the flow is in development)
go build -o rke2-airgap-manager ./cmd/install

# Run (elevated privileges required for ISO mount and system-level checks)
sudo ./rke2-airgap-manager
```

---

## Configuration

Create `~/.lifecycle.conf` before running (used by upcoming install and upgrade flows):

```
ARTIFACTORY_KEY=...
DOCKER_USER=...
DOCKER_TOKEN=...
```

The config package validates that all three values are present.

---

## Install flow (in development)

```
Collect config interactively
  └─ FQDN + TLS certs  OR  bare IP (self-signed)
  └─ network interface, node name, hardware profile

Validate TLS cert/key pair

Generate install configuration
  └─ profile-aware storage sizing

Teardown existing RKE2 (idempotent)

Install RKE2  →  kubectl  →  Helm  →  local-path-provisioner  →  tools

Download ISO (air-gap mode)

Configure ingress (optional)

Verify node and pod health
```

---

## Upgrade flow (in development)

```
Detect current versions  (OS, Helm, RKE2, ISO)
Query latest versions    (Artifactory folders + Docker Hub OCI tags)

Present diff, ask which components to upgrade

[online path]
  check - download - apply - verify

[airgap path]
  Disk-space pre-flight  →  delete stale ISOs if needed
  Unmount old ISO loop mounts
  Download new ISO  →  verify integrity
  Refresh local chart index from new ISO
  Apply upgrade from local bundle
  Delete superseded ISOs

Monitor pod health
```

---

## Notes

- All sensitive URLs, registry paths, namespaces, and credential values have been removed.
- This repository contains only generic logic suitable for demonstration and portfolio purposes.
- In air-gap mode the cluster itself has no external access; the tool fetches the ISO from a configured artifact repository.