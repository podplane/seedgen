// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"fmt"
	"io"

	"github.com/netsy-dev/netsy/pkg/datafile"

	"github.com/podplane/seedgen/internal/seedgen/defaults"
)

// WriteSnapshot normalises and renumbers records to look freshly created
// (Revision = 1..N, Created=true, Deleted=false, CreateRevision=Revision,
// PrevRevision=0, Version=1, no lease/dek/timestamps), then writes them as
// a Netsy snapshot file. The renumbering is required for Netsy's bootstrap
// integrity check, which enforces COUNT(records) == MAX(revision).
func WriteSnapshot(w io.Writer, records []*datafile.Record, leaderID string) error {
	out := make([]*datafile.Record, len(records))
	for i, record := range records {
		rev := int64(i + 1)
		value, err := defaults.TransformValue(record.Key, record.Value)
		if err != nil {
			return err
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
			LeaderID:       leaderID,
			ReplicatedAt:   nil,
		}
	}
	if err := datafile.WriteSnapshot(w, out, leaderID); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}
