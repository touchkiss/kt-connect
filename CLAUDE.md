# CLAUDE.md

Guide for Claude Code (`claude.ai/code`) in this repo.

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

Integration test prereqs in `testing/integration/README.md`:
- `kubectl` must point at a test cluster
- `ktctl` must already be built and available on `PATH`
- passwordless `sudo` is expected
- integration entry point: `testing/integration/go.sh [--keep-proof|--cleanup-only]`

## Repository shape

- `cmd/ktctl` is the main CLI entrypoint.
- `cmd/shadow`, `cmd/router`, `cmd/navigator` are helper binaries for in-cluster or traffic-support flows.
- `pkg/kt/command` holds Cobra wiring and top workflows for connect / exchange / mesh / preview / forward / clean / recover.
- `pkg/kt/command/options` defines CLI/config-backed options plus runtime store for session state across setup/teardown.
- `pkg/kt/service/cluster` is Kubernetes access layer. `types.go` defines central `KubernetesInterface`; most cluster behavior sits behind it.
- `pkg/kt/transmission` contains local-to-cluster transport primitives, especially port-forward and session-tied forwarding.
- `pkg/shadow` contains shadow workload logic; `pkg/shadow/proxy` is HTTP proxy that injects/rewrites `Baggage` lane headers.
- `pkg/common` and `pkg/kt/util` contain shared constants, OS helpers, filesystem paths, logging, small utilities.
- `openspec/` contains change specs and archived design/task records.

## Big-picture architecture

`ktctl` is Cobra CLI. High-level lifecycle:
1. Parse options into `pkg/kt/command/options`
2. Run `general.Prepare()` to configure logging, kubeconfig resolution, namespace/context defaults, time-difference handling, and the Kubernetes clientset
3. Start a component-specific workflow (`connect`, `exchange`, `mesh`, etc.)
4. Persist runtime/session state in `opt.Store`
5. On process exit, call `general.CleanupWorkspace()` to tear down local and cluster-side resources

Architecture details that matter:

### 1. Options vs runtime state are intentionally split

`opt.Get()` returns parsed/static command options. `opt.Store` is mutable runtime store for values discovered/created during session (clientset, rest config, generated shadow names, lane/session IDs, etc.).

If value must survive until cleanup, usually put in `opt.Store`, not only parsed options.

### 2. Cluster operations are funneled through `cluster.Ins()`

`pkg/kt/service/cluster/types.go` defines interface used by command code. Main seam for tests and for understanding what resources workflow creates/deletes. When tracing behavior, look for `cluster.Ins().<method>` calls, not raw client-go in command packages.

### 3. Connect mode is transport + DNS + shadow orchestration

`connect` is not one tunnel. It combines:
- local privilege and option validation in `pkg/kt/command/connect.go`
- DNS setup and shadow workload creation in `pkg/kt/command/connect/common.go`
- a transport implementation in `pkg/kt/command/connect/tun2socks.go` or `sshuttle.go`
- Kubernetes port-forwarding in `pkg/kt/transmission/portforward.go`

For lane-aware connect, `common.go` is key integration point: decides shadow naming/labels/envs, starts session-scoped forwarding, passes lane metadata into shadow binary.

### 4. Shadow workload behavior is split between cluster setup and in-pod proxying

Cluster side creates shadow workload and injects environment/labels/annotations. Runtime behavior inside workload lives under `cmd/shadow` and `pkg/shadow/proxy`.

Current lane implementation uses shadow proxy to rewrite outbound `Baggage` header so exactly one normalized `lane=<lane>` member is present.

### 5. Cleanup is centralized and state-driven

`general.CleanupWorkspace()` is final teardown path after command execution. If feature creates cluster resources, verify creation + cleanup. Recent lane-aware cleanup relies on session-scoped labels, not only stored resource names, so teardown logic is critical.

## Existing guidance from repo docs

Important points from repo docs:
- Project is Kubernetes development tool with four headline workflows: Connect, Exchange, Mesh, Preview (`README.md`).
- Lane-aware shadow pod change archived in `openspec/changes/archive/2026-04-03-lane-aware-shadow-pod/`; when revisiting, also check synced main spec at `openspec/specs/lane-aware-shadow-pod/spec.md`.

## Notes for future edits

- Prefer package-level focused tests over only `go test ./...` while iterating; many behaviors isolated in command/service packages.
- For connect-related changes, review both setup and teardown before calling change complete.
- If touching lane-aware behavior, relevant coverage lives in:
  - `pkg/kt/command/connect/common_test.go`
  - `pkg/kt/command/general/teardown_test.go`
  - `pkg/shadow/proxy/proxy_test.go`
  - `cmd/shadow/main_test.go`
