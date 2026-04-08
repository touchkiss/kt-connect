package cluster

import (
	"strings"
	"testing"

	"github.com/alibaba/kt-connect/pkg/kt/util"
)

func TestBuildLaneWorkloadSelector(t *testing.T) {
	labels := map[string]string{
		util.KtRole:               util.RoleConnectShadow,
		util.KtLane:               "baseline",
		util.KtSession:            "session-1",
		"sidecar.istio.io/inject": "true",
		"control-by":              "kt",
	}

	selector := buildLaneWorkloadSelector(labels)
	if selector[util.KtRole] != util.RoleConnectShadow || selector[util.KtLane] != "baseline" || selector[util.KtSession] != "session-1" {
		t.Fatalf("unexpected selector: %#v", selector)
	}
	if selector["sidecar.istio.io/inject"] != "true" {
		t.Fatalf("expected sidecar inject in selector, got %#v", selector)
	}
	if selector["control-by"] != "" {
		t.Fatalf("unexpected noisy label leaked into selector: %#v", selector)
	}
}

func TestBuildLaneEnvoyFilterObject(t *testing.T) {
	selector := map[string]string{
		util.KtRole:               util.RoleConnectShadow,
		util.KtLane:               "baseline",
		util.KtSession:            "session-1",
		"sidecar.istio.io/inject": "true",
	}
	obj := buildLaneEnvoyFilterObject("kt-lane-session-1", "mixin", selector, "baseline")

	if obj.GetName() != "kt-lane-session-1" || obj.GetNamespace() != "mixin" {
		t.Fatalf("unexpected metadata: name=%s ns=%s", obj.GetName(), obj.GetNamespace())
	}

	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("missing spec: %#v", obj.Object)
	}
	workloadSelector, ok := spec["workloadSelector"].(map[string]any)
	if !ok {
		t.Fatalf("missing workloadSelector: %#v", spec)
	}
	labels, ok := workloadSelector["labels"].(map[string]string)
	if ok {
		if labels[util.KtLane] != "baseline" {
			t.Fatalf("unexpected selector labels: %#v", labels)
		}
	} else {
		labelsAny, ok2 := workloadSelector["labels"].(map[string]any)
		if !ok2 || labelsAny[util.KtLane] != "baseline" {
			t.Fatalf("unexpected selector labels: %#v", workloadSelector["labels"])
		}
	}

	patches, ok := spec["configPatches"].([]any)
	if !ok || len(patches) == 0 {
		t.Fatalf("missing configPatches: %#v", spec)
	}
	firstPatch := patches[0].(map[string]any)
	if firstPatch["applyTo"] != "HTTP_FILTER" {
		t.Fatalf("unexpected applyTo: %#v", firstPatch)
	}
	patch := firstPatch["patch"].(map[string]any)
	if patch["operation"] != "INSERT_BEFORE" {
		t.Fatalf("unexpected patch operation: %#v", patch)
	}
	value := patch["value"].(map[string]any)
	if value["name"] != "envoy.filters.http.lua" {
		t.Fatalf("unexpected filter name: %#v", value)
	}
	typedConfig := value["typed_config"].(map[string]any)
	inlineCode, _ := typedConfig["inlineCode"].(string)
	if typedConfig["@type"] != "type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua" {
		t.Fatalf("unexpected typed config type: %#v", typedConfig)
	}
	if inlineCode == "" || !containsAll(inlineCode, "baggage", "local lane = \"baseline\"", "table.insert(members, \"lane=\" .. lane)") {
		t.Fatalf("unexpected lua code: %s", inlineCode)
	}
}

func containsAll(text string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(text, sub) {
			return false
		}
	}
	return true
}
