package cluster

import (
	"context"
	"fmt"
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	laneShadowKeystoreVolumeName = "kt-shadow-keystore"
	laneShadowKeystoreMountPath  = "/var/kt-connect/keystore"
	laneShadowKeystoreInitName   = "kt-shadow-keystore-init"
)

// GetOrCreateShadow create shadow pod or deployment
func (k *Kubernetes) GetOrCreateShadow(name string, labels, annotations, envs map[string]string, exposePorts string, portNameDict map[int]string) (
	string, string, string, error) {
	// record context data
	opt.Store.Shadow = name

	// extra labels must be applied after origin labels
	for key, val := range util.String2Map(opt.Get().Global.WithLabel) {
		labels[key] = val
	}
	for key, val := range util.String2Map(opt.Get().Global.WithAnnotation) {
		annotations[key] = val
	}
	if lane, exists := labels[util.KtLane]; exists {
		opt.Store.Lane = lane
	}
	if session, exists := labels[util.KtSession]; exists {
		opt.Store.Session = session
	}
	annotations[util.KtUser] = util.GetLocalUserName()
	resourceMeta := ResourceMeta{
		Name:        name,
		Namespace:   opt.Get().Global.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}
	sshKeyMeta := SSHkeyMeta{
		SshConfigMapName: name,
		PrivateKeyPath:   util.PrivateKeyPath(name),
	}

	ports := map[string]int{}
	if exposePorts != "" {
		portPairs := strings.Split(exposePorts, ",")
		for _, exposePort := range portPairs {
			_, port, err := util.ParsePortMapping(exposePort)
			if err != nil {
				log.Warn().Err(err).Msgf("invalid port")
			} else {
				// TODO: assume port using http protocol for istio constraint, should support user-defined protocol
				name = fmt.Sprintf("http-%d", port)
				if n, exists := portNameDict[port]; exists {
					name = n
				}
				ports[name] = port
			}
		}
	}

	if opt.Store.Component == util.ComponentConnect && (opt.Get().Connect.ShareShadow || labels[util.KtSession] != "") {
		pod, generator, err2 := k.tryGetExistingShadows(&resourceMeta, &sshKeyMeta)
		if err2 != nil {
			return "", "", "", err2
		}
		if pod != nil && generator != nil {
			return pod.Status.PodIP, pod.Name, generator.PrivateKeyPath, nil
		}
	}

	podMeta := PodMetaAndSpec{
		Meta:          &resourceMeta,
		Image:         opt.Get().Global.Image,
		ContainerName: "kt-connect-shadow",
		Envs:          envs,
		Ports:         ports,
	}
	if labels[util.KtLane] != "" && labels[util.KtSession] != "" {
		enableExplicitLaneShadowTemplate(&podMeta)
	}
	return k.createShadow(&podMeta, &sshKeyMeta)
}

func (k *Kubernetes) createShadow(metaAndSpec *PodMetaAndSpec, sshKeyMeta *SSHkeyMeta) (
	podIP string, podName string, privateKeyPath string, err error) {

	generator, err := util.Generate(sshKeyMeta.PrivateKeyPath)
	if err != nil {
		return
	}

	configMap, err := k.createConfigMapWithSshKey(metaAndSpec.Meta.Labels, sshKeyMeta.SshConfigMapName, metaAndSpec.Meta.Namespace, generator)
	if err != nil {
		return
	}
	log.Info().Msgf("Successful create config map %v", configMap.Name)

	pod, err := k.createAndGetPod(metaAndSpec, sshKeyMeta.SshConfigMapName)
	if err != nil {
		return
	}
	return pod.Status.PodIP, pod.Name, generator.PrivateKeyPath, nil
}

