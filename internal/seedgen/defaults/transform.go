// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

var transforms = []transform{
	objectTransform{apiPrefix: "cert-manager.io/", kind: "Certificate", mutate: resetReadyCondition},
	objectTransform{apiPrefix: "cert-manager.io/", kind: "Issuer", mutate: resetReadyCondition},
	objectTransform{apiPrefix: "cert-manager.io/", kind: "ClusterIssuer", mutate: resetReadyCondition},
	objectTransform{apiPrefix: "helm.toolkit.fluxcd.io/", kind: "HelmRelease", mutate: resetReadyCondition},
	objectTransform{apiPrefix: "policy.cert-manager.io/", kind: "CertificateRequestPolicy", mutate: resetReadyCondition},
	objectTransform{apiVersion: "apps/v1", kind: "Deployment", mutate: resetAvailableCondition},
	objectTransform{apiVersion: "apps/v1", kind: "DaemonSet", mutate: resetDaemonSetAvailability},
	objectTransform{apiVersion: "v1", kind: "Service", mutate: preferDualStackService},
	keyTransform{key: "/registry/services/specs/default/kubernetes", mutate: setServiceDualStack("198.18.0.1", "fdc6::1")},
	keyTransform{key: "/registry/services/specs/platform-coredns/platform-coredns", mutate: setServiceDualStack("198.19.255.254", "fdc6::ffff")},
}

func resetReadyCondition(obj map[string]any) bool {
	return setConditionStatus(obj, "Ready", "True", "False")
}

func resetAvailableCondition(obj map[string]any) bool {
	return setConditionStatus(obj, "Available", "True", "False")
}

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

func preferDualStackService(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false
	}
	return setServiceDualStackFields(spec, "", "")
}

func setServiceDualStack(ipv4, ipv6 string) func(map[string]any) bool {
	return func(obj map[string]any) bool {
		if obj["kind"] != "Service" || stringValue(obj["apiVersion"]) != "v1" {
			return false
		}
		spec, ok := obj["spec"].(map[string]any)
		if !ok {
			return false
		}
		return setServiceDualStackFields(spec, ipv4, ipv6)
	}
}

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

func stringSliceEqual(value any, want []string) bool {
	items, ok := value.([]any)
	if !ok || len(items) != len(want) {
		return false
	}
	for i, item := range items {
		if item != want[i] {
			return false
		}
	}
	return true
}
