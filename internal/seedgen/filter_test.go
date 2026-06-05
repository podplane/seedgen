// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/pkg/pipeline"
)

// TestFilterIncludeAndExclude verifies that filtering preserves included
// records unless they also match an exclude rule.
func TestFilterIncludeAndExclude(t *testing.T) {
	t.Parallel()
	include, err := pipeline.LoadRulesBytes([]byte(`{
		"prefixes": ["/registry/namespaces/", "/registry/secrets/"]
	}`), "include")
	if err != nil {
		t.Fatalf("include: %v", err)
	}
	exclude, err := pipeline.LoadRulesBytes([]byte(`{
		"substrings": ["/sh.helm.release.v1."]
	}`), "exclude")
	if err != nil {
		t.Fatalf("exclude: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/namespaces/default")},
		{Revision: 2, Key: []byte("/registry/events/default/foo")},                      // not in include
		{Revision: 3, Key: []byte("/registry/secrets/flux/sh.helm.release.v1.flux.v1")}, // excluded
		{Revision: 4, Key: []byte("/registry/secrets/default/my-secret")},
	}
	kept := Filter(records, include, exclude)
	if len(kept) != 2 {
		t.Fatalf("want 2 kept, got %d", len(kept))
	}
	if string(kept[0].Key) != "/registry/namespaces/default" {
		t.Fatalf("unexpected first kept: %s", kept[0].Key)
	}
	if string(kept[1].Key) != "/registry/secrets/default/my-secret" {
		t.Fatalf("unexpected second kept: %s", kept[1].Key)
	}
}
