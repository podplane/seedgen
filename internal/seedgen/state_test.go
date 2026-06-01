// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

func TestCurrentStateAppliesTombstonesAndLastWriteWins(t *testing.T) {
	t.Parallel()
	records := []*datafile.Record{
		{Revision: 1, Key: []byte("/a"), Value: []byte("v1")},
		{Revision: 2, Key: []byte("/b"), Value: []byte("v1")},
		{Revision: 3, Key: []byte("/a"), Value: []byte("v2")},
		{Revision: 4, Key: []byte("/b"), Deleted: true},
		{Revision: 5, Key: []byte("/c"), Value: []byte("v1")},
	}
	got := CurrentState(records)
	if len(got) != 2 {
		t.Fatalf("want 2 surviving records, got %d", len(got))
	}
	if string(got[0].Key) != "/a" || string(got[0].Value) != "v2" {
		t.Fatalf("want /a=v2 first, got key=%s value=%s", got[0].Key, got[0].Value)
	}
	if string(got[1].Key) != "/c" || string(got[1].Value) != "v1" {
		t.Fatalf("want /c=v1 second, got key=%s value=%s", got[1].Key, got[1].Value)
	}
}

func TestCurrentStateUnordered(t *testing.T) {
	t.Parallel()
	// Records arrive out of revision order; CurrentState must still apply them
	// in the right order so the tombstone wins.
	records := []*datafile.Record{
		{Revision: 2, Key: []byte("/a"), Deleted: true},
		{Revision: 1, Key: []byte("/a"), Value: []byte("v1")},
	}
	got := CurrentState(records)
	if len(got) != 0 {
		t.Fatalf("want 0 surviving records, got %d", len(got))
	}
}
