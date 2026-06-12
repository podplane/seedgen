// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/pkg/pipeline"
)

const initialNetsyKey = "_netsy"

// WriteOptions configures seed snapshot output.
type WriteOptions struct {
	LeaderID         string
	Transforms       pipeline.Transforms
	VerifyComponents string
}

// WriteSnapshot normalises and renumbers records to look freshly created
// (Revision = 1..N, Created=true, Deleted=false, CreateRevision=Revision,
// PrevRevision=0, Version=1, no lease/dek/timestamps), then writes them as
// a Netsy snapshot file. The renumbering is required for Netsy's bootstrap
// integrity check, which enforces COUNT(records) == MAX(revision).
func WriteSnapshot(w io.Writer, records []*datafile.Record, opts WriteOptions) error {
	out, err := prepareRecordsForWrite(records, opts)
	if err != nil {
		return err
	}
	if err := datafile.WriteSnapshot(w, out, opts.LeaderID); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

// WriteRecordFiles writes one JSON file per emitted seed record under dir.
// Each file is named from the record key with slashes replaced by underscores.
func WriteRecordFiles(dir string, records []*datafile.Record, opts WriteOptions) error {
	out, err := prepareRecordsForWrite(records, opts)
	if err != nil {
		return err
	}
	if err := removeExistingRecordFiles(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create record files directory %s: %w", dir, err)
	}
	seen := make(map[string]string, len(out))
	for _, record := range out {
		key := string(record.Key)
		name := recordFileNameForKey(key)
		if existing, ok := seen[name]; ok {
			return fmt.Errorf("record file name collision for %q and %q: %s", existing, key, name)
		}
		seen[name] = key

		value, err := recordValueJSON(record.Value)
		if err != nil {
			return fmt.Errorf("decode record %s value as JSON: %w", key, err)
		}
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create record file %s: %w", path, err)
		}
		enc := json.NewEncoder(f)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}{Key: key, Value: value}); err != nil {
			_ = f.Close()
			return fmt.Errorf("write record file %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close record file %s: %w", path, err)
		}
	}
	return nil
}

// prepareRecordsForWrite orders, transforms, verifies, renumbers, and
// normalises records into the exact records emitted by seed writers.
func prepareRecordsForWrite(records []*datafile.Record, opts WriteOptions) ([]*datafile.Record, error) {
	ordered, err := orderRecordsForWrite(records)
	if err != nil {
		return nil, err
	}
	verifier, err := newComponentsVerifier(opts.VerifyComponents)
	if err != nil {
		return nil, err
	}
	out := make([]*datafile.Record, len(ordered))
	for i, record := range ordered {
		rev := int64(i + 1)
		value, err := transformValue(opts.Transforms, record.Key, record.Value)
		if err != nil {
			return nil, err
		}
		if verifier != nil {
			verifier.Check(record.Key, value)
		}
		out[i] = &datafile.Record{
			Revision:       rev,
			Key:            record.Key,
			Created:        true,
			Deleted:        false,
			CreateRevision: rev,
			PrevRevision:   0,
			Version:        1,
			Lease:          0,
			Dek:            0,
			Value:          value,
			CreatedAt:      nil,
			CompactedAt:    nil,
			LeaderID:       opts.LeaderID,
			ReplicatedAt:   nil,
		}
	}
	if verifier != nil {
		if err := verifier.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// removeExistingRecordFiles removes stale JSON record files from dir while
// leaving subdirectories and non-JSON files untouched.
func removeExistingRecordFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read record files directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale record file %s: %w", path, err)
		}
	}
	return nil
}

// recordFileNameForKey returns the JSON file name for key by replacing forward
// slashes with underscores and appending .json.
func recordFileNameForKey(key string) string {
	return strings.ReplaceAll(key, "/", "_") + ".json"
}

// recordValueJSON returns value as JSON suitable for embedding in a record file.
// Empty values are emitted as null for Netsy's internal records.
func recordValueJSON(value []byte) (json.RawMessage, error) {
	if len(bytes.TrimSpace(value)) == 0 {
		return json.RawMessage(`null`), nil
	}
	var raw json.RawMessage
	if err := json.Unmarshal(value, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// orderRecordsForWrite preserves input order except for Netsy's initial
// internal record, which must be emitted first so it receives revision 1 after
// renumbering.
func orderRecordsForWrite(records []*datafile.Record) ([]*datafile.Record, error) {
	initialIndex := -1
	for i, record := range records {
		if string(record.Key) != initialNetsyKey {
			continue
		}
		if initialIndex != -1 {
			return nil, fmt.Errorf("multiple %s records in seed output", initialNetsyKey)
		}
		initialIndex = i
	}
	if initialIndex <= 0 {
		return records, nil
	}

	ordered := make([]*datafile.Record, 0, len(records))
	ordered = append(ordered, records[initialIndex])
	ordered = append(ordered, records[:initialIndex]...)
	ordered = append(ordered, records[initialIndex+1:]...)
	return ordered, nil
}
