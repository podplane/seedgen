// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"fmt"
	"io"

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
	ordered, err := orderRecordsForWrite(records)
	if err != nil {
		return err
	}
	verifier, err := newComponentsVerifier(opts.VerifyComponents)
	if err != nil {
		return err
	}
	out := make([]*datafile.Record, len(ordered))
	for i, record := range ordered {
		rev := int64(i + 1)
		value, err := transformValue(opts.Transforms, record.Key, record.Value)
		if err != nil {
			return err
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
			return err
		}
	}
	if err := datafile.WriteSnapshot(w, out, opts.LeaderID); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
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
