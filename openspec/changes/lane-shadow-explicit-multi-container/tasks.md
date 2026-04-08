## 1. Template Model Extension

- [x] 1.1 Extend `PodMetaAndSpec` in `pkg/kt/service/cluster/pod.go` with optional explicit template fields (main container config, init fragments, sidecar constraints, pod annotations, volumes/volumeMount fragments)
- [x] 1.2 Keep zero-value defaults backward-compatible so existing non-lane callers do not change behavior

## 2. Pod/Deployment Rendering Refactor

- [x] 2.1 Refactor `createPod` and `createDeployment` in `pkg/kt/service/cluster/helper.go` to assemble `Containers` and `InitContainers` from template fields
- [x] 2.2 Support sidecar resource annotation passthrough on Pod/PodTemplate metadata in lane path

## 3. Lane-Only Explicit Multi-Container Shadow Path

- [x] 3.1 In `pkg/kt/service/cluster/shadow_pod.go`, enable explicit multi-container template only when lane session metadata exists
- [x] 3.2 Add minimal keystore init fragment wiring (init command fragment + required volume/volumeMount links) in lane template branch
- [x] 3.3 Preserve non-lane path behavior and existing shadow lifecycle (create/reuse/cleanup)

## 4. Lane Annotation Input Unification

- [x] 4.1 In `pkg/kt/command/connect/common.go`, centralize lane annotation generation including `sidecar.istio.io/inject=true`
- [x] 4.2 Add sidecar resource annotations (`sidecar.istio.io/*`) to lane annotation input and pass template-related parameters into shadow creation

## 5. Automated Test Coverage

- [x] 5.1 Extend `pkg/kt/command/connect/common_test.go` to verify lane input includes required sidecar annotations and template toggles
- [x] 5.2 Add cluster-layer template rendering tests (new test file under `pkg/kt/service/cluster/`) to assert lane path contains main container + init fragments + sidecar constraints + keystore fragment
- [x] 5.3 Add non-lane regression assertions to ensure legacy single-container rendering remains unchanged

## 6. Spec and Verification

- [x] 6.1 Update `openspec/specs/lane-aware-shadow-pod/spec.md` to reflect explicit multi-container template requirements and template-level regression tests
- [x] 6.2 Run targeted tests for touched packages and record verification commands/results in change notes

## Verification Notes

- `go test ./pkg/kt/command/connect -run 'TestGetOrCreateShadowUsesLaneSessionMetadata|TestBuildShadowAnnotationsForLane' -count=1` -> passed
- `go test ./pkg/kt/service/cluster -run 'TestCreatePodKeepsLegacySingleContainerDefaults|TestCreatePodBuildsExplicitLaneTemplate' -count=1` -> passed
- `go test ./pkg/kt/command/connect ./pkg/kt/service/cluster -count=1` -> passed
- `go vet ./pkg/... ./cmd/...` -> failed on pre-existing issue in `pkg/kt/command/general/setup.go:59` (`signal.Notify` uses an unbuffered channel), unrelated to this change
