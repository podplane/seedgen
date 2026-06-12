// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import "github.com/netsy-dev/netsy/pkg/datafile"

// Pipeline groups the include rules, exclude rules, transforms, and optional
// expectation check used by a seedgen command.
type Pipeline struct {
	IncludeRules  func(string) (*Rules, error)
	ExcludeRules  func(string) (*Rules, error)
	Transforms    func(string) Transforms
	CheckExpected func(string, []*datafile.Record) error
}

// WithDefaults fills unset pipeline hooks with no-op implementations.
func (p Pipeline) WithDefaults() Pipeline {
	if p.IncludeRules == nil {
		p.IncludeRules = func(string) (*Rules, error) { return &Rules{}, nil }
	}
	if p.ExcludeRules == nil {
		p.ExcludeRules = func(string) (*Rules, error) { return &Rules{}, nil }
	}
	if p.Transforms == nil {
		p.Transforms = func(string) Transforms { return nil }
	}
	if p.CheckExpected == nil {
		p.CheckExpected = func(string, []*datafile.Record) error { return nil }
	}
	return p
}
