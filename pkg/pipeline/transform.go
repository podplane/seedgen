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

// Transform is one key selector plus optional JSON and protobuf mutations.
type Transform interface {
	matchesKey(key string) bool
	applyJSON(key string, obj map[string]any) bool
	applyProtobuf(key string, obj runtime.Object) bool
}

// Transforms is an ordered list of seed output transforms.
type Transforms []Transform

// PrefixTransform applies mutations to keys under Prefix whose object kind and
// API version match the transform metadata.
type PrefixTransform struct {
	Prefix         string
	APIVersion     string
	APIPrefix      string
	Kind           string
	MutateJSON     func(map[string]any) bool
	MutateProtobuf func(runtime.Object) bool
}

// matchesKey reports whether a registry key belongs to this prefix transform.
func (t PrefixTransform) matchesKey(key string) bool {
	return strings.HasPrefix(key, t.Prefix)
}

// applyJSON runs a JSON transform when the object has the API version and kind
// declared by the transform table.
func (t PrefixTransform) applyJSON(_ string, obj map[string]any) bool {
	if t.MutateJSON == nil {
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
	return t.MutateJSON(obj)
}

// applyProtobuf runs a typed-object transform when the decoded Kubernetes
// object has the API version and kind declared by the transform table.
func (t PrefixTransform) applyProtobuf(_ string, obj runtime.Object) bool {
	if t.MutateProtobuf == nil {
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
	return t.MutateProtobuf(obj)
}

// KeyTransform applies mutations to one exact key.
type KeyTransform struct {
	Key            string
	MutateJSON     func(map[string]any) bool
	MutateProtobuf func(runtime.Object) bool
}

// matchesKey reports whether a registry key exactly matches this key transform.
func (t KeyTransform) matchesKey(key string) bool {
	return key == t.Key
}

// applyJSON runs a JSON transform when the storage key exactly matches the key
// declared by the transform table.
func (t KeyTransform) applyJSON(key string, obj map[string]any) bool {
	if t.MutateJSON == nil || key != t.Key {
		return false
	}
	return t.MutateJSON(obj)
}

// applyProtobuf runs a typed-object transform when the storage key exactly
// matches the key declared by the transform table.
func (t KeyTransform) applyProtobuf(key string, obj runtime.Object) bool {
	if t.MutateProtobuf == nil || key != t.Key {
		return false
	}
	return t.MutateProtobuf(obj)
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
