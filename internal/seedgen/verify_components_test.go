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

// TestWriteSnapshotFailsForMissingComponentsManifestImage verifies that seed
// images must exactly match the normalized manifest set.
func TestWriteSnapshotFailsForMissingComponentsManifestImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "components.json")
	if err := os.WriteFile(manifestPath, []byte(`{"components":{"images":[{"image":"docker.io/library/caddy:2"}]}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/registry/deployments/default/caddy"), Value: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"containers":[{"image":"caddy:2"}]}}}}`)},
	}
	var buf bytes.Buffer
	err := WriteSnapshot(&buf, records, WriteOptions{LeaderID: "seed", VerifyComponents: manifestPath})
	if err == nil {
		t.Fatalf("WriteSnapshot succeeded with an unnormalized seed image missing from manifest")
	}
	if !strings.Contains(err.Error(), "caddy:2") || !strings.Contains(err.Error(), "/registry/deployments/default/caddy") {
		t.Fatalf("WriteSnapshot error = %v, want missing image and key", err)
	}
}
