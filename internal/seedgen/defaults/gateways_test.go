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
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestGatewayProfiles(t *testing.T) {
	p := Pipeline()
	for _, profile := range []string{"minimal", "platform", "agentic"} {
		inc, err := p.IncludeRules(profile)
		if err != nil {
			t.Fatal(err)
		}
		exc, err := p.ExcludeRules(profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-sourcegateway/sourcegateway",
			"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-router/agent-router",
			"/registry/apiextensions.k8s.io/customresourcedefinitions/gitrepositories.sourcegateway.dev",
			"/registry/apiextensions.k8s.io/customresourcedefinitions/objectstoragebuckets.sourcegateway.dev",
			"/registry/clusterroles/agent-router:platform-agent-router",
			"/registry/clusterrolebindings/agent-router:platform-agent-router",
			"/registry/clusterroles/envoy-ai-gateway-inference-pool-reader:platform-agent-router",
			"/registry/mutatingwebhookconfigurations/envoy-ai-gateway-gateway-pod-mutator.platform-agent-router",
			"/registry/deployments/platform-agent-router/agent-router",
			"/registry/secrets.podplane.dev/secretproviderbindings/platform-sourcegateway/sourcegateway",
			"/registry/configmaps/platform-sourcegateway/sourcegateway",
			"/registry/networkpolicies/platform-sourcegateway/sourcegateway-ingress",
		} {
			if got := inc.Matches(key) && !exc.Matches(key); got != (profile == "agentic") {
				t.Errorf("%s: %s kept=%v", profile, key, got)
			}
		}
		for _, key := range []string{
			"/registry/sourcegateway.dev/gitrepositories/platform-sourcegateway/user",
			"/registry/sourcegateway.dev/objectstoragebuckets/platform-sourcegateway/user",
			"/registry/agentgateway.dev/agentgatewaybackends/platform-agentgateway/user",
			"/registry/secrets/platform-agent-router/ca",
			"/registry/secrets/platform-agent-router/sh.helm.release.v1.agent-router.v1",
			"/registry/gateway.networking.k8s.io/gatewayclasses/agentgateway",
			"/registry/agentgateway.dev/agentgatewayparameters/platform-agentgateway/agentgateway",
			"/registry/secrets/platform-sourcegateway/sh.helm.release.v1.sourcegateway.v1",
			"/registry/secrets/platform-components/sh.helm.release.v1.platform-components.v1",
			"/registry/secrets-store.csi.x-k8s.io/secretproviderclasses/platform-sourcegateway/sourcegateway",
			"/registry/secrets-api.podplane.dev/secretproviderkeyspaces/platform-sourcegateway/old-provider.sourcegateway",
		} {
			if inc.Matches(key) && !exc.Matches(key) {
				t.Errorf("%s leaks %s", profile, key)
			}
		}
		for _, name := range []string{"aigatewayroutes", "aiservicebackends", "backendsecuritypolicies", "gatewayconfigs", "mcproutes", "quotapolicies"} {
			key := "/registry/apiextensions.k8s.io/customresourcedefinitions/" + name + ".aigateway.envoyproxy.io"
			if got := inc.Matches(key) && !exc.Matches(key); got != (profile == "agentic") {
				t.Errorf("%s: %s kept=%v", profile, key, got)
			}
			key = "/registry/aigateway.envoyproxy.io/" + name + "/platform-agent-router/user"
			if inc.Matches(key) && !exc.Matches(key) {
				t.Errorf("%s leaks %s", profile, key)
			}
		}
		role := &rbacv1.ClusterRole{Rules: []rbacv1.PolicyRule{{APIGroups: []string{"sourcegateway.dev"}}, {APIGroups: []string{"aigateway.envoyproxy.io"}}, {APIGroups: []string{"source.toolkit.fluxcd.io"}}}}
		p.Transforms(profile).TransformObject("/registry/clusterroles/admin", role)
		want := 1
		if profile == "agentic" {
			want = 3
		}
		if len(role.Rules) != want {
			t.Fatalf("%s RBAC: %#v", profile, role.Rules)
		}
		jsonRole := []byte(`{"rules":[{"apiGroups":["sourcegateway.dev"]},{"apiGroups":["aigateway.envoyproxy.io"]},{"apiGroups":["source.toolkit.fluxcd.io"]}]}`)
		out, err := p.Transforms(profile).TransformValue([]byte("/registry/clusterroles/admin"), jsonRole)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "sourcegateway.dev") != (profile == "agentic") || strings.Contains(string(out), "aigateway.envoyproxy.io") != (profile == "agentic") || !strings.Contains(string(out), "source.toolkit.fluxcd.io") {
			t.Fatalf("%s JSON RBAC: %s", profile, out)
		}
	}
	webhookKey := "/registry/mutatingwebhookconfigurations/envoy-ai-gateway-gateway-pod-mutator.platform-agent-router"
	webhook := &admissionv1.MutatingWebhookConfiguration{Webhooks: []admissionv1.MutatingWebhook{{ClientConfig: admissionv1.WebhookClientConfig{CABundle: []byte("captured")}}}}
	p.Transforms("agentic").TransformObject(webhookKey, webhook)
	if len(webhook.Webhooks[0].ClientConfig.CABundle) != 0 {
		t.Fatal("captured webhook CA retained")
	}
	prepared, err := seedgen.PrepareRecordsForWrite([]*datafile.Record{{Key: []byte(webhookKey), Value: []byte(`{"webhooks":[{"clientConfig":{"caBundle":"captured","service":{"name":"agent-router"}}}]}`)}}, seedgen.WriteOptions{Transforms: p.Transforms("agentic")})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 1 || prepared[0].Revision != 1 || strings.Contains(string(prepared[0].Value), "captured") || !strings.Contains(string(prepared[0].Value), "agent-router") {
		t.Fatalf("webhook seed output: %#v", prepared)
	}
	key := "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-sourcegateway/sourcegateway"
	value := []byte(`{"spec":{"values":{"platform":{"sourceGateway":{"providerName":"old-provider"}},"sourcegateway":{"encryptionKey":{"value":"persistent-key"},"fullnameOverride":"sourcegateway","persistence":{"enabled":true}}}}}`)
	out, err := seedgen.PrepareRecordsForWrite([]*datafile.Record{{Key: []byte(key), Value: value}}, seedgen.WriteOptions{Transforms: p.Transforms("agentic")})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Revision != 1 {
		t.Fatalf("non-contiguous records: %#v", out)
	}
	if s := string(out[0].Value); strings.Contains(s, "old-provider") || strings.Contains(s, "persistent-key") || !strings.Contains(s, "persistence") {
		t.Fatal(s)
	}
	for _, key := range []string{
		"/registry/secrets.podplane.dev/secretproviderbindings/platform-sourcegateway/sourcegateway",
		"/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components",
	} {
		value := []byte(`{"spec":{"providerName":"old-provider","values":{"platform":{"components":{"values":{"sourcegateway":{"platform":{"sourceGateway":{"providerName":"old-provider"}},"sourcegateway":{"encryptionKey":{"value":"persistent-key"},"servingTLS":{"keep":true}}}}}}}}}`)
		if strings.Contains(key, "helmreleases") {
			value = []byte(strings.Replace(string(value), `"providerName":"old-provider","values"`, `"values"`, 1))
		} else {
			value = []byte(`{"spec":{"providerName":"old-provider","secretName":"sourcegateway"}}`)
		}
		got, err := p.Transforms("agentic").TransformValue([]byte(key), value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(got), "old-provider") {
			t.Fatal(string(got))
		}
		if strings.Contains(key, "helmreleases") && (strings.Contains(string(got), "persistent-key") || !strings.Contains(string(got), "servingTLS")) {
			t.Fatal(string(got))
		}
	}
	for _, deploymentKey := range []string{"/registry/deployments/platform-sourcegateway/sourcegateway", "/registry/deployments/platform-agent-router/agent-router"} {
		replicas := int32(1)
		deployment := &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas}}
		p.Transforms("agentic").TransformObject(deploymentKey, deployment)
		if *deployment.Spec.Replicas != 0 {
			t.Fatalf("%s must wait for runtime reconciliation", deploymentKey)
		}
		got, err := p.Transforms("agentic").TransformValue([]byte(deploymentKey), []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"replicas":1}}`))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), `"replicas":0`) {
			t.Fatal(string(got))
		}
	}
}

func TestAgentRouterSeedValues(t *testing.T) {
	p := Pipeline()
	for _, parent := range []bool{false, true} {
		key := "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-agent-router/agent-router"
		values := map[string]any{"ai-gateway-helm": map[string]any{"controller": map[string]any{"mcp": map[string]any{"sessionEncryption": map[string]any{
			"seed": "captured-primary", "existingSecret": "captured-secret", "iterations": 100000,
			"fallback": map[string]any{"seed": "captured-fallback", "iterations": 90000},
		}}}}}
		if parent {
			key = "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components"
			values = map[string]any{"platform": map[string]any{"components": map[string]any{"values": map[string]any{"agent-router": values}}}}
		}
		input, err := json.Marshal(map[string]any{"spec": map[string]any{"values": values}})
		if err != nil {
			t.Fatal(err)
		}
		out, err := seedgen.PrepareRecordsForWrite([]*datafile.Record{{Key: []byte(key), Value: input}}, seedgen.WriteOptions{Transforms: p.Transforms("agentic")})
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0].Revision != 1 || strings.Contains(string(out[0].Value), "captured-") || !strings.Contains(string(out[0].Value), "90000") || !strings.Contains(string(out[0].Value), "100000") {
			t.Fatalf("session seed leaked or non-secret configuration lost: %#v", out)
		}
	}
	for _, profile := range []string{"platform", "agentic"} {
		for _, host := range []string{"agent-router.platform-agent-router.svc.cluster.local", "other-extension.example"} {
			input := `{"spec":{"values":{"gateway-helm":{"config":{"envoyGateway":{"extensionApis":{"enableSDSSecretRef":true},"extensionManager":{"service":{"fqdn":{"hostname":"` + host + `"}}}}}}}}}`
			key := []byte("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-envoy-gateway/envoy-gateway")
			out, err := p.Transforms(profile).TransformValue(key, []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			wantExtension := profile == "agentic" || host == "other-extension.example"
			if strings.Contains(string(out), "extensionManager") != wantExtension || !strings.Contains(string(out), "enableSDSSecretRef") {
				t.Fatalf("%s extension cleanup: %s", profile, out)
			}
		}
	}
}
