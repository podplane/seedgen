// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/internal/buildvars"
	"github.com/podplane/seedgen/internal/seedgen"
	"github.com/podplane/seedgen/internal/seedgen/defaults"
	"github.com/podplane/seedgen/pkg/pipeline"
	"github.com/spf13/cobra"
)

type options struct {
	input       string
	cluster     string
	output      string
	name        string
	leaderID    string
	includeFile string
	excludeFile string
	expect      string
	verify      string
	dryRun      bool
}

// NewRootCmd builds the root cobra command for the seedgen CLI. When a custom
// pipeline is provided, it replaces the built-in Podplane pipeline.
func NewRootCmd(pipelines ...pipeline.Pipeline) *cobra.Command {
	activePipeline := defaults.Pipeline()
	if len(pipelines) > 0 {
		activePipeline = pipelines[0].WithDefaults()
	}
	var opts options
	cmd := &cobra.Command{
		Use:           "seedgen",
		Short:         "Produce a Podplane seed from a running Podplane cluster's on-disk Netsy state",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long: `Podplane Seed Generator

seedgen reads the latest snapshot and chunk files written by Netsy to its
object-storage directory, replays them into the current cluster state, applies
include and exclude rules, renumbers the surviving records' revisions
sequentially, and writes the result as a fresh Netsy .netsy snapshot file
suitable for use as a Podplane seed.`,
		Version: buildvars.BuildVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, opts, activePipeline)
		},
	}
	cmd.Flags().StringVarP(&opts.input, "input", "i", "", "Directory containing snapshots/ and chunks/ (a Netsy bucket root on disk; overrides --cluster)")
	cmd.Flags().StringVar(&opts.cluster, "cluster", "default", "Local Podplane cluster id; shortcut for ~/.podplane/data/s3/buckets/<id>-netsy")
	cmd.Flags().StringVarP(&opts.output, "output", "o", ".", "Directory to write the .netsy file and key reports")
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Seed name used for <name>.netsy and reports/<name>/")
	cmd.Flags().StringVar(&opts.leaderID, "leader-id", "seed", "LeaderID stamped on the output snapshot")
	cmd.Flags().StringVar(&opts.includeFile, "include", "", "Path to a JSONC include file overriding the pipeline default")
	cmd.Flags().StringVar(&opts.excludeFile, "exclude", "", "Path to a JSONC exclude file overriding the pipeline default")
	cmd.Flags().StringVar(&opts.expect, "expect", "", "Check for expected records based on the type of seed. Defaults to --name when name is recommended or minimal. Options: recommended, minimal, or none")
	cmd.Flags().StringVar(&opts.verify, "verify-components", "", "Path to components.json manifest; fail if emitted seed images are absent from it")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Run the full pipeline and write key reports but do not write the output file")
	return cmd
}

// run executes the root command using activePipeline for filtering,
// transforming, and expectation checks.
func run(cmd *cobra.Command, opts options, activePipeline pipeline.Pipeline) error {
	expect, reportName, err := resolveExpectAndReportName(cmd, opts)
	if err != nil {
		return err
	}
	outputPath := resolveOutputPath(opts)
	inputDir, err := resolveInputDir(cmd, opts)
	if err != nil {
		return err
	}
	include, err := loadRules(opts.includeFile, activePipeline.IncludeRules)
	if err != nil {
		return fmt.Errorf("include rules: %w", err)
	}
	exclude, err := loadRules(opts.excludeFile, activePipeline.ExcludeRules)
	if err != nil {
		return fmt.Errorf("exclude rules: %w", err)
	}
	records, err := seedgen.ReadAll(inputDir)
	if err != nil {
		return err
	}
	current := seedgen.CurrentState(records)
	kept, includedKeys, excludedKeys, ignoredKeys := classifyKeys(current, include, exclude)
	if err := activePipeline.CheckExpected(expect, kept); err != nil {
		return err
	}
	reportsDir := resolveReportsDir(opts, reportName)
	if err := writeKeyReports(reportsDir, includedKeys, excludedKeys, ignoredKeys); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d total read\nwrote reports to %s:\n- %d included\n- %d excluded\n- %d ignored\n", len(records), reportsDir, len(includedKeys), len(excludedKeys), len(ignoredKeys))

	if opts.dryRun {
		if opts.verify != "" {
			writeOpts := seedgen.WriteOptions{LeaderID: opts.leaderID, Transforms: activePipeline.Transforms, VerifyComponents: opts.verify}
			if err := seedgen.WriteSnapshot(io.Discard, kept, writeOpts); err != nil {
				return err
			}
		}
		fmt.Fprintln(os.Stderr, "dry run: skipping snapshot write")
		return nil
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output %s: %w", outputPath, err)
	}
	defer f.Close()
	writeOpts := seedgen.WriteOptions{LeaderID: opts.leaderID, Transforms: activePipeline.Transforms, VerifyComponents: opts.verify}
	if err := seedgen.WriteSnapshot(f, kept, writeOpts); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", outputPath)
	return nil
}