func (k *Kubernetes) createAndGetPod(metaAndSpec *PodMetaAndSpec, sshcm string) (*coreV1.Pod, error) {
	if opt.Get().Global.UseShadowDeployment {
		if err := k.createShadowDeployment(metaAndSpec, sshcm); err != nil {
			return nil, err
		}
		log.Info().Msgf("Creating shadow deployment %s in namespace %s", metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace)
		delete(metaAndSpec.Meta.Labels, util.ControlBy)
		pods, err := k.WaitPodsReady(metaAndSpec.Meta.Labels, metaAndSpec.Meta.Namespace, opt.Get().Global.PodCreationTimeout)
		if err != nil {
			return nil, err
		}
		return &pods[0], nil
	} else {
		if err := k.createShadowPod(metaAndSpec, sshcm); err != nil {
			return nil, err
		}
		log.Info().Msgf("Deploying shadow pod %s in namespace %s", metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace)
		return k.WaitPodReady(metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace, opt.Get().Global.PodCreationTimeout)
	}
}

func filterRunningPods(pods []coreV1.Pod) []coreV1.Pod {
	runningPods := make([]coreV1.Pod, 0)
	for _, pod := range pods {
		if pod.Status.Phase == coreV1.PodRunning && pod.DeletionTimestamp == nil {
			runningPods = append(runningPods, pod)
		}
	}
	return runningPods
}

// createShadowDeployment create shadow deployment
func (k *Kubernetes) createShadowDeployment(metaAndSpec *PodMetaAndSpec, sshcm string) error {
	deployment := createDeployment(metaAndSpec)
	k.appendSshVolume(&deployment.Spec.Template.Spec, sshcm)
	if _, err := k.Clientset.AppsV1().Deployments(metaAndSpec.Meta.Namespace).
		Create(context.TODO(), deployment, metav1.CreateOptions{}); err != nil {
		return err
	}
	SetupHeartBeat(metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace, k.UpdateDeploymentHeartBeat)
	return nil
}

// createShadowPod create shadow pod
func (k *Kubernetes) createShadowPod(metaAndSpec *PodMetaAndSpec, sshcm string) error {
	pod := createPod(metaAndSpec)
	k.appendSshVolume(&pod.Spec, sshcm)
	if _, err := k.Clientset.CoreV1().Pods(metaAndSpec.Meta.Namespace).
		Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		return err
	}
	SetupHeartBeat(metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace, k.UpdatePodHeartBeat)
	return nil
}

func (k *Kubernetes) appendSshVolume(podSpec *coreV1.PodSpec, sshcm string) {
	podSpec.Containers[0].VolumeMounts = appendUniqueVolumeMounts(podSpec.Containers[0].VolumeMounts, coreV1.VolumeMount{
		Name:      "ssh-public-key",
		MountPath: fmt.Sprintf("/root/%s", util.SshAuthKey),
	})
	podSpec.Volumes = appendUniqueVolumes(podSpec.Volumes, getSSHVolume(sshcm))
}

