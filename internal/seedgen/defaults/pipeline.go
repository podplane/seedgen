// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	_ "embed"
	"fmt"

	"github.com/podplane/seedgen/internal/seedgen"
	"github.com/podplane/seedgen/pkg/pipeline"
)

//go:embed include.jsonc
var defaultIncludeJSONC []byte

//go:embed exclude.jsonc
var defaultExcludeJSONC []byte

//go:embed minimal/include.jsonc
var minimalIncludeJSONC []byte

//go:embed minimal/exclude.jsonc
var minimalExcludeJSONC []byte

//go:embed recommended/include.jsonc
var recommendedIncludeJSONC []byte

//go:embed recommended/exclude.jsonc
var recommendedExcludeJSONC []byte

// Pipeline returns the built-in Podplane seed generation pipeline.
func Pipeline() pipeline.Pipeline {
	return pipeline.Pipeline{
		IncludeRules: includeRules,
		ExcludeRules: excludeRules,
		Transforms: func(profile string) pipeline.Transforms {
			if profile == "minimal" {
				return MinimalTransforms
			}
			return Transforms
		},
		CheckExpected: seedgen.CheckExpected,
	}
}

// includeRules returns the built-in include rules for profile.
func includeRules(profile string) (*pipeline.Rules, error) {
	common, err := pipeline.LoadRulesBytes(defaultIncludeJSONC, "<embedded include.jsonc>")
	if err != nil {
		return nil, err
	}
	switch profile {
	case "minimal":
		minimal, err := pipeline.LoadRulesBytes(minimalIncludeJSONC, "<embedded minimal/include.jsonc>")
		if err != nil {
			return nil, err
		}
		return pipeline.MergeRules(common, minimal), nil
	case "recommended":
		minimal, err := pipeline.LoadRulesBytes(minimalIncludeJSONC, "<embedded minimal/include.jsonc>")
		if err != nil {
			return nil, err
		}
		recommended, err := pipeline.LoadRulesBytes(recommendedIncludeJSONC, "<embedded recommended/include.jsonc>")
		if err != nil {
			return nil, err
		}
		return pipeline.MergeRules(common, minimal, recommended), nil
	case "", "none":
		return common, nil
	default:
		return nil, fmt.Errorf("unknown seed profile %q", profile)
	}
}

// excludeRules returns the built-in exclude rules for profile.
func excludeRules(profile string) (*pipeline.Rules, error) {
	common, err := pipeline.LoadRulesBytes(defaultExcludeJSONC, "<embedded exclude.jsonc>")
	if err != nil {
		return nil, err
	}
	switch profile {
	case "minimal":
		minimal, err := pipeline.LoadRulesBytes(minimalExcludeJSONC, "<embedded minimal/exclude.jsonc>")
		if err != nil {
			return nil, err
		}
		return pipeline.MergeRules(common, minimal), nil
	case "recommended":
		recommended, err := pipeline.LoadRulesBytes(recommendedExcludeJSONC, "<embedded recommended/exclude.jsonc>")
		if err != nil {
			return nil, err
		}
		return pipeline.MergeRules(common, recommended), nil
	case "", "none":
		return common, nil
	default:
		return nil, fmt.Errorf("unknown seed profile %q", profile)
	}
}