// resolveExpectAndReportName returns the --expect value and report directory
// name implied by the CLI flags. The seed name drives --expect for the built-in
// published seed names; custom names require an explicit --expect value.
func resolveExpectAndReportName(cmd *cobra.Command, opts options) (expect, reportName string, err error) {
	expectSet := cmd.Flags().Changed("expect")
	if opts.name == "" {
		if !opts.dryRun {
			return "", "", fmt.Errorf("--name is required")
		}
		if !expectSet {
			return "", "", fmt.Errorf("--expect is required when --dry-run is used without --name")
		}
		return opts.expect, opts.expect, nil
	}
	if filepath.Base(opts.name) != opts.name {
		return "", "", fmt.Errorf("--name must be a file name, not a path")
	}
	if expectSet {
		return opts.expect, opts.name, nil
	}
	if opts.name == "recommended" || opts.name == "minimal" {
		return opts.name, opts.name, nil
	}
	return "", "", fmt.Errorf("--expect is required when --name is not recommended or minimal")
}

// resolveOutputPath returns the snapshot path derived from the output directory
// and seed name.
func resolveOutputPath(opts options) string {
	return filepath.Join(opts.output, opts.name+".netsy")
}

// resolveReportsDir returns the directory used for key reports.
func resolveReportsDir(opts options, reportName string) string {
	return filepath.Join(opts.output, "reports", reportName)
}

// classifyKeys partitions current records according to the include and exclude
// rules used for seed output. Included keys are the records that will be
// written to the seed, excluded keys matched include rules but were dropped by
// exclude rules, and ignored keys did not match include rules at all.
func classifyKeys(records []*datafile.Record, include, exclude *pipeline.Rules) (kept []*datafile.Record, includedKeys, excludedKeys, ignoredKeys []string) {
	kept = make([]*datafile.Record, 0, len(records))
	includedKeys = make([]string, 0, len(records))
	excludedKeys = make([]string, 0, len(records))
	ignoredKeys = make([]string, 0, len(records))
	for _, record := range records {
		key := string(record.Key)
		if !include.Matches(key) {
			ignoredKeys = append(ignoredKeys, key)
			continue
		}
		if exclude.Matches(key) {
			excludedKeys = append(excludedKeys, key)
			continue
		}
		kept = append(kept, record)
		includedKeys = append(includedKeys, key)
	}
	return kept, includedKeys, excludedKeys, ignoredKeys
}

// writeKeyReports writes the three key report files that explain how the current
// record set was classified by the include and exclude rules.
func writeKeyReports(dir string, includedKeys, excludedKeys, ignoredKeys []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create key reports directory %s: %w", dir, err)
	}
	for name, keys := range map[string][]string{
		"included.txt": includedKeys,
		"excluded.txt": excludedKeys,
		"ignored.txt":  ignoredKeys,
	} {
		if err := writeKeyReport(filepath.Join(dir, name), keys); err != nil {
			return err
		}
	}
	return nil
}

// writeKeyReport writes one alphanumerically sorted key per line to path,
// replacing any existing file.
func writeKeyReport(path string, keys []string) error {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create key report %s: %w", path, err)
	}
	defer f.Close()
	for _, key := range sorted {
		if _, err := fmt.Fprintln(f, key); err != nil {
			return fmt.Errorf("write key report %s: %w", path, err)
		}
	}
	return nil
}

// resolveInputDir returns the directory to read records from.
//
// When --input is set it wins (and --cluster must not also be set
// explicitly). Otherwise the bucket path is derived from --cluster, which
// defaults to "default" to match the Podplane local-cluster convention.
func resolveInputDir(cmd *cobra.Command, opts options) (string, error) {
	inputSet := cmd.Flags().Changed("input")
	clusterSet := cmd.Flags().Changed("cluster")
	if inputSet && clusterSet {
		return "", fmt.Errorf("--input and --cluster are mutually exclusive")
	}
	if inputSet {
		return opts.input, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".podplane", "data", "s3", "buckets", opts.cluster+"-netsy"), nil
}

// loadRules returns fallback rules when path is empty, or the file's parsed
// rules otherwise.
func loadRules(path string, fallback func() (*pipeline.Rules, error)) (*pipeline.Rules, error) {
	if path == "" {
		return fallback()
	}
	return pipeline.LoadRulesFile(path)
}
