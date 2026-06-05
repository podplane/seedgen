// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
)

type transform interface {
	matchesKey(key string) bool
	applyJSON(key string, obj map[string]any) bool
	applyProtobuf(key string, obj runtime.Object) bool
}

type prefixTransform struct {
	prefix         string
	apiVersion     string
	apiPrefix      string
	kind           string
	mutateJSON     func(map[string]any) bool
	mutateProtobuf func(runtime.Object) bool
}

// matchesKey reports whether a registry key belongs to this prefix transform.
func (t prefixTransform) matchesKey(key string) bool {
	return strings.HasPrefix(key, t.prefix)
}

// applyJSON runs a JSON transform when the object has the API version and kind
// declared by the transform table.
func (t prefixTransform) applyJSON(_ string, obj map[string]any) bool {
	if t.mutateJSON == nil {
		return false
	}
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
	return t.mutateJSON(obj)
}

// applyProtobuf runs a typed-object transform when the decoded Kubernetes object
// has the API version and kind declared by the transform table.
func (t prefixTransform) applyProtobuf(_ string, obj runtime.Object) bool {
	if t.mutateProtobuf == nil {
		return false
	}
	gvk := obj.GetObjectKind().GroupVersionKind()
	apiVersion := gvk.GroupVersion().String()
	if t.apiVersion != "" && apiVersion != t.apiVersion {
		return false
	}
	if t.apiPrefix != "" && !strings.HasPrefix(apiVersion, t.apiPrefix) {
		return false
	}
	if gvk.Kind != t.kind {
		return false
	}
	return t.mutateProtobuf(obj)
}

type keyTransform struct {
	key            string
	mutateJSON     func(map[string]any) bool
	mutateProtobuf func(runtime.Object) bool
}

// matchesKey reports whether a registry key exactly matches this key transform.
func (t keyTransform) matchesKey(key string) bool {
	return key == t.key
}

// applyJSON runs a JSON transform when the storage key exactly matches the key
// declared by the transform table.
func (t keyTransform) applyJSON(key string, obj map[string]any) bool {
	if t.mutateJSON == nil || key != t.key {
		return false
	}
	return t.mutateJSON(obj)
}

// applyProtobuf runs a typed-object transform when the storage key exactly
// matches the key declared by the transform table.
func (t keyTransform) applyProtobuf(key string, obj runtime.Object) bool {
	if t.mutateProtobuf == nil || key != t.key {
		return false
	}
	return t.mutateProtobuf(obj)
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
		if t.applyJSON(recordKey, obj) {
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

// TransformObject applies default seed-output transforms to a typed Kubernetes
// object decoded from protobuf storage.
func TransformObject(key string, obj runtime.Object) bool {
	var changed bool
	for _, t := range matchingTransforms(key) {
		if t.applyProtobuf(key, obj) {
			changed = true
		}
	}
	return changed
}

// HasTransform reports whether any seed transform is configured for key.
func HasTransform(key string) bool {
	return len(matchingTransforms(key)) > 0
}

// matchingTransforms returns all transforms whose key selector matches key.
func matchingTransforms(key string) []transform {
	var matched []transform
	for _, t := range transforms {
		if t.matchesKey(key) {
			matched = append(matched, t)
		}
	}
	return matched
}

// stringValue returns value as a string, or an empty string for non-strings.
func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
