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
	include, err := p.IncludeRules("none")
	if err != nil {
		t.Fatalf("IncludeRules(none): %v", err)
	}
	if len(include.Prefixes) == 0 {
		t.Fatal("default include rules should have prefixes")
	}
	exclude, err := p.ExcludeRules("none")
	if err != nil {
		t.Fatalf("ExcludeRules(none): %v", err)
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

	minimalInclude, err := p.IncludeRules("minimal")
	if err != nil {
		t.Fatalf("minimal IncludeRules: %v", err)
	}
	minimalExclude, err := p.ExcludeRules("minimal")
	if err != nil {
		t.Fatalf("minimal ExcludeRules: %v", err)
	}
	if !minimalInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd") {
		t.Fatal("minimal include rules should match core Flux HelmRelease")
	}
	if minimalInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cert-manager/cert-manager") {
		t.Fatal("minimal include rules should not match addon HelmRelease")
	}
	if !minimalExclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cert-manager/cert-manager") {
		t.Fatal("minimal exclude rules should drop recommended addon Flux resources")
	}
	recommendedInclude, err := p.IncludeRules("recommended")
	if err != nil {
		t.Fatalf("recommended IncludeRules: %v", err)
	}
	recommendedExclude, err := p.ExcludeRules("recommended")
	if err != nil {
		t.Fatalf("recommended ExcludeRules: %v", err)
	}
	if !recommendedInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cert-manager/cert-manager") {
		t.Fatal("recommended include rules should match addon HelmRelease")
	}
	if !recommendedInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-podplane-operator/podplane-operator") {
		t.Fatal("recommended include rules should match podplane-operator HelmRelease")
	}
	if !recommendedInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-secrets-store-csi-driver/secrets-store-csi-driver") {
		t.Fatal("recommended include rules should match Secrets Store CSI Driver HelmRelease")
	}
	if !recommendedInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-secrets-store-csi-provider-openbao/secrets-store-csi-provider-openbao") {
		t.Fatal("recommended include rules should match Secrets Store CSI provider HelmReleases")
	}
	if !recommendedInclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-zot-registry/zot-registry") {
		t.Fatal("recommended include rules should match zot-registry HelmRelease")
	}
	if !recommendedInclude.Matches("/registry/secrets.podplane.dev/secretproviderbindings/platform-aok/aok-source-controller") {
		t.Fatal("recommended include rules should match SecretProviderBinding records")
	}
	if !recommendedInclude.Matches("/registry/secrets-api.podplane.dev/secretproviderkeyspaces/platform-aok/aws-secrets-manager.aok-source-controller") {
		t.Fatal("recommended include rules should match SecretProviderKeyspace records")
	}
	if recommendedExclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cert-manager/cert-manager") {
		t.Fatal("recommended exclude rules should not inherit minimal addon excludes")
	}
	if !minimalExclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-podplane-operator/podplane-operator") {
		t.Fatal("minimal exclude rules should drop podplane-operator HelmRelease")
	}
	if !minimalExclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-secrets-store-csi-provider-openbao/secrets-store-csi-provider-openbao") {
		t.Fatal("minimal exclude rules should drop Secrets Store CSI provider HelmReleases")
	}
	if !minimalExclude.Matches("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-zot-registry/zot-registry") {
		t.Fatal("minimal exclude rules should drop zot-registry HelmRelease")
	}
	if !minimalExclude.Matches("/registry/secrets-store.csi.x-k8s.io/secretproviderclasses/platform-aok/aok-source-controller") {
		t.Fatal("minimal exclude rules should drop generated SecretProviderClass records")
	}
	if !minimalExclude.Matches("/registry/secrets.podplane.dev/secretproviderbindings/platform-aok/aok-source-controller") {
		t.Fatal("minimal exclude rules should drop SecretProviderBinding records")
	}
	if !minimalExclude.Matches("/registry/secrets-api.podplane.dev/secretproviderkeyspaces/platform-aok/aws-secrets-manager.aok-source-controller") {
		t.Fatal("minimal exclude rules should drop SecretProviderKeyspace records")
	}
	if !minimalExclude.Matches("/registry/apiregistration.k8s.io/apiservices/v1beta1.secrets-api.podplane.dev") {
		t.Fatal("minimal exclude rules should drop Podplane secrets APIService records")
	}
	if !minimalExclude.Matches("/registry/validatingadmissionpolicies/platform-podplane-operator-spc-restriction-vap") {
		t.Fatal("minimal exclude rules should drop pod-side SecretProviderClass admission policy records")
	}
	if !minimalExclude.Matches("/registry/clusterroles/platform-secrets-store-csi-provider-openbao-csi-provider-clusterrole") {
		t.Fatal("minimal exclude rules should drop Secrets Store CSI provider ClusterRoles")
	}
	if !minimalExclude.Matches("/registry/clusterrolebindings/platform-secrets-store-csi-provider-openbao-csi-provider-clusterrolebinding") {
		t.Fatal("minimal exclude rules should drop Secrets Store CSI provider ClusterRoleBindings")
	}
	if !minimalExclude.Matches("/registry/csidrivers/secrets-store.csi.k8s.io") {
		t.Fatal("minimal exclude rules should drop Secrets Store CSI Driver CSIDriver records")
	}
	if !minimalExclude.Matches("/registry/validatingadmissionpolicies/platform-deny-editor-shell-vap") {
		t.Fatal("minimal exclude rules should drop editor shell admission policies")
	}
	if !minimalExclude.Matches("/registry/validatingadmissionpolicybindings/platform-deny-editor-shell-vapb") {
		t.Fatal("minimal exclude rules should drop editor shell admission policy bindings")
	}

	value := []byte(`{"apiVersion":"v1","kind":"Service","spec":{"clusterIP":"198.18.0.1","clusterIPs":["198.18.0.1"],"ipFamilies":["IPv4"],"ipFamilyPolicy":"SingleStack"}}`)
	got, err := p.Transforms("none").TransformValue([]byte("/registry/services/specs/default/kubernetes"), value)
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

