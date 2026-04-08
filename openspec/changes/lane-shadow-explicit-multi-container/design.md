## Context

当前 lane 场景下的 shadow 创建路径已经具备命名、标签、注解和会话清理能力，但 workload 渲染仍以通用单容器模板为核心，`sidecar.istio.io/inject` 仅作为触发条件。该实现在不同集群注入链路下可观测结果不稳定，且很难通过单元测试断言 Pod 模板是否满足多容器约束。

本次设计目标是在不改动业务转发协议与现网 Istio 规则的前提下，把 lane 分支升级为显式模板：明确主容器、init 片段、sidecar 约束和资源注解，并使关键字段可测试。

## Goals / Non-Goals

**Goals:**
- lane 场景下显式渲染 shadow 多容器模板（主容器 + init 片段 + sidecar 约束）。
- 模板中可稳定携带 sidecar 资源注解与 keystore init 片段。
- 非 lane 路径保持现有行为和兼容性。
- 增加模板级单测，覆盖 lane 分支与非 lane 回退分支。

**Non-Goals:**
- 不改造 `kt-connect-shadow` 的业务代理协议与转发语义。
- 不变更集群侧 VirtualService/DestinationRule 或全局注入策略。
- 不引入新的 CLI 参数。

## Decisions

### 决策 1：在 `PodMetaAndSpec` 增加“显式模板字段”，并保持默认兼容
- 方案：为 `PodMetaAndSpec` 增加可选字段（如 init 模板片段、附加容器、额外卷/挂载、lane 模板开关）。
- 原因：改动最小且能复用现有 `createPod/createDeployment` 入口；非 lane 调用不需要改行为。
- 备选：单独新增 lane 专用 workload builder。
- 取舍：专用 builder 更隔离，但代码重复更高，短期不符合“最小变更”。

### 决策 2：仅 lane 路径启用显式多容器模板
- 方案：`GetOrCreateShadow` 根据 lane 是否存在切换模板分支。
- 原因：限制 blast radius，保证无 `--lane` 的路径不回归。
- 备选：统一全部 shadow 使用新模板。
- 取舍：统一改造长期更整洁，但风险和验证成本更高。

### 决策 3：sidecar 资源约束以注解透传为主，容器结构由模板显式表达
- 方案：保留 `sidecar.istio.io/inject=true`，并在 lane 分支增加 sidecar 资源相关注解。
- 原因：兼容现有注入机制，避免手写 `istio-proxy` 镜像配置。
- 备选：手写完整 `istio-init/istio-proxy` 容器定义。
- 取舍：手写方案可控性更高，但与集群版本耦合强、维护成本高。

### 决策 4：keystore init 先落地“片段能力 + 存在性可测试”
- 方案：实现最小 keystore init 片段（命令、卷、挂载占位）并确保可在单测中断言。
- 原因：先建立结构化 contract，再按环境补全具体镜像与凭据绑定细节。
- 备选：一次性做完整生产级 keystore 初始化流程。
- 取舍：一次到位复杂度高，易引入环境依赖导致不稳定。

## Risks / Trade-offs

- [风险] 显式模板字段与集群注入行为冲突（重复字段或顺序冲突）
  → 缓解：lane 分支灰度启用；仅增加必要字段并避免硬编码完整 Istio 容器。
- [风险] 模板字段扩展影响旧调用路径
  → 缓解：新增字段均为可选默认值；非 lane 分支不进入新逻辑。
- [风险] keystore init 片段依赖外部 Secret/ConfigMap
  → 缓解：第一阶段仅要求片段与挂载 contract 存在，环境绑定通过可配置输入补齐。

## Migration Plan

1. 扩展 `PodMetaAndSpec` 模型与 `createPod/createDeployment` 渲染能力（保持默认兼容）。
2. 在 `GetOrCreateShadow` lane 分支启用显式多容器模板与 keystore init 片段。
3. 在 `getOrCreateShadow` 统一 lane 注解输入（含 sidecar 资源注解）。
4. 增加/更新模板级单测并运行目标包测试。
5. 如出现兼容问题，回滚 lane 模板分支到旧路径，保留已完成的 header 注入与清理逻辑。

## Open Questions

- keystore init 片段来源采用哪种策略：内置常量、配置注入，还是注解透传？
- sidecar 验收是否仅做字段存在性，还是增加资源阈值断言？
