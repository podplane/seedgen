// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"bytes"
	"encoding/json"
	"strings"
)

type transform interface {
	apply(key string, obj map[string]any) bool
}

type objectTransform struct {
	apiVersion string
	apiPrefix  string
	kind       string
	mutate     func(map[string]any) bool
}

func (t objectTransform) apply(_ string, obj map[string]any) bool {
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

func (t keyTransform) apply(key string, obj map[string]any) bool {
	if key != t.key {
		return false
	}
	return t.mutate(obj)
}

// TransformValue applies default seed-output JSON transforms for resources
// whose live status/spec fields should not be trusted when bootstrapping a new
// cluster.
func TransformValue(key, value []byte) []byte {
	var obj map[string]any
	if err := json.Unmarshal(value, &obj); err != nil {
		return value
	}
	var changed bool
	recordKey := string(key)
	for _, t := range transforms {
		if t.apply(recordKey, obj) {
			changed = true
		}
	}
	if !changed {
		return value
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return value
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