func (k *Kubernetes) tryGetExistingShadows(resourceMeta *ResourceMeta, sshKeyMeta *SSHkeyMeta) (*coreV1.Pod, *util.SSHGenerator, error) {
	var app *appV1.Deployment
	var pod *coreV1.Pod
	if opt.Get().Global.UseShadowDeployment {
		app2, err := k.GetDeployment(resourceMeta.Name, resourceMeta.Namespace)
		if err != nil {
			// shared deployment not found is ok, return without error
			return nil, nil, nil
		}
		app = app2
		podList, err := k.GetPodsByLabel(app.Spec.Selector.MatchLabels, resourceMeta.Namespace)
		if err != nil || len(podList.Items) == 0 {
			log.Error().Err(err).Msgf("Found shadow deployment '%s' but cannot fetch it's pod", resourceMeta.Name)
			return nil, nil, err
		} else if len(podList.Items) > 1 {
			log.Warn().Msgf("Found more than one shadow pod with labels %v", app.Spec.Selector.MatchLabels)
			return nil, nil, err
		}
		pod = &podList.Items[0]
	} else {
		pod2, err := k.GetPod(resourceMeta.Name, resourceMeta.Namespace)
		if err != nil {
			// shared pod not found is ok, return without error
			return nil, nil, nil
		}
		pod = pod2
	}

	configMap, err := k.GetConfigMap(sshKeyMeta.SshConfigMapName, resourceMeta.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			if pod.DeletionTimestamp == nil {
				log.Error().Msgf("Found shadow pod without configmap. Please delete the pod '%s'", resourceMeta.Name)
			} else {
				_, err = k.WaitPodTerminate(resourceMeta.Name, resourceMeta.Namespace)
				if k8sErrors.IsNotFound(err) {
					// Pod already terminated
					return nil, nil, nil
				}
			}
		}
		return nil, nil, err
	}

	generator := util.NewSSHGenerator(configMap.Data[util.SshAuthPrivateKey], configMap.Data[util.SshAuthKey], sshKeyMeta.PrivateKeyPath)

	if err = util.WritePrivateKey(generator.PrivateKeyPath, []byte(configMap.Data[util.SshAuthPrivateKey])); err != nil {
		return nil, nil, err
	}

	if opt.Get().Global.UseShadowDeployment {
		log.Info().Msgf("Found shadow daemon deployment, reuse it")
		if err = k.IncreaseDeploymentRef(resourceMeta.Name, resourceMeta.Namespace); err != nil {
			return nil, nil, err
		}
	} else {
		log.Info().Msgf("Found shadow daemon pod, reuse it")
		if err = k.IncreasePodRef(resourceMeta.Name, resourceMeta.Namespace); err != nil {
			return nil, nil, err
		}
	}
	return pod, generator, nil
}

func getSSHVolume(volume string) coreV1.Volume {
	sshVolume := coreV1.Volume{
		Name: "ssh-public-key",
		VolumeSource: coreV1.VolumeSource{
			ConfigMap: &coreV1.ConfigMapVolumeSource{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: volume,
				},
				Items: []coreV1.KeyToPath{
					{
						Key:  util.SshAuthKey,
						Path: "authorized_keys",
					},
				},
			},
		},
	}
	return sshVolume
}

func enableExplicitLaneShadowTemplate(metaAndSpec *PodMetaAndSpec) {
	metaAndSpec.TemplateAnnotations = util.MergeMap(metaAndSpec.TemplateAnnotations, filterSidecarAnnotations(metaAndSpec.Meta.Annotations))
	metaAndSpec.MainContainerVolumeMounts = appendUniqueVolumeMounts(metaAndSpec.MainContainerVolumeMounts,
		coreV1.VolumeMount{
			Name:      laneShadowKeystoreVolumeName,
			MountPath: laneShadowKeystoreMountPath,
		},
		coreV1.VolumeMount{
			Name:      "ssh-public-key",
			MountPath: fmt.Sprintf("/root/%s", util.SshAuthKey),
		},
	)
	metaAndSpec.InitContainers = append(metaAndSpec.InitContainers, coreV1.Container{
		Name:    laneShadowKeystoreInitName,
		Image:   metaAndSpec.Image,
		Command: []string{"sh", "-c", fmt.Sprintf("mkdir -p %[1]s && cp /root/%[2]s %[1]s/authorized_keys", laneShadowKeystoreMountPath, util.SshAuthKey)},
		VolumeMounts: []coreV1.VolumeMount{
			{
				Name:      laneShadowKeystoreVolumeName,
				MountPath: laneShadowKeystoreMountPath,
			},
			{
				Name:      "ssh-public-key",
				MountPath: fmt.Sprintf("/root/%s", util.SshAuthKey),
			},
		},
	})
	metaAndSpec.Volumes = appendUniqueVolumes(metaAndSpec.Volumes, coreV1.Volume{
		Name: laneShadowKeystoreVolumeName,
		VolumeSource: coreV1.VolumeSource{
			EmptyDir: &coreV1.EmptyDirVolumeSource{},
		},
	})
}

func filterSidecarAnnotations(annotations map[string]string) map[string]string {
	sidecarAnnotations := map[string]string{}
	for key, value := range annotations {
		if strings.HasPrefix(key, "sidecar.istio.io/") {
			sidecarAnnotations[key] = value
		}
	}
	return sidecarAnnotations
}