func TestTransformsRemoveComponentsGitSecretRef(t *testing.T) {
	t.Parallel()

	value := []byte(`{"apiVersion":"source.toolkit.fluxcd.io/v1","kind":"GitRepository","metadata":{"name":"podplane-components"},"spec":{"url":"https://github.com/podplane/components.git","secretRef":{"name":"podplane-components-git"},"ref":{"branch":"main"}}}`)
	got, err := Transforms.TransformValue([]byte("/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components"), value)
	if err != nil {
		t.Fatalf("TransformValue(GitRepository): %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed GitRepository: %v", err)
	}
	spec := obj["spec"].(map[string]any)
	if _, ok := spec["secretRef"]; ok {
		t.Fatalf("components GitRepository secretRef was not removed: %s", got)
	}
}

func TestMinimalTransformsResetRecommendedState(t *testing.T) {
	t.Parallel()

	helmRelease := []byte(`{"apiVersion":"helm.toolkit.fluxcd.io/v2","kind":"HelmRelease","metadata":{"name":"platform-components"},"spec":{"values":{"platform":{"components":{"apps":{"traefik":{"enabled":true}}}}}}}`)
	got, err := MinimalTransforms.TransformValue([]byte("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components"), helmRelease)
	if err != nil {
		t.Fatalf("TransformValue(platform-components): %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed HelmRelease: %v", err)
	}
	if _, ok := obj["spec"].(map[string]any)["values"]; ok {
		t.Fatalf("minimal platform-components values were not removed: %s", got)
	}

	rangeAllocation := []byte(`{"apiVersion":"v1","kind":"RangeAllocation","data":"IAAAAAAAAAAAAAg=","range":"30000-32767"}`)
	got, err = MinimalTransforms.TransformValue([]byte("/registry/ranges/servicenodeports"), rangeAllocation)
	if err != nil {
		t.Fatalf("TransformValue(servicenodeports): %v", err)
	}
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed RangeAllocation: %v", err)
	}
	if obj["data"] != "" {
		t.Fatalf("servicenodeports data = %v, want empty", obj["data"])
	}

	clusterRole := []byte(`{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"ClusterRole","rules":[{"apiGroups":["cert-manager.io"],"resources":["certificates"],"verbs":["get"]},{"apiGroups":["secrets-api.podplane.dev"],"resources":["publickeys"],"verbs":["get"]},{"apiGroups":["secrets-api.podplane.dev"],"resources":["secretproviderkeyspaces"],"verbs":["get"]},{"apiGroups":["secrets.podplane.dev"],"resources":["secretproviderbindings"],"verbs":["get"]},{"apiGroups":["secrets-store.csi.x-k8s.io"],"resources":["secretproviderclasses"],"verbs":["get"]},{"apiGroups":["source.toolkit.fluxcd.io"],"resources":["gitrepositories"],"verbs":["get"]}]}`)
	got, err = MinimalTransforms.TransformValue([]byte("/registry/clusterroles/view"), clusterRole)
	if err != nil {
		t.Fatalf("TransformValue(view ClusterRole): %v", err)
	}
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed ClusterRole: %v", err)
	}
	rules := obj["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("rules length = %d, want 1: %s", len(rules), got)
	}
	apiGroups := rules[0].(map[string]any)["apiGroups"].([]any)
	if apiGroups[0] != "source.toolkit.fluxcd.io" {
		t.Fatalf("remaining apiGroups = %#v, want Flux rule", apiGroups)
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
