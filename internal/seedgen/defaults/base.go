// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type transform interface {
	matchesKey(key string) bool
	apply(key string, obj map[string]any) bool
}

type prefixTransform struct {
	prefix     string
	apiVersion string
	apiPrefix  string
	kind       string
	mutate     func(map[string]any) bool
}

func (t prefixTransform) matchesKey(key string) bool {
	return strings.HasPrefix(key, t.prefix)
}

func (t prefixTransform) apply(_ string, obj map[string]any) bool {
	apiVersion := stringValue(obj["apiVersion"])
	if t.apiVersion != "" && apiVersion != t.apiVersion {
		return false
	}
	if t.apiPrefix != "" && !strings.HasPrefix(apiVersion, t.apiPrefix) {
		return false
	}
	if obj["kind"] != t.kind {
		return false
	}
	return t.mutate(obj)
}

type keyTransform struct {
	key    string
	mutate func(map[string]any) bool
}

func (t keyTransform) matchesKey(key string) bool {
	return key == t.key
}

func (t keyTransform) apply(key string, obj map[string]any) bool {
	if key != t.key {
		return false
	}
	return t.mutate(obj)
}

// TransformValue applies default seed-output JSON transforms for resources
// whose live status/spec fields should not be trusted when bootstrapping a new
// cluster.
func TransformValue(key, value []byte) ([]byte, error) {
	recordKey := string(key)
	matched := matchingTransforms(recordKey)
	if len(matched) == 0 {
		return value, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(value, &obj); err != nil {
		return nil, fmt.Errorf("decode %s as JSON: %w", recordKey, err)
	}
	var changed bool
	for _, t := range matched {
		if t.apply(recordKey, obj) {
			changed = true
		}
	}
	if !changed {
		return value, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return nil, fmt.Errorf("encode transformed value for %s as JSON: %w", recordKey, err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func matchingTransforms(key string) []transform {
	var matched []transform
	for _, t := range transforms {
		if t.matchesKey(key) {
			matched = append(matched, t)
		}
	}
	return matched
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
