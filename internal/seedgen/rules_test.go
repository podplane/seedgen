// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import "testing"

func TestRulesMatches(t *testing.T) {
	t.Parallel()
	rules, err := LoadRulesBytes([]byte(`{
		"keys": ["/registry/health"],
		"prefixes": ["/registry/events/"],
		"substrings": ["/sh.helm.release.v1."]
	}`), "test")
	if err != nil {
		t.Fatalf("LoadRulesBytes: %v", err)
	}
	cases := []struct {
		name string
		key  string
		want bool
	}{
		{"exact match", "/registry/health", true},
		{"prefix match", "/registry/events/default/foo", true},
		{"substring match", "/registry/secrets/flux-system/sh.helm.release.v1.flux.v1", true},
		{"no match", "/registry/namespaces/default", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rules.Matches(tc.key); got != tc.want {
				t.Fatalf("Matches(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestRulesMatchesNil(t *testing.T) {
	t.Parallel()
	var rules *Rules
	if rules.Matches("/registry/foo") {
		t.Fatal("nil Rules should never match")
	}
}

func TestDefaultRulesParse(t *testing.T) {
	t.Parallel()
	include, err := DefaultIncludeRules()
	if err != nil {
		t.Fatalf("DefaultIncludeRules: %v", err)
	}
	if len(include.Prefixes) == 0 {
		t.Fatal("default include rules should have prefixes")
	}
	exclude, err := DefaultExcludeRules()
	if err != nil {
		t.Fatalf("DefaultExcludeRules: %v", err)
	}
	if len(exclude.Prefixes) == 0 {
		t.Fatal("default exclude rules should have prefixes")
	}
}
