## ADDED Requirements

### Requirement: `ktctl connect` SHALL accept a lane parameter
The `ktctl connect` command MUST support a `--lane <lane>` argument.

#### Scenario: Provide lane on connect
- **WHEN** user runs `ktctl connect --namespace <ns> --lane <lane>`
- **THEN** the system starts a connect session bound to `<lane>`
- **AND THEN** subsequent outbound requests from the local debug traffic path are treated as belonging to `<lane>`

### Requirement: System SHALL create a lane-aware shadow workload on connect
When `--lane` is provided, the system MUST create (or reconcile) a lane-aware shadow workload in the target namespace.

#### Scenario: Shadow workload is created
- **WHEN** user runs `ktctl connect --namespace <ns> --lane <lane>`
- **THEN** the system creates a shadow Deployment/Pod in namespace `<ns>`
- **AND THEN** the shadow workload has a main container named `kt-connect-shadow`

### Requirement: Shadow workload MUST have key Istio injection capabilities
The shadow workload MUST have the key Istio injection capabilities consistent with in-cluster caller pods, including `istio-init` and `istio-proxy`.

#### Scenario: Istio init and proxy exist
- **WHEN** the shadow workload is created in a namespace where Istio injection is available
- **THEN** the resulting Pod contains `istio-init` and `istio-proxy` containers

### Requirement: Local debug outbound traffic MUST egress via the shadow workload
When `--lane` is provided, local debug traffic MUST NOT directly access downstream Services; instead it MUST egress via the shadow workload.

#### Scenario: Downstream access uses shadow proxy
- **WHEN** a locally debugged request attempts to access a downstream Service inside the cluster
- **THEN** the request is routed through the shadow workload as an egress proxy

### Requirement: `kt-connect-shadow` MUST inject `baggage: lane=<lane>`
For each outbound request represented by local debug traffic, `kt-connect-shadow` MUST inject `baggage: lane=<lane>` so that Istio routing can match the corresponding subset.

#### Scenario: Inject baggage lane
- **WHEN** the shadow workload proxies an outbound request to a downstream Service
- **THEN** the outbound request includes header `baggage: lane=<lane>`

### Requirement: disconnect MUST clean up shadow resources
On `ktctl disconnect` (or equivalent disconnect action), the system MUST delete all shadow resources created for the connect session.

#### Scenario: Cleanup on disconnect
- **WHEN** user disconnects a connect session created with `--lane <lane>`
- **THEN** the system deletes the associated shadow workload resources from the target namespace
