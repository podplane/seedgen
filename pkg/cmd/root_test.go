// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/pkg/pipeline"
)

// TestClassifyKeys verifies that include and exclude rules partition keys into
// the command's report groups.
func TestClassifyKeys(t *testing.T) {
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
		{Key: []byte("/registry/namespaces/default")},
		{Key: []byte("/registry/events/default/foo")},
		{Key: []byte("/registry/secrets/flux/sh.helm.release.v1.flux.v1")},
	}

	kept, includedKeys, excludedKeys, ignoredKeys := classifyKeys(records, include, exclude)
	if len(kept) != 1 || string(kept[0].Key) != "/registry/namespaces/default" {
		t.Fatalf("kept = %#v, want namespace record", kept)
	}
	if !slices.Equal(includedKeys, []string{"/registry/namespaces/default"}) {
		t.Fatalf("includedKeys = %#v", includedKeys)
	}
	if !slices.Equal(excludedKeys, []string{"/registry/secrets/flux/sh.helm.release.v1.flux.v1"}) {
		t.Fatalf("excludedKeys = %#v", excludedKeys)
	}
	if !slices.Equal(ignoredKeys, []string{"/registry/events/default/foo"}) {
		t.Fatalf("ignoredKeys = %#v", ignoredKeys)
	}
}

// TestWriteKeyReports verifies that the command writes one report file for
// each key classification.
func TestWriteKeyReports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := writeKeyReports(dir, []string{"included"}, []string{"excluded"}, []string{"ignored"}); err != nil {
		t.Fatalf("writeKeyReports: %v", err)
	}
	for name, want := range map[string]string{
		"included.txt": "included\n",
		"excluded.txt": "excluded\n",
		"ignored.txt":  "ignored\n",
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(data) != want {
			t.Fatalf("%s = %q, want %q", name, data, want)
		}
	}
}
