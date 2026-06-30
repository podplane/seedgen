// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"github.com/podplane/seedgen/pkg/pipeline"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MinimalTransforms extends the built-in transforms with minimal-profile
// cleanup for records that intentionally overlap with the recommended profile.
var MinimalTransforms = append(append(pipeline.Transforms{}, Transforms...),
	pipeline.KeyTransform{Key: "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components", JSONTransforms: []pipeline.JSONTransform{resetPlatformComponentsValues}},
	pipeline.KeyTransform{Key: "/registry/ranges/servicenodeports", JSONTransforms: []pipeline.JSONTransform{resetServiceNodePortsRange}},
	pipeline.KeyTransform{Key: "/registry/clusterroles/admin", JSONTransforms: []pipeline.JSONTransform{removeAddonAggregatedRBACRules}, ProtobufTransforms: []pipeline.ProtobufTransform{removeAddonAggregatedRBACRulesObject}},
	pipeline.KeyTransform{Key: "/registry/clusterroles/edit", JSONTransforms: []pipeline.JSONTransform{removeAddonAggregatedRBACRules}, ProtobufTransforms: []pipeline.ProtobufTransform{removeAddonAggregatedRBACRulesObject}},
	pipeline.KeyTransform{Key: "/registry/clusterroles/view", JSONTransforms: []pipeline.JSONTransform{removeAddonAggregatedRBACRules}, ProtobufTransforms: []pipeline.ProtobufTransform{removeAddonAggregatedRBACRulesObject}},
)

// minimalAddonRBACAPIGroups lists recommended-only API groups whose aggregated
// RBAC rules should be removed from default ClusterRoles in minimal seeds.
var minimalAddonRBACAPIGroups = map[string]struct{}{
	"acme.cert-manager.io":       {},
	"cert-manager.io":            {},
	"secrets-api.podplane.dev":   {},
	"secrets.podplane.dev":       {},
	"secrets-store.csi.x-k8s.io": {},
}

// resetPlatformComponentsValues removes recommended addon values from the
// platform-components HelmRelease so the chart reconciles with its core-only
// defaults in minimal seeds.
func resetPlatformComponentsValues(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := spec["values"]; !ok {
		return false
	}
	delete(spec, "values")
	return true
}

// resetServiceNodePortsRange clears stale NodePort allocator bits that may have
// been consumed by recommended-only LoadBalancer/NodePort Services.
func resetServiceNodePortsRange(obj map[string]any) bool {
	if obj["data"] == "" {
		return false
	}
	obj["data"] = ""
	return true
}

// removeAddonAggregatedRBACRules removes cert-manager/acme rules aggregated
// into Kubernetes' default admin/edit/view ClusterRoles by recommended addons.
func removeAddonAggregatedRBACRules(obj map[string]any) bool {
	rules, ok := obj["rules"].([]any)
	if !ok {
		return false
	}
	out := make([]any, 0, len(rules))
	var changed bool
	for _, rule := range rules {
		if ruleHasAddonAPIGroup(rule) {
			changed = true
			continue
		}
		out = append(out, rule)
	}
	if !changed {
		return false
	}
	obj["rules"] = out
	return true
}

// removeAddonAggregatedRBACRulesObject removes recommended-only addon RBAC
// rules from protobuf-decoded Kubernetes ClusterRole records.
func removeAddonAggregatedRBACRulesObject(obj runtime.Object) bool {
	clusterRole, ok := obj.(*rbacv1.ClusterRole)
	if !ok {
		return false
	}
	out := make([]rbacv1.PolicyRule, 0, len(clusterRole.Rules))
	var changed bool
	for _, rule := range clusterRole.Rules {
		remove := false
		for _, apiGroup := range rule.APIGroups {
			if _, ok := minimalAddonRBACAPIGroups[apiGroup]; ok {
				remove = true
				break
			}
		}
		if remove {
			changed = true
			continue
		}
		out = append(out, rule)
	}
	if !changed {
		return false
	}
	clusterRole.Rules = out
	return true
}

// ruleHasAddonAPIGroup reports whether an RBAC rule grants access to an addon
// API group that should not be present in the minimal profile.
func ruleHasAddonAPIGroup(rule any) bool {
	ruleMap, ok := rule.(map[string]any)
	if !ok {
		return false
	}
	apiGroups, ok := ruleMap["apiGroups"].([]any)
	if !ok {
		return false
	}
	for _, item := range apiGroups {
		apiGroup, ok := item.(string)
		if !ok {
			continue
		}
		if _, ok := minimalAddonRBACAPIGroups[apiGroup]; ok {
			return true
		}
	}
	return false
}
