# rke2-airgap-lifecycle-manager

A Go-based automation tool that manages the full lifecycle of an RKE2 cluster in air-gap environments — from fresh install through ongoing upgrades. It orchestrates version detection, ISO handling, installer execution, token exchange, and Helm reconciliation through Kubernetes.

---

## Why this tool exists

Air-gap RKE2 clusters require a multi-step, error-prone process: tearing down and reinstalling RKE2, downloading an OS ISO, writing a platform answers file into a PVC, exchanging a one-time access code for a JWT, and triggering a Helm-based reconciliation loop — all before a single application pod starts.

During install, several pods failed to start because the object-storage access secret is written with bootstrap placeholder credentials, and dependent pods crash before the real credentials are available. This tool automates the full flow end-to-end, including the secret-patch workaround that unblocks those pods.

---

## Features

- Detects current versions: platform OS build, Helm operator, RKE2, and airgap ISO
- Queries an artifact repository and OCI registry to find the latest available builds
- Generates the platform configuration answers file with profile-aware storage sizing
- Validates TLS certificate and key pair: expiry, FQDN/SAN match, public-key consistency
- Manages the full install sequence: RKE2, kubectl, Helm, local-path-provisioner, system tools
- Handles ISO lifecycle: disk-space pre-flight, cleanup of superseded ISOs, unmount of stale loop mounts, download with progress, integrity check
- Writes configuration into the setup PVC via `sudo tee` (PVC directory is root-owned at write time)
- Exchanges a one-time access code from setup-pod logs for a JWT
- Triggers upgrade via the platform's internal state-machine API
- Patches the object-storage access secret once real credentials appear, then restarts affected pods
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
  installer/      installer download and execution
  kube/           Kubernetes interactions (nodes, pods, secrets, CRs)
  responses/      platform answers-file generation and PVC write
  certcheck/      TLS certificate validation
  token/          access-code scraping, token exchange, upgrade trigger
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

Run installer  →  write responses to PVC

Access code (from setup-pod logs)  →  JWT  →  trigger Helm reconciliation

Wait for storage-user secret  →  patch object-storage secret  →  restart pods

[rke2 ingress mode]  →  apply ingress bridge Service
```

---

## Upgrade flow

```
Detect current versions  (OS build, Helm operator, RKE2, ISO)
Query latest versions    (Artifactory folders + Docker Hub OCI tags)

Present diff, ask operator which components to upgrade

[online path]
  Patch platformResponses.json with new Helm version
  Download installer  →  run -b (batch)
  Access code  →  JWT  →  trigger reconciliation
  helm upgrade <operator> oci://...

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
- `InsecureSkipVerify` is used for internal cluster HTTPS calls; edge nodes commonly use self-signed certificates.
