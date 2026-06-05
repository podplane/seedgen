// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	_ "embed"

	"github.com/podplane/seedgen/internal/seedgen"
	"github.com/podplane/seedgen/pkg/pipeline"
)

//go:embed include.jsonc
var defaultIncludeJSONC []byte

//go:embed exclude.jsonc
var defaultExcludeJSONC []byte

// Pipeline returns the built-in Podplane seed generation pipeline.
func Pipeline() pipeline.Pipeline {
	return pipeline.Pipeline{
		IncludeRules: func() (*pipeline.Rules, error) {
			return pipeline.LoadRulesBytes(defaultIncludeJSONC, "<embedded include.jsonc>")
		},
		ExcludeRules: func() (*pipeline.Rules, error) {
			return pipeline.LoadRulesBytes(defaultExcludeJSONC, "<embedded exclude.jsonc>")
		},
		Transforms:    Transforms,
		CheckExpected: seedgen.CheckExpected,
	}
}
