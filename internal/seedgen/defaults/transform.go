// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"slices"

	"github.com/podplane/seedgen/pkg/pipeline"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Transforms is the built-in Podplane seed transform table.
var Transforms = pipeline.Transforms{
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/certificates/", APIPrefix: "cert-manager.io/", Kind: "Certificate", MutateJSON: resetReadyCondition},
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/issuers/", APIPrefix: "cert-manager.io/", Kind: "Issuer", MutateJSON: resetReadyCondition},
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/clusterissuers/", APIPrefix: "cert-manager.io/", Kind: "ClusterIssuer", MutateJSON: resetReadyCondition},
	pipeline.PrefixTransform{Prefix: "/registry/helm.toolkit.fluxcd.io/helmreleases/", APIPrefix: "helm.toolkit.fluxcd.io/", Kind: "HelmRelease", MutateJSON: resetReadyCondition},
	pipeline.PrefixTransform{Prefix: "/registry/policy.cert-manager.io/certificaterequestpolicies/", APIPrefix: "policy.cert-manager.io/", Kind: "CertificateRequestPolicy", MutateJSON: resetReadyCondition},
	pipeline.PrefixTransform{Prefix: "/registry/deployments/", APIVersion: "apps/v1", Kind: "Deployment", MutateJSON: resetAvailableCondition, MutateProtobuf: resetDeploymentAvailableCondition},
	pipeline.PrefixTransform{Prefix: "/registry/daemonsets/", APIVersion: "apps/v1", Kind: "DaemonSet", MutateJSON: resetDaemonSetAvailability, MutateProtobuf: resetTypedDaemonSetAvailability},
	pipeline.PrefixTransform{Prefix: "/registry/services/specs/", APIVersion: "v1", Kind: "Service", MutateJSON: preferDualStackService, MutateProtobuf: preferDualStackServiceObject},
	pipeline.KeyTransform{Key: "/registry/services/specs/default/kubernetes", MutateJSON: setServiceDualStack("198.18.0.1", "fdc6::1"), MutateProtobuf: setServiceDualStackObject("198.18.0.1", "fdc6::1")},
	pipeline.KeyTransform{Key: "/registry/services/specs/platform-coredns/platform-coredns", MutateJSON: setServiceDualStack("198.19.255.254", "fdc6::ffff"), MutateProtobuf: setServiceDualStackObject("198.19.255.254", "fdc6::ffff")},
}

// resetReadyCondition marks a True Ready condition as False in a JSON object.
func resetReadyCondition(obj map[string]any) bool {
	return setConditionStatus(obj, "Ready", "True", "False")
}

// resetAvailableCondition marks a True Available condition as False in a JSON
// object.
func resetAvailableCondition(obj map[string]any) bool {
	return setConditionStatus(obj, "Available", "True", "False")
}

// resetDeploymentAvailableCondition marks live Deployment availability as not
// yet established so a bootstrapped cluster reconciles it from its own state.
func resetDeploymentAvailableCondition(obj runtime.Object) bool {
	deployment := obj.(*appsv1.Deployment)
	var changed bool
	for i := range deployment.Status.Conditions {
		condition := &deployment.Status.Conditions[i]
		if condition.Type != appsv1.DeploymentAvailable || condition.Status != corev1.ConditionTrue {
			continue
		}
		condition.Status = corev1.ConditionFalse
		changed = true
	}
	return changed
}

// setConditionStatus changes matching status conditions in a JSON object from
// one status value to another.
func setConditionStatus(obj map[string]any, conditionType, from, to string) bool {
	status, ok := obj["status"].(map[string]any)
	if !ok {
		return false
	}
	conditions, ok := status["conditions"].([]any)
	if !ok {
		return false
	}
	var changed bool
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok || condition["type"] != conditionType || condition["status"] != from {
			continue
		}
		condition["status"] = to
		changed = true
	}
	return changed
}

// resetDaemonSetAvailability clears live DaemonSet availability counters in a
// JSON object.
func resetDaemonSetAvailability(obj map[string]any) bool {
	status, ok := obj["status"].(map[string]any)
	if !ok {
		return false
	}
	var changed bool
	for _, key := range []string{"numberReady", "numberAvailable"} {
		if status[key] == nil || status[key] == float64(0) {
			continue
		}
		status[key] = float64(0)
		changed = true
	}
	return changed
}

