// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/internal/buildvars"
	"github.com/podplane/seedgen/internal/seedgen"
	"github.com/spf13/cobra"
)

type options struct {
	input       string
	cluster     string
	output      string
	leaderID    string
	includeFile string
	excludeFile string
	expect      string
	logs        string
	dryRun      bool
}

// NewRootCmd builds the root cobra command for the seedgen CLI.
func NewRootCmd() *cobra.Command {
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
			return run(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.input, "input", "", "Directory containing snapshots/ and chunks/ (a Netsy bucket root on disk; overrides --cluster)")
	cmd.Flags().StringVar(&opts.cluster, "cluster", "default", "Local Podplane cluster id; shortcut for ~/.podplane/data/s3/buckets/<id>-netsy")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output .netsy file path (required unless --dry-run)")
	cmd.Flags().StringVar(&opts.leaderID, "leader-id", "seed", "LeaderID stamped on the output snapshot")
	cmd.Flags().StringVar(&opts.includeFile, "include", "", "Path to a JSONC include file overriding the embedded default")
	cmd.Flags().StringVar(&opts.excludeFile, "exclude", "", "Path to a JSONC exclude file overriding the embedded default")
	cmd.Flags().StringVar(&opts.expect, "expect", "recommended", "Check for expected records based on the type of seed. Options: recommended (default), minimal, or none")
	cmd.Flags().StringVar(&opts.logs, "logs", "logs", "Directory to write included.txt, excluded.txt, and ignored.txt key logs")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Run the full pipeline and write key logs but do not write the output file")
	return cmd
}

func run(cmd *cobra.Command, opts options) error {
	if !opts.dryRun && opts.output == "" {
		return fmt.Errorf("--output is required (or pass --dry-run)")
	}
	inputDir, err := resolveInputDir(cmd, opts)
	if err != nil {
		return err
	}
	include, err := loadRules(opts.includeFile, seedgen.DefaultIncludeRules)
	if err != nil {
		return fmt.Errorf("include rules: %w", err)
	}
	exclude, err := loadRules(opts.excludeFile, seedgen.DefaultExcludeRules)
	if err != nil {
		return fmt.Errorf("exclude rules: %w", err)
	}
	records, err := seedgen.ReadAll(inputDir)
	if err != nil {
		return err
	}
	current := seedgen.CurrentState(records)
	kept, includedKeys, excludedKeys, ignoredKeys := classifyKeys(current, include, exclude)
	if err := seedgen.CheckExpected(opts.expect, kept); err != nil {
		return err
	}
	if err := writeKeyReports(opts.logs, includedKeys, excludedKeys, ignoredKeys); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d total read\nwrote to %s:\n- %d included\n- %d excluded\n- %d ignored\n", len(records), opts.logs, len(includedKeys), len(excludedKeys), len(ignoredKeys))

	if opts.dryRun {
		fmt.Fprintln(os.Stderr, "dry run: skipping snapshot write")
		return nil
	}

	f, err := os.Create(opts.output)
	if err != nil {
		return fmt.Errorf("create output %s: %w", opts.output, err)
	}
	defer f.Close()
	if err := seedgen.WriteSnapshot(f, kept, opts.leaderID); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", opts.output)
	return nil
}

// classifyKeys partitions current records according to the include and exclude
// rules used for seed output. Included keys are the records that will be
// written to the seed, excluded keys matched include rules but were dropped by
// exclude rules, and ignored keys did not match include rules at all.
func classifyKeys(records []*datafile.Record, include, exclude *seedgen.Rules) (kept []*datafile.Record, includedKeys, excludedKeys, ignoredKeys []string) {
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

// writeKeyReports writes the three key log files that explain how the current
// record set was classified by the include and exclude rules.
func writeKeyReports(dir string, includedKeys, excludedKeys, ignoredKeys []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create key logs directory %s: %w", dir, err)
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

// loadRules returns the embedded default rules when path is empty, or the
// file's parsed rules otherwise.
func loadRules(path string, fallback func() (*seedgen.Rules, error)) (*seedgen.Rules, error) {
	if path == "" {
		return fallback()
	}
	return seedgen.LoadRulesFile(path)
}
