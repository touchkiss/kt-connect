package cluster

import (
	"fmt"
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubernetesClient(kubeConfig string) (clientset *kubernetes.Clientset, err error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}
	clientset, err = kubernetes.NewForConfig(config)
	return
}

func createService(metaAndSpec *SvcMetaAndSpec) *coreV1.Service {
	var servicePorts []coreV1.ServicePort
	metaAndSpec.Meta.Annotations = util.MapPut(metaAndSpec.Meta.Annotations, util.KtLastHeartBeat, util.GetTimestamp())
	metaAndSpec.Meta.Labels = util.MergeMap(metaAndSpec.Meta.Labels, map[string]string{util.ControlBy: util.KubernetesToolkit})

	for srcPort, targetPort := range metaAndSpec.Ports {
		servicePorts = append(servicePorts, coreV1.ServicePort{
			Name:       fmt.Sprintf("kt-%d", srcPort),
			Port:       int32(srcPort),
			TargetPort: intstr.FromInt(targetPort),
		})
	}

	service := &coreV1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        metaAndSpec.Meta.Name,
			Namespace:   metaAndSpec.Meta.Namespace,
			Labels:      metaAndSpec.Meta.Labels,
			Annotations: metaAndSpec.Meta.Annotations,
		},
		Spec: coreV1.ServiceSpec{
			Selector: metaAndSpec.Selectors,
			Type:     coreV1.ServiceTypeClusterIP,
			Ports:    servicePorts,
		},
	}
	if metaAndSpec.External {
		service.Spec.Type = coreV1.ServiceTypeLoadBalancer
	}
	return service
}

func createDeployment(metaAndSpec *PodMetaAndSpec) *appV1.Deployment {
	metaAndSpec.Meta.Annotations = util.MapPut(metaAndSpec.Meta.Annotations, util.KtRefCount, "1")
	metaAndSpec.Meta.Annotations = util.MapPut(metaAndSpec.Meta.Annotations, util.KtLastHeartBeat, util.GetTimestamp())
	podAnnotations := util.MergeMap(metaAndSpec.Meta.Annotations, metaAndSpec.TemplateAnnotations)

	var originLabels = make(map[string]string, 0)
	for k, v := range metaAndSpec.Meta.Labels {
		originLabels[k] = v
	}
	metaAndSpec.Meta.Labels = util.MergeMap(metaAndSpec.Meta.Labels, map[string]string{util.ControlBy: util.KubernetesToolkit})

	return &appV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        metaAndSpec.Meta.Name,
			Namespace:   metaAndSpec.Meta.Namespace,
			Labels:      metaAndSpec.Meta.Labels,
			Annotations: podAnnotations,
		},
		Spec: appV1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: originLabels,
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      originLabels,
					Annotations: podAnnotations,
				},
				Spec: createPodSpec(metaAndSpec),
			},
		},
	}
}

func createPod(metaAndSpec *PodMetaAndSpec) *coreV1.Pod {
	metaAndSpec.Meta.Annotations = util.MapPut(metaAndSpec.Meta.Annotations, util.KtRefCount, "1")
	metaAndSpec.Meta.Annotations = util.MapPut(metaAndSpec.Meta.Annotations, util.KtLastHeartBeat, util.GetTimestamp())
	metaAndSpec.Meta.Labels = util.MergeMap(metaAndSpec.Meta.Labels, map[string]string{util.ControlBy: util.KubernetesToolkit})
	podAnnotations := util.MergeMap(metaAndSpec.Meta.Annotations, metaAndSpec.TemplateAnnotations)

	pod := &coreV1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        metaAndSpec.Meta.Name,
			Namespace:   metaAndSpec.Meta.Namespace,
			Labels:      metaAndSpec.Meta.Labels,
			Annotations: podAnnotations,
		},
		Spec: createPodSpec(metaAndSpec),
	}

	if opt.Get().Global.ImagePullSecret != "" {
		addImagePullSecret(pod, opt.Get().Global.ImagePullSecret)
	}

	if opt.Get().Global.NodeSelector != "" {
		pod.Spec.NodeSelector = util.String2Map(opt.Get().Global.NodeSelector)
	}

	return pod
}

func createPodSpec(metaAndSpec *PodMetaAndSpec) coreV1.PodSpec {
	mainContainer := createContainer(metaAndSpec.ContainerName, metaAndSpec.Image, []string{}, metaAndSpec.Envs, metaAndSpec.Ports)
	if len(metaAndSpec.MainContainerCommand) > 0 {
		mainContainer.Command = metaAndSpec.MainContainerCommand
	}
	if len(metaAndSpec.MainContainerArgs) > 0 {
		mainContainer.Args = metaAndSpec.MainContainerArgs
	}
	mainContainer.VolumeMounts = appendUniqueVolumeMounts(mainContainer.VolumeMounts, metaAndSpec.MainContainerVolumeMounts...)

	containers := []coreV1.Container{mainContainer}
	containers = append(containers, metaAndSpec.AdditionalContainers...)

	spec := coreV1.PodSpec{
		ServiceAccountName: opt.Get().Global.ServiceAccount,
		Containers:         containers,
		InitContainers:     append([]coreV1.Container{}, metaAndSpec.InitContainers...),
		Volumes:            append([]coreV1.Volume{}, metaAndSpec.Volumes...),
	}
	return spec
}

func appendUniqueVolumeMounts(base []coreV1.VolumeMount, mounts ...coreV1.VolumeMount) []coreV1.VolumeMount {
	result := append([]coreV1.VolumeMount{}, base...)
	for _, mount := range mounts {
		exists := false
		for i, current := range result {
			if current.Name == mount.Name && current.MountPath == mount.MountPath {
				result[i] = mount
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, mount)
		}
	}
	return result
}

func appendUniqueVolumes(base []coreV1.Volume, volumes ...coreV1.Volume) []coreV1.Volume {
	result := append([]coreV1.Volume{}, base...)
	for _, volume := range volumes {
		exists := false
		for i, current := range result {
			if current.Name == volume.Name {
				result[i] = volume
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, volume)
		}
	}
	return result
}

func createContainer(name, image string, args []string, envs map[string]string, ports map[string]int) coreV1.Container {
	var envVar []coreV1.EnvVar
	for k, v := range envs {
		envVar = append(envVar, coreV1.EnvVar{Name: k, Value: v})
	}
	var pullPolicy coreV1.PullPolicy
	if opt.Get().Global.ForceUpdate {
		pullPolicy = "Always"
	} else {
		pullPolicy = "IfNotPresent"
	}
	container := coreV1.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: pullPolicy,
		Args:            args,
		Env:             envVar,
		SecurityContext: &coreV1.SecurityContext{
			Capabilities: &coreV1.Capabilities{
				Add: []coreV1.Capability{
					"AUDIT_WRITE",
				},
			},
		},
		Ports: []coreV1.ContainerPort{},
		Resources: coreV1.ResourceRequirements{
			Limits:   coreV1.ResourceList{},
			Requests: coreV1.ResourceList{},
		},
	}
	if opt.Get().Global.PodQuota != "" {
		addResourceLimit(&container, opt.Get().Global.PodQuota)
	}
	for name, port := range ports {
		container.Ports = append(container.Ports, coreV1.ContainerPort{
			Name:          name,
			Protocol:      coreV1.ProtocolTCP,
			ContainerPort: int32(port),
		})
	}
	return container
}
