package connect

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alibaba/kt-connect/pkg/common"
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/service/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/service/dns"
	"github.com/alibaba/kt-connect/pkg/kt/transmission"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"
	coreV1 "k8s.io/api/core/v1"
)

var clusterIns = cluster.Ins

func setupDns(shadowPodName, shadowPodIp string) error {
	if strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeHosts) {
		log.Info().Msgf("Setting up dns in hosts mode")
		dump2HostsNamespaces := ""
		pos := len(util.DnsModeHosts)
		if len(opt.Get().Connect.DnsMode) > pos+1 && opt.Get().Connect.DnsMode[pos:pos+1] == ":" {
			dump2HostsNamespaces = opt.Get().Connect.DnsMode[pos+1:]
		}
		if err := dumpToHost(dump2HostsNamespaces); err != nil {
			return err
		}
	} else if opt.Get().Connect.DnsMode == util.DnsModePodDns {
		log.Info().Msgf("Setting up dns in pod mode")
		return dns.SetNameServer(shadowPodIp)
	} else if strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeLocalDns) {
		log.Info().Msgf("Setting up dns in local mode")
		svcToIp, headlessPods := getServiceHosts(opt.Get().Global.Namespace, true)
		if err := dns.DumpHosts(svcToIp, ""); err != nil {
			return err
		}
		watchServicesAndPods(opt.Get().Global.Namespace, svcToIp, headlessPods, true)

		forwardedPodPort, err := setupShadowPortForward(shadowPodName, common.StandardDnsPort, "shadow DNS")
		if err != nil {
			return err
		}

		dnsPort := util.AlternativeDnsPort
		if util.IsWindows() {
			dnsPort = common.StandardDnsPort
		} else if util.IsMacos() {
			dnsPort = opt.Get().Connect.DnsPort
		}
		// must set up name server before change dns config
		// otherwise the upstream name server address will be incorrect in linux
		if err := dns.SetupLocalDns(forwardedPodPort, dnsPort, getDnsOrder(opt.Get().Connect.DnsMode)); err != nil {
			log.Error().Err(err).Msgf("Failed to setup local dns server")
			return err
		}
		return dns.SetNameServer(fmt.Sprintf("%s:%d", common.Localhost, dnsPort))
	} else {
		return fmt.Errorf("invalid dns mode: '%s', supportted mode are %s, %s, %s", opt.Get().Connect.DnsMode,
			util.DnsModeLocalDns, util.DnsModePodDns, util.DnsModeHosts)
	}
	return nil
}

func getDnsOrder(dnsMode string) []string {
	if !strings.Contains(dnsMode, ":") {
		return []string{util.DnsOrderCluster, util.DnsOrderUpstream}
	}
	return strings.Split(strings.SplitN(dnsMode, ":", 2)[1], ",")
}

func setupShadowPortForward(shadowPodName string, remotePort int, target string) (int, error) {
	localPort := util.GetRandomTcpPort()
	if opt.Get().Connect.Lane == "" {
		if _, err := transmission.SetupPortForwardToLocal(shadowPodName, remotePort, localPort); err != nil {
			return 0, fmt.Errorf("setup %s port-forward for pod %s: %w", target, shadowPodName, err)
		}
		return localPort, nil
	}
	gone, err := transmission.SetupSessionPortForwardToLocal(shadowPodName, remotePort, localPort)
	if err != nil {
		return 0, fmt.Errorf("setup %s port-forward for pod %s: %w", target, shadowPodName, err)
	}
	watchConnectionGone(target, gone, terminateConnectSession)
	return localPort, nil
}

func watchConnectionGone(target string, gone <-chan int, onGone func()) {
	if gone == nil {
		return
	}
	go func() {
		<-gone
		log.Warn().Msgf("%s connection interrupted, ending connect session", target)
		if onGone != nil {
			onGone()
		}
	}()
}

func terminateConnectSession() {
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to locate current process for session shutdown")
		return
	}
	if err = process.Signal(os.Interrupt); err != nil {
		log.Warn().Err(err).Msg("Failed to signal current process for session shutdown")
	}
}

func watchServicesAndPods(namespace string, svcToIp map[string]string, headlessPods []string, shortDomainOnly bool) {
	setupTime := time.Now().Unix()
	go clusterIns().WatchService("", namespace,
		func(svc *coreV1.Service) {
			// ignore add service event during watch setup
			if time.Now().Unix()-setupTime > 3 {
				svcToIp, headlessPods = getServiceHosts(namespace, shortDomainOnly)
				_ = dns.DumpHosts(svcToIp, namespace)
			}
		},
		func(svc *coreV1.Service) {
			svcToIp, headlessPods = getServiceHosts(namespace, shortDomainOnly)
			_ = dns.DumpHosts(svcToIp, namespace)
		}, nil)
	go clusterIns().WatchPod("", namespace, nil, func(pod *coreV1.Pod) {
		if util.Contains(headlessPods, pod.Name) {
			// it may take some time for new pod get assign an ip
			time.Sleep(5 * time.Second)
			svcToIp, headlessPods = getServiceHosts(namespace, shortDomainOnly)
			_ = dns.DumpHosts(svcToIp, namespace)
		}
	}, nil)
}

