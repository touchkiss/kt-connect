## Why

kt-connect 当前的本地调试流量通常绕过集群内现网调用方 Pod 的 Sidecar/Init 注入与链路信息，导致在启用泳道（lane）/subset 路由的 Istio 环境中无法稳定命中期望的 subset。需要提供一种“与现网调用方一致的关键注入能力 + 可控 lane 标识注入”的方式，让本地调试流量在不改动现网路由规则的前提下正确走到对应泳道。

## What Changes

- `ktctl connect` 新增 `--lane <lane>` 参数，用于声明本地调试会话的 lane。
- connect 时在目标 namespace 创建并维护一个 lane-aware shadow 工作负载（Pod/Deployment），用于代理本地调试流量的出站访问。
- shadow 工作负载具备与现网调用方 Pod 一致的关键 Istio 注入能力（至少包含 `istio-init` 与 `istio-proxy`）。
- shadow 工作负载主容器为 `kt-connect-shadow`。
- `kt-connect-shadow` 代表本地流量访问下游服务时，固定注入 `baggage: lane=<lane>`。
- 本地调试流量不再直接访问下游 Service，而是通过 shadow 工作负载代理出站。
- disconnect 时自动清理与该会话相关的 shadow 资源。

## Capabilities

### New Capabilities
- `lane-aware-shadow-pod`: 为 `ktctl connect` 提供 lane-aware shadow 工作负载的创建/维护/清理，以及通过 shadow 代理出站并注入 `baggage: lane=<lane>` 的能力。

### Modified Capabilities
- 

## Impact

- CLI：`ktctl connect` 参数解析与帮助信息需要更新。
- Kubernetes：connect/disconnect 需要新增对 shadow Deployment/Pod、Service（如需要）、RBAC/OwnerReference/Label 选择器等资源的管理。
- 网络路径：本地到集群内服务的访问链路将改为经由 shadow 工作负载转发出站。
- Istio：依赖现网已有 VirtualService/DestinationRule 的 subset 规则；本变更不修改现网规则，但需要确保 shadow 工作负载具备 `istio-init`/`istio-proxy` 注入并可携带 baggage header。
