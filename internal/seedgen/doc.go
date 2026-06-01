// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

// Package seedgen implements the core seedgen pipeline: read a Netsy
// object-storage directory (snapshot + chunks), flatten the record history
// into the current value per key, apply include and exclude rules, renumber
// surviving records to 1..N, and write the result as a Netsy snapshot file.
//
// The flattening step is equivalent to a full LSM-style compaction (drop
// history, drop tombstones), but more aggressive than Netsy's own
// compaction. Netsy compaction preserves rows for revision contiguity and
// only nulls out their values; seedgen discards the old revisions and
// tombstoned keys entirely, then renumbers what remains.
package seedgen
