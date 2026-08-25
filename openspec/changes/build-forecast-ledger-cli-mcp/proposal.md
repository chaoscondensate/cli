## Why

Forecast Ledger v1.0.0 has a published data and cryptographic contract, but its reference scripts do not provide a safe, cohesive user workflow for creating, sealing, timestamping, publishing, revealing, and independently verifying forecast ledgers. A portable Go CLI and MCP server are needed so people and AI agents can use the same audited domain operations on macOS, Linux, and Windows without depending on Python.

## What Changes

- Add a single Go binary named `forecast-ledger`, built with `urfave/cli/v3` and distributed for supported macOS, Linux, and Windows architectures.
- Embed the exact Forecast Ledger v1.0.0 schema and implement its structural, format, semantic, canonicalization, and cryptographic checks offline. The initial contract is pinned to schema commit `e409463d702888fefd253b32f21b9b2f864aabed` and schema SHA-256 `e63bdd01f0241aa4d94d5ccc45e84bcea70a6a7fd46ab77cff4802b3f8b8fc65`.
- Require an explicit `--file` (`-f`) for every ledger operation; never guess a ledger when multiple files may exist. Require stable question and forecast IDs for record-specific operations.
- Add commands for ledger initialization and inspection, full validation, platform/question management, append-only public forecasts and forecast updates, question resolution, canonical target generation, sealing, revealing, OpenTimestamps lifecycle management, layered verification, and portable evidence-package creation and verification.
- Make `forecast-ledger forecast seal` the atomic high-level operation that generates the ciphertext and hides plaintext. Do not expose partial "encrypt" or "hide" mutations that can leave an invalid or unsafe ledger state.
- Use OpenTimestamps as the only timestamp protocol supported by v1. RFC 3161 input is out of scope and must be rejected as unsupported.
- Add an MCP server over stdio that exposes the same domain operations with structured schemas, explicit ledger paths, bounded filesystem access, deterministic errors, and secret redaction.
- Add project guidance (`AGENTS.md`), English documentation, examples, conformance fixtures, release automation, checksums, SBOM/provenance, and a cross-platform test matrix.
- Keep the tool open source and suitable for non-commercial use. Do not claim that the separate Chaos Condensate practice or website is non-commercial.

## Capabilities

### New Capabilities

- `cli-contract`: Discoverable, scriptable, English CLI behavior, explicit file selection, stable output/error conventions, safe mutation semantics, and cross-platform distribution.
- `ledger-management`: Create, read, validate, inspect, and update Forecast Ledger v1 files while enforcing append-only forecast history and typed question lifecycles.
- `forecast-cryptography`: Build RFC 8785 `forecast-envelope/v1` targets and perform interoperable `forecast-seal/v1` seal and reveal workflows without leaking secrets.
- `timestamp-evidence`: Create, upgrade, inspect, and verify OpenTimestamps receipts and synchronize integrity metadata with exact target artifacts.
- `layered-verification`: Independently report schema/semantic validity, content binding, existence timing, reveal validity, and outcome-evidence status without collapsing distinct claims, and build or verify portable evidence packages without assuming a publication transport.
- `mcp-server`: Expose safe CLI-parity resources and tools over MCP stdio using the same application services and validation rules.

### Modified Capabilities

None. This is the first behavior change in the repository.

## Impact

- Introduces a new Go module and shared domain/application packages used by both CLI and MCP adapters.
- Adds dependencies for `urfave/cli/v3`, the official MCP Go SDK, JSON Schema/YAML processing, RFC 8785-compatible canonicalization, ChaCha20-Poly1305, and a pure-Go OpenTimestamps implementation or adapter proven against the official client.
- Produces local ledger, target, receipt, key-file, manifest, and evidence-package files. Network access is explicit and limited to OpenTimestamps calendars and the selected Bitcoin verification source; placing a package on any hosting or distribution service is outside the CLI.
- Treats Forecast Ledger v1.0.0, its reference validator, cryptographic code, and test vector as interoperability sources; the CLI must not weaken their constraints.
- Does not add scoring/ranking storage, platform imports, digital authorship signatures, HTTP MCP transport, hosted publication APIs, source-control automation, or RFC 3161 support in v1.
