// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"strings"

	"github.com/podplane/seedgen/pkg/pipeline"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const envoyGatewayDeploymentKey = "/registry/deployments/platform-envoy-gateway/envoy-gateway"

// agentRouterEnvoyGatewayDeploymentKey is Agent Router's dedicated Envoy
// Gateway controller, which carries the Agent Router extension.
const agentRouterEnvoyGatewayDeploymentKey = "/registry/deployments/platform-agent-router/envoy-gateway"

// AgenticTransforms adds shared platform bootstrap behavior.
var AgenticTransforms = append(append(pipeline.Transforms{}, Transforms...),
	pipeline.KeyTransform{Key: "/registry/deployments/platform-sourcegateway/sourcegateway", JSONTransforms: []pipeline.JSONTransform{scaleDeploymentToZero}, ProtobufTransforms: []pipeline.ProtobufTransform{scaleDeploymentToZeroObject}},
	pipeline.KeyTransform{Key: "/registry/deployments/platform-agent-router/agent-router", JSONTransforms: []pipeline.JSONTransform{scaleDeploymentToZero}, ProtobufTransforms: []pipeline.ProtobufTransform{scaleDeploymentToZeroObject}},
	pipeline.KeyTransform{Key: agentRouterEnvoyGatewayDeploymentKey, JSONTransforms: []pipeline.JSONTransform{scaleDeploymentToZero}, ProtobufTransforms: []pipeline.ProtobufTransform{scaleDeploymentToZeroObject}},
	pipeline.KeyTransform{Key: envoyGatewayDeploymentKey, JSONTransforms: []pipeline.JSONTransform{scaleDeploymentToZero}, ProtobufTransforms: []pipeline.ProtobufTransform{scaleDeploymentToZeroObject}},
)

// PlatformTransforms also removes agentic reconciliation and aggregated permissions.
var PlatformTransforms = platformTransforms()

func platformTransforms() pipeline.Transforms {
	out := append(pipeline.Transforms{}, AgenticTransforms...)
	out = append(out, pipeline.KeyTransform{Key: "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components", JSONTransforms: []pipeline.JSONTransform{removeAgenticValues}})
	for _, name := range []string{"admin", "edit", "view"} {
		out = append(out, pipeline.KeyTransform{Key: "/registry/clusterroles/" + name, JSONTransforms: []pipeline.JSONTransform{removeAgenticRules}, ProtobufTransforms: []pipeline.ProtobufTransform{removeAgenticRulesObject}})
	}
	return out
}

func removeAgenticValues(obj map[string]any) bool {
	current := obj
	for _, key := range []string{"spec", "values", "platform", "components"} {
		next, ok := current[key].(map[string]any)
		if !ok {
			return false
		}
		current = next
	}
	changed := false
	for _, section := range []string{"apps", "crds", "values"} {
		entries, _ := current[section].(map[string]any)
		for _, name := range []string{"agent-sandbox", "agent-sandbox-crds", "sourcegateway", "sourcegateway-crds", "agent-router", "agent-router-crds"} {
			if _, ok := entries[name]; ok {
				delete(entries, name)
				changed = true
			}
		}
	}
	return changed
}

func removeAgenticRules(obj map[string]any) bool {
	rules, ok := obj["rules"].([]any)
	if !ok {
		return false
	}
	out := make([]any, 0, len(rules))
	for _, item := range rules {
		rule, _ := item.(map[string]any)
		groups, _ := rule["apiGroups"].([]any)
		remove := false
		for _, group := range groups {
			if name, ok := group.(string); ok && agenticAPIGroup(name) {
				remove = true
			}
		}
		if !remove {
			out = append(out, item)
		}
	}
	if len(out) == len(rules) {
		return false
	}
	obj["rules"] = out
	return true
}

func removeAgenticRulesObject(obj runtime.Object) bool {
	role, ok := obj.(*rbacv1.ClusterRole)
	if !ok {
		return false
	}
	out := make([]rbacv1.PolicyRule, 0, len(role.Rules))
	for _, rule := range role.Rules {
		remove := false
		for _, group := range rule.APIGroups {
			if agenticAPIGroup(group) {
				remove = true
			}
		}
		if !remove {
			out = append(out, rule)
		}
	}
	if len(out) == len(role.Rules) {
		return false
	}
	role.Rules = out
	return true
}

func agenticAPIGroup(group string) bool {
	return group == "sourcegateway.dev" || group == "aigateway.envoyproxy.io" || group == "agents.x-k8s.io" || strings.HasSuffix(group, ".agents.x-k8s.io")
}

// scaleDeploymentToZero prevents a seeded Deployment from creating Pods before
// its runtime dependencies have been reconciled.
func scaleDeploymentToZero(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok || spec["replicas"] == float64(0) {
		return false
	}
	spec["replicas"] = float64(0)
	return true
}

// scaleDeploymentToZeroObject prevents a protobuf-decoded seeded Deployment
// from creating Pods before its runtime dependencies have been reconciled.
func scaleDeploymentToZeroObject(obj runtime.Object) bool {
	deployment := obj.(*appsv1.Deployment)
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == 0 {
		return false
	}
	replicas := int32(0)
	deployment.Spec.Replicas = &replicas
	return true
}
