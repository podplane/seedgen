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

var expectedProfiles = map[string]expectedProfile{
	"minimal": {
		keys: []string{
			"_netsy",
			"/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cilium-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/fluxcd-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/gateway-api-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-rbac",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cilium/cilium",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-coredns/coredns",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd",
		},
	},
	"recommended": {
		keys: []string{
			"_netsy",
			"/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/agent-sandbox-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cilium-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/envoy-gateway-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/fluxcd-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/gateway-api-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-rbac",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/podplane-operator-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/secrets-store-csi-driver-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-sandbox/agent-sandbox",
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
			"/registry/secrets-store.csi.x-k8s.io/secretproviderclasses/platform-envoy-gateway/platform-envoy-gateway-ingress-certificates",
			"/registry/secrets-store.csi.x-k8s.io/secretproviderclasses/platform-podplane-operator/platform-podplane-operator",
		},
	},
}

// CheckExpected verifies that the kept record set contains the records expected
// for profile. An empty profile, or "none", disables the check.
func CheckExpected(profile string, records []*datafile.Record) error {
	if profile == "" || profile == "none" {
		return nil
	}
	expected, ok := expectedProfiles[profile]
	if !ok {
		return fmt.Errorf("unknown expected seed profile %q (want recommended, minimal, or none)", profile)
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
