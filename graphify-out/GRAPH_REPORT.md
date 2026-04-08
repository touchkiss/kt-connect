# Graph Report - .graphify_scope  (2026-04-07)

## Corpus Check
- Corpus is ~37,548 words - fits in a single context window. You may not need a graph.

## Summary
- 656 nodes · 897 edges · 81 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 240 edges (avg confidence: 0.5)
- Token cost: 0 input · 0 output

## God Nodes (most connected - your core abstractions)
1. `Kubernetes` - 12 edges
2. `Kubernetes` - 9 edges
3. `Kubernetes` - 9 edges
4. `DnsServer` - 8 edges
5. `CleanupWorkspace()` - 7 edges
6. `Kubernetes` - 7 edges
7. `CheckClusterResources()` - 6 edges
8. `setupDns()` - 6 edges
9. `calculateMinimalIpRange()` - 6 edges
10. `ipToBin()` - 6 edges

## Surprising Connections (you probably didn't know these)
- None detected - all connections are within the same source files.

## Communities

### Community 0 - "Lane Cleanup Core"
Cohesion: 0.05
Nodes (40): analysisConfigAnnotation(), analysisExpiredConfigmaps(), analysisExpiredDeployments(), analysisExpiredPods(), analysisExpiredServices(), analysisLockAndOrphanServices(), buildLaneShadowName(), buildShadowAnnotations() (+32 more)

### Community 1 - "Custom Config DNS"
Cohesion: 0.04
Nodes (12): fetchNameServerInConf(), GetNameServer(), Expose(), exposeLocalService(), ResourceMeta, SSHkeyMeta, ForwardPodToLocal(), ForwardRemotePortsViaSshTunnel() (+4 more)

### Community 2 - "CLI Config Commands"
Cohesion: 0.05
Nodes (30): createResolverFile(), getAllDomainSuffixes(), HandleExtraDomainMapping(), RestoreNameServer(), SetNameServer(), getMeshLabels(), ManualMesh(), Mesh() (+22 more)

### Community 3 - "Pod Operations"
Cohesion: 0.1
Nodes (11): execute(), handlePodEvent(), Kubernetes, PodMetaAndSpec, GetDeploymentByResourceName(), getDeploymentByService(), getServiceByDeployment(), GetServiceByResourceName() (+3 more)

### Community 4 - "Proxy And Teardown"
Cohesion: 0.15
Nodes (20): isLaneBaggageMember(), mergeBaggage(), New(), cleanLaneSessionResources(), cleanLocalFiles(), cleanService(), cleanShadowPodAndConfigMap(), CleanupWorkspace() (+12 more)

### Community 5 - "Service Deployment Ops"
Cohesion: 0.09
Nodes (10): handleServiceEvent(), Kubernetes, SvcMetaAndSpec, Channel, Cli, Kubernetes, KubernetesInterface, Sshuttle (+2 more)

### Community 6 - "SSH Tunnel Utilities"
Cohesion: 0.15
Nodes (13): Cli, disconnectRemotePort(), encodePrivateKeyToPEM(), encodePublicKey(), Generate(), generatePrivateKey(), getSshTunnelAddress(), handleBrokenTunnel() (+5 more)

### Community 7 - "System Utilities"
Cohesion: 0.15
Nodes (6): GetDaemonRunning(), GetTime(), GetTimestamp(), IsProcessExist(), watchPidFile(), WritePidFile()

### Community 8 - "CIDR Calculations"
Cohesion: 0.3
Nodes (13): binToIpRange(), calculateMinimalIpRange(), calculateMinimalIpv6Range(), decToBin(), excludeIpFromRange(), getPodIps(), getServiceIps(), ipRangeToBin() (+5 more)

### Community 9 - "Shadow Pod Setup"
Cohesion: 0.27
Nodes (4): enableExplicitLaneShadowTemplate(), filterSidecarAnnotations(), getSSHVolume(), Kubernetes

