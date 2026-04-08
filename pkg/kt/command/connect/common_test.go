package connect

import (
	"testing"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/service/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/util"
)

type shadowCreateStub struct {
	cluster.KubernetesInterface
	shadowName           string
	shadowLabels         map[string]string
	shadowAnnotations    map[string]string
	shadowEnvs           map[string]string
	envoyFilterName      string
	envoyFilterNamespace string
	envoyFilterLabels    map[string]string
	envoyFilterLane      string
}

func (s *shadowCreateStub) GetOrCreateShadow(name string, labels, annotations, envs map[string]string, portsToExpose string, portNameDict map[int]string) (string, string, string, error) {
	s.shadowName = name
	s.shadowLabels = cloneStringMap(labels)
	s.shadowAnnotations = cloneStringMap(annotations)
	s.shadowEnvs = cloneStringMap(envs)
	return "127.0.0.1", "shadow-pod", "/tmp/private.key", nil
}

func (s *shadowCreateStub) ApplyLaneEnvoyFilter(name, namespace string, labels map[string]string, lane string) error {
	s.envoyFilterName = name
	s.envoyFilterNamespace = namespace
	s.envoyFilterLabels = cloneStringMap(labels)
	s.envoyFilterLane = lane
	return nil
}

func TestGetEnvsIncludesLaneForShadowProxy(t *testing.T) {
	originalLane := opt.Get().Connect.Lane
	defer func() {
		opt.Get().Connect.Lane = originalLane
	}()

	opt.Get().Connect.Lane = "test-lane"
	envs := getEnvs()
	if envs["KT_LANE"] != "test-lane" {
		t.Fatalf("expected KT_LANE env, got %#v", envs)
	}
}

func TestGetOrCreateShadowUsesLaneSessionMetadata(t *testing.T) {
	stub := &shadowCreateStub{}
	originalClusterIns := clusterIns
	clusterIns = func() cluster.KubernetesInterface {
		return stub
	}
	defer func() {
		clusterIns = originalClusterIns
	}()

	originalNamespace := opt.Get().Global.Namespace
	originalLane := opt.Get().Connect.Lane
	originalSession := opt.Store.Session
	originalUseShadowDeployment := opt.Get().Global.UseShadowDeployment
	originalLaneEnvoyFilter := opt.Store.LaneEnvoyFilter
	defer func() {
		opt.Get().Global.Namespace = originalNamespace
		opt.Get().Connect.Lane = originalLane
		opt.Store.Session = originalSession
		opt.Get().Global.UseShadowDeployment = originalUseShadowDeployment
		opt.Store.LaneEnvoyFilter = originalLaneEnvoyFilter
	}()

	opt.Get().Global.Namespace = "test-ns"
	opt.Get().Connect.Lane = "test-lane"
	opt.Store.Session = "session-1"
	opt.Get().Global.UseShadowDeployment = true
	opt.Store.LaneEnvoyFilter = ""

	_, _, _, err := getOrCreateShadow()
	if err != nil {
		t.Fatalf("getOrCreateShadow returned error: %v", err)
	}
	if stub.shadowName != "kt-connect-shadow-test-ns-test-lane-session-1" {
		t.Fatalf("expected lane-aware shadow name, got %q", stub.shadowName)
	}
	if stub.shadowLabels[util.KtRole] != util.RoleConnectShadow || stub.shadowLabels[util.KtLane] != "test-lane" || stub.shadowLabels[util.KtSession] != "session-1" {
		t.Fatalf("expected lane session labels, got %#v", stub.shadowLabels)
	}
	if stub.shadowLabels["sidecar.istio.io/inject"] != "true" {
		t.Fatalf("expected sidecar inject label, got %#v", stub.shadowLabels)
	}
	if stub.shadowAnnotations["sidecar.istio.io/inject"] != "true" {
		t.Fatalf("expected istio inject annotation, got %#v", stub.shadowAnnotations)
	}
	if stub.shadowAnnotations["sidecar.istio.io/proxyCPU"] == "" ||
		stub.shadowAnnotations["sidecar.istio.io/proxyCPULimit"] == "" ||
		stub.shadowAnnotations["sidecar.istio.io/proxyMemory"] == "" ||
		stub.shadowAnnotations["sidecar.istio.io/proxyMemoryLimit"] == "" {
		t.Fatalf("expected sidecar resource annotations, got %#v", stub.shadowAnnotations)
	}
	if stub.shadowEnvs["KT_LANE"] != "test-lane" {
		t.Fatalf("expected lane env to be forwarded, got %#v", stub.shadowEnvs)
	}
	if stub.envoyFilterName != "kt-lane-session-1" || stub.envoyFilterNamespace != "test-ns" || stub.envoyFilterLane != "test-lane" {
		t.Fatalf("expected lane envoy filter to be applied, got name=%q namespace=%q lane=%q", stub.envoyFilterName, stub.envoyFilterNamespace, stub.envoyFilterLane)
	}
	if stub.envoyFilterLabels["sidecar.istio.io/inject"] != "true" {
		t.Fatalf("expected sidecar inject label on envoy filter selector, got %#v", stub.envoyFilterLabels)
	}
}

func TestBuildShadowAnnotationsForLane(t *testing.T) {
	annotations := buildShadowAnnotations("test-lane")
	if annotations["sidecar.istio.io/inject"] != "true" {
		t.Fatalf("expected istio inject annotation, got %#v", annotations)
	}
	if annotations["sidecar.istio.io/proxyCPU"] == "" ||
		annotations["sidecar.istio.io/proxyCPULimit"] == "" ||
		annotations["sidecar.istio.io/proxyMemory"] == "" ||
		annotations["sidecar.istio.io/proxyMemoryLimit"] == "" {
		t.Fatalf("expected sidecar resource annotations, got %#v", annotations)
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
