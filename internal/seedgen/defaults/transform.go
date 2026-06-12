// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"fmt"
	"net/netip"
	"slices"
	"sync/atomic"

	"github.com/podplane/seedgen/pkg/pipeline"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	firstGeneratedServiceIP = 2
	lastGeneratedServiceIP  = 254
)

var nextGeneratedServiceIP atomic.Uint64

func init() {
	nextGeneratedServiceIP.Store(firstGeneratedServiceIP)
}

// Transforms is the built-in Podplane seed transform table.
var Transforms = pipeline.Transforms{
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/certificates/", APIPrefix: "cert-manager.io/", Kind: "Certificate", JSONTransforms: []pipeline.JSONTransform{resetReadyCondition}},
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/issuers/", APIPrefix: "cert-manager.io/", Kind: "Issuer", JSONTransforms: []pipeline.JSONTransform{resetReadyCondition}},
	pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/clusterissuers/", APIPrefix: "cert-manager.io/", Kind: "ClusterIssuer", JSONTransforms: []pipeline.JSONTransform{resetReadyCondition}},
	pipeline.PrefixTransform{Prefix: "/registry/helm.toolkit.fluxcd.io/helmreleases/", APIPrefix: "helm.toolkit.fluxcd.io/", Kind: "HelmRelease", JSONTransforms: []pipeline.JSONTransform{resetReadyCondition}},
	pipeline.PrefixTransform{Prefix: "/registry/policy.cert-manager.io/certificaterequestpolicies/", APIPrefix: "policy.cert-manager.io/", Kind: "CertificateRequestPolicy", JSONTransforms: []pipeline.JSONTransform{resetReadyCondition}},
	pipeline.PrefixTransform{Prefix: "/registry/deployments/", APIVersion: "apps/v1", Kind: "Deployment", JSONTransforms: []pipeline.JSONTransform{resetAvailableCondition, normalizePodTemplateSpecImages}, ProtobufTransforms: []pipeline.ProtobufTransform{resetDeploymentAvailableCondition, normalizeTypedPodTemplateSpecImages}},
	pipeline.PrefixTransform{Prefix: "/registry/daemonsets/", APIVersion: "apps/v1", Kind: "DaemonSet", JSONTransforms: []pipeline.JSONTransform{resetDaemonSetAvailability, normalizePodTemplateSpecImages}, ProtobufTransforms: []pipeline.ProtobufTransform{resetTypedDaemonSetAvailability, normalizeTypedPodTemplateSpecImages}},
	pipeline.PrefixTransform{Prefix: "/registry/statefulsets/", APIVersion: "apps/v1", Kind: "StatefulSet", JSONTransforms: []pipeline.JSONTransform{normalizePodTemplateSpecImages}, ProtobufTransforms: []pipeline.ProtobufTransform{normalizeTypedPodTemplateSpecImages}},
	pipeline.PrefixTransform{Prefix: "/registry/services/specs/", APIVersion: "v1", Kind: "Service", JSONTransforms: []pipeline.JSONTransform{preferDualStackService, normalizeGeneratedServiceClusterIP}, ProtobufTransforms: []pipeline.ProtobufTransform{preferDualStackServiceObject, normalizeGeneratedServiceClusterIPObject}},
	pipeline.KeyTransform{Key: "/registry/services/specs/default/kubernetes", JSONTransforms: []pipeline.JSONTransform{setServiceDualStack("198.18.0.1", "fdc6::1")}, ProtobufTransforms: []pipeline.ProtobufTransform{setServiceDualStackObject("198.18.0.1", "fdc6::1")}},
	pipeline.KeyTransform{Key: "/registry/services/specs/platform-coredns/platform-coredns", JSONTransforms: []pipeline.JSONTransform{setServiceDualStack("198.19.255.254", "fdc6::ffff")}, ProtobufTransforms: []pipeline.ProtobufTransform{setServiceDualStackObject("198.19.255.254", "fdc6::ffff")}},
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

// normalizePodTemplateSpecImages normalizes image references under a workload's
// JSON pod template spec.
func normalizePodTemplateSpecImages(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false
	}
	template, ok := spec["template"].(map[string]any)
	if !ok {
		return false
	}
	podSpec, ok := template["spec"].(map[string]any)
	if !ok {
		return false
	}
	var changed bool
	for _, key := range []string{"initContainers", "containers", "ephemeralContainers"} {
		items, ok := podSpec[key].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			container, ok := item.(map[string]any)
			if !ok {
				continue
			}
			image, ok := container["image"].(string)
			if !ok || image == "" {
				continue
			}
			if normalized := pipeline.NormalizeImageRef(image); normalized != image {
				container["image"] = normalized
				changed = true
			}
		}
	}
	return changed
}

