# Podplane Seed Generator — Agent Development Guide

`seedgen` is a Go CLI that produces Podplane seed files which are Netsy
snapshot files (`.netsy`) by reading a running Netsy cluster's snapshots
and chunks from disk, flattening the record history into the current 
value per key, filtering it down to the records suitable for a template,
renumbering revisions to satisfy Netsy's bootstrap integrity check, and
writing a fresh snapshot.

The flattening step is equivalent to a full LSM-style compaction (drop
history, drop tombstones), but more aggressive than Netsy's own compaction
which preserves rows for revision contiguity and only nulls out their
values.

## Build & Test Commands

- **Setup**: `make setup` — verify required tools and enable git hooks
- **Build**: `make build` — builds the `seedgen` binary with build info via ldflags
- **Test**: `make test` — run tests with the race detector
- **Format**: `make fmt`
- **Lint**: `make lint`
- **Precommit**: `make precommit` — formatting check + lint (read-only)
- **Clean**: `make clean` — remove `bin/`

## Design constraints

- **No CGO, no SQLite.** This tool reads the on-disk Netsy object-storage
  layout via the public `github.com/netsy-dev/netsy/pkg/datafile` package
  only. The Podplane local VM's fake S3 backs that storage with a normal
  filesystem directory; we point `seedgen` at that directory.
- **Bootstrap integrity invariant.** Netsy's
  `localdb.VerifyIntegrity()` requires `COUNT(records) == MAX(revision)`.
  Any output snapshot from `seedgen` must therefore have contiguous revisions
  `1..N` with one record per revision. The pipeline renumbers surviving
  records to guarantee this.
- **No history, no tombstones in output.** A template snapshot should look
  like a freshly-created cluster: each surviving record is emitted as
  `Created=true, Deleted=false, CreateRevision=Revision, PrevRevision=0,
  Version=1`.

## Directory layout

```
cmd/                     N/A (single binary; entry point is main.go)
internal/
├── buildvars/           Build-time linker var package, parallels podplane/.
├── cmd/                 Cobra command wiring; thin glue between flags and the
│                        seedgen package.
└── seedgen/             Core library:
    ├── defaults/        Embedded default include.jsonc and exclude.jsonc.
    ├── read.go          Walk a directory's snapshots/ + chunks/, parse via
    │                    pkg/datafile.
    ├── rules.go         JSONC rules loader (keys, prefixes, substrings).
    ├── state.go         Flatten record history to current value per key.
    ├── filter.go        Apply include and exclude rules.
    └── write.go         Renumber + normalise + WriteSnapshot.
```

## Testing

- Standard library `testing` only. Table-driven where useful.
- Fakes/synthetic fixtures over mocks: build snapshot and chunk files with
  `pkg/datafile.WriteSnapshot` / `WriteChunk` inside `t.TempDir()`, then run
  the seedgen pipeline over them.
- Always assert `len(output) == maxRevision(output)` in tests that exercise
  the writer; that's the property Netsy bootstrap will check.
- For seed-content debugging and workflow verification, prefer the JSON files
  written under `records/<seed-name>/`; they are the intended inspectable view
  of what `seedgen` emitted. Do not require `read-netsy-file` unless the task is
  specifically to validate or inspect the binary `.netsy` snapshot format.

## Code style

- File headers: Podplane Apache-2.0 header on every file.
- Package docs: use `doc.go` for any package-level documentation.
- Imports: stdlib → third-party → local (`github.com/podplane/seedgen/*`).
- Errors: named returns `(result Type, err error)`, early returns,
  `fmt.Errorf()` wrapping.
- Logging: prefer returning errors. If logging is needed, use `log/slog`.
- Dependencies: never add a new Go module without explicit confirmation. Keep
  the dependency surface tiny — `github.com/netsy-dev/netsy/pkg/datafile`,
  `github.com/spf13/cobra`, `github.com/tidwall/jsonc` only.