// resetTypedDaemonSetAvailability clears live DaemonSet availability counters
// that should be recomputed after the seed is bootstrapped.
func resetTypedDaemonSetAvailability(obj runtime.Object) bool {
	daemonSet := obj.(*appsv1.DaemonSet)
	var changed bool
	if daemonSet.Status.NumberReady != 0 {
		daemonSet.Status.NumberReady = 0
		changed = true
	}
	if daemonSet.Status.NumberAvailable != 0 {
		daemonSet.Status.NumberAvailable = 0
		changed = true
	}
	return changed
}

// preferDualStackService mutates a JSON Service to prefer dual-stack networking
// while preserving its existing cluster IP allocation.
func preferDualStackService(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false
	}
	return setServiceDualStackFields(spec, "", "")
}

// preferDualStackServiceObject mutates a typed Service to prefer dual-stack
// networking while preserving its existing cluster IP allocation.
func preferDualStackServiceObject(obj runtime.Object) bool {
	service := obj.(*corev1.Service)
	return setTypedServiceDualStackFields(&service.Spec, "", "")
}

// setServiceDualStack returns a JSON Service mutation for services whose
// cluster IPs are part of the seed template.
func setServiceDualStack(ipv4, ipv6 string) func(map[string]any) bool {
	return func(obj map[string]any) bool {
		if obj["kind"] != "Service" || obj["apiVersion"] != "v1" {
			return false
		}
		spec, ok := obj["spec"].(map[string]any)
		if !ok {
			return false
		}
		return setServiceDualStackFields(spec, ipv4, ipv6)
	}
}

// setServiceDualStackObject returns a typed Service mutation for services whose
// cluster IPs are part of the seed template.
func setServiceDualStackObject(ipv4, ipv6 string) func(runtime.Object) bool {
	return func(obj runtime.Object) bool {
		service := obj.(*corev1.Service)
		return setTypedServiceDualStackFields(&service.Spec, ipv4, ipv6)
	}
}

// setServiceDualStackFields mutates a JSON Service spec toward the Podplane
// seed's preferred dual-stack shape.
func setServiceDualStackFields(spec map[string]any, ipv4, ipv6 string) bool {
	var changed bool
	if !stringSliceEqual(spec["ipFamilies"], []string{"IPv4", "IPv6"}) {
		spec["ipFamilies"] = []string{"IPv4", "IPv6"}
		changed = true
	}
	if spec["ipFamilyPolicy"] != "PreferDualStack" {
		spec["ipFamilyPolicy"] = "PreferDualStack"
		changed = true
	}
	if ipv4 != "" && spec["clusterIP"] != ipv4 {
		spec["clusterIP"] = ipv4
		changed = true
	}
	if ipv4 != "" {
		clusterIPs := []string{ipv4}
		if ipv6 != "" {
			clusterIPs = append(clusterIPs, ipv6)
		}
		if !stringSliceEqual(spec["clusterIPs"], clusterIPs) {
			spec["clusterIPs"] = clusterIPs
			changed = true
		}
	}
	return changed
}

// setTypedServiceDualStackFields mutates a Service spec toward the Podplane
// seed's preferred dual-stack shape.
func setTypedServiceDualStackFields(spec *corev1.ServiceSpec, ipv4, ipv6 string) bool {
	var changed bool
	wantFamilies := []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
	if !slices.Equal(spec.IPFamilies, wantFamilies) {
		spec.IPFamilies = wantFamilies
		changed = true
	}
	wantPolicy := corev1.IPFamilyPolicyPreferDualStack
	if spec.IPFamilyPolicy == nil || *spec.IPFamilyPolicy != wantPolicy {
		spec.IPFamilyPolicy = &wantPolicy
		changed = true
	}
	if ipv4 != "" && spec.ClusterIP != ipv4 {
		spec.ClusterIP = ipv4
		changed = true
	}
	if ipv4 != "" {
		clusterIPs := []string{ipv4}
		if ipv6 != "" {
			clusterIPs = append(clusterIPs, ipv6)
		}
		if !slices.Equal(spec.ClusterIPs, clusterIPs) {
			spec.ClusterIPs = clusterIPs
			changed = true
		}
	}
	return changed
}

// stringSliceEqual reports whether a JSON array value equals a string slice.
func stringSliceEqual(value any, want []string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	return slices.EqualFunc(items, want, func(item any, want string) bool { return item == want })
}