// normalizeTypedPodTemplateSpecImages normalizes image references under a typed
// workload pod template spec.
func normalizeTypedPodTemplateSpecImages(obj runtime.Object) bool {
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		return normalizeTypedPodSpecImages(&workload.Spec.Template.Spec)
	case *appsv1.DaemonSet:
		return normalizeTypedPodSpecImages(&workload.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		return normalizeTypedPodSpecImages(&workload.Spec.Template.Spec)
	default:
		return false
	}
}

// normalizeTypedPodSpecImages normalizes image references in typed pod spec
// container lists.
func normalizeTypedPodSpecImages(podSpec *corev1.PodSpec) bool {
	var changed bool
	for i := range podSpec.InitContainers {
		if normalized := pipeline.NormalizeImageRef(podSpec.InitContainers[i].Image); normalized != podSpec.InitContainers[i].Image {
			podSpec.InitContainers[i].Image = normalized
			changed = true
		}
	}
	for i := range podSpec.Containers {
		if normalized := pipeline.NormalizeImageRef(podSpec.Containers[i].Image); normalized != podSpec.Containers[i].Image {
			podSpec.Containers[i].Image = normalized
			changed = true
		}
	}
	for i := range podSpec.EphemeralContainers {
		if normalized := pipeline.NormalizeImageRef(podSpec.EphemeralContainers[i].Image); normalized != podSpec.EphemeralContainers[i].Image {
			podSpec.EphemeralContainers[i].Image = normalized
			changed = true
		}
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

// normalizeGeneratedServiceClusterIP replaces a generated JSON Service IPv4
// cluster IP with the next deterministic seed IP.
func normalizeGeneratedServiceClusterIP(obj map[string]any) bool {
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return false
	}
	namespace, _ := metadata["namespace"].(string)
	name, _ := metadata["name"].(string)
	if namespace == "default" && name == "kubernetes" {
		return false
	}
	if namespace == "platform-coredns" && name == "platform-coredns" {
		return false
	}
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return false
	}
	clusterIP, ok := spec["clusterIP"].(string)
	addr, err := netip.ParseAddr(clusterIP)
	if !ok || err != nil || !addr.Unmap().Is4() {
		return false
	}
	ip := nextGeneratedServiceClusterIP()
	if spec["clusterIP"] == ip && stringSliceEqual(spec["clusterIPs"], []string{ip}) {
		return false
	}
	spec["clusterIP"] = ip
	spec["clusterIPs"] = []string{ip}
	return true
}

// normalizeGeneratedServiceClusterIPObject replaces a generated typed Service
// IPv4 cluster IP with the next deterministic seed IP.
func normalizeGeneratedServiceClusterIPObject(obj runtime.Object) bool {
	service := obj.(*corev1.Service)
	if service.Namespace == "default" && service.Name == "kubernetes" {
		return false
	}
	if service.Namespace == "platform-coredns" && service.Name == "platform-coredns" {
		return false
	}
	addr, err := netip.ParseAddr(service.Spec.ClusterIP)
	if err != nil || !addr.Unmap().Is4() {
		return false
	}
	ip := nextGeneratedServiceClusterIP()
	if service.Spec.ClusterIP == ip && slices.Equal(service.Spec.ClusterIPs, []string{ip}) {
		return false
	}
	service.Spec.ClusterIP = ip
	service.Spec.ClusterIPs = []string{ip}
	return true
}

// nextGeneratedServiceClusterIP returns the next deterministic generated
// Service IPv4 address in the Podplane seed range.
func nextGeneratedServiceClusterIP() string {
	next := nextGeneratedServiceIP.Add(1) - 1
	if next > lastGeneratedServiceIP {
		panic(fmt.Sprintf("too many generated Service cluster IPs: exceeded 198.18.0.%d", lastGeneratedServiceIP))
	}
	return fmt.Sprintf("198.18.0.%d", next)
}

// setServiceDualStack returns a JSON Service transform for services whose
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

// setServiceDualStackObject returns a typed Service transform for services whose
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
