// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"strings"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

func TestCheckExpectedRecommended(t *testing.T) {
	t.Parallel()
	records := make([]*datafile.Record, 0)
	for _, key := range expectedProfiles["recommended"].keys {
		records = append(records, &datafile.Record{Key: []byte(key)})
	}

	if err := CheckExpected("recommended", records); err != nil {
		t.Fatalf("CheckExpected recommended error = %v", err)
	}
}

func TestCheckExpectedRecommendedReportsMissing(t *testing.T) {
	t.Parallel()
	err := CheckExpected("recommended", []*datafile.Record{
		{Key: []byte("/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components")},
	})
	if err == nil {
		t.Fatalf("CheckExpected recommended succeeded with missing records")
	}
	msg := err.Error()
	for _, want := range []string{
		"platform-envoy-gateway/envoy-gateway",
		"platform-envoy-gateway/platform-envoy-gateway",
		"platform-http-to-https-redirect-httproute",
		"platform-envoy-gateway-ingress-certificates",
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
