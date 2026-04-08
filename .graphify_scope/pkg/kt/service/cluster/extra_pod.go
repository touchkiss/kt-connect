package cluster

import (
	"context"
	"fmt"
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRouterPod create router pod
func (k *Kubernetes) CreateRouterPod(name string, labels, annotations map[string]string, ports map[int]int) (*coreV1.Pod, error) {
	targetPorts := map[string]int{}
	for _, remotePort := range ports {
		targetPorts[fmt.Sprintf("router-%d", remotePort)] = remotePort
	}
	metaAndSpec := &PodMetaAndSpec{
		Meta: &ResourceMeta{
			Name:        name,
			Namespace:   opt.Get().Global.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Image:         opt.Get().Mesh.RouterImage,
		ContainerName: util.DefaultContainer,
		Envs:          map[string]string{},
		Ports:         targetPorts,
		IsLeaf:        true,
	}
	pod := createPod(metaAndSpec)
	if _, err := k.Clientset.CoreV1().Pods(metaAndSpec.Meta.Namespace).
		Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	SetupHeartBeat(metaAndSpec.Meta.Name, metaAndSpec.Meta.Namespace, k.UpdatePodHeartBeat)
	log.Info().Msgf("Router pod %s created", name)
	return k.WaitPodReady(name, opt.Get().Global.Namespace, opt.Get().Global.PodCreationTimeout)
}

// CreateRectifierPod create pod for rectify time difference
func (k *Kubernetes) CreateRectifierPod(name string) (*coreV1.Pod, error) {
	metaAndSpec := &PodMetaAndSpec{
		Meta: &ResourceMeta{
			Name:        name,
			Namespace:   opt.Get().Global.Namespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Image:         opt.Get().Global.Image,
		ContainerName: util.DefaultContainer,
		Envs:          map[string]string{},
		Ports:         map[string]int{},
		IsLeaf:        true,
	}
	pod := createPod(metaAndSpec)
	pod.Spec.Containers[0].Command = []string{"tail", "-f", "/dev/null"}
	if _, err := k.Clientset.CoreV1().Pods(metaAndSpec.Meta.Namespace).
		Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	log.Debug().Msgf("Rectify pod %s created", name)
	return k.WaitPodReady(name, opt.Get().Global.Namespace, opt.Get().Global.PodCreationTimeout)
}
