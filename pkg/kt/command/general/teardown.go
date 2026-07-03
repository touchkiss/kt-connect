package general

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/service/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/service/dns"
	"github.com/alibaba/kt-connect/pkg/kt/service/tun"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var clusterIns = cluster.Ins
var tunIns = tun.Ins

// CleanupWorkspace clean workspace
func CleanupWorkspace() {
	log.Debug().Msgf("Cleaning workspace")
	cleanLocalFiles()
	if opt.Store.Component == util.ComponentConnect {
		recoverGlobalHostsAndProxy()
		shutdownTunDevice()
	}

	if opt.Store.Component == util.ComponentExchange {
		recoverExchangedTarget()
	} else if opt.Store.Component == util.ComponentMesh {
		recoverAutoMeshRoute()
	}
	cleanService()
	cleanShadowPodAndConfigMap()
}

// shutdownTunDevice stops the tun2socks engine so the tun (utun) device is
// destroyed deterministically on session end, not only via the ToSocks signal
// goroutine. Skipped when no tun device was created (--disableTunDevice).
func shutdownTunDevice() {
	if opt.Get().Connect.DisableTunDevice {
		return
	}
	if err := tunIns().Shutdown(); err != nil {
		log.Debug().Err(err).Msgf("Failed to shutdown tun device")
	}
}

func recoverGlobalHostsAndProxy() {
	if strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeHosts) ||
		strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeLocalDns) {
		log.Debug().Msg("Dropping hosts records ...")
		dns.DropHosts()
	}
	if strings.HasPrefix(opt.Get().Connect.DnsMode, util.DnsModeLocalDns) {
		if err := tun.Ins().RestoreRoute(); err != nil {
			log.Debug().Err(err).Msgf("Failed to restore route table")
		}
	}
}

func cleanLocalFiles() {
	if opt.Store.Component == "" {
		return
	}
	pidFile := fmt.Sprintf("%s/%s-%d.pid", util.KtPidDir, opt.Store.Component, os.Getpid())
	if err := os.Remove(pidFile); os.IsNotExist(err) {
		log.Debug().Msgf("Pid file %s not exist", pidFile)
	} else if err != nil {
		log.Debug().Err(err).Msgf("Remove pid file %s failed", pidFile)
	} else {
		log.Info().Msgf("Removed pid file %s", pidFile)
	}

	if opt.Store.Shadow != "" {
		for _, sshcm := range strings.Split(opt.Store.Shadow, ",") {
			file := util.PrivateKeyPath(sshcm)
			if err := os.Remove(file); os.IsNotExist(err) {
				log.Debug().Msgf("Key file %s not exist", file)
			} else if err != nil {
				log.Debug().Msgf("Remove key file %s failed", pidFile)
			} else {
				log.Info().Msgf("Removed key file %s", file)
			}
		}
	}
}

func recoverExchangedTarget() {
	if opt.Store.Origin == "" {
		// process exit before target exchanged
		return
	}
	if opt.Get().Exchange.Mode == util.ExchangeModeScale {
		log.Info().Msgf("Recovering origin deployment %s", opt.Store.Origin)
		err := clusterIns().ScaleTo(opt.Store.Origin, opt.Get().Global.Namespace, &opt.Store.Replicas)
		if err != nil {
			log.Error().Err(err).Msgf("Scale deployment %s to %d failed",
				opt.Store.Origin, opt.Store.Replicas)
		}
		// wait for scale complete
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		go func() {
			waitDeploymentRecoverComplete()
			ch <- os.Interrupt
		}()
		_ = <-ch
	} else if opt.Get().Exchange.Mode == util.ExchangeModeSelector {
		RecoverOriginalService(opt.Store.Origin, opt.Get().Global.Namespace)
		log.Info().Msgf("Original service %s recovered", opt.Store.Origin)
	}
}

func recoverAutoMeshRoute() {
	if opt.Store.Router != "" {
		routerPod, err := clusterIns().GetPod(opt.Store.Router, opt.Get().Global.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Router pod has been removed unexpectedly")
			// in case of router pod gone, try recover origin service via runtime store
			if opt.Store.Origin != "" {
				recoverService(opt.Store.Origin)
			}
			return
		}
		if shouldDelRouter, err2 := clusterIns().DecreasePodRef(opt.Store.Router, opt.Get().Global.Namespace); err2 != nil {
			log.Error().Err(err2).Msgf("Decrease router pod %s reference failed", opt.Store.Shadow)
		} else if shouldDelRouter {
			routerConfig := routerPod.Annotations[util.KtConfig]
			config := util.String2Map(routerConfig)
			recoverService(config["service"])
			if err = clusterIns().RemovePod(opt.Store.Router, opt.Get().Global.Namespace); err != nil {
				log.Warn().Err(err).Msgf("Failed to remove router pod")
			}
		} else {
			stdout, stderr, err3 := clusterIns().ExecInPod(util.DefaultContainer, opt.Store.Router, opt.Get().Global.Namespace,
				util.RouterBin, "remove", opt.Store.Mesh)
			log.Debug().Msgf("Stdout: %s", stdout)
			log.Debug().Msgf("Stderr: %s", stderr)
			if err3 != nil {
				log.Warn().Err(err3).Msgf("Failed to remove version %s from router pod", opt.Store.Mesh)
			}
		}
	}
}

