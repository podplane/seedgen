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
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cert-manager-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/cilium-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/fluxcd-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/gateway-api-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/platform-rbac",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/traefik-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/trust-manager-crds",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-sandbox/agent-sandbox",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cert-manager/cert-manager",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-certs/platform-certs",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cilium/cilium",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-coredns/coredns",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-traefik/traefik",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-trust-manager/trust-manager",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-trust/platform-trust",
			"/registry/cert-manager.io/clusterissuers/platform-ingress-selfsigned-clusterissuer",
			"/registry/cert-manager.io/clusterissuers/platform-service-certificate-clusterissuer",
			"/registry/gateway.networking.k8s.io/gateways/platform-traefik/platform-traefik-gateway",
			"/registry/gateway.networking.k8s.io/httproutes/platform-traefik/platform-http-to-https-redirect-httproute",
			"/registry/cert-manager.io/issuers/platform-trust-manager/platform-trust-manager",
			"/registry/policy.cert-manager.io/certificaterequestpolicies/platform-ingress-certificate-crp",
			"/registry/policy.cert-manager.io/certificaterequestpolicies/platform-service-certificate-crp",
			"/registry/policy.cert-manager.io/certificaterequestpolicies/trust-manager-policy",
			"/registry/traefik.io/tlsstores/platform-traefik/default",
		},
		prefixes: []string{
			"/registry/cert-manager.io/certificates/platform-traefik/platform-traefik-default-",
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
