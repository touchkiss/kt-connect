## MODIFIED Requirements

### Requirement: System SHALL create a lane-aware shadow workload on connect
When `--lane` is provided, the system MUST create (or reconcile) a lane-aware shadow workload in the target namespace.

#### Scenario: Shadow workload is created
- **WHEN** user runs `ktctl connect --namespace <ns> --lane <lane>`
- **THEN** the system creates a shadow Deployment/Pod in namespace `<ns>`
- **AND THEN** the shadow workload has a main container named `kt-connect-shadow`
- **AND THEN** the lane path uses an explicit multi-container template contract that is inspectable in rendered PodSpec

### Requirement: Shadow workload MUST have key Istio injection capabilities
The shadow workload MUST have the key Istio injection capabilities consistent with in-cluster caller pods, including `istio-init` and `istio-proxy`.

#### Scenario: Istio init and proxy exist
- **WHEN** the shadow workload is created in a namespace where Istio injection is available
- **THEN** the resulting Pod contains `istio-init` and `istio-proxy` containers
- **AND THEN** lane shadow template contains explicit init/sidecar constraints and sidecar resource annotations
- **AND THEN** lane shadow template contains a keystore init fragment with required volume/volumeMount links

## ADDED Requirements

### Requirement: Lane shadow template MUST be regression-tested at template level
The implementation MUST provide automated tests that validate lane/non-lane template rendering behavior for shadow workloads.

#### Scenario: Lane template renders required structures
- **WHEN** tests render lane shadow Pod/Deployment templates
- **THEN** assertions verify presence of main container, init container fragments, sidecar-related constraints/annotations, and keystore init fragment

#### Scenario: Non-lane path remains backward-compatible
- **WHEN** tests render non-lane shadow templates
- **THEN** assertions verify legacy single-container behavior remains unchanged unless explicitly configured otherwise

