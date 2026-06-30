// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

// TestWriteSnapshotVerifiesComponentsManifest verifies that manifest images are
// normalized before comparison with emitted seed images.
func TestWriteSnapshotVerifiesComponentsManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "components.json")
	if err := os.WriteFile(manifestPath, []byte(`{"components":{"images":[{"image":"caddy:2"}]}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/deployments/default/caddy"), Value: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"containers":[{"image":"docker.io/library/caddy:2"}]}}}}`)},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, records, WriteOptions{LeaderID: "seed", VerifyComponents: manifestPath}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
}

// TestWriteSnapshotVerifiesMirroredComponentsManifestImage verifies that image
// mirror prefixes are stripped before comparison with upstream manifest images.
func TestWriteSnapshotVerifiesMirroredComponentsManifestImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "components.json")
	if err := os.WriteFile(manifestPath, []byte(`{"components":{"images":[{"image":"quay.io/cilium/cilium:v1.16.3@sha256:62d2a09bbef840a46099ac4c69421c90f84f28d018d479749049011329aa7f28"}]}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/daemonsets/platform-cilium/cilium"), Value: []byte(`{"apiVersion":"apps/v1","kind":"DaemonSet","spec":{"template":{"spec":{"containers":[{"image":"default-registry.local/mirror/quay.io/cilium/cilium:v1.16.3@sha256:62d2a09bbef840a46099ac4c69421c90f84f28d018d479749049011329aa7f28"}]}}}}`)},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, records, WriteOptions{LeaderID: "seed", VerifyComponents: manifestPath}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
}

// TestWriteSnapshotVerifiesRepositoryOnlyComponentsManifestImage verifies that
// repository-only Helm value image fields are accepted when the manifest lists
// a tagged image for the same repository.
func TestWriteSnapshotVerifiesRepositoryOnlyComponentsManifestImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "components.json")
	if err := os.WriteFile(manifestPath, []byte(`{"components":{"images":[{"image":"ghcr.io/fluxcd/helm-controller:v1.5.3"}]}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-fluxcd/fluxcd"), Value: []byte(`{"apiVersion":"helm.toolkit.fluxcd.io/v2","kind":"HelmRelease","spec":{"values":{"flux2":{"helmController":{"image":"default-registry.local/mirror/ghcr.io/fluxcd/helm-controller"}}}}}`)},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, records, WriteOptions{LeaderID: "seed", VerifyComponents: manifestPath}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
}

// TestWriteSnapshotFailsForMissingComponentsManifestImage verifies that seed
// images must be present in the normalized manifest set.
func TestWriteSnapshotFailsForMissingComponentsManifestImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "components.json")
	if err := os.WriteFile(manifestPath, []byte(`{"components":{"images":[{"image":"docker.io/library/caddy:2"}]}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/deployments/default/caddy"), Value: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"containers":[{"image":"busybox:1.37"}]}}}}`)},
	}
	var buf bytes.Buffer
	err := WriteSnapshot(&buf, records, WriteOptions{LeaderID: "seed", VerifyComponents: manifestPath})
	if err == nil {
		t.Fatalf("WriteSnapshot succeeded with a seed image missing from manifest")
	}
	if !strings.Contains(err.Error(), "busybox:1.37") || !strings.Contains(err.Error(), "/registry/deployments/default/caddy") {
		t.Fatalf("WriteSnapshot error = %v, want missing image and key", err)
	}
}
