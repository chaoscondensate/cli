# Forecast Ledger CLI

[![CI](https://github.com/chaoscondensate/cli/actions/workflows/ci.yml/badge.svg)](https://github.com/chaoscondensate/cli/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

`forecast-ledger` is an open-source, local-first command-line tool and MCP
server for creating and checking portable forecast records. It is designed for
people and teams who want an explicit history of what they predicted, when they
recorded it, and what evidence is available later.

The broader project is described at [chaoscondensate.com](https://chaoscondensate.com/).
The interoperable data contract is maintained in the
[Forecast Ledger schema repository](https://github.com/chaoscondensate/schema).

> [!IMPORTANT]
> This project is in active pre-release development. Local validation, status,
> version metadata, document parsing, schema checks, and core storage foundations
> are available. Authoring, sealing, OpenTimestamps, publication packages, and
> the MCP tool surface are still being completed. Do not treat the current build
> as a finished evidence system.

## Why Forecast Ledger?

A forecast is more useful when its history remains inspectable. Forecast Ledger
keeps the question, quantitative forecast, later revisions, resolution evidence,
and optional cryptographic timing material in a portable JSON or YAML document.

The CLI is designed around a few strict rules:

- Every ledger operation names its file explicitly with `--file` or `-f`.
- Validation is local and never downloads a schema.
- Forecast revisions append a new record instead of rewriting history.
- Secrets and unrevealed forecast material never belong in normal output.
- Git and hosted services are optional; a ledger is an ordinary portable file.
- Verification reports what the evidence supports without claiming authorship,
  completeness, truth, or an exact self-reported creation time.

## Install

### Homebrew

Stable releases are available from the project tap:

```sh
brew install chaoscondensate/tap/forecast-ledger
```

### Release archive

Download the archive for your platform from
[GitHub Releases](https://github.com/chaoscondensate/cli/releases), verify it
against `checksums.txt`, and place `forecast-ledger` or
`forecast-ledger.exe` on your `PATH`.

Official release targets are macOS, Linux, and Windows on amd64 and arm64.

### Build from source

Go 1.27 or newer is required:

```sh
git clone https://github.com/chaoscondensate/cli.git
cd cli
make build
./dist/forecast-ledger version --json
```

## Quick start

Every ledger command requires an explicit file. Validate a local JSON or YAML
ledger without network access:

```sh
forecast-ledger validate --file ledger.yaml
```

Read a compact ledger and integrity summary:

```sh
forecast-ledger status --file ledger.yaml
```

Read-only commands that support stdin accept `--file -`:

```sh
forecast-ledger validate --file - < ledger.json
```

Use stable JSON output in scripts:

```sh
forecast-ledger --json status --file ledger.yaml
```

Inspect the binary and exact embedded contract:

```sh
forecast-ledger version --json
```

Run `forecast-ledger --help` or `forecast-ledger <command> --help` for the
complete command tree and examples. Commands that are still under development
fail explicitly; they do not silently modify a ledger.

## Planned workflows

The first complete release is intended to support:

- JSON and YAML ledger initialization and source-preserving authoring;
- platforms, typed questions, public forecasts, and append-only revisions;
- binary, multiple-choice, numeric, and date forecast values;
- sealed forecasts using the published `forecast-seal/v1` profile;
- reveal verification without discarding the original commitment evidence;
- canonical target generation and OpenTimestamps receipts;
- layered local verification and portable evidence packages;
- a permission-gated MCP stdio server backed by the same application services.

Progress and accepted behavior are tracked in the repository's
[`openspec`](openspec/) directory.

## Evidence boundaries

Forecast Ledger can help demonstrate that specific bytes existed before a
cryptographic timestamp bound. It cannot by itself prove who authored a record,
that no forecasts were omitted, that a forecast or outcome is true, or that a
self-reported timestamp is exact. A pending receipt is not verified timing, and
filesystem, hosting, Git, or archive timestamps are not substitutes for
cryptographic evidence.

Keep protected key files out of repositories, backups intended for publication,
shell arguments, logs, and evidence packages. The security model and supported
OpenTimestamps subset will be documented before the first stable release.

## Development

```sh
gofmt -w cmd internal
go mod verify
go test ./...
go vet ./...
```

Create a complete local release snapshot with:

```sh
make release-snapshot
```

See the [build guide](docs/development/build.md),
[dependency review](docs/development/dependencies.md), and
[release runbook](docs/development/releasing.md) for details. Contributors and
AI coding agents should read [AGENTS.md](AGENTS.md) before changing behavior.

## Contributing and support

Issues and focused pull requests are welcome. Please use
[GitHub Issues](https://github.com/chaoscondensate/cli/issues) for bugs, feature
proposals, documentation gaps, and release problems. For security-sensitive
reports, do not publish secrets, private ledgers, or unrevealed forecast material
in an issue.

## License

Licensed under the [Apache License 2.0](LICENSE).

The embedded Forecast Ledger contract and conformance fixtures retain their
upstream attribution; see [`third_party/forecast-ledger`](third_party/forecast-ledger/).
