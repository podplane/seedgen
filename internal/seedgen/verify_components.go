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

const helmReleaseKeyPrefix = "/registry/helm.toolkit.fluxcd.io/helmreleases/"

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
	repos        map[string]struct{}
	missing      map[string]map[string]struct{}
	mirrored     map[string]map[string]struct{}
	upstream     map[string]map[string]struct{}
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
	repos := make(map[string]struct{})
	for _, item := range manifest.Components.Images {
		if item.Image == "" {
			continue
		}
		image := pipeline.NormalizeImageRef(item.Image)
		images[image] = struct{}{}
		repo, _ := imageRepository(image)
		repos[repo] = struct{}{}
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("components manifest %s contains no images", path)
	}
	return &componentsVerifier{
		manifestPath: path,
		images:       images,
		repos:        repos,
		missing:      make(map[string]map[string]struct{}),
		mirrored:     make(map[string]map[string]struct{}),
		upstream:     make(map[string]map[string]struct{}),
	}, nil
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
	normalized := normalizeSeedImageRef(image)
	if _, ok := v.images[normalized]; ok {
		v.recordMatchedImage(key, image)
		return
	}
	repo, hasVersion := imageRepository(normalized)
	if strings.HasPrefix(key, helmReleaseKeyPrefix) && !hasVersion {
		if _, ok := v.repos[repo]; ok {
			v.recordMatchedImage(key, image)
			return
		}
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
	if len(v.missing) == 0 && (len(v.mirrored) == 0 || len(v.upstream) == 0) {
		return nil
	}
	var b strings.Builder
	if len(v.missing) > 0 {
		v.writeMissingError(&b)
	}
	if len(v.mirrored) > 0 && len(v.upstream) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		v.writeMixedMirrorError(&b)
	}
	return fmt.Errorf("%s", b.String())
}

// recordMatchedImage records whether a manifest-matched seed image used the
// local component mirror or the upstream image reference directly.
func (v *componentsVerifier) recordMatchedImage(key, image string) {
	images := v.upstream
	if strings.Contains(image, "/mirror/") {
		images = v.mirrored
	}
	keys := images[image]
	if keys == nil {
		keys = make(map[string]struct{})
		images[image] = keys
	}
	keys[key] = struct{}{}
}

// writeMissingError appends the missing-image verifier failure to b.
func (v *componentsVerifier) writeMissingError(b *strings.Builder) {
	images := make([]string, 0, len(v.missing))
	for image := range v.missing {
		images = append(images, image)
	}
	sort.Strings(images)
	fmt.Fprintf(b, "seed contains %d image(s) missing from components manifest %s:", len(images), v.manifestPath)
	for _, image := range images {
		keys := make([]string, 0, len(v.missing[image]))
		for key := range v.missing[image] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintf(b, "\n- %s", image)
		for _, key := range keys {
			fmt.Fprintf(b, "\n  - %s", key)
		}
	}
}

// writeMixedMirrorError appends the mixed mirrored/upstream verifier failure
func (v *componentsVerifier) writeMixedMirrorError(b *strings.Builder) {
	fmt.Fprintf(b, "seed contains mixed mirrored and upstream component images; regenerate from a cluster where component image refs are consistently all mirrored or all upstream")
	v.writeImageExamples(b, "mirrored", v.mirrored)
	v.writeImageExamples(b, "upstream", v.upstream)
}

// writeImageExamples appends a small deterministic set of image examples
func (v *componentsVerifier) writeImageExamples(b *strings.Builder, label string, images map[string]map[string]struct{}) {
	names := make([]string, 0, len(images))
	for image := range images {
		names = append(names, image)
	}
	sort.Strings(names)
	limit := len(names)
	if limit > 3 {
		limit = 3
	}
	fmt.Fprintf(b, "\n%s examples:", label)
	for _, image := range names[:limit] {
		keys := make([]string, 0, len(images[image]))
		for key := range images[image] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintf(b, "\n- %s", image)
		if len(keys) > 0 {
			fmt.Fprintf(b, "\n  - %s", keys[0])
		}
	}
}

// normalizeSeedImageRef strips the local component image mirror prefix before
// comparing seed images against the upstream images listed in components.json.
func normalizeSeedImageRef(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if _, rest, ok := strings.Cut(image, "/mirror/"); ok {
		return pipeline.NormalizeImageRef(rest)
	}
	return pipeline.NormalizeImageRef(image)
}

// imageRepository returns the repository portion of image and reports whether
// the image included a tag or digest.
func imageRepository(image string) (string, bool) {
	image = strings.TrimSpace(image)
	if at := strings.Index(image, "@"); at >= 0 {
		return image[:at], true
	}
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon > lastSlash {
		return image[:lastColon], true
	}
	return image, false
}
