// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tidwall/jsonc"
)

//go:embed defaults/include.jsonc
var defaultIncludeJSONC []byte

//go:embed defaults/exclude.jsonc
var defaultExcludeJSONC []byte

// Rules captures one side of the filter (include or exclude). A record
// matches the rule set if its key matches any entry in Keys, Prefixes,
// or Substrings.
type Rules struct {
	Keys       []string `json:"keys"`
	Prefixes   []string `json:"prefixes"`
	Substrings []string `json:"substrings"`

	keySet map[string]struct{}
}

// DefaultIncludeRules returns the embedded default include rules.
func DefaultIncludeRules() (*Rules, error) {
	return LoadRulesBytes(defaultIncludeJSONC, "<embedded include.jsonc>")
}

// DefaultExcludeRules returns the embedded default exclude rules.
func DefaultExcludeRules() (*Rules, error) {
	return LoadRulesBytes(defaultExcludeJSONC, "<embedded exclude.jsonc>")
}

// LoadRulesFile reads and parses a JSONC rules file from disk.
func LoadRulesFile(path string) (*Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file %s: %w", path, err)
	}
	return LoadRulesBytes(data, path)
}

// LoadRulesBytes parses a JSONC rules payload.
func LoadRulesBytes(data []byte, source string) (*Rules, error) {
	var rules Rules
	if err := json.Unmarshal(jsonc.ToJSON(data), &rules); err != nil {
		return nil, fmt.Errorf("decode rules %s: %w", source, err)
	}
	rules.keySet = make(map[string]struct{}, len(rules.Keys))
	for _, key := range rules.Keys {
		rules.keySet[key] = struct{}{}
	}
	return &rules, nil
}

// Matches reports whether key matches any rule in the set.
func (r *Rules) Matches(key string) bool {
	if r == nil {
		return false
	}
	if _, ok := r.keySet[key]; ok {
		return true
	}
	for _, prefix := range r.Prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	for _, sub := range r.Substrings {
		if strings.Contains(key, sub) {
			return true
		}
	}
	return false
}
