// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

// fileEntry describes one snapshot or chunk file on disk along with its
// parsed revision.
type fileEntry struct {
	path     string
	revision int64
}

// ReadAll returns every record contained in the latest snapshot under
// inputDir followed by every chunk above that snapshot's revision, in
// revision order. inputDir must contain a snapshots/ and/or chunks/
// subdirectory as written by Netsy's datastore package.
//
// When no snapshot file is present every chunk is read. When a snapshot is
// present, only chunks with a revision strictly greater than that snapshot's
// revision are read (older chunks are already covered by the snapshot and
// would be redundant). When neither subdirectory exists the result is empty.
func ReadAll(inputDir string) ([]*datafile.Record, error) {
	snapshot, err := latestSnapshot(inputDir)
	if err != nil {
		return nil, err
	}
	var fromRevision int64
	var records []*datafile.Record
	if snapshot != nil {
		fromRevision = snapshot.revision
		snapRecords, err := readSnapshotFile(snapshot.path)
		if err != nil {
			return nil, err
		}
		records = append(records, snapRecords...)
	}
	chunks, err := chunksAbove(inputDir, fromRevision)
	if err != nil {
		return nil, err
	}
	for _, chunk := range chunks {
		chunkRecords, err := readChunkFile(chunk.path)
		if err != nil {
			return nil, err
		}
		records = append(records, chunkRecords...)
	}
	return records, nil
}

// latestSnapshot returns the snapshot file with the highest revision under
// inputDir/snapshots, or nil when the directory is missing or empty.
func latestSnapshot(inputDir string) (*fileEntry, error) {
	dir := filepath.Join(inputDir, "snapshots")
	entries, err := listNetsyFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].revision > entries[j].revision })
	latest := entries[0]
	return &latest, nil
}

// chunksAbove returns every chunk file under inputDir/chunks whose revision
// is strictly greater than fromRevision, sorted oldest first.
func chunksAbove(inputDir string, fromRevision int64) ([]fileEntry, error) {
	dir := filepath.Join(inputDir, "chunks")
	entries, err := listNetsyFiles(dir)
	if err != nil {
		return nil, err
	}
	var kept []fileEntry
	for _, entry := range entries {
		if entry.revision > fromRevision {
			kept = append(kept, entry)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].revision < kept[j].revision })
	return kept, nil
}

// listNetsyFiles returns every .netsy file under root with a parsed
// revision derived from its filename. A missing root directory is treated
// as empty rather than an error so callers can run against directories
// that have only a snapshot or only chunks.
func listNetsyFiles(root string) ([]fileEntry, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	var entries []fileEntry
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".netsy") {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ".netsy")
		revision, err := strconv.ParseInt(base, 10, 64)
		if err != nil {
			return nil
		}
		entries = append(entries, fileEntry{path: path, revision: revision})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk %s: %w", root, walkErr)
	}
	return entries, nil
}

// readSnapshotFile opens a snapshot file and returns its records.
func readSnapshotFile(path string) ([]*datafile.Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open snapshot %s: %w", path, err)
	}
	defer f.Close()
	records, err := datafile.ReadSnapshot(f)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	return records, nil
}

// readChunkFile opens a chunk file and returns its records.
func readChunkFile(path string) ([]*datafile.Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open chunk %s: %w", path, err)
	}
	defer f.Close()
	records, err := datafile.ReadChunk(f)
	if err != nil {
		return nil, fmt.Errorf("read chunk %s: %w", path, err)
	}
	return records, nil
}
