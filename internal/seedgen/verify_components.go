// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/podplane/seedgen/pkg/pipeline"
)

// componentsManifest models the image list in a components manifest file.
type componentsManifest struct {
	Components struct {
		Images []struct {
			Image string `json:"image"`
		} `json:"images"`
	} `json:"components"`
}

// componentsVerifier checks emitted seed images against a components manifest.
type componentsVerifier struct {
	manifestPath string
	images       map[string]struct{}
	missing      map[string]map[string]struct{}
}

// newComponentsVerifier loads an optional components manifest verifier.
func newComponentsVerifier(path string) (*componentsVerifier, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read components manifest %s: %w", path, err)
	}
	var manifest componentsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode components manifest %s: %w", path, err)
	}
	images := make(map[string]struct{})
	for _, item := range manifest.Components.Images {
		if item.Image == "" {
			continue
		}
		images[pipeline.NormalizeImageRef(item.Image)] = struct{}{}
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("components manifest %s contains no images", path)
	}
	return &componentsVerifier{manifestPath: path, images: images, missing: make(map[string]map[string]struct{})}, nil
}

// Check records any JSON image fields in value that are absent from the
// manifest.
func (v *componentsVerifier) Check(key, value []byte) {
	var obj any
	if err := json.Unmarshal(value, &obj); err != nil {
		return
	}
	v.checkValue(string(key), obj)
}

// checkValue recursively walks decoded JSON and checks fields named image.
func (v *componentsVerifier) checkValue(key string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for field, child := range typed {
			if field == "image" {
				image, ok := child.(string)
				if ok && image != "" {
					v.checkImage(key, image)
				}
				continue
			}
			v.checkValue(key, child)
		}
	case []any:
		for _, child := range typed {
			v.checkValue(key, child)
		}
	}
}

// checkImage records image as missing for key when the manifest does not list
// it.
func (v *componentsVerifier) checkImage(key, image string) {
	if _, ok := v.images[image]; ok {
		return
	}
	keys := v.missing[image]
	if keys == nil {
		keys = make(map[string]struct{})
		v.missing[image] = keys
	}
	keys[key] = struct{}{}
}

// Err reports all missing images collected during verification.
func (v *componentsVerifier) Err() error {
	if len(v.missing) == 0 {
		return nil
	}
	images := make([]string, 0, len(v.missing))
	for image := range v.missing {
		images = append(images, image)
	}
	sort.Strings(images)
	var b strings.Builder
	fmt.Fprintf(&b, "seed contains %d image(s) missing from components manifest %s:", len(images), v.manifestPath)
	for _, image := range images {
		keys := make([]string, 0, len(v.missing[image]))
		for key := range v.missing[image] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "\n- %s", image)
		for _, key := range keys {
			fmt.Fprintf(&b, "\n  - %s", key)
		}
	}
	return fmt.Errorf("%s", b.String())
}
