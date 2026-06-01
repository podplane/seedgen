// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

func TestReadAllSnapshotAndChunks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSnapshotFixture(t, dir, 10, []*datafile.Record{
		{Revision: 1, Key: []byte("/a"), Value: []byte("v1"), Created: true, CreateRevision: 1, Version: 1, LeaderID: "fixture"},
		{Revision: 10, Key: []byte("/b"), Value: []byte("v1"), Created: true, CreateRevision: 10, Version: 1, LeaderID: "fixture"},
	})
	writeChunkFixture(t, dir, 11, []*datafile.Record{
		{Revision: 11, Key: []byte("/c"), Value: []byte("v1"), Created: true, CreateRevision: 11, Version: 1, LeaderID: "fixture"},
	})
	// Stale chunk below latest snapshot revision must be ignored.
	writeChunkFixture(t, dir, 5, []*datafile.Record{
		{Revision: 5, Key: []byte("/d"), Value: []byte("v1"), Created: true, CreateRevision: 5, Version: 1, LeaderID: "fixture"},
	})

	got, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	wantKeys := []string{"/a", "/b", "/c"}
	if len(got) != len(wantKeys) {
		t.Fatalf("want %d records, got %d", len(wantKeys), len(got))
	}
	for i, want := range wantKeys {
		if string(got[i].Key) != want {
			t.Errorf("record %d: key=%s, want %s", i, got[i].Key, want)
		}
	}
}

func TestReadAllChunksOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// No snapshot at all; chunks should all be read regardless of revision.
	writeChunkFixture(t, dir, 1, []*datafile.Record{
		{Revision: 1, Key: []byte("/a"), Value: []byte("v1"), Created: true, CreateRevision: 1, Version: 1, LeaderID: "fixture"},
	})
	writeChunkFixture(t, dir, 7, []*datafile.Record{
		{Revision: 7, Key: []byte("/b"), Value: []byte("v1"), Created: true, CreateRevision: 7, Version: 1, LeaderID: "fixture"},
	})

	got, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	wantKeys := []string{"/a", "/b"}
	if len(got) != len(wantKeys) {
		t.Fatalf("want %d records, got %d", len(wantKeys), len(got))
	}
	for i, want := range wantKeys {
		if string(got[i].Key) != want {
			t.Errorf("record %d: key=%s, want %s", i, got[i].Key, want)
		}
	}
}

func TestReadAllMissingDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	got, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll on empty dir: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 records, got %d", len(got))
	}
}

func writeSnapshotFixture(t *testing.T, dir string, revision int64, records []*datafile.Record) {
	t.Helper()
	snapshotDir := filepath.Join(dir, "snapshots")
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		t.Fatalf("mkdir snapshots: %v", err)
	}
	path := filepath.Join(snapshotDir, fmt.Sprintf("%019d.netsy", revision))
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	defer f.Close()
	if err := datafile.WriteSnapshot(f, records, "fixture"); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
}

func writeChunkFixture(t *testing.T, dir string, revision int64, records []*datafile.Record) {
	t.Helper()
	partition := revision % 10000
	chunkDir := filepath.Join(dir, "chunks", fmt.Sprintf("%04d", partition))
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		t.Fatalf("mkdir chunks: %v", err)
	}
	path := filepath.Join(chunkDir, fmt.Sprintf("%019d.netsy", revision))
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create chunk: %v", err)
	}
	defer f.Close()
	if err := datafile.WriteChunk(f, records, "fixture"); err != nil {
		t.Fatalf("WriteChunk: %v", err)
	}
}