func recoverService(originSvcName string) {
	RecoverOriginalService(originSvcName, opt.Get().Global.Namespace)
	log.Info().Msgf("Original service %s recovered", originSvcName)

	stuntmanSvcName := originSvcName + util.StuntmanServiceSuffix
	if err := clusterIns().RemoveService(stuntmanSvcName, opt.Get().Global.Namespace); err != nil {
		log.Error().Err(err).Msgf("Failed to remove stuntman service %s", stuntmanSvcName)
	}
	log.Info().Msgf("Stuntman service %s removed", stuntmanSvcName)
}

func RecoverOriginalService(svcName, namespace string) {
	if svc, err := clusterIns().GetService(svcName, namespace); err != nil {
		log.Error().Err(err).Msgf("Original service %s not found", svcName)
		return
	} else {
		var selector map[string]string
		if svc.Annotations == nil {
			log.Warn().Msgf("No annotation found in service %s, skipping", svcName)
			return
		}
		originSelector, exists := svc.Annotations[util.KtSelector]
		if !exists {
			log.Warn().Msgf("No selector annotation found in service %s, skipping", svcName)
			return
		}
		if err = json.Unmarshal([]byte(originSelector), &selector); err != nil {
			log.Error().Err(err).Msgf("Failed to unmarshal original selector of service %s", svcName)
			return
		}
		svc.Spec.Selector = selector
		delete(svc.Annotations, util.KtSelector)
		if _, err = clusterIns().UpdateService(svc); err != nil {
			log.Error().Err(err).Msgf("Failed to recover selector of original service %s", svcName)
		}
	}
}

func waitDeploymentRecoverComplete() {
	ok := false
	counts := opt.Get().Exchange.RecoverWaitTime / 5
	for i := 0; i < counts; i++ {
		deployment, err := clusterIns().GetDeployment(opt.Store.Origin, opt.Get().Global.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Cannot fetch original deployment %s", opt.Store.Origin)
			break
		} else if deployment.Status.ReadyReplicas == opt.Store.Replicas {
			ok = true
			break
		} else {
			log.Info().Msgf("Wait for deployment %s recover ...", opt.Store.Origin)
			time.Sleep(5 * time.Second)
		}
	}
	if !ok {
		log.Warn().Msgf("Deployment %s recover timeout", opt.Store.Origin)
	}
}

func cleanService() {
	if opt.Store.Service != "" {
		log.Info().Msgf("Cleaning service %s", opt.Store.Service)
		err := clusterIns().RemoveService(opt.Store.Service, opt.Get().Global.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Delete service %s failed", opt.Store.Service)
		}
	}
}

func cleanShadowPodAndConfigMap() {
	var err error
	if shouldCleanLaneSessionResources() {
		cleanLaneSessionResources()
		return
	}
	if shouldCleanLaneEnvoyFilters() {
		defer cleanLaneSessionEnvoyFilters(map[string]string{
			util.KtRole:    util.RoleConnectShadow,
			util.KtLane:    opt.Store.Lane,
			util.KtSession: opt.Store.Session,
		}, opt.Get().Global.Namespace)
	}
	if opt.Store.Shadow != "" {
		shouldDelWithShared := false
		if opt.Get().Connect.ShareShadow {
			// There is always exactly one shadow pod or deployment for connect
			if opt.Get().Global.UseShadowDeployment {
				shouldDelWithShared, err = clusterIns().DecreaseDeploymentRef(opt.Store.Shadow, opt.Get().Global.Namespace)
			} else {
				shouldDelWithShared, err = clusterIns().DecreasePodRef(opt.Store.Shadow, opt.Get().Global.Namespace)
			}
			if err != nil {
				log.Error().Err(err).Msgf("Decrease shadow daemon %s ref count failed", opt.Store.Shadow)
			}
		}
		if shouldDelWithShared || !opt.Get().Connect.ShareShadow {
			for _, shadow := range strings.Split(opt.Store.Shadow, ",") {
				log.Info().Msgf("Cleaning configmap %s", shadow)
				err = clusterIns().RemoveConfigMap(shadow, opt.Get().Global.Namespace)
				if err != nil {
					log.Error().Err(err).Msgf("Delete configmap %s failed", shadow)
				}
				log.Info().Msgf("Cleaning shadow pod %s", shadow)
				if opt.Get().Global.UseShadowDeployment {
					err = clusterIns().RemoveDeployment(shadow, opt.Get().Global.Namespace)
				} else {
					err = clusterIns().RemovePod(shadow, opt.Get().Global.Namespace)
				}
				if err != nil {
					log.Error().Err(err).Msgf("Delete shadow pod %s failed", shadow)
				}
			}
		}
		if opt.Get().Exchange.Mode == util.ExchangeModeEphemeral {
			for _, shadow := range strings.Split(opt.Store.Shadow, ",") {
				log.Info().Msgf("Removing ephemeral container of pod %s", shadow)
				err = clusterIns().RemoveEphemeralContainer(util.KtExchangeContainer, shadow, opt.Get().Global.Namespace)
				if err != nil {
					log.Error().Err(err).Msgf("Remove ephemeral container of pod %s failed", shadow)
				}
			}
		}
	}
}

