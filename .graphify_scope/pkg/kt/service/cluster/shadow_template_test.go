package cluster

import (
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	coreV1 "k8s.io/api/core/v1"
	"testing"
)

func TestCreatePodKeepsLegacySingleContainerDefaults(t *testing.T) {
	originalServiceAccount := opt.Get().Global.ServiceAccount
	defer func() {
		opt.Get().Global.ServiceAccount = originalServiceAccount
	}()

	opt.Get().Global.ServiceAccount = "legacy-sa"

	pod := createPod(&PodMetaAndSpec{
		Meta: &ResourceMeta{
			Name:        "shadow",
			Namespace:   "default",
			Labels:      map[string]string{"app": "shadow"},
			Annotations: map[string]string{},
		},
		Image:         "shadow:latest",
		ContainerName: "kt-connect-shadow",
		Envs:          map[string]string{"A": "B"},
		Ports:         map[string]int{"http": 8080},
	})

	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected single container, got %d", len(pod.Spec.Containers))
	}
	if len(pod.Spec.InitContainers) != 0 {
		t.Fatalf("expected no init containers, got %#v", pod.Spec.InitContainers)
	}
	if len(pod.Spec.Volumes) != 0 {
		t.Fatalf("expected no extra volumes, got %#v", pod.Spec.Volumes)
	}
	if pod.Spec.Containers[0].Name != "kt-connect-shadow" {
		t.Fatalf("expected main container name preserved, got %q", pod.Spec.Containers[0].Name)
	}
	if pod.Spec.ServiceAccountName != "legacy-sa" {
		t.Fatalf("expected service account to stay unchanged, got %q", pod.Spec.ServiceAccountName)
	}
}

func TestCreatePodBuildsExplicitLaneTemplate(t *testing.T) {
	pod := createPod(&PodMetaAndSpec{
		Meta: &ResourceMeta{
			Name:      "shadow",
			Namespace: "default",
			Labels: map[string]string{
				util.KtRole:    util.RoleConnectShadow,
				util.KtLane:    "blue",
				util.KtSession: "session-1",
			},
			Annotations: map[string]string{},
		},
		Image:         "shadow:latest",
		ContainerName: "kt-connect-shadow",
		Envs:          map[string]string{"KT_LANE": "blue"},
		Ports:         map[string]int{"http": 8080},
		TemplateAnnotations: map[string]string{
			"sidecar.istio.io/inject":           "true",
			"sidecar.istio.io/proxyCPU":         "100m",
			"sidecar.istio.io/proxyCPULimit":    "200m",
			"sidecar.istio.io/proxyMemory":      "128Mi",
			"sidecar.istio.io/proxyMemoryLimit": "256Mi",
		},
		MainContainerVolumeMounts: []coreV1.VolumeMount{
			{
				Name:      "keystore",
				MountPath: "/var/kt-connect/keystore",
			},
			{
				Name:      "ssh-public-key",
				MountPath: "/root/.ssh/authorized_keys",
			},
		},
		InitContainers: []coreV1.Container{
			{
				Name:    "kt-shadow-keystore-init",
				Image:   "shadow:latest",
				Command: []string{"sh", "-c", "echo init"},
				VolumeMounts: []coreV1.VolumeMount{
					{
						Name:      "keystore",
						MountPath: "/var/kt-connect/keystore",
					},
					{
						Name:      "ssh-public-key",
						MountPath: "/root/.ssh/authorized_keys",
					},
				},
			},
		},
		Volumes: []coreV1.Volume{
			{
				Name: "keystore",
				VolumeSource: coreV1.VolumeSource{
					EmptyDir: &coreV1.EmptyDirVolumeSource{},
				},
			},
		},
	})

	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected a single main container, got %d", len(pod.Spec.Containers))
	}
	if pod.Spec.Containers[0].Name != "kt-connect-shadow" {
		t.Fatalf("expected main container name, got %q", pod.Spec.Containers[0].Name)
	}
	if len(pod.Spec.InitContainers) != 1 || pod.Spec.InitContainers[0].Name != "kt-shadow-keystore-init" {
		t.Fatalf("expected keystore init fragment, got %#v", pod.Spec.InitContainers)
	}
	if got := pod.Annotations["sidecar.istio.io/inject"]; got != "true" {
		t.Fatalf("expected sidecar injection annotation, got %q", got)
	}
	if got := pod.Annotations["sidecar.istio.io/proxyMemoryLimit"]; got != "256Mi" {
		t.Fatalf("expected sidecar resource annotation, got %q", got)
	}
	if len(pod.Spec.Volumes) != 1 || pod.Spec.Volumes[0].Name != "keystore" {
		t.Fatalf("expected keystore volume fragment, got %#v", pod.Spec.Volumes)
	}
	if len(pod.Spec.Containers[0].VolumeMounts) != 2 {
		t.Fatalf("expected explicit volume mount fragments, got %#v", pod.Spec.Containers[0].VolumeMounts)
	}
}