### Community 10 - "Connect Clean Commands"
Cohesion: 0.29
Nodes (8): Clean(), isEmpty(), NewCleanCommand(), checkPermissionAndOptions(), Connect(), NewConnectCommand(), preCheck(), silenceCleanup()

### Community 11 - "Teardown Tests"
Cohesion: 0.27
Nodes (7): assertLaneSessionLabels(), cleanupClusterStub, cloneLabels(), TestCleanShadowPodAndConfigMapContinuesWhenLaneSessionCleanupFails(), TestCleanShadowPodAndConfigMapRemovesLaneSessionResourcesByLabel(), withCleanupStub(), withLaneSessionCleanupState()

### Community 12 - "Connect Common Tests"
Cohesion: 0.21
Nodes (3): cloneStringMap(), shadowCreateStub, TestGetOrCreateShadowUsesLaneSessionMetadata()

### Community 13 - "Ephemeral Exchange"
Cohesion: 0.35
Nodes (10): ByEphemeralContainer(), createEphemeralContainer(), exchangeWithEphemeralContainer(), getListenedPorts(), getPodsOfResource(), getPodsOfService(), isEphemeralContainerReady(), randPort() (+2 more)

### Community 14 - "CIDR Tests"
Cohesion: 0.22
Nodes (3): buildPod(), buildService(), TestKubernetes_ClusterCidr()

### Community 15 - "Windows Tun Routing"
Cohesion: 0.31
Nodes (4): Cli, getInterfaceIndex(), getKtRouteRecords(), RouteRecord

### Community 16 - "Unix DNS Server"
Cohesion: 0.33
Nodes (1): DnsServer

### Community 17 - "Deployment Lifecycle"
Cohesion: 0.33
Nodes (1): Kubernetes

### Community 18 - "Workload Builders"
Cohesion: 0.33
Nodes (5): appendUniqueVolumeMounts(), createContainer(), createDeployment(), createPod(), createPodSpec()

### Community 19 - "DNS Server Core"
Cohesion: 0.36
Nodes (7): DnsServer, getDnsAddresses(), getIngressDomains(), query(), SetupLocalDns(), toARecord(), wildcardMatch()

### Community 20 - "DNS Cache Utils"
Cohesion: 0.36
Nodes (5): getCacheKey(), notExpired(), NsEntry, ReadCache(), WriteCache()

### Community 21 - "Collection Helpers"
Cohesion: 0.29
Nodes (2): MapContains(), MapEquals()

### Community 22 - "Auto Mesh Flow"
Cohesion: 0.46
Nodes (7): AutoMesh(), createRouter(), createShadowService(), createStuntmanService(), isNameUsable(), sanityCheck(), toPortMapParameter()

### Community 23 - "Heartbeat Tracking"
Cohesion: 0.29
Nodes (2): HeartBeatStatus, SetupTimeDifference()

### Community 24 - "Linux DNS Setup"
Cohesion: 0.46
Nodes (6): restoreIptables(), RestoreNameServer(), restoreResolvConf(), SetNameServer(), setupIptables(), setupResolvConf()

### Community 25 - "Port Forwarding"
Cohesion: 0.57
Nodes (6): closeGone(), closeStop(), createPortForwarder(), parseReqHost(), SetupPortForwardToLocal(), SetupSessionPortForwardToLocal()

### Community 26 - "Hosts File Sync"
Cohesion: 0.67
Nodes (6): DropHosts(), DumpHosts(), getHostsPath(), loadHostsFile(), mergeLines(), updateHostsFile()

### Community 27 - "Logger Lifecycle"
Cohesion: 0.4
Nodes (3): CleanBackgroundLogs(), FileWriter, isExpired()

### Community 28 - "Recover Command"
Cohesion: 0.6
Nodes (4): checkAndMarkUnlock(), fetchTargetRole(), NewRecoverCommand(), Recover()

### Community 29 - "Birdseye Status"
Cohesion: 0.6
Nodes (4): Birdseye(), NewBirdseyeCommand(), showConnectors(), showServiceStatus()

