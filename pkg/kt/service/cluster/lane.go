package cluster

import (
	"context"
	"fmt"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var laneEnvoyFilterGVR = schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "envoyfilters"}

func (k *Kubernetes) ApplyLaneEnvoyFilter(name, namespace string, labels map[string]string, lane string) error {
	if lane == "" {
		return nil
	}
	if opt.Store.RestConfig == nil {
		return fmt.Errorf("rest config not initialized")
	}

	client, err := dynamic.NewForConfig(opt.Store.RestConfig)
	if err != nil {
		return fmt.Errorf("create dynamic client: %w", err)
	}

	selector := buildLaneWorkloadSelector(labels)
	if len(selector) == 0 {
		return fmt.Errorf("cannot build workload selector from labels")
	}

	obj := buildLaneEnvoyFilterObject(name, namespace, selector, lane)
	resource := client.Resource(laneEnvoyFilterGVR).Namespace(namespace)
	if _, err = resource.Create(context.TODO(), obj, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("Created lane envoy filter %s in namespace %s", name, namespace)
		return nil
	} else if !k8sErrors.IsAlreadyExists(err) {
		return fmt.Errorf("create lane envoy filter %s: %w", name, err)
	}

	current, getErr := resource.Get(context.TODO(), name, metav1.GetOptions{})
	if getErr != nil {
		return fmt.Errorf("get existing lane envoy filter %s: %w", name, getErr)
	}
	obj.SetResourceVersion(current.GetResourceVersion())
	if _, err = resource.Update(context.TODO(), obj, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update lane envoy filter %s: %w", name, err)
	}
	log.Info().Msgf("Updated lane envoy filter %s in namespace %s", name, namespace)
	return nil
}

func buildLaneWorkloadSelector(labels map[string]string) map[string]string {
	selector := map[string]string{}
	for _, key := range []string{util.KtRole, util.KtLane, util.KtSession, util.KtTarget, "sidecar.istio.io/inject"} {
		if value := labels[key]; value != "" {
			selector[key] = value
		}
	}
	return selector
}

func buildLaneEnvoyFilterObject(name, namespace string, selector map[string]string, lane string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "networking.istio.io/v1alpha3",
		"kind":       "EnvoyFilter",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]any{
				util.KtRole:    selector[util.KtRole],
				util.KtLane:    selector[util.KtLane],
				util.KtSession: selector[util.KtSession],
			},
		},
		"spec": map[string]any{
			"workloadSelector": map[string]any{
				"labels": selector,
			},
			"configPatches": []any{
				map[string]any{
					"applyTo": "HTTP_FILTER",
					"match": map[string]any{
						"context": "SIDECAR_OUTBOUND",
						"listener": map[string]any{
							"filterChain": map[string]any{
								"filter": map[string]any{
									"name": "envoy.filters.network.http_connection_manager",
									"subFilter": map[string]any{
										"name": "envoy.filters.http.router",
									},
								},
							},
						},
					},
					"patch": map[string]any{
						"operation": "INSERT_BEFORE",
						"value": map[string]any{
							"name": "envoy.filters.http.lua",
							"typed_config": map[string]any{
								"@type":      "type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua",
								"inlineCode": buildLaneLuaCode(lane),
							},
						},
					},
				},
			},
		},
	}}
}

func buildLaneLuaCode(lane string) string {
	return fmt.Sprintf(`function envoy_on_request(handle)
  local lane = %q
  local baggage = handle:headers():get("baggage")
  if baggage == nil or baggage == "" then
    handle:headers():replace("baggage", "lane=" .. lane)
    return
  end

  local members = {}
  for member in string.gmatch(baggage, '([^,]+)') do
    local trimmed = member:gsub("^%%s*(.-)%%s*$", "%%1")
    local key = trimmed:match("^([^=]+)=")
    if key ~= nil then
      key = key:gsub("^%%s*(.-)%%s*$", "%%1"):lower()
    end
    if trimmed ~= "" and key ~= "lane" then
      table.insert(members, trimmed)
    end
  end

  table.insert(members, "lane=" .. lane)
  handle:headers():replace("baggage", table.concat(members, ","))
end`, lane)
}
