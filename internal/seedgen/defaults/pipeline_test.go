// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"encoding/json"
	"testing"
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
