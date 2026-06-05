// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/pkg/pipeline"
)

// Filter returns the subset of records whose keys match the include rules
// and do not match the exclude rules. Input order is preserved.
func Filter(records []*datafile.Record, include, exclude *pipeline.Rules) []*datafile.Record {
	kept := make([]*datafile.Record, 0, len(records))
	for _, record := range records {
		key := string(record.Key)
		if !include.Matches(key) {
			continue
		}
		if exclude.Matches(key) {
			continue
		}
		kept = append(kept, record)
	}
	return kept
}
