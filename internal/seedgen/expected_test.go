// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"strings"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

func TestCheckExpectedPlatformProfiles(t *testing.T) {
	t.Parallel()
	for _, profile := range []string{"platform", "agentic"} {
		records := make([]*datafile.Record, 0)
		for _, key := range expectedProfiles[profile].keys {
			records = append(records, &datafile.Record{Key: []byte(key)})
		}
		if err := CheckExpected(profile, records); err != nil {
			t.Fatal(err)
		}
		if profile == "platform" {
			if err := CheckExpected("agentic", records); err == nil || !strings.Contains(err.Error(), "agent-sandbox-crds") || !strings.Contains(err.Error(), "platform-agent-sandbox/agent-sandbox") {
				t.Fatalf("agentic should require Sandbox: %v", err)
			}
		}
		// Every required record is individually guarded, including runtime policy.
		for i, record := range records {
			missing := append(append([]*datafile.Record{}, records[:i]...), records[i+1:]...)
			if err := CheckExpected(profile, missing); err == nil || !strings.Contains(err.Error(), string(record.Key)) {
				t.Fatalf("%s missing %s: %v", profile, record.Key, err)
			}
		}
	}
}

func TestCheckExpectedPlatformReportsMissing(t *testing.T) {
	t.Parallel()
	err := CheckExpected("platform", []*datafile.Record{
		{Key: []byte("/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components")},
	})
	if err == nil {
		t.Fatalf("CheckExpected platform succeeded with missing records")
	}
	msg := err.Error()
	for _, want := range []string{
		"platform-envoy-gateway/envoy-gateway",
		"platform-envoy-gateway/platform-envoy-gateway",
		"platform-http-to-https-redirect-httproute",
		"platform-podplane-operator/platform-podplane-operator",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing error did not mention %q: %s", want, msg)
		}
	}
}

func TestCheckExpectedUnknownProfile(t *testing.T) {
	t.Parallel()
	if err := CheckExpected("custom", nil); err == nil {
		t.Fatalf("CheckExpected custom succeeded")
	}
}

func TestCheckExpectedNoneDisablesCheck(t *testing.T) {
	t.Parallel()
	if err := CheckExpected("none", nil); err != nil {
		t.Fatalf("CheckExpected none error = %v", err)
	}
}