func shouldCleanLaneSessionResources() bool {
	return opt.Store.Component == util.ComponentConnect && opt.Store.Lane != "" && opt.Store.Session != "" && opt.Store.Shadow == ""
}

func cleanLaneSessionResources() {
	labels := map[string]string{
		util.KtRole:    util.RoleConnectShadow,
		util.KtLane:    opt.Store.Lane,
		util.KtSession: opt.Store.Session,
	}
	namespace := opt.Get().Global.Namespace
	defer cleanLaneSessionEnvoyFilters(labels, namespace)
	if configs, err := clusterIns().GetConfigMapsByLabel(labels, namespace); err != nil {
		log.Error().Err(err).Msg("List lane session configmaps failed")
	} else {
		for _, configMap := range configs.Items {
			log.Info().Msgf("Cleaning configmap %s", configMap.Name)
			if err = clusterIns().RemoveConfigMap(configMap.Name, namespace); err != nil {
				log.Error().Err(err).Msgf("Delete configmap %s failed", configMap.Name)
			}
		}
	}
	if opt.Get().Global.UseShadowDeployment {
		if deployments, err := clusterIns().GetDeploymentsByLabel(labels, namespace); err != nil {
			log.Error().Err(err).Msg("List lane session deployments failed")
		} else {
			for _, deployment := range deployments.Items {
				log.Info().Msgf("Cleaning shadow deployment %s", deployment.Name)
				if err = clusterIns().RemoveDeployment(deployment.Name, namespace); err != nil {
					log.Error().Err(err).Msgf("Delete shadow deployment %s failed", deployment.Name)
				}
			}
		}
		return
	}
	if pods, err := clusterIns().GetPodsByLabel(labels, namespace); err != nil {
		log.Error().Err(err).Msg("List lane session pods failed")
	} else {
		for _, pod := range pods.Items {
			log.Info().Msgf("Cleaning shadow pod %s", pod.Name)
			if err = clusterIns().RemovePod(pod.Name, namespace); err != nil {
				log.Error().Err(err).Msgf("Delete shadow pod %s failed", pod.Name)
			}
		}
	}
}

func shouldCleanLaneEnvoyFilters() bool {
	return opt.Store.Component == util.ComponentConnect && opt.Store.Lane != "" && opt.Store.Session != ""
}

func cleanLaneSessionEnvoyFilters(labels map[string]string, namespace string) {
	if opt.Store.RestConfig == nil {
		log.Warn().Msg("Skip lane envoyfilter cleanup: rest config not initialized")
		return
	}
	client, err := dynamic.NewForConfig(opt.Store.RestConfig)
	if err != nil {
		log.Error().Err(err).Msg("Create dynamic client for lane envoyfilter cleanup failed")
		return
	}
	gvr := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "envoyfilters"}
	resource := client.Resource(gvr).Namespace(namespace)
	if opt.Store.LaneEnvoyFilter != "" {
		if err = resource.Delete(context.TODO(), opt.Store.LaneEnvoyFilter, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			log.Error().Err(err).Msgf("Delete lane envoyfilter %s failed", opt.Store.LaneEnvoyFilter)
		} else {
			log.Info().Msgf("Cleaned lane envoyfilter %s", opt.Store.LaneEnvoyFilter)
		}
	}
	selector := fmt.Sprintf("%s=%s,%s=%s,%s=%s", util.KtRole, labels[util.KtRole], util.KtLane, labels[util.KtLane], util.KtSession, labels[util.KtSession])
	list, err := resource.List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error().Err(err).Msg("List lane envoyfilters failed")
		return
	}
	for _, item := range list.Items {
		if opt.Store.LaneEnvoyFilter != "" && item.GetName() == opt.Store.LaneEnvoyFilter {
			continue
		}
		if err = resource.Delete(context.TODO(), item.GetName(), metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			log.Error().Err(err).Msgf("Delete lane envoyfilter %s failed", item.GetName())
			continue
		}
		log.Info().Msgf("Cleaned lane envoyfilter %s", item.GetName())
	}
}
