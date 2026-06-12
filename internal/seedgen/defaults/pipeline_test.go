// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"encoding/json"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPipeline verifies the built-in pipeline can parse its rules and run a
// representative JSON transform.
func TestPipeline(t *testing.T) {
	t.Parallel()
	p := Pipeline()
	include, err := p.IncludeRules()
	if err != nil {
		t.Fatalf("IncludeRules: %v", err)
	}
	if len(include.Prefixes) == 0 {
		t.Fatal("default include rules should have prefixes")
	}
	exclude, err := p.ExcludeRules()
	if err != nil {
		t.Fatalf("ExcludeRules: %v", err)
	}
	if len(exclude.Prefixes) == 0 {
		t.Fatal("default exclude rules should have prefixes")
	}
	if !include.Matches("/registry/services/specs/default/kubernetes") {
		t.Fatal("default include rules should match the Kubernetes Service spec")
	}
	if !include.Matches("_netsy") {
		t.Fatal("default include rules should match Netsy's initial record")
	}
	if !include.Matches("/registry/ipaddresses/198.19.255.254") || !include.Matches("/registry/ipaddresses/fdc6::ffff") {
		t.Fatal("default include rules should match static service IPAddress records")
	}
	if include.Matches("/registry/ipaddresses/198.18.5.149") {
		t.Fatal("default include rules should not match generated IPAddress records")
	}
	if !exclude.Matches("/registry/events/default/example") {
		t.Fatal("default exclude rules should match Kubernetes events")
	}

	value := []byte(`{"apiVersion":"v1","kind":"Service","spec":{"clusterIP":"198.18.0.1","clusterIPs":["198.18.0.1"],"ipFamilies":["IPv4"],"ipFamilyPolicy":"SingleStack"}}`)
	got, err := p.Transforms.TransformValue([]byte("/registry/services/specs/default/kubernetes"), value)
	if err != nil {
		t.Fatalf("TransformValue: %v", err)
	}
	var service map[string]any
	if err := json.Unmarshal(got, &service); err != nil {
		t.Fatalf("decode transformed service: %v", err)
	}
	spec := service["spec"].(map[string]any)
	if spec["ipFamilyPolicy"] != "PreferDualStack" {
		t.Fatalf("ipFamilyPolicy = %v, want PreferDualStack", spec["ipFamilyPolicy"])
	}
	clusterIPs := spec["clusterIPs"].([]any)
	if len(clusterIPs) != 2 || clusterIPs[0] != "198.18.0.1" || clusterIPs[1] != "fdc6::1" {
		t.Fatalf("clusterIPs = %#v, want default IPv4 and IPv6", clusterIPs)
	}
}

func TestTransformsNormalizeWorkloadImages(t *testing.T) {
	t.Parallel()
	value := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"initContainers":[{"name":"init","image":"coredns/coredns:v1.12.1"}],"containers":[{"name":"app","image":"caddy:2"}],"ephemeralContainers":[{"name":"debug","image":"localhost/debug:latest"}]}}}}`)
	got, err := Transforms.TransformValue([]byte("/registry/deployments/platform-example/example"), value)
	if err != nil {
		t.Fatalf("TransformValue: %v", err)
	}
	var deployment map[string]any
	if err := json.Unmarshal(got, &deployment); err != nil {
		t.Fatalf("decode transformed deployment: %v", err)
	}
	podSpec := deployment["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
	if image := containerImage(t, podSpec, "initContainers", 0); image != "docker.io/coredns/coredns:v1.12.1" {
		t.Fatalf("init image = %q, want normalized Docker Hub repo", image)
	}
	if image := containerImage(t, podSpec, "containers", 0); image != "docker.io/library/caddy:2" {
		t.Fatalf("container image = %q, want normalized Docker Hub library repo", image)
	}
	if image := containerImage(t, podSpec, "ephemeralContainers", 0); image != "localhost/debug:latest" {
		t.Fatalf("ephemeral image = %q, want unchanged explicit localhost registry", image)
	}

	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "registry", Image: "registry:3"}},
		}}},
	}
	if !Transforms.TransformObject("/registry/statefulsets/platform-registry/platform-registry", statefulSet) {
		t.Fatalf("TransformObject reported no StatefulSet change")
	}
	if image := statefulSet.Spec.Template.Spec.Containers[0].Image; image != "docker.io/library/registry:3" {
		t.Fatalf("StatefulSet image = %q, want normalized Docker Hub library repo", image)
	}
}

func TestNormalizeServiceClusterIPs(t *testing.T) {
	nextGeneratedServiceIP.Store(firstGeneratedServiceIP)
	cases := []struct {
		key        string
		namespace  string
		name       string
		clusterIP  string
		wantIP     string
		wantIPList []string
	}{
		{"/registry/services/specs/platform-alpha/alpha", "platform-alpha", "alpha", "198.18.5.149", "198.18.0.2", []string{"198.18.0.2"}},
		{"/registry/services/specs/platform-zeta/zeta", "platform-zeta", "zeta", "198.19.202.64", "198.18.0.3", []string{"198.18.0.3"}},
		{"/registry/services/specs/default/kubernetes", "default", "kubernetes", "198.18.0.1", "198.18.0.1", []string{"198.18.0.1", "fdc6::1"}},
		{"/registry/services/specs/platform-headless/headless", "platform-headless", "headless", "None", "None", []string{"None"}},
		{"/registry/services/specs/platform-coredns/platform-coredns", "platform-coredns", "platform-coredns", "198.19.255.254", "198.19.255.254", []string{"198.19.255.254", "fdc6::ffff"}},
	}
	for _, tc := range cases {
		got, err := Transforms.TransformValue([]byte(tc.key), serviceValueFixture(tc.namespace, tc.name, tc.clusterIP))
		if err != nil {
			t.Fatalf("TransformValue(%s): %v", tc.key, err)
		}
		assertServiceClusterIP(t, got, tc.wantIP, tc.wantIPList)
	}
}

// containerImage returns one image field from a decoded JSON pod spec fixture.
func containerImage(t *testing.T, podSpec map[string]any, field string, index int) string {
	t.Helper()
	items := podSpec[field].([]any)
	container := items[index].(map[string]any)
	return container["image"].(string)
}

func serviceValueFixture(namespace, name, clusterIP string) []byte {
	return []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"namespace":"` + namespace + `","name":"` + name + `"},"spec":{"clusterIP":"` + clusterIP + `","clusterIPs":["` + clusterIP + `"],"ipFamilies":["IPv4","IPv6"],"ipFamilyPolicy":"PreferDualStack"}}`)
}

func assertServiceClusterIP(t *testing.T, value []byte, wantIP string, wantIPList []string) {
	t.Helper()
	var service map[string]any
	if err := json.Unmarshal(value, &service); err != nil {
		t.Fatalf("decode service: %v", err)
	}
	spec := service["spec"].(map[string]any)
	if got := spec["clusterIP"]; got != wantIP {
		t.Fatalf("clusterIP = %v, want %s", got, wantIP)
	}
	clusterIPs := spec["clusterIPs"].([]any)
	if len(clusterIPs) != len(wantIPList) {
		t.Fatalf("clusterIPs = %#v, want %#v", clusterIPs, wantIPList)
	}
	for i, want := range wantIPList {
		if clusterIPs[i] != want {
			t.Fatalf("clusterIPs = %#v, want %#v", clusterIPs, wantIPList)
		}
	}
}
