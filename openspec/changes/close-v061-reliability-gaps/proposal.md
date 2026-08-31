## Why

The v0.6.1 release has several confirmed gaps between its documented contracts and its observable behavior: CLI redaction changes JSON shape, CLI and MCP classify equivalent outcomes differently, interrupted ledger replacement can leave users unable to mutate a ledger, optional outcome-source retrieval can connect to an address that was never checked, and the required vulnerability scan cannot analyze the selected Go toolchain. These are release-correctness and security-boundary defects, so they should be fixed before broader refactoring or performance work from the review is considered.

## What Changes

- Preserve the exact `encoding/json` field shape while recursively redacting secret-valued keys, including embedded fields, `omitempty`, custom marshalers, byte slices, and tag options.
- Move successful, planned, unchanged, partial-failure, and verification outcome classification below the adapters so equivalent CLI and MCP calls return the same code, safe message, and typed public data. Preserve safe `target check` and timestamp reports even when their application category is non-zero.
- **BREAKING** Correct the accidental v0.6.1 CLI timestamp-verification nesting and divergent MCP outcome codes. Consumers that relied on `data.TimestampArtifactResult`, `timestamp.stamped`, unconditional `*.updated`, or unconditional `publication.built` must adopt the documented flat data and shared outcome codes.
- Recover a valid unfinished single-ledger replacement journal automatically while holding the ledger writer lock, then continue the requested mutation against the recovered bytes. Refuse ambiguous, altered, malformed, or invalid recovery state without deleting evidence or guessing.
- Bind optional outcome-source HTTPS requests to the public IP addresses that passed policy, reapply the policy to redirects and each connection, and reject CGNAT plus other non-public or reserved destinations.
- Select a `govulncheck` invocation compatible with Go 1.27, pin it consistently in contributor and release documentation, and run it in CI so the documented security check cannot silently drift.
- Remove only dead helpers, placeholder parameters, duplicate work, or unreachable branches directly touched by these fixes. Do not split `internal/service`, introduce a general-purpose command framework, change the Forecast Ledger v1.3.0 contract, or optimize path/document algorithms without measurements and separate acceptance criteria.

## Capabilities

### New Capabilities

- `transport-result-contracts`: Stable JSON shape, service-owned outcome classification, safe partial-result preservation, and CLI/MCP parity.
- `ledger-write-recovery`: Automatic, lock-scoped recovery of valid interrupted single-ledger replacements and safe refusal of ambiguous recovery state.
- `outcome-source-network-safety`: Connection-bound public-destination enforcement for optional outcome-source HTTPS retrieval.
- `dependency-security-gates`: A pinned, toolchain-compatible vulnerability scan that is documented and enforced in CI.

### Modified Capabilities

- None.

## Impact

- Affects `internal/presentation`, transport-neutral result contracts in `internal/service`, CLI and MCP dispatch/presentation, `internal/storage` ledger transactions, optional outcome-source retrieval, tests and generated result references, CI, and contributor/security/dependency documentation.
- Changes preview JSON and MCP outcome contracts only where v0.6.1 currently contradicts the documented shared-service behavior; ledger bytes, Forecast Ledger v1.3.0 schema and fixtures, canonical targets, sealed values, RFC 3161 evidence, and publication package bytes do not change.
- May update the pinned `golang.org/x/vuln` command version or invocation but does not add a runtime dependency.
