// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"sort"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

// CurrentState flattens a record history into the current value per key.
// Records are processed in revision order. A non-deleted record sets or
// replaces the current value for its key; a deleted record (tombstone) drops
// the key entirely. The returned slice contains one record per surviving key,
// sorted by the original revision of that surviving record.
func CurrentState(records []*datafile.Record) []*datafile.Record {
	sorted := make([]*datafile.Record, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Revision < sorted[j].Revision })

	current := map[string]*datafile.Record{}
	for _, record := range sorted {
		key := string(record.Key)
		if record.Deleted {
			delete(current, key)
			continue
		}
		current[key] = record
	}

	survivors := make([]*datafile.Record, 0, len(current))
	for _, record := range current {
		survivors = append(survivors, record)
	}
	sort.Slice(survivors, func(i, j int) bool { return survivors[i].Revision < survivors[j].Revision })
	return survivors
}