func dumpToHost(targetNamespaces string) error {
	namespacesToDump := []string{opt.Get().Global.Namespace}
	if targetNamespaces != "" {
		namespacesToDump = []string{}
		for _, ns := range strings.Split(targetNamespaces, ",") {
			namespacesToDump = append(namespacesToDump, ns)
		}
	}
	hosts := map[string]string{}
	for _, namespace := range namespacesToDump {
		log.Debug().Msgf("Search service in %s namespace ...", namespace)
		svcToIp, headlessPods := getServiceHosts(namespace, false)
		watchServicesAndPods(namespace, svcToIp, headlessPods, false)
		for svc, ip := range svcToIp {
			hosts[svc] = ip
		}
	}
	return dns.DumpHosts(hosts, "")
}

func getServiceHosts(namespace string, shortDomainOnly bool) (map[string]string, []string) {
	hosts := make(map[string]string)
	podNames := make([]string, 0)
	services, err := clusterIns().GetAllServiceInNamespace(namespace)
	if err == nil {
		for _, service := range services.Items {
			ip := service.Spec.ClusterIP
			if ip == "" || ip == "None" {
				pods, err2 := clusterIns().GetPodsByLabel(service.Spec.Selector, namespace)
				if err2 != nil || len(pods.Items) == 0 {
					continue
				}
				for _, p := range pods.Items {
					ip = p.Status.PodIP
					if ip != "" {
						podNames = append(podNames, p.Name)
						break
					}
				}
				log.Debug().Msgf("Headless service found: %s.%s %s", service.Name, namespace, ip)
			} else {
				log.Debug().Msgf("Service found: %s.%s %s", service.Name, namespace, ip)
			}
			if shortDomainOnly {
				hosts[service.Name] = ip
			} else {
				if namespace == opt.Get().Global.Namespace {
					hosts[service.Name] = ip
				}
				hosts[fmt.Sprintf("%s.%s", service.Name, namespace)] = ip
				hosts[fmt.Sprintf("%s.%s.svc.%s", service.Name, namespace, opt.Get().Connect.ClusterDomain)] = ip
			}
		}
	}
	return hosts, podNames
}

func getOrCreateShadow() (string, string, string, error) {
	shadowPodName := fmt.Sprintf("kt-connect-shadow-%s", strings.ToLower(util.RandomString(5)))
	if opt.Get().Connect.Lane != "" {
		shadowPodName = buildLaneShadowName()
	}
	if opt.Get().Connect.ShareShadow {
		shadowPodName = fmt.Sprintf("kt-connect-shadow-daemon")
	}

	annotations := buildShadowAnnotations(opt.Get().Connect.Lane)
	labels := getLabels()
	endPointIP, podName, privateKeyPath, err := clusterIns().GetOrCreateShadow(shadowPodName, labels,
		annotations, getEnvs(), "", map[int]string{})
	if err != nil {
		return "", "", "", err
	}

	if opt.Get().Connect.Lane != "" {
		envoyFilterName := fmt.Sprintf("kt-lane-%s", opt.Store.Session)
		if err := clusterIns().ApplyLaneEnvoyFilter(envoyFilterName, opt.Get().Global.Namespace, labels, opt.Get().Connect.Lane); err != nil {
			return "", "", "", err
		}
		opt.Store.LaneEnvoyFilter = envoyFilterName
	}

	return endPointIP, podName, privateKeyPath, nil
}

func buildLaneShadowName() string {
	if opt.Store.Session == "" {
		opt.Store.Session = fmt.Sprintf("s%s-%s", util.GetTimestamp(), strings.ToLower(util.RandomString(5)))
	}
	return fmt.Sprintf("kt-connect-shadow-%s-%s-%s", opt.Get().Global.Namespace, opt.Get().Connect.Lane, opt.Store.Session)
}

func getEnvs() map[string]string {
	envs := make(map[string]string)
	localDomains := dns.GetLocalDomains()
	if localDomains != "" {
		log.Debug().Msgf("Found local domains: %s", localDomains)
		envs[common.EnvVarLocalDomains] = localDomains
	}
	if strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeLocalDns) {
		envs[common.EnvVarDnsProtocol] = "tcp"
	} else {
		envs[common.EnvVarDnsProtocol] = "udp"
	}
	if opt.Get().Global.Debug {
		envs[common.EnvVarLogLevel] = "debug"
	} else {
		envs[common.EnvVarLogLevel] = "info"
	}
	if opt.Get().Connect.Lane != "" {
		envs[common.EnvVarLane] = opt.Get().Connect.Lane
	}
	return envs
}

func getLabels() map[string]string {
	labels := map[string]string{
		util.KtRole: util.RoleConnectShadow,
	}
	if opt.Get().Connect.Lane != "" {
		labels[util.KtLane] = opt.Get().Connect.Lane
		labels["sidecar.istio.io/inject"] = "true"
		if opt.Store.Session == "" {
			opt.Store.Session = fmt.Sprintf("s%s-%s", util.GetTimestamp(), strings.ToLower(util.RandomString(5)))
		}
		labels[util.KtSession] = opt.Store.Session
	}
	if opt.Get().Global.UseShadowDeployment {
		labels[util.KtTarget] = util.RandomString(20)
	}
	return labels
}

func buildShadowAnnotations(lane string) map[string]string {
	if lane == "" {
		return map[string]string{}
	}
	return map[string]string{
		"sidecar.istio.io/inject":           "true",
		"sidecar.istio.io/proxyCPU":         "1000m",
		"sidecar.istio.io/proxyCPULimit":    "1500m",
		"sidecar.istio.io/proxyMemory":      "200Mi",
		"sidecar.istio.io/proxyMemoryLimit": "500Mi",
	}
}
