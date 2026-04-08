## Why

当前 `ktctl connect --lane` 的 shadow 工作负载仍主要依赖 `sidecar.istio.io/inject` 的隐式注入路径，Pod 模板本身是单容器模型，无法在代码层稳定表达并验证多容器约束（init/sidecar 结构、资源注解、keystore init 片段）。这会导致在不同集群注入策略下结果不一致，难以保证泳道场景的可重复性与可测试性。

## What Changes

- 将 lane 场景下的 shadow workload 从“隐式注入 + 单容器模板”升级为“显式多容器模板”。
- 在模板中明确支持：主容器 `kt-connect-shadow`、`initContainers`（含 keystore init 片段）、sidecar 约束与 sidecar 资源注解透传。
- 保持非 `--lane` 路径行为不变，继续使用现有单容器路径与现有 connect/disconnect 生命周期。
- 补齐模板级自动化测试，覆盖 lane/非 lane 分支与关键字段断言（init/sidecar/keystore 片段）。

## Capabilities

### New Capabilities

- 无

### Modified Capabilities

- `lane-aware-shadow-pod`: 将 lane-aware shadow 的规格要求从“触发注入”升级为“显式多容器模板 + 可验证模板约束（init/sidecar/keystore 片段）”，并要求对应测试覆盖。

## Impact

- 受影响代码：`pkg/kt/service/cluster/helper.go`、`pkg/kt/service/cluster/shadow_pod.go`、`pkg/kt/command/connect/common.go`、`pkg/kt/service/cluster/pod.go`。
- 受影响测试：扩展 `pkg/kt/command/connect/common_test.go`，新增或补强 cluster 层模板渲染测试。
- 对外 CLI/API：无新增命令参数；`--lane` 语义增强为显式模板保障。
- 运行时影响：lane 场景 Pod 结构更明确、可控，非 lane 路径保持兼容。
