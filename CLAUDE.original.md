# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and test commands

- `make mod` — run `go mod tidy -compat=1.17`
- `make test` — run the full Go test suite with coverage output under `artifacts/report/coverage/`
- `make check` — run `go vet ./pkg/... ./cmd/...`
- `make ktctl` — build the `ktctl` CLI for linux/darwin/windows into `artifacts/`
- `make shadow-local` — build the shadow binary for local debugging
- `make shadow` — build the linux shadow binary and docker image
- `make router` — build the linux router binary and docker image
- `make navigator-local` — build the navigator binary for local debugging
- `make release` — build release artifacts with goreleaser snapshot mode
- `make clean` — remove `artifacts`, `output`, and `dist`

Common direct Go commands:
- `go test ./...` — run all tests
- `go test ./pkg/kt/command/connect` — run a single package
- `go test ./pkg/shadow/proxy -run TestProxyReplacesLaneAcrossMultipleBaggageHeaders -count=1` — run one named test
- `go test ./pkg/kt/command/connect ./pkg/kt/command/general ./pkg/shadow/proxy ./cmd/shadow` — focused verification set for lane-aware connect work

Integration test prerequisites are documented in `testing/integration/README.md`:
- `kubectl` must point at a test cluster
- `ktctl` must already be built and available on `PATH`
- passwordless `sudo` is expected
- integration entry point: `testing/integration/go.sh [--keep-proof|--cleanup-only]`

## Repository shape

- `cmd/ktctl` is the main CLI entrypoint.
- `cmd/shadow`, `cmd/router`, and `cmd/navigator` are helper binaries that run inside the cluster or support specific traffic flows.
- `pkg/kt/command` holds Cobra command wiring and the top-level workflows for connect / exchange / mesh / preview / forward / clean / recover.
- `pkg/kt/command/options` defines CLI/config-backed option structs plus the runtime store used to carry session state across setup and teardown.
- `pkg/kt/service/cluster` is the Kubernetes access layer. `types.go` defines the central `KubernetesInterface`; most cluster-side behavior is implemented behind that interface.
- `pkg/kt/transmission` contains the local-to-cluster transport primitives, especially port-forward setup and session-tied forwarding behavior.
- `pkg/shadow` contains the shadow workload logic; `pkg/shadow/proxy` is the HTTP proxy that injects/rewrites `Baggage` lane headers.
- `pkg/common` and `pkg/kt/util` contain shared constants, OS helpers, filesystem paths, logging, and small cross-cutting utilities.
- `openspec/` contains change specs and archived design/task records for feature work.

## Big-picture architecture

`ktctl` is a Cobra CLI that always follows the same high-level lifecycle:
1. Parse options into `pkg/kt/command/options`
2. Run `general.Prepare()` to configure logging, kubeconfig resolution, namespace/context defaults, time-difference handling, and the Kubernetes clientset
3. Start a component-specific workflow (`connect`, `exchange`, `mesh`, etc.)
4. Persist runtime/session state in `opt.Store`
5. On process exit, call `general.CleanupWorkspace()` to tear down local and cluster-side resources

A few architecture details matter when editing behavior:

### 1. Options vs runtime state are intentionally split

`opt.Get()` returns the parsed/static command options. `opt.Store` is the mutable runtime store for things discovered or created during a session (clientset, rest config, generated shadow names, lane/session IDs, etc.).

If a value must survive until cleanup, it usually belongs in `opt.Store`, not only in the parsed command options.

### 2. Cluster operations are funneled through `cluster.Ins()`

`pkg/kt/service/cluster/types.go` defines the interface used by command code. This is the main seam for tests and for understanding what resources a workflow creates or deletes. When tracing behavior, look for `cluster.Ins().<method>` calls rather than raw client-go usage in command packages.

### 3. Connect mode is transport + DNS + shadow orchestration

The `connect` command is not just one tunnel. It combines:
- local privilege and option validation in `pkg/kt/command/connect.go`
- DNS setup and shadow workload creation in `pkg/kt/command/connect/common.go`
- a transport implementation in `pkg/kt/command/connect/tun2socks.go` or `sshuttle.go`
- Kubernetes port-forwarding in `pkg/kt/transmission/portforward.go`

For lane-aware connect, `common.go` is the key integration point: it decides shadow naming/labels/envs, starts session-scoped forwarding, and passes lane metadata into the shadow binary.

### 4. Shadow workload behavior is split between cluster setup and in-pod proxying

The cluster side creates a shadow workload and injects environment/labels/annotations. The runtime behavior inside that workload lives under `cmd/shadow` and `pkg/shadow/proxy`.

The current lane implementation uses the shadow proxy to rewrite the outbound `Baggage` header so exactly one normalized `lane=<lane>` member is present.

### 5. Cleanup is centralized and state-driven

`general.CleanupWorkspace()` is the final teardown path after command execution. If a feature creates cluster resources, verify both creation and cleanup paths. Recent lane-aware cleanup work relies on session-scoped labels rather than only stored resource names, so teardown logic is a critical part of feature behavior.

## Existing guidance from repo docs

Important points from the repo documentation worth carrying forward:
- The project is a Kubernetes development tool with four headline workflows: Connect, Exchange, Mesh, and Preview (`README.md`).
- The lane-aware shadow pod change has been archived into `openspec/changes/archive/2026-04-03-lane-aware-shadow-pod/`; when revisiting that feature, also check the synced main spec at `openspec/specs/lane-aware-shadow-pod/spec.md`.

## Notes for future edits

- Prefer package-level focused tests over only `go test ./...` while iterating; many behaviors are isolated in command/service packages.
- For connect-related changes, review both setup and teardown paths before concluding a change is complete.
- If you touch lane-aware behavior, relevant coverage currently lives in:
  - `pkg/kt/command/connect/common_test.go`
  - `pkg/kt/command/general/teardown_test.go`
  - `pkg/shadow/proxy/proxy_test.go`
  - `cmd/shadow/main_test.go`
