## 1. CLI & Session Model

- [x] 1.1 Add `--lane <lane>` flag to `ktctl connect` command and update help/usage text
- [x] 1.2 Validate `--lane` value (non-empty, allowed charset) and plumb it into connect session context/state
- [x] 1.3 Ensure lane is persisted for the session so disconnect can locate resources created by the same session

## 2. Shadow Workload Manifests & Idempotent Reconcile

- [x] 2.1 Define shadow workload naming/labeling conventions (include namespace + session id + lane) to avoid collisions
- [x] 2.2 Implement rendering of Kubernetes resources for shadow workload (Deployment and any required Service)
- [x] 2.3 Add Istio injection trigger to shadow workload via label/annotation (no manual `istio-init`/`istio-proxy` spec)
- [x] 2.4 Implement apply/reconcile logic on connect (create or update existing shadow resources)
- [x] 2.5 Implement readiness detection for shadow workload (wait for Pod ready before routing local traffic through it)

## 3. Local Traffic Routing via Shadow

- [x] 3.1 Identify existing local→cluster routing mode in kt-connect (tun/port-forward/proxy) and choose the minimal integration point for shadow egress
- [x] 3.2 Implement local outbound path change: route downstream service access through the shadow workload instead of direct service access when `--lane` is set
- [x] 3.3 Implement connectivity mechanism to shadow workload (e.g., port-forward to `kt-connect-shadow`) and lifecycle management tied to connect session

## 4. `kt-connect-shadow` Behavior (Baggage Injection)

- [x] 4.1 Implement `kt-connect-shadow` as an outbound proxy that forwards requests to original destinations
- [x] 4.2 Inject `baggage: lane=<lane>` on each proxied outbound request; merge with existing `baggage` header if present (override/append lane key only)
- [x] 4.3 Add unit/integration tests for header injection behavior (existing baggage, no baggage, multiple keys)

## 5. Disconnect Cleanup

- [x] 5.1 Implement resource cleanup on disconnect using label selectors scoped to the session id (avoid deleting other sessions)
- [x] 5.2 Ensure cleanup is best-effort and idempotent (re-run safe) and handles partial creation failures

## 6. Verification

- [x] 6.1 Add automated tests covering: connect creates shadow workload, lane flag is honored, and disconnect removes resources
- [x] 6.2 Manual verification checklist: `ktctl connect --namespace <ns> --lane <lane>` creates injected pod; requests egress via shadow; downstream routing matches subset by `baggage: lane=<lane>`