### Community 30 - "ConfigMap Lifecycle"
Cohesion: 0.33
Nodes (1): Kubernetes

### Community 31 - "Route CLI Variant"
Cohesion: 0.47
Nodes (1): Cli

### Community 32 - "Route CLI Variant"
Cohesion: 0.47
Nodes (1): Cli

### Community 33 - "Interpretable Reader"
Cohesion: 0.4
Nodes (1): InterpretableReader

### Community 34 - "String Utils Tests"
Cohesion: 0.4
Nodes (0): 

### Community 35 - "Collection Tests"
Cohesion: 0.4
Nodes (0): 

### Community 36 - "Exchange Command"
Cohesion: 0.7
Nodes (3): Exchange(), NewExchangeCommand(), toTypeAndName()

### Community 37 - "Forward Command"
Cohesion: 0.7
Nodes (3): Forward(), NewForwardCommand(), parsePort()

### Community 38 - "Set Config Command"
Cohesion: 0.5
Nodes (2): Set(), setConfigValue()

### Community 39 - "Route Error Type"
Cohesion: 0.4
Nodes (1): AllRouteFailError

### Community 40 - "Nginx Route Reload"
Cohesion: 0.7
Nodes (4): reloadRouteConf(), removeRouteConf(), WriteAndReloadRouteConf(), writeRouteConf()

### Community 41 - "Unix DNS Tests"
Cohesion: 0.5
Nodes (0): 

### Community 42 - "DNS Error Type"
Cohesion: 0.5
Nodes (1): DomainNotExistError

### Community 43 - "SSH Key Tests"
Cohesion: 0.5
Nodes (0): 

### Community 44 - "Background Runner"
Cohesion: 0.5
Nodes (0): 

### Community 45 - "Scale Exchange Logic"
Cohesion: 0.83
Nodes (3): ByScale(), getExchangeAnnotation(), getExchangeLabels()

### Community 46 - "Unset Config Command"
Cohesion: 0.67
Nodes (2): Unset(), unsetConfigValue()

### Community 47 - "Get Config Command"
Cohesion: 0.67
Nodes (2): Get(), getConfigValue()

### Community 48 - "Namespace Watchers"
Cohesion: 0.5
Nodes (1): Kubernetes

### Community 49 - "Hosts Tests"
Cohesion: 0.5
Nodes (0): 

### Community 50 - "Hack CLI Install"
Cohesion: 0.5
Nodes (1): Cli

### Community 51 - "Unix Admin Checks"
Cohesion: 0.67
Nodes (0): 

### Community 52 - "Windows Admin Checks"
Cohesion: 0.67
Nodes (0): 

### Community 53 - "Port Forward Tests"
Cohesion: 0.67
Nodes (0): 

### Community 54 - "Option Config Model"
Cohesion: 0.67
Nodes (1): OptionConfig

### Community 55 - "Show Profile Command"
Cohesion: 0.67
Nodes (0): 

### Community 56 - "Drop Profile Command"
Cohesion: 0.67
Nodes (0): 

### Community 57 - "Load Profile Command"
Cohesion: 0.67
Nodes (0): 

### Community 58 - "Service Locking"
Cohesion: 0.67
Nodes (0): 

### Community 59 - "Ephemeral Container Ops"
Cohesion: 0.67
Nodes (1): Kubernetes

### Community 60 - "Router Rectifier Pods"
Cohesion: 0.67
Nodes (1): Kubernetes

### Community 61 - "DNS Server Tests"
Cohesion: 0.67
Nodes (0): 

### Community 62 - "Route Config File"
Cohesion: 0.67
Nodes (0): 

### Community 63 - "Network Tests"
Cohesion: 1.0
Nodes (0): 

### Community 64 - "Auto Mesh Tests"
Cohesion: 1.0
Nodes (0): 

### Community 65 - "Runtime Store"
Cohesion: 1.0
Nodes (1): RuntimeStore

### Community 66 - "Usage Template"
Cohesion: 1.0
Nodes (0): 

### Community 67 - "Sorter Tests"
Cohesion: 1.0
Nodes (0): 

