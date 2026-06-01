# Podplane Seed Generator

A utility for creating Podplane seed files from [Netsy](https://netsy.dev) chunks & snapshots.

`seedgen` is a Go tool that turns the on-disk state of a running
Netsy cluster into a fresh `.netsy` snapshot file suitable
for use as a Podplane seed file (e.g. the one downloaded by
`podplane deps download` / `podplane hooks netsy-seed` as `recommended.netsy`).

## How it works

1. Read the latest snapshot and every chunk above its revision from a Netsy
   object-storage directory using the public
   [`github.com/netsy-dev/netsy/pkg/datafile`](https://pkg.go.dev/github.com/netsy-dev/netsy/pkg/datafile)
   package.
2. Flatten the record history into the current value per key (deleted records
   drop the key, otherwise the latest revision wins).
3. Apply include and exclude rules to drop runtime/per-cluster records that
   should not appear in a template (events, leases, helm release state, etc.).
4. Renumber surviving records' revisions sequentially `1..N` and normalise them
   to look freshly created. This is required because Netsy's bootstrap
   integrity check requires `total records == max revision`.
5. Write the result as an uncompressed Netsy snapshot file.

The `seedgen` tool runs entirely against the host-side fake S3 directory used by a
Podplane local VM and has no dependency on SQLite, CGO, or a running cluster.

## Usage

```
seedgen --output <file> [flags]                       # reads the "default" local cluster
seedgen --cluster <id>  --output <file> [flags]
seedgen --input <dir>   --output <file> [flags]
seedgen --dry-run       [flags]                       # any of the above without writing
seedgen --expect recommended --output recommended.netsy
```

The kept keys are always printed to stderr as a summary; the snapshot itself
is written to the file given by `--output`.

| Flag          | Default     | Description                                                                                |
| ------------- | ----------- | ------------------------------------------------------------------------------------------ |
| `--cluster`   | `default`   | Local Podplane cluster id; shortcut for `~/.podplane/data/s3/buckets/<id>-netsy`.          |
| `--input`     | (unset)     | Directory containing `snapshots/` and `chunks/` (a Netsy bucket root on disk). Overrides `--cluster`. |
| `--output`    | (required)  | Output `.netsy` file path. Required unless `--dry-run` is set.                             |
| `--leader-id` | `seed`      | `LeaderID` stamped on the output snapshot.                                                 |
| `--include`   | (embedded)  | Path to a JSONC include file overriding the embedded default.                              |
| `--exclude`   | (embedded)  | Path to a JSONC exclude file overriding the embedded default.                              |
| `--expect`    | `recommended` | Check for expected records based on the type of seed. Options: `recommended`, `minimal`, or `none`. |
| `--dry-run`   | `false`     | Run the full pipeline and print kept keys but do not write the output file.                |

The default `--expect recommended` guard is intended for published seed
snapshots. The check runs after the current-state flattening and
include/exclude filters, so it catches both clusters that were exported too
early and filter rules that accidentally drop required platform records. Use
`--expect minimal` for minimal seeds or `--expect none` for ad-hoc/custom/debug
exports.

## Include / exclude rules

Both files share the same JSONC shape:

```jsonc
{
  // Exact key matches
  "keys": [
    "/registry/health"
  ],
  // Match if the key starts with this string
  "prefixes": [
    "/registry/events/"
  ],
  // Match if the key contains this string anywhere
  "substrings": [
    "/sh.helm.release.v1."
  ]
}
```

A record is kept iff it matches **any** include rule AND does **not** match any
exclude rule.

## Install

```
go install github.com/podplane/seedgen@latest
```

## Building and testing

```
make build      # builds bin/seedgen
make test       # runs unit tests with the race detector
make precommit  # gofmt + golangci-lint
```

## Learn More

Learn more about Podplane at the official project website: [podplane.dev](https://podplane.dev)

## License

Podplane is licensed under the Apache License, Version 2.0.
Copyright The Podplane Authors.

See the [LICENSE](./LICENSE) file for details.
