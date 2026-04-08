## Context

- kt-connect 现状：`ktctl connect` 将本地调试流量以“本地进程 →（某种隧道/转发）→ 集群服务”的方式接入，但该路径往往绕过集群内现网调用方 Pod 的 Istio 注入（`istio-init`/`istio-proxy`）以及调用方侧的请求头/链路上下文，导致在 Istio 基于 lane/subset 的路由场景下难以稳定命中目标 subset。
- 目标：引入一个与现网调用方具备关键注入能力一致的 shadow 工作负载，让本地流量的“出站”发生在该工作负载内，由 Envoy/iptables 接管，并在应用层固定注入 `baggage: lane=<lane>`。
- 约束：
  - 不修改现网 VirtualService / DestinationRule。
  - 不修改现网泳道命名与匹配字段。
  - 不实现本地进程直接加入 mesh。
  - 仅支持固定格式 `baggage: lane=<lane>`（不支持任意 header 路由）。
  - 如果ktctl connect命令不带有--lane参数，则保持现有行为不变。

## Goals / Non-Goals

**Goals:**
- `ktctl connect` 支持 `--lane <lane>`，使一次 connect 会话与一个 lane 绑定。
- connect 时在目标 namespace 创建 lane-aware shadow 工作负载（优先 Deployment，便于滚动与稳定性），主容器为 `kt-connect-shadow`。
- shadow 工作负载启用 Istio 注入能力，至少包含 `istio-init` 与 `istio-proxy`，确保其拥有与现网调用方一致的“出站接管”路径。
- 本地调试流量访问集群其他服务时，统一改为经由 shadow 工作负载代理出站；由 `kt-connect-shadow` 在转发请求时注入 `baggage: lane=<lane>`。
- disconnect 自动清理会话相关的 shadow 资源。

**Non-Goals:**
- 不调整/生成/变更任何现网 Istio 路由配置（VirtualService/DestinationRule）。
- 不实现本地进程 sidecar 注入或直接加入 mesh。
- 不提供自定义 header、权重、路由表达式等高级路由能力。

## Decisions

1. **Shadow 形态：Deployment + Service（必要时）**
   - 选择 Deployment：在 connect 生命周期内更稳定，支持重建/漂移处理；Pod 直接创建也可行，但更难处理重启与一致性。
   - 是否需要 Service：取决于本地到 shadow 的连通方式。若 kt-connect 采用 port-forward，则不需要 Service；若通过集群内地址访问 shadow（例如 NodePort/ClusterIP + tunnel），可能需要 Service。设计上优先采用 `kubectl port-forward`/等价机制，减少额外资源。

2. **Istio 注入方式：使用 label/annotation 触发 sidecar 注入**
   - 对 namespace：若目标 namespace 已开启 auto-injection，则 Deployment 只需遵循默认即可。
   - 对 pod：若 namespace 未开启注入，则在 shadow Pod/Deployment 上显式添加注入 annotation（例如 `sidecar.istio.io/inject: "true"`）并确保相关 webhook 生效。
   - 不尝试手动拼装 `istio-init`/`istio-proxy` 容器 spec（容易与集群 Istio 版本/配置漂移）。

3. **Header 注入位置：在 `kt-connect-shadow` 应用层实现**
   - 通过 HTTP 代理实现：shadow 主容器作为显式 HTTP proxy（或透明代理），对每个出站请求强制添加 `baggage: lane=<lane>`。
   - 验证范围以 shadow 工作负载、代理转发、baggage 注入和会话级清理这些可观察行为为准；不将额外控制面资源作为本次变更的验收前提。

4. **本地流量到 shadow 的路径：复用现有 connect 隧道能力**
   - kt-connect 现有 connect 一般已有与集群交互的通道（例如建立到某个 Pod 的转发）。本设计在不重构整体连接模型的前提下，将“下游访问”这一路径从直连改为经 shadow。
   - 具体实现需在 specs/tasks 阶段与现有 tun/forward 模块对齐（如 TUN 模式、HTTP proxy 模式或 SOCKS 模式）。

5. **资源标识与幂等**
   - Shadow 资源命名包含：namespace + connect 会话标识（如用户/机器/时间戳或现有 session id）+ lane，避免并发 connect 冲突。
   - 使用 label 统一标识与选择：如 `app=kt-connect-shadow`, `kt-connect/session=<id>`, `kt-connect/lane=<lane>`。
   - disconnect 通过 label selector 清理，保证幂等与容错。

## Risks / Trade-offs

- [Istio 注入不可用/被禁用] → 连接前检查目标 namespace 的注入条件（webhook 可用性/权限），必要时在输出中给出明确错误与修复建议。
- [Shadow 代理协议不匹配导致部分流量不可达] → 初期限定支持的流量类型（例如仅 HTTP/HTTPS），在 specs 中明确；对非支持协议给出清晰报错。
- [Header 注入与现网链路冲突] → 仅追加/覆盖 `baggage` 中的 `lane` 键；若请求已有 `baggage`，合并而非替换（确保不破坏其它 baggage）。
- [资源清理不完整导致残留] → 统一 owner label + final cleanup；disconnect 时再做一次“best-effort 扫描清理”。
- [多用户并发/同 namespace 多 lane] → 资源命名与 selector 必须包含 session 维度，避免误删。

## Manual Verification Checklist

1. Run `ktctl connect --namespace <ns> --lane <lane>` against a namespace where Istio injection is available.
2. Confirm the shadow workload is created in `<ns>` and carries lane/session labels plus `sidecar.istio.io/inject: "true"`.
3. Confirm the resulting shadow Pod contains `istio-proxy` (and `istio-init` where the mesh mode injects it).
4. From the local debug flow, trigger a downstream in-cluster HTTP request that should traverse the lane-aware path.
5. Verify the request egresses through the shadow workload and the outbound request contains `baggage: lane=<lane>`.
6. Confirm downstream Istio subset routing matches the expected lane behavior.
7. Disconnect the session and confirm only the current session's shadow resources are removed.
8. Re-run disconnect once more to confirm cleanup remains best-effort and idempotent.