### Community 68 - "Kube Client Tests"
Cohesion: 1.0
Nodes (0): 

### Community 69 - "Service Tests"
Cohesion: 1.0
Nodes (0): 

### Community 70 - "Ingress Queries"
Cohesion: 1.0
Nodes (1): Kubernetes

### Community 71 - "Lane Envoy Filter"
Cohesion: 1.0
Nodes (1): Kubernetes

### Community 72 - "Scale Deployment Tests"
Cohesion: 1.0
Nodes (0): 

### Community 73 - "Socks Tunnel CLI"
Cohesion: 1.0
Nodes (1): Cli

### Community 74 - "Darwin DNS Tests"
Cohesion: 1.0
Nodes (0): 

### Community 75 - "Kt Config Types"
Cohesion: 1.0
Nodes (1): KtConf

### Community 76 - "Shared Constants"
Cohesion: 1.0
Nodes (0): 

### Community 77 - "Unix Constants"
Cohesion: 1.0
Nodes (0): 

### Community 78 - "Windows Constants"
Cohesion: 1.0
Nodes (0): 

### Community 79 - "Ingress Helpers"
Cohesion: 1.0
Nodes (0): 

### Community 80 - "Lane Metadata"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **33 isolated node(s):** `NsEntry`, `SSHGenerator`, `ConnectOptions`, `ExchangeOptions`, `MeshOptions` (+28 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Network Tests`** (2 nodes): `network_test.go`, `TestExtractHostIp()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Auto Mesh Tests`** (2 nodes): `auto_test.go`, `Test_toPortMapParameter()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Runtime Store`** (2 nodes): `store.go`, `RuntimeStore`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Usage Template`** (2 nodes): `usage.go`, `UsageTemplate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Sorter Tests`** (2 nodes): `sorter_test.go`, `TestSortServiceArray()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Kube Client Tests`** (2 nodes): `helper_test.go`, `Test_getKubernetesClient()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Service Tests`** (2 nodes): `service_test.go`, `TestKubernetes_CreateService()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Ingress Queries`** (2 nodes): `Kubernetes`, `.GetAllIngressInNamespace()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Lane Envoy Filter`** (2 nodes): `Kubernetes`, `.ApplyLaneEnvoyFilter()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Scale Deployment Tests`** (2 nodes): `deployment_test.go`, `TestKubernetes_ScaleTo()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Socks Tunnel CLI`** (2 nodes): `Cli`, `.ToSocks()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Darwin DNS Tests`** (2 nodes): `dns_darwin_test.go`, `Test_getAllDomainSuffixes()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Kt Config Types`** (2 nodes): `type.go`, `KtConf`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Shared Constants`** (1 nodes): `const.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Unix Constants`** (1 nodes): `const_unix.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Windows Constants`** (1 nodes): `const_windows.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Ingress Helpers`** (1 nodes): `igress.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Lane Metadata`** (1 nodes): `lane.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Are the 6 inferred relationships involving `CleanupWorkspace()` (e.g. with `cleanLocalFiles()` and `recoverGlobalHostsAndProxy()`) actually correct?**
  _`CleanupWorkspace()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `NsEntry`, `SSHGenerator`, `ConnectOptions` to the rest of the system?**
  _33 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Lane Cleanup Core` be split into smaller, more focused modules?**
  _Cohesion score 0.05 - nodes in this community are weakly interconnected._
- **Should `Custom Config DNS` be split into smaller, more focused modules?**
  _Cohesion score 0.04 - nodes in this community are weakly interconnected._
- **Should `CLI Config Commands` be split into smaller, more focused modules?**
  _Cohesion score 0.05 - nodes in this community are weakly interconnected._
- **Should `Pod Operations` be split into smaller, more focused modules?**
  _Cohesion score 0.1 - nodes in this community are weakly interconnected._
- **Should `Service Deployment Ops` be split into smaller, more focused modules?**
  _Cohesion score 0.09 - nodes in this community are weakly interconnected._