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

//go:embed platform/include.jsonc
var platformIncludeJSONC []byte

//go:embed platform/exclude.jsonc
var platformExcludeJSONC []byte

//go:embed agentic/include.jsonc
var agenticIncludeJSONC []byte

//go:embed agentic/exclude.jsonc
var agenticExcludeJSONC []byte

// Pipeline returns the built-in Podplane seed generation pipeline.
func Pipeline() pipeline.Pipeline {
	return pipeline.Pipeline{
		IncludeRules: includeRules,
		ExcludeRules: excludeRules,
		Transforms: func(profile string) pipeline.Transforms {
			switch profile {
			case "minimal":
				return MinimalTransforms
			case "platform":
				return PlatformTransforms
			case "agentic":
				return AgenticTransforms
			default:
				return Transforms
			}
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
	case "platform", "agentic":
		minimal, err := pipeline.LoadRulesBytes(minimalIncludeJSONC, "<embedded minimal/include.jsonc>")
		if err != nil {
			return nil, err
		}
		platform, err := pipeline.LoadRulesBytes(platformIncludeJSONC, "<embedded platform/include.jsonc>")
		if err != nil {
			return nil, err
		}
		if profile == "agentic" {
			agentic, err := pipeline.LoadRulesBytes(agenticIncludeJSONC, "<embedded agentic/include.jsonc>")
			if err != nil {
				return nil, err
			}
			return pipeline.MergeRules(common, minimal, platform, agentic), nil
		}
		return pipeline.MergeRules(common, minimal, platform), nil
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
	case "platform", "agentic":
		shared, err := pipeline.LoadRulesBytes(agenticExcludeJSONC, "<embedded agentic/exclude.jsonc>")
		if err != nil {
			return nil, err
		}
		if profile == "platform" {
			platform, err := pipeline.LoadRulesBytes(platformExcludeJSONC, "<embedded platform/exclude.jsonc>")
			if err != nil {
				return nil, err
			}
			return pipeline.MergeRules(common, shared, platform), nil
		}
		return pipeline.MergeRules(common, shared), nil
	case "", "none":
		return common, nil
	default:
		return nil, fmt.Errorf("unknown seed profile %q", profile)
	}
}
