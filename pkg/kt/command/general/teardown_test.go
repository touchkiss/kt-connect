package general

import (
	"fmt"
	"testing"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/service/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/util"
)

type cleanupClusterStub struct {
	cluster.KubernetesInterface
	deploymentsRemoved   []string
	configMapsRemoved    []string
	deploymentList       *appV1.DeploymentList
	configMapList        *coreV1.ConfigMapList
	deploymentRemoveErrs map[string]error
	configMapRemoveErrs  map[string]error
	labelsSeen           []map[string]string
}

func (s *cleanupClusterStub) GetDeploymentsByLabel(labels map[string]string, namespace string) (*appV1.DeploymentList, error) {
	s.labelsSeen = append(s.labelsSeen, cloneLabels(labels))
	return s.deploymentList, nil
}

func (s *cleanupClusterStub) GetConfigMapsByLabel(labels map[string]string, namespace string) (*coreV1.ConfigMapList, error) {
	s.labelsSeen = append(s.labelsSeen, cloneLabels(labels))
	return s.configMapList, nil
}

func (s *cleanupClusterStub) RemoveDeployment(name, namespace string) error {
	s.deploymentsRemoved = append(s.deploymentsRemoved, name)
	if s.deploymentRemoveErrs != nil {
		if err, ok := s.deploymentRemoveErrs[name]; ok {
			return err
		}
	}
	return nil
}

func (s *cleanupClusterStub) RemoveConfigMap(name, namespace string) error {
	s.configMapsRemoved = append(s.configMapsRemoved, name)
	if s.configMapRemoveErrs != nil {
		if err, ok := s.configMapRemoveErrs[name]; ok {
			return err
		}
	}
	return nil
}

func TestCleanShadowPodAndConfigMapRemovesLaneSessionResourcesByLabel(t *testing.T) {
	stub := &cleanupClusterStub{
		deploymentList: &appV1.DeploymentList{Items: []appV1.Deployment{{}, {}}},
		configMapList:  &coreV1.ConfigMapList{Items: []coreV1.ConfigMap{{}, {}}},
	}
	stub.deploymentList.Items[0].Name = "shadow-a"
	stub.deploymentList.Items[1].Name = "shadow-b"
	stub.configMapList.Items[0].Name = "shadow-a"
	stub.configMapList.Items[1].Name = "shadow-b"

	withCleanupStub(t, stub)
	withLaneSessionCleanupState(t)

	cleanShadowPodAndConfigMap()

	if len(stub.deploymentsRemoved) != 2 {
		t.Fatalf("expected 2 lane session deployments to be removed by label, got %v", stub.deploymentsRemoved)
	}
	if len(stub.configMapsRemoved) != 2 {
		t.Fatalf("expected 2 lane session configmaps to be removed by label, got %v", stub.configMapsRemoved)
	}
	assertLaneSessionLabels(t, stub.labelsSeen)
}

func TestCleanShadowPodAndConfigMapContinuesWhenLaneSessionCleanupFails(t *testing.T) {
	stub := &cleanupClusterStub{
		deploymentList: &appV1.DeploymentList{Items: []appV1.Deployment{{}, {}}},
		configMapList:  &coreV1.ConfigMapList{Items: []coreV1.ConfigMap{{}, {}}},
		deploymentRemoveErrs: map[string]error{
			"shadow-a": fmt.Errorf("deployment delete failed"),
		},
		configMapRemoveErrs: map[string]error{
			"shadow-a": fmt.Errorf("configmap delete failed"),
		},
	}
	stub.deploymentList.Items[0].Name = "shadow-a"
	stub.deploymentList.Items[1].Name = "shadow-b"
	stub.configMapList.Items[0].Name = "shadow-a"
	stub.configMapList.Items[1].Name = "shadow-b"

	withCleanupStub(t, stub)
	withLaneSessionCleanupState(t)

	cleanShadowPodAndConfigMap()

	if len(stub.deploymentsRemoved) != 2 {
		t.Fatalf("expected cleanup to continue across deployment delete failures, got %v", stub.deploymentsRemoved)
	}
	if len(stub.configMapsRemoved) != 2 {
		t.Fatalf("expected cleanup to continue across configmap delete failures, got %v", stub.configMapsRemoved)
	}
}

func withCleanupStub(t *testing.T, stub cluster.KubernetesInterface) {
	t.Helper()
	originalClusterIns := clusterIns
	clusterIns = func() cluster.KubernetesInterface {
		return stub
	}
	t.Cleanup(func() {
		clusterIns = originalClusterIns
	})
}

func withLaneSessionCleanupState(t *testing.T) {
	t.Helper()
	originalShadow := opt.Store.Shadow
	originalSession := opt.Store.Session
	originalComponent := opt.Store.Component
	originalStoreLane := opt.Store.Lane
	originalConnectLane := opt.Get().Connect.Lane
	originalUseShadowDeployment := opt.Get().Global.UseShadowDeployment
	t.Cleanup(func() {
		opt.Store.Shadow = originalShadow
		opt.Store.Session = originalSession
		opt.Store.Component = originalComponent
		opt.Store.Lane = originalStoreLane
		opt.Get().Connect.Lane = originalConnectLane
		opt.Get().Global.UseShadowDeployment = originalUseShadowDeployment
	})

	opt.Store.Shadow = ""
	opt.Store.Session = "session-1"
	opt.Store.Component = util.ComponentConnect
	opt.Store.Lane = "lane-a"
	opt.Get().Connect.Lane = ""
	opt.Get().Global.UseShadowDeployment = true
}

func assertLaneSessionLabels(t *testing.T, labelsSeen []map[string]string) {
	t.Helper()
	if len(labelsSeen) != 2 {
		t.Fatalf("expected 2 label lookups, got %d", len(labelsSeen))
	}
	for _, labels := range labelsSeen {
		if labels[util.KtRole] != util.RoleConnectShadow {
			t.Fatalf("expected %s label, got %v", util.KtRole, labels)
		}
		if labels[util.KtLane] != "lane-a" {
			t.Fatalf("expected %s label to use persisted lane, got %v", util.KtLane, labels)
		}
		if labels[util.KtSession] != "session-1" {
			t.Fatalf("expected %s label, got %v", util.KtSession, labels)
		}
	}
}

func cloneLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}
