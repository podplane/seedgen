// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tidwall/jsonc"
)

// Rules captures one side of the filter (include or exclude). A record
// matches the rule set if its key matches any entry in Keys, Prefixes,
// or Substrings.
type Rules struct {
	Keys       []string `json:"keys"`
	Prefixes   []string `json:"prefixes"`
	Substrings []string `json:"substrings"`

	keySet map[string]struct{}
}

// MergeRules returns one rule set containing every key, prefix, and substring
// from rules in order.
func MergeRules(rules ...*Rules) *Rules {
	merged := &Rules{}
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		merged.Keys = append(merged.Keys, rule.Keys...)
		merged.Prefixes = append(merged.Prefixes, rule.Prefixes...)
		merged.Substrings = append(merged.Substrings, rule.Substrings...)
	}
	merged.rebuild()
	return merged
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
	rules.rebuild()
	return &rules, nil
}

// rebuild refreshes the exact-key lookup table after parsing or merging rules.
func (r *Rules) rebuild() {
	r.keySet = make(map[string]struct{}, len(r.Keys))
	for _, key := range r.Keys {
		r.keySet[key] = struct{}{}
	}
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
