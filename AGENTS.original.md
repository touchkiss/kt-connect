# AGENTS.md
Agent-focused contributor guide for `github.com/alibaba/kt-connect`.
For autonomous coding agents operating in this repository.

## 1) Mission context
- `kt-connect` is a Kubernetes development toolkit.
- Primary workflows: `connect`, `exchange`, `mesh`, `preview`.
- CLI entrypoint: `cmd/ktctl`.
- Helper binaries: `cmd/shadow`, `cmd/router`, `cmd/navigator`.
- Command orchestration: `pkg/kt/command`.
- Kubernetes operations should go through `pkg/kt/service/cluster`.

## 2) Build, lint, and test commands
Preferred Make targets (match repository conventions):
- `make mod` -> `go mod tidy -compat=1.17`.
- `make test` -> full tests + coverage output in `artifacts/report/coverage/`.
- `make check` -> `go vet ./pkg/... ./cmd/...`.
- `make ktctl` -> build multi-platform `ktctl` binaries in `artifacts/`.
- `make shadow-local` -> build local debug `shadow` binary.
- `make shadow` -> build linux `shadow` binary + docker image.
- `make router` -> build linux `router` binary + docker image.
- `make navigator-local` -> build local debug `navigator` binary.
- `make release` -> snapshot release via goreleaser.
- `make clean` -> clean `artifacts`, `output`, `dist`.

Direct Go commands for faster inner-loop validation:
- `go test ./...` (all tests).
- `go test ./pkg/kt/command/connect` (single package).
- `go test ./pkg/kt/command/connect -run TestName -count=1` (single test by name).
- `go test ./pkg/shadow/proxy -run TestProxyReplacesLaneAcrossMultipleBaggageHeaders -count=1` (real single-test example).
- `go test ./pkg/kt/command/connect ./pkg/kt/command/general ./pkg/shadow/proxy ./cmd/shadow` (focused lane regression set).
- `go vet ./pkg/... ./cmd/...` (same scope as `make check`).

Recommended validation order before declaring success:
1. Run targeted tests for changed package(s).
2. Run `make check`.
3. Run `make test` when change is cross-package or lifecycle-related.

## 3) Integration test prerequisites
- Entry script: `testing/integration/go.sh [--keep-proof|--cleanup-only]`.
- Preconditions:
  - `kubectl` points to a reachable cluster.
  - `ktctl` is built and available on `PATH`.
  - passwordless `sudo` is available.
- Avoid unattended integration loops without verifying prerequisites.

## 4) Architecture constraints
- Keep parsed/static options in `opt.Get()`.
- Keep mutable runtime/session state in `opt.Store`.
- If teardown needs data, persist it in `opt.Store` during setup.
- Prefer `cluster.Ins().<method>` over ad-hoc client-go usage in command packages.
- Connect flow touches command setup, DNS mode, transport (`tun2socks` / `sshuttle`), and port-forwarding.
- Cleanup is centralized in `general.CleanupWorkspace()`; validate setup and teardown paths.

Lane-sensitive behavior notes:
- Lane metadata affects connect setup, shadow pod/deployment labels/annotations/envs, and shadow proxy behavior.
- Key lane-related tests:
  - `pkg/kt/command/connect/common_test.go`
  - `pkg/kt/command/general/teardown_test.go`
  - `pkg/shadow/proxy/proxy_test.go`
  - `cmd/shadow/main_test.go`

## 5) Go style guide for this repo

### Imports
- Use standard Go grouping (stdlib, third-party, local module).
- Honor goimports local prefix: `github.com/alibaba/kt-connect`.
- Keep aliases minimal and meaningful.
- Remove unused imports.

### Formatting
- Run `gofmt` (or editor equivalent) before finalizing.
- Keep code readable; lint allows long lines up to 140 chars, but prefer clarity.
- Do not manually align code with spaces.

### Types and interfaces
- Prefer concrete types unless an interface seam is needed for testing or abstraction.
- Reuse existing boundaries such as `KubernetesInterface`.
- Avoid unnecessary API/signature churn.
- Initialize maps/slices when mutation requires it; use zero values intentionally.

### Naming conventions
- Follow idiomatic Go casing (`CamelCase` exported, `camelCase` unexported).
- Use concise domain names (`shadowPodName`, `namespace`, `lane`).
- Keep acronym casing consistent (`DNS`, `IP`, `ID`, `CIDR`).
- Boolean helpers should read naturally (example: `shouldCleanLaneSessionResources`).

### Error handling
- Return errors; avoid panics in normal flows.
- Wrap propagated errors with context: `fmt.Errorf("...: %w", err)`.
- Log where context is useful, but avoid repetitive log-and-return chains.
- Preserve existing behavior contracts unless intentional and documented.

### Logging
- Use `zerolog` (`github.com/rs/zerolog/log`).
- Include actionable context (resource name, namespace, operation).
- Keep hot-path logs concise.

### Control flow and concurrency
- Keep goroutine lifecycles explicit; avoid leaked watchers/channels.
- Respect existing watch callback patterns and teardown contracts.
- Avoid arbitrary sleeps unless required by eventual consistency and explained by code context.

### Comments and docs
- Add comments only for non-obvious logic/constraints.
- Keep comments accurate and close to the behavior they describe.
- Update docs/tests when external behavior changes.

## 6) Testing guidelines for agents
- Use focused package tests during iteration.
- For bug fixes, add or update a test that fails before and passes after.
- Delay broad test runs until targeted tests are green.
- Use `-count=1` for deterministic reruns while debugging.
- If changing cleanup/resource lifecycle, include interrupted/error-path validation.

## 7) Linting and static analysis profile
- Repo includes `.golangci.yml` with broad linter coverage.
- Current notable settings:
  - `goimports` local prefix: `github.com/alibaba/kt-connect`.
  - `govet` shadow checks enabled.
  - `errcheck` strict for blank assignments and type assertions.
  - `funlen` and cyclomatic complexity thresholds are configured.
- Even when CI uses only `go vet`, write code compatible with common golangci-lint checks.

## 8) Cursor/Copilot rule file discovery
Requested rule files status in this repo:
- `.cursor/rules/`: not present.
- `.cursorrules`: not present.
- `.github/copilot-instructions.md`: not present.

Related instruction-like files that do exist:
- `.cursor/commands/opsx-*.md`
- `.cursor/skills/openspec-*/SKILL.md`
- `.github/prompts/opsx-*.prompt.md`
- `.github/skills/openspec-*/SKILL.md`

If dedicated Cursor/Copilot rule files are added later, merge their constraints here.

## 9) Agent execution checklist
Before editing:
1. Identify affected command/service boundaries.
2. Find existing tests in touched packages.
3. Confirm whether both setup and teardown paths are impacted.

Before final response:
1. Run targeted tests.
2. Run `make check`.
3. Run broader tests as needed.
4. Report exactly what was run and what remains unverified.

Keep changes minimal, reversible, and aligned with existing patterns.
