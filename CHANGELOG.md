# Changelog

Notable user-visible changes to Forecast Ledger CLI are recorded here. Release
tags and downloadable files are published on GitHub Releases.

## Unreleased

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
