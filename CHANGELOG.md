# Changelog

Notable user-visible changes to Forecast Ledger CLI are recorded here. Release
tags and downloadable files are published on GitHub Releases.

## Unreleased

## 0.2.4 - 2026-08-27

### Fixed

- Give each MCP parity-test call an independent deadline so slower release
  runners cannot exhaust one shared test budget before later operations.

## 0.2.3 - 2026-08-27

### Fixed

- Report never-built targets as ordered `not_applicable` results and continue
  `target check --all`, while retaining hard failures for missing referenced or
  mismatched evidence.
- Default initial recorded times from one operation-clock observation instead
  of copying an explicit historical ledger creation time; inclusive time
  boundaries now use inclusive wording.
- Omit unknown diagnostic positions instead of printing line/column zero, and
  keep semantic field order when JSON or YAML records are inserted or replaced.
- Make MCP root errors identify safe root classes, flags, and route IDs without
  exposing configured absolute paths.
- Show retained target, receipt, Bitcoin height, anchored-before, and
  verification-time evidence in forecast and layered verification output while
  keeping offline values clearly labeled as stored.
- Keep writer locking fail-fast even when a long `--timeout` is supplied; callers
  that intentionally contend remain responsible for serialization or bounded
  retry with backoff.

### Testing

- Add regressions derived from the v0.2.2 dogfooding findings across CLI modes,
  MCP, JSON/YAML structural edits, fixed clocks, concurrent writers, and stored
  OpenTimestamps evidence.

## 0.2.2 - 2026-08-26

### Fixed

- Preserve retained target and OpenTimestamps evidence while authoring later
  forecasts and question lifecycle changes.
- Keep YAML timestamp inputs, structured validation diagnostics, and private
  input permission errors safe and consistent across CLI transports.
- Preserve readable expanded JSON/YAML ledger formatting as records are added
  or revealed.
- Return exit 9 after presenting pending or incomplete layered and package
  verification reports instead of remapping them to an internal failure.
- Show type-aware question, forecast, and verification details in human and
  plain output without exposing sealed data or internal Go union fields.
- Omit mutating MCP tools from read-only discovery and report static tool
  effects separately from the server's current mode.

### Testing

- Add a reproducible multi-question dogfood lifecycle and run it in native
  macOS, Linux, and Windows CI jobs.

## 0.2.1 - 2026-08-26

### Fixed

- Validate Windows owner-only key ACLs structurally instead of comparing a
  non-canonical SDDL rendering.
- Snapshot artifact-root identity from an open handle so replacement is
  detected reliably on Windows.

## 0.2.0 - 2026-08-26

### Added

- Complete ledger, platform, question, public forecast, sealed forecast,
  target, timestamp, verification, publication, and MCP command surfaces.
- Explicit `--file` selection for every ledger operation; eligible local
  read-only commands also accept stdin with `--file -`.
- Portable publication packages that do not require Git or a hosted service.
- A local stdio MCP server with explicit named roots, optional whole-server
  read-only/offline modes, bounded requests, and a separate default-off reveal
  capability.

### Changed

- Removed hidden preview placeholders: advertised leaf commands now invoke real
  shared application services.
- Kept Forecast Ledger schema compatibility exact at v1.0.0. The CLI does not
  guess or download migrations for another schema version.

### Security and evidence limits

- OpenTimestamps support is experimental until differential, public-calendar
  liveness, native-platform, and independent-review gates pass.
- Sealed operations write protected key files before ledger mutation and report
  recoverable retained-key states after an interrupted commit.
- Verification does not prove authorship, completeness, forecast truth,
  calibration, exact self-reported time, or outcome-source authority.
- A cryptographically valid anchor that is not before the known outcome is
  retained with a warning and fails the layered timing-sufficiency check.
- Public timestamp calls reveal request timing and, during Bitcoin checks, the
  block heights of interest to the fixed public services.

## 0.1.1 - 2026-08-25

- Published local ledger validation, status, and build/schema version metadata.
