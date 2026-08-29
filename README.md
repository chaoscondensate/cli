# Forecast Ledger CLI

[![CI](https://github.com/chaoscondensate/cli/actions/workflows/ci.yml/badge.svg)](https://github.com/chaoscondensate/cli/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Create and independently check portable forecast evidence without requiring
Git or a hosted service.

Forecast Ledger CLI provides the `forecast-ledger` command and a local MCP
server for portable forecast records. It is intended for individual
forecasters, forecasting teams and researchers, and developers who need a
reviewable file format and automation interface instead of a required hosted
account.

The broader project is described at [chaoscondensate.com](https://chaoscondensate.com/).
The interoperable data contract is maintained in the
[Forecast Ledger schema repository](https://github.com/chaoscondensate/schema).
User-visible changes are tracked in the [changelog](CHANGELOG.md).

> [!IMPORTANT]
> **Status: Preview and unaudited.** Release `v0.4.0` implements the complete
> CLI and MCP command surface: authoring, sealed forecasts, canonical
> targets, experimental RFC 3161 timestamp evidence, layered verification, and
> portable publication packages. RFC 3161 support remains experimental until
> the tracked native-platform and independent-review gates are complete. The project has no recorded independent
> security or cryptographic audit. Do not treat the current build as a finished
> evidence system.

Release archives target macOS, Linux, and Windows on amd64 and arm64. The CLI
can check whether a ledger follows the pinned data contract and report the
evidence actually present. Even after the planned evidence workflows are
complete, it will not by itself prove authorship, ledger completeness, forecast
truth, an exact self-reported time, or the correctness of an outcome source.

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

See the [complete installation guide](docs/getting-started/install.md) for
checksum verification, upgrades, removal, and archive fallback instructions.

Published releases provide Homebrew, platform archives, native Linux packages,
and a Windows Chocolatey package. Check the selected release's asset list before
using a package command.

### Homebrew

Stable releases are available from the project tap:

```sh
brew install chaoscondensate/tap/forecast-ledger
```

### Linux packages

Download the package for your architecture from
[GitHub Releases](https://github.com/chaoscondensate/cli/releases):

- Debian and Ubuntu: `.deb`
- Fedora, RHEL, and openSUSE: `.rpm`
- Alpine Linux: `.apk`
- Arch Linux: `.pkg.tar.zst`

Both x86-64 and ARM64 packages are built. These are downloadable release
packages, not hosted APT, RPM, APK, or Arch repositories, so download the new
package before upgrading.

### Windows

Windows x86-64 releases include a Chocolatey `.nupkg` alongside the `.zip`.
Windows ARM64 uses the native `.zip` archive. The Chocolatey package is attached
to GitHub Releases rather than published to the public Chocolatey repository.

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

Every ledger command requires an explicit file. Start with an empty ledger:

```sh
forecast-ledger init \
  --file ledger.yaml \
  --ledger-id my-forecasts \
  --timezone Europe/London \
  --forecaster-id me \
  --forecaster-name "My Name"
```

The embedded Forecast Ledger v1.2.0 contract permits zero questions and questions
with zero forecasts. Add them when they are ready; `--input` is optional on
`init`, and `initial_forecast` is optional in question input. See [Create a
ledger](docs/getting-started/create-ledger.md) for the empty-first, combined,
dry-run, team, and sealed-key workflows.

Root display and current forecaster metadata can later be changed with a closed
patch, without rewriting question or forecast history:

```sh
forecast-ledger ledger update --file ledger.yaml --input metadata-patch.yaml
```

Platform records can be managed locally with `platform add`, `update`, `list`,
`show`, and approved `remove`; see [Manage platform records](docs/how-to/manage-platforms.md).

Typed questions can be added before their first forecast, updated within the v1
evidence rules, listed, shown, resolved, annulled, or disputed; see
[Manage questions and resolutions](docs/how-to/manage-questions.md).

Append the first public forecast, or a later revision without modifying an
earlier record:

```sh
forecast-ledger forecast add \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-001 \
  --input forecast.yaml
```

See [Manage public forecasts](docs/how-to/manage-public-forecasts.md) for typed
values, global IDs, supersession, dry-run, list/show, and stdin behavior.

Keep a forecast private until an authenticated reveal:

```sh
forecast-ledger forecast seal \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --input private-forecast.yaml \
  --key-file f-launch-002.key
forecast-ledger forecast reveal \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --key-file f-launch-002.key \
  --yes
```

Read [Seal and reveal forecasts](docs/how-to/seal-and-reveal-forecasts.md)
before handling private material or protected keys.

Build or check the exact canonical bytes used by later evidence:

```sh
forecast-ledger target build \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002
forecast-ledger target check \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002
```

See [Build and check forecast targets](docs/how-to/build-targets.md) for the
projection, deterministic paths, `--all`, collision behavior, and evidence
limits. Checking a forecast whose target was never retained succeeds with
`not_applicable` and guidance to run `target build`; it does not pretend that
missing bytes passed verification.

Create experimental RFC 3161 evidence with an explicit timestamp authority and
a retained PEM trust bundle, then inspect or verify it locally:

```sh
forecast-ledger timestamp stamp \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --tsa-url https://tsa.example.com/ \
  --ca-bundle trust/tsa-ca.pem
forecast-ledger timestamp status --file ledger.yaml --question q-launch --forecast f-launch-002
forecast-ledger timestamp verify --file ledger.yaml --question q-launch --forecast f-launch-002
```

There is no built-in TSA list and no system-root fallback. Stamp sends one
SHA-256 RFC 3161 request to the named public HTTPS endpoint and retains the
request (`.tsq`), response (`.tsr`), exact target, and ledger-relative CA
bundle. Status and verify make no timestamp-service network request. Repeat
stamp with another TSA URL to retain independent entries. Read [Timestamp
forecasts](docs/how-to/timestamp-forecasts.md) before choosing and retaining
trust material.

Run all evidence layers locally, or opt into outcome-source reachability checks:

```sh
forecast-ledger verify --file ledger.yaml --offline
forecast-ledger verify --file ledger.yaml
```

Verification reports content binding, existence timing, reveal authentication,
and outcome evidence separately. Normal human and `--plain` output include the
complete ordered layer matrix; `--json` adds the same matrix as stable data. See
[Verify evidence](docs/how-to/verify-evidence.md). Overall `pass` requires at
least one applicable forecast-evidence layer. An empty or all-not-applicable
selection returns `no_evidence`, `incomplete`, and exit 9.

Build a standalone package without Git or a hosted service, then verify its
manifest and evidence offline:

```sh
forecast-ledger publish build --file ledger.yaml --output evidence-package
forecast-ledger publish verify \
  --file evidence-package/ledger/ledger.yaml \
  --manifest evidence-package/manifest.json
```

Publication verification has no network option. It verifies the packaged
request, response, target, and CA bytes locally. Publication follows evidence
paths recorded in the selected ledger. A
standalone target merely sitting beside the ledger is not packaged until the
ledger references it. Manifest and file integrity remain visible even when the
evidence aggregate is `no_evidence` or incomplete. See
[Build and verify publication packages](docs/how-to/publish-evidence.md).

Run the local MCP stdio adapter with explicit named roots:

```sh
forecast-ledger mcp serve \
  --ledger-root main=/data/forecast-ledgers \
  --output-root packages=/data/forecast-packages \
  --secret-root keys=/data/forecast-secrets
```

The default MCP server is read-write and online within those roots. Use
`--read-only` or `--offline` to limit the whole server. In read-only mode,
mutating tools are omitted from discovery and direct calls to their names are
unknown-tool errors. Reveal remains absent unless `--allow-reveal` is explicitly
set. See [Run the MCP server](docs/how-to/run-mcp.md).

Validate a local JSON or YAML ledger without network access:

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

Stdin contains only ledger bytes, so it cannot resolve sibling targets or
timestamp artifacts. Commands that inspect or mutate evidence therefore require a real
`--file` path. In YAML input, quote RFC 3339 timestamps, for example
`forecasted_at: "2026-09-01T09:00:00Z"`; this keeps examples portable across
YAML parsers even though the CLI safely normalizes timestamp-tagged scalars in
known timestamp fields.

Use stable JSON output in scripts:

```sh
forecast-ledger --json status --file ledger.yaml
```

Inspect the binary and exact embedded contract:

```sh
forecast-ledger version --json
```

Run `forecast-ledger --help` for the commands available in the installed
release. The current source has no visible placeholder leaf: every advertised
command has a connected application action. Installed Preview releases may have
a smaller surface, so check that binary's help and `version --json` output.

## Available workflow groups

The current source supports:

- source-preserving platform, question, and forecast authoring after JSON/YAML initialization;
- platforms, typed questions, public forecasts, and append-only revisions;
- binary, multiple-choice, numeric, and date forecast values;
- sealed forecasts using the published `forecast-seal/v1` profile;
- reveal verification without discarding the original commitment evidence;
- canonical target generation and RFC 3161 request/response evidence;
- layered local verification and portable evidence packages;
- a root-confined MCP stdio server backed by the same application services.

Progress and accepted behavior are tracked in the repository's
[`openspec`](openspec/) directory.

## Evidence boundaries

Forecast Ledger can help demonstrate that specific bytes existed before a
cryptographic timestamp bound. It cannot by itself prove who authored a record,
that no forecasts were omitted, that a forecast or outcome is true, or that a
self-reported timestamp is exact. Pending RFC 3161 evidence is not verified timing, and
filesystem, hosting, Git, or archive timestamps are not substitutes for
cryptographic evidence.

Keep protected key files out of repositories, backups intended for publication,
shell arguments or logs. Publication packages intentionally include the exact
retained public CA bundle. The constrained RFC 3161 profile is experimental,
uses SHA-256 only, excludes system roots and revocation/LTV lookup, and rejects
unsupported or weak input instead of guessing.

## Development

```sh
gofmt -w cmd internal
go mod verify
go test ./...
go vet ./...
```

Create a local archive and Linux-package snapshot with:

```sh
make release-snapshot
```

Chocolatey package generation requires the Windows-only `choco` executable and
is checked separately by CI on Windows.

See the [build guide](docs/development/build.md),
[dependency review](docs/development/dependencies.md), and
[release runbook](docs/development/releasing.md) for details. Contributors and
AI coding agents should read [AGENTS.md](AGENTS.md) before changing behavior.

## Contributing and support

Issues and focused pull requests are welcome. Read the
[contribution guide](CONTRIBUTING.md) before changing behavior. Please use
[GitHub Issues](https://github.com/chaoscondensate/cli/issues) for bugs, feature
proposals, documentation gaps, and release problems. For security-sensitive
reports, do not publish secrets, private ledgers, or unrevealed forecast material
in an issue. Use the [security policy](SECURITY.md) to report a suspected
vulnerability privately. Community participation is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md), which also provides a confidential
reporting route.

See the [support guide](SUPPORT.md) for usage, bug, schema, security, conduct,
and broader Chaos Condensate routes and their boundaries.
Project roles and decision rights are defined in [governance](GOVERNANCE.md).

## License

Original project material is licensed under the
[Apache License 2.0](LICENSE), SPDX identifier `Apache-2.0`. See the
[licensing policy](LICENSE_POLICY.md) and [third-party notices](THIRD_PARTY_NOTICES.md).

The embedded Forecast Ledger contract and conformance fixtures retain their
upstream attribution; see [`third_party/forecast-ledger`](third_party/forecast-ledger/).
