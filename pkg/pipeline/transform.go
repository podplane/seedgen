// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
)

// NormalizeImageRef expands Docker Hub shorthand image references to explicit
// registry/repository form.
func NormalizeImageRef(value string) string {
	value = strings.TrimSpace(value)
	if slash := strings.Index(value, "/"); slash >= 0 {
		first := value[:slash]
		if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
			return value
		}
		return "docker.io/" + value
	}
	return "docker.io/library/" + value
}

// Transform is one key selector plus optional JSON and protobuf transform chains.
type Transform interface {
	matchesKey(key string) bool
	applyJSON(key string, obj map[string]any) bool
	applyProtobuf(key string, obj runtime.Object) bool
}

// Transforms is an ordered list of seed output transforms.
type Transforms []Transform

// JSONTransform mutates a decoded JSON object and reports whether it changed.
type JSONTransform func(map[string]any) bool

// ProtobufTransform mutates a decoded Kubernetes runtime object and reports
// whether it changed.
type ProtobufTransform func(runtime.Object) bool

// PrefixTransform applies transforms to keys under Prefix whose object kind and
// API version match the transform metadata.
type PrefixTransform struct {
	Prefix             string
	APIVersion         string
	APIPrefix          string
	Kind               string
	JSONTransforms     []JSONTransform
	ProtobufTransforms []ProtobufTransform
}

// matchesKey reports whether a registry key belongs to this prefix transform.
func (t PrefixTransform) matchesKey(key string) bool {
	return strings.HasPrefix(key, t.Prefix)
}

// applyJSON runs JSON transforms when the object has the API version and kind
// declared by the transform table.
func (t PrefixTransform) applyJSON(_ string, obj map[string]any) bool {
	if t.JSONTransforms == nil {
		return false
	}
	apiVersion, _ := obj["apiVersion"].(string)
	if t.APIVersion != "" && apiVersion != t.APIVersion {
		return false
	}
	if t.APIPrefix != "" && !strings.HasPrefix(apiVersion, t.APIPrefix) {
		return false
	}
	if obj["kind"] != t.Kind {
		return false
	}
	return applyJSONTransforms(t.JSONTransforms, obj)
}

// applyProtobuf runs protobuf transforms when the decoded Kubernetes
// object has the API version and kind declared by the transform table.
func (t PrefixTransform) applyProtobuf(_ string, obj runtime.Object) bool {
	if t.ProtobufTransforms == nil {
		return false
	}
	gvk := obj.GetObjectKind().GroupVersionKind()
	apiVersion := gvk.GroupVersion().String()
	if t.APIVersion != "" && apiVersion != t.APIVersion {
		return false
	}
	if t.APIPrefix != "" && !strings.HasPrefix(apiVersion, t.APIPrefix) {
		return false
	}
	if gvk.Kind != t.Kind {
		return false
	}
	return applyProtobufTransforms(t.ProtobufTransforms, obj)
}

// KeyTransform applies transforms to one exact key.
type KeyTransform struct {
	Key                string
	JSONTransforms     []JSONTransform
	ProtobufTransforms []ProtobufTransform
}

// matchesKey reports whether a registry key exactly matches this key transform.
func (t KeyTransform) matchesKey(key string) bool {
	return key == t.Key
}

// applyJSON runs JSON transforms when the storage key exactly matches the key
// declared by the transform table.
func (t KeyTransform) applyJSON(key string, obj map[string]any) bool {
	if t.JSONTransforms == nil || key != t.Key {
		return false
	}
	return applyJSONTransforms(t.JSONTransforms, obj)
}

// applyProtobuf runs protobuf transforms when the storage key exactly
// matches the key declared by the transform table.
func (t KeyTransform) applyProtobuf(key string, obj runtime.Object) bool {
	if t.ProtobufTransforms == nil || key != t.Key {
		return false
	}
	return applyProtobufTransforms(t.ProtobufTransforms, obj)
}

// applyJSONTransforms runs each JSON transform and reports whether any changed
// the object.
func applyJSONTransforms(transforms []JSONTransform, obj map[string]any) bool {
	var changed bool
	for _, mutate := range transforms {
		if mutate(obj) {
			changed = true
		}
	}
	return changed
}

// applyProtobufTransforms runs each protobuf transform and reports whether any
// changed the object.
func applyProtobufTransforms(transforms []ProtobufTransform, obj runtime.Object) bool {
	var changed bool
	for _, mutate := range transforms {
		if mutate(obj) {
			changed = true
		}
	}
	return changed
}

// TransformValue applies JSON transforms to a record value.
func (ts Transforms) TransformValue(key, value []byte) ([]byte, error) {
	recordKey := string(key)
	matched := ts.matching(recordKey)
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

// TransformObject applies protobuf transforms to a decoded Kubernetes object.
func (ts Transforms) TransformObject(key string, obj runtime.Object) bool {
	var changed bool
	for _, t := range ts.matching(key) {
		if t.applyProtobuf(key, obj) {
			changed = true
		}
	}
	return changed
}

// HasTransform reports whether any seed transform is configured for key.
func (ts Transforms) HasTransform(key string) bool {
	return len(ts.matching(key)) > 0
}

// matching returns all transforms whose key selector matches key.
func (ts Transforms) matching(key string) []Transform {
	var matched []Transform
	for _, t := range ts {
		if t.matchesKey(key) {
			matched = append(matched, t)
		}
	}
	return matched
}
