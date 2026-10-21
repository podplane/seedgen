// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"fmt"
	"strings"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

type expectedProfile struct {
	keys     []string
	prefixes []string
}

// runtimePolicyKeys are platform-runtime's records. Every managed seed
// needs all of them: the validating policy alone rejects restored Pods.
var runtimePolicyKeys = []string{
	"/registry/runtimeclasses/gvisor",
	"/registry/runtimeclasses/runc",
	"/registry/mutatingadmissionpolicies/platform-runtime-class-default-map",
	"/registry/mutatingadmissionpolicybindings/platform-runtime-class-default-mapb",
	"/registry/validatingadmissionpolicies/platform-runtime-class-vap",
	"/registry/validatingadmissionpolicybindings/platform-runtime-class-vapb",
}

var expectedProfiles = map[string]expectedProfile{
	"minimal": {
		keys: append([]string{
			"_netsy",
			"/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cilium-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/fluxcd-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/gateway-api-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-rbac",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-runtime",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cilium/cilium",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-coredns/coredns",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd",
		}, runtimePolicyKeys...),
	},
	"platform": {
		keys: append([]string{
			"_netsy",
			"/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cilium-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/envoy-gateway-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/fluxcd-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/gateway-api-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-rbac",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-runtime",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/podplane-operator-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/secrets-store-csi-driver-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cilium/cilium",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-coredns/coredns",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-envoy-gateway/envoy-gateway",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-podplane-operator/podplane-operator",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-secrets-store-csi-driver/secrets-store-csi-driver",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-zot-registry/zot-registry",
			"/registry/gateway.envoyproxy.io/envoyproxies/platform-envoy-gateway/platform-envoy-gateway",
			"/registry/gateway.networking.k8s.io/gatewayclasses/platform-envoy-gateway",
			"/registry/gateway.networking.k8s.io/gateways/platform-envoy-gateway/platform-envoy-gateway",
			"/registry/gateway.networking.k8s.io/httproutes/platform-envoy-gateway/platform-http-to-https-redirect-httproute",
			"/registry/secrets-store.csi.x-k8s.io/secretproviderclasses/platform-podplane-operator/platform-podplane-operator",
		}, runtimePolicyKeys...),
	},
}

func init() {
	platform := expectedProfiles["platform"]
	expectedProfiles["agentic"] = expectedProfile{keys: append(append([]string{}, platform.keys...),
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-sourcegateway/sourcegateway",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-router/agent-router",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/sourcegateway-crds",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/agent-router-crds",
		"/registry/mutatingwebhookconfigurations/envoy-ai-gateway-gateway-pod-mutator.platform-agent-router",
		"/registry/secrets.podplane.dev/secretproviderbindings/platform-sourcegateway/sourcegateway",
		"/registry/deployments/platform-sourcegateway/sourcegateway",
		"/registry/services/specs/platform-sourcegateway/sourcegateway",
		"/registry/serviceaccounts/platform-sourcegateway/sourcegateway",
		"/registry/configmaps/platform-sourcegateway/sourcegateway",
		"/registry/networkpolicies/platform-sourcegateway/sourcegateway-ingress",
		"/registry/namespaces/platform-sourcegateway",
		"/registry/namespaces/platform-agent-router",
		"/registry/deployments/platform-agent-router/agent-router",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/aigatewayroutes.aigateway.envoyproxy.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/aiservicebackends.aigateway.envoyproxy.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/backendsecuritypolicies.aigateway.envoyproxy.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/gatewayconfigs.aigateway.envoyproxy.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/mcproutes.aigateway.envoyproxy.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/quotapolicies.aigateway.envoyproxy.io",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/agent-sandbox-crds",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-sandbox/agent-sandbox",
	)}
}

// CheckExpected verifies that the kept record set contains the records expected
// for profile. An empty profile, or "none", disables the check.
func CheckExpected(profile string, records []*datafile.Record) error {
	if profile == "" || profile == "none" {
		return nil
	}
	expected, ok := expectedProfiles[profile]
	if !ok {
		return fmt.Errorf("unknown expected seed profile %q (want platform, agentic, minimal, or none)", profile)
	}

	keys := make(map[string]struct{}, len(records))
	for _, record := range records {
		keys[string(record.Key)] = struct{}{}
	}

	var missing []string
	for _, key := range expected.keys {
		if _, ok := keys[key]; !ok {
			missing = append(missing, key)
		}
	}
	for _, prefix := range expected.prefixes {
		var found bool
		for key := range keys {
			if strings.HasPrefix(key, prefix) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, prefix+"*")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %d expected %s seed records:\n%s", len(missing), profile, strings.Join(missing, "\n"))
	}
	return nil
}
