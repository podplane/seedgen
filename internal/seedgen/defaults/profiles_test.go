// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/internal/seedgen"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Exercise the public pipeline hooks, including broad common-rule leakage.
func TestPlatformProfiles(t *testing.T) {
	p := Pipeline()
	sandboxKeys := []string{
		"/registry/namespaces/platform-agent-sandbox",
		"/registry/serviceaccounts/platform-agent-sandbox/agent-sandbox",
		"/registry/roles/platform-agent-sandbox/agent-sandbox-controller",
		"/registry/rolebindings/platform-agent-sandbox/agent-sandbox-controller",
		"/registry/deployments/platform-agent-sandbox/agent-sandbox",
		"/registry/services/specs/platform-agent-sandbox/agent-sandbox",
		"/registry/clusterroles/agent-sandbox-controller",
		"/registry/clusterrolebindings/agent-sandbox-controller",
		"/registry/clusterroles/platform-agent-sandbox-admin",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/sandboxes.agents.x-k8s.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/sandboxclaims.extensions.agents.x-k8s.io",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-sandbox/agent-sandbox",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-cluster/agent-sandbox-crds",
		"/registry/source.toolkit.fluxcd.io/helmcharts/platform-components/platform-agent-sandbox-agent-sandbox",
		"/registry/source.toolkit.fluxcd.io/helmcharts/platform-components/platform-cluster-agent-sandbox-crds",
		"/registry/secrets/platform-agent-sandbox/sh.helm.release.v1.agent-sandbox.v1",
		"/registry/secrets/platform-cluster/sh.helm.release.v1.platform-agent-sandbox-crds.v1",
	}
	for _, profile := range []string{"platform", "agentic"} {
		t.Run(profile, func(t *testing.T) {
			include, err := p.IncludeRules(profile)
			if err != nil {
				t.Fatal(err)
			}
			exclude, err := p.ExcludeRules(profile)
			if err != nil {
				t.Fatal(err)
			}
			if !exclude.Matches("/registry/configmaps/platform-agent-sandbox/kube-root-ca.crt") {
				t.Fatal("common per-cluster CA exclusion must remain")
			}
			for _, key := range sandboxKeys {
				kept := include.Matches(key) && !exclude.Matches(key)
				if kept != (profile == "agentic") {
					t.Errorf("%s kept = %v", key, kept)
				}
			}
			for _, key := range []string{
				"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-envoy-gateway/envoy-gateway",
				"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-podplane-operator/podplane-operator",
				"/registry/clusterroles/view",
			} {
				if !include.Matches(key) || exclude.Matches(key) {
					t.Errorf("missing shared record %s", key)
				}
			}
			for _, key := range []string{"/registry/secrets/platform-envoy-gateway/node", "/registry/secrets/platform-envoy-gateway/bundle-example", "/registry/daemonsets/platform-envoy-gateway/platform-envoy-gateway"} {
				if !exclude.Matches(key) {
					t.Errorf("missing Envoy exclusion %s", key)
				}
			}

			parentKey := "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components"
			parent := []byte(`{"apiVersion":"helm.toolkit.fluxcd.io/v2","kind":"HelmRelease","spec":{"values":{"platform":{"components":{"apps":{"agent-sandbox":{"enabled":true},"envoy-gateway":{"enabled":true}},"crds":{"agent-sandbox-crds":{"enabled":true},"envoy-gateway-crds":{"enabled":true}},"values":{"agent-sandbox":{"replicas":2},"envoy-gateway":{"custom":"retain"}}}}}}}`)
			prepared, err := seedgen.PrepareRecordsForWrite([]*datafile.Record{{Key: []byte(parentKey), Value: parent}}, seedgen.WriteOptions{Transforms: p.Transforms(profile)})
			if err != nil {
				t.Fatal(err)
			}
			if len(prepared) != 1 || prepared[0].Revision != 1 {
				t.Fatalf("non-contiguous output: %#v", prepared)
			}
			got := prepared[0].Value
			if strings.Contains(string(got), "agent-sandbox") != (profile == "agentic") {
				t.Fatalf("parent Sandbox values: %s", got)
			}
			if !strings.Contains(string(got), "retain") || !strings.Contains(string(got), "envoy-gateway-crds") {
				t.Fatalf("lost other overrides: %s", got)
			}

			for _, name := range []string{"admin", "edit", "view"} {
				key := "/registry/clusterroles/" + name
				role := []byte(`{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"ClusterRole","rules":[{"apiGroups":["agents.x-k8s.io"],"resources":["sandboxes"],"verbs":["get"]},{"apiGroups":["gateway.networking.k8s.io"],"resources":["gateways"],"verbs":["get"]}]}`)
				got, err := p.Transforms(profile).TransformValue([]byte(key), role)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(got), "agents.x-k8s.io") != (profile == "agentic") || !strings.Contains(string(got), "gateway.networking.k8s.io") {
					t.Fatalf("RBAC: %s", got)
				}
				typed := &rbacv1.ClusterRole{Rules: []rbacv1.PolicyRule{{APIGroups: []string{"agents.x-k8s.io"}}, {APIGroups: []string{"gateway.networking.k8s.io"}}}}
				p.Transforms(profile).TransformObject(key, typed)
				want := 1
				if profile == "agentic" {
					want = 2
				}
				if len(typed.Rules) != want {
					t.Fatalf("typed RBAC: %#v", typed.Rules)
				}
			}
			value := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"replicas":1}}`)
			got, err = p.Transforms(profile).TransformValue([]byte(envoyGatewayDeploymentKey), value)
			if err != nil {
				t.Fatal(err)
			}
			var obj map[string]any
			if err := json.Unmarshal(got, &obj); err != nil {
				t.Fatal(err)
			}
			if obj["spec"].(map[string]any)["replicas"] != float64(0) {
				t.Fatalf("Envoy not scaled: %s", got)
			}
			replicas := int32(1)
			deployment := &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas}}
			p.Transforms(profile).TransformObject(envoyGatewayDeploymentKey, deployment)
			if *deployment.Spec.Replicas != 0 {
				t.Fatal("typed Envoy not scaled")
			}
		})
	}
	for _, obsolete := range []string{"recommended", "app"} {
		if _, err := p.IncludeRules(obsolete); err == nil {
			t.Fatalf("%s include alias remains", obsolete)
		}
		if _, err := p.ExcludeRules(obsolete); err == nil {
			t.Fatalf("%s exclude alias remains", obsolete)
		}
		if err := p.CheckExpected(obsolete, nil); err == nil {
			t.Fatalf("%s guard alias remains", obsolete)
		}
	}
}

// TestProfilesExcludeLocalStorage checks no seed profile keeps the cluster-scoped
// leftovers of the local CSI storage stack, while keeping shared records that
// sit under the same broad include prefixes.
func TestProfilesExcludeLocalStorage(t *testing.T) {
	p := Pipeline()
	excluded := []string{
		"/registry/apiextensions.k8s.io/customresourcedefinitions/lvmvolumes.local.openebs.io",
		"/registry/apiregistration.k8s.io/apiservices/v1alpha1.local.openebs.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/volumesnapshots.snapshot.storage.k8s.io",
		"/registry/apiextensions.k8s.io/customresourcedefinitions/volumegroupsnapshots.groupsnapshot.storage.k8s.io",
		"/registry/apiregistration.k8s.io/apiservices/v1beta2.groupsnapshot.storage.k8s.io",
		"/registry/clusterroles/openebs-lvm-provisioner-role",
		"/registry/clusterrolebindings/openebs-lvm-provisioner-binding",
		"/registry/csidrivers/local.csi.openebs.io",
		"/registry/priorityclasses/platform-csi-openebs-lvm-lvm-localpv-csi-node-critical",
		"/registry/storageclasses/platform-openebs-lvm-default",
	}
	kept := []string{
		"/registry/apiextensions.k8s.io/customresourcedefinitions/xbackends.gateway.networking.x-k8s.io",
		"/registry/priorityclasses/system-node-critical",
	}
	for _, profile := range []string{"minimal", "platform", "agentic"} {
		t.Run(profile, func(t *testing.T) {
			include, err := p.IncludeRules(profile)
			if err != nil {
				t.Fatal(err)
			}
			exclude, err := p.ExcludeRules(profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range excluded {
				if !include.Matches(key) {
					t.Errorf("%s no longer matches an include rule; drop it from this test", key)
				}
				if !exclude.Matches(key) {
					t.Errorf("%s kept", key)
				}
			}
			for _, key := range kept {
				if !include.Matches(key) || exclude.Matches(key) {
					t.Errorf("%s dropped", key)
				}
			}
		})
	}
}
