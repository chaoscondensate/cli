## Context

See `proposal.md` for motivation. The repository currently implements the project foundation, document/validation/storage layers, and the read-only validate/status CLI increment; later authoring, cryptography, timestamp, publication, and MCP operations remain planned. The authoritative interoperability source is the republished Forecast Ledger v1.0.0 contract at commit `e409463d702888fefd253b32f21b9b2f864aabed`, not the floating state of a tag or website. The exact schema file has SHA-256 `e63bdd01f0241aa4d94d5ccc45e84bcea70a6a7fd46ab77cff4802b3f8b8fc65`; the current release archive has SHA-256 `a3d6afcf8a3cd9b9e9a650ebac684cbe2f155a81db309797d77694b5f4b9bbda`.

The older Research folder is useful for threat models and operational use cases, but it describes a pre-release v0.5 shape and non-final protocols. Where it differs, the published v1 schema, English documentation, reference validator, crypto implementation, and deterministic vector win. In particular, v1 has stable forecast IDs, a non-recursive `forecast-envelope/v1`, `forecast-seal/v1`, and OpenTimestamps as its only timestamp protocol. Hash-only `forecast-commit/v1`, RFC 3161, snapshot protocols, and older prediction field names are not v1 CLI contracts.

The CLI must follow the human-first and composable behavior in the Command Line Interface Guidelines: explicit external actions, examples-first help, primary output on stdout, diagnostics on stderr, stable JSON, no secret argv/environment inputs, TTY-aware presentation, and a single binary. All public source, interfaces, documentation, fixtures, and project guidance are English.

## Goals / Non-Goals

**Goals:**

- One cohesive domain implementation shared by CLI and MCP adapters.
- Exact offline compatibility with Forecast Ledger v1 structure, semantics, target bytes, and seal vectors.
- Explicit, recoverable workflows for many independent ledger files.
- Safe public and sealed forecast lifecycles, OTS evidence, layered verification, and portable evidence packages.
- Reproducible pure-Go release artifacts for macOS, Linux, and Windows.

**Non-Goals:**

- Designing a new ledger schema, commitment protocol, encryption scheme, timestamp protocol, or authorship signature.
- Supporting the historical v0.5 Research schema or hash-only `forecast-commit/v1` in the first release.
- RFC 3161, hosted publisher APIs, platform imports, scoring/ranking persistence, HTTP MCP, or automatic secret-manager integrations.
- Proving authorship, completeness, truth, or substantive outcome-source quality.
- Treating the separate Chaos Condensate commercial practice as part of the open-source CLI's non-commercial positioning.

## Decisions

### 1. Use a hexagonal single-binary architecture

The executable name is `forecast-ledger`, with module path `github.com/chaoscondensate/cli`. One `main` selects either the CLI adapter or MCP stdio adapter; both call the same application services. The intended package boundaries are:

```text
cmd/forecast-ledger       process entrypoint
internal/app              transport-neutral errors and shared contracts
internal/buildinfo        build, source, schema, Go, and MCP metadata
internal/service          use-case orchestration shared by CLI and MCP
internal/ledger           typed v1 model, selectors, lifecycle rules
internal/document         JSON/YAML source tree and format-preserving patches
internal/validation       embedded schema plus semantic checks
internal/canonical        bounded RFC 8785/JCS implementation
internal/forecastcrypto   target, seal, reveal, and vector verification
internal/timestamp/ots    pure-Go OTS backend and Bitcoin-source adapters
internal/publication      evidence-package manifests, build, and verification
internal/storage          locks, safe paths, atomic/recoverable writes
internal/presentation     human, plain, and stable JSON results/errors
internal/adapters/cli     urfave commands and flags
internal/adapters/mcp     MCP tools, resources, permissions, and stdio
```

Adapters do not invoke each other through subprocesses. Domain errors carry stable codes and structured details; each adapter maps them to exit codes or MCP tool errors. This prevents validation, locking, and cryptographic behavior from drifting.

Alternative considered: implement the MCP server by shelling out to CLI commands. Rejected because it would duplicate serialization, complicate cancellation and locking, leak secrets through process boundaries, and make errors less structured.

### 2. Pin and embed the released contract by content

The repository vendors the exact v1 schema, reference fixtures needed for conformance, crypto vector, and source attribution under a versioned internal data directory. Builds embed those bytes and expose the supported schema commit and digest in `forecast-ledger version --json`. Runtime validation never downloads a schema or follows a remote `$ref`.

The release tag was moved during the RFC removal while retaining version 1.0.0. Therefore, a tag URL alone is not sufficient for build reproducibility. Dependency updates use an explicit review task that records old/new commit and digest, runs the upstream fixture suite through the Go implementation, and requires a compatibility decision.

Alternative considered: fetch the `$schema` URL at runtime. Rejected because it introduces network dependence, mutable behavior, and a supply-chain boundary into validation.

### 3. Parse once into a source tree and a typed semantic model

JSON and YAML readers reject duplicate keys, aliases or nesting that exceed resource limits, invalid UTF-8/surrogates, floats where v1 forbids them, and values outside I-JSON-safe bounds. The parser retains a source tree with comments, key order, style, and newline information and separately maps it into a typed domain model.

Mutations patch only selected source nodes, then remap and run complete post-validation. A transaction obtains an exclusive cross-platform lock, writes a temporary sibling file, flushes it, and uses an OS-specific safe-replace implementation. A journal records enough state to recover if Windows replacement semantics or a crash prevents one-step replacement. No command mutates an anchored immutable forecast statement; allowed reveal and integrity metadata additions follow the published envelope exclusions.

Alternative considered: unmarshal and reserialize the entire file. Rejected because it would destroy YAML comments/order, create noisy publication diffs, and make accidental semantic changes harder to review.

### 4. Use a noun-verb command tree with one explicit ledger flag

The CLI surface is:

```text
forecast-ledger
├── init
├── validate
├── status
├── platform add|update|list|show|remove
├── question add|update|list|show|resolve
├── forecast add|list|show|seal|reveal
├── target build|check
├── timestamp stamp|upgrade|verify|status
├── verify
├── publish build|verify
├── mcp serve
├── completion
└── version
```

Every ledger command has a required `--file/-f`. Record-specific commands require `--question-id` and `--forecast-id`; array positions and timestamps are never identity selectors. `forecast add --supersedes-forecast-id` is the only forecast-update mechanism. Complex typed inputs use `--input <file>` or `--input -`; small non-secret scalar flags remain available for common cases. Target and timestamp commands require one forecast selector unless `--all` is explicit.

`validate` is local and never performs an implicit network call. `verify` runs the layered verifier; network-dependent OTS checks are explicit and honor timeouts. `publish build` creates a local evidence package at an explicit output path, while `publish verify` validates an existing package. Neither command uploads data or requires source-control metadata.

Global behavior flags are `--json`, `--plain`, `--quiet`, `--verbose`, `--no-color`, `--no-input`, `--dry-run`, and `--timeout` where applicable. Exit categories are: 0 success, 1 internal, 2 usage, 3 invalid data, 4 not found, 5 conflict/precondition, 6 cryptographic/proof failure, 7 local I/O, 8 network/remote, 9 pending/not ready, 10 unavailable in the current release, and 130 interrupted. Planned commands may remain registered for structural development tests, but they stay hidden from normal help and return the unavailable category until their application service is connected.

Alternative considered: expose separate `encrypt`, `hide`, and `commit` commands. Rejected because the published contract has one atomic sealed state; partial primitives make it easy to publish plaintext or leave a record that cannot be revealed.

### 5. Implement exact target and sealing profiles, not general-purpose crypto

The canonicalizer implements only the bounded RFC 8785 subset required by Forecast Ledger: UTF-16 property ordering, I-JSON-safe integers, UTF-8 output, exact strings, and no floats. It does not rely on incidental `encoding/json` map ordering. `forecast-envelope/v1` projection is a typed allowlist matching the published reference function; `integrity` is excluded to prevent recursion and `key_hint` is excluded for rotation.

Sealing uses Go's OS-backed `crypto/rand`, fresh 32-byte salt and key, a fresh 12-byte nonce, SHA-256 commitment, and ChaCha20-Poly1305 with the exact published JCS plaintext and AAD. A new key is generated per forecast. The key is written first to an explicit new file: POSIX mode 0600 and an owner-only Windows ACL where supported. Failure to secure the key aborts before ledger mutation. The key path must be outside evidence-package output roots; weak placement or permissions are a hard error unless an explicitly documented recovery override is used.

Reveal decrypts and validates before opening a write transaction. The public mirror is generated from authenticated plaintext rather than user input, and the original ciphertext, commitment, nonce, and envelope relationship remain. No log or result includes pre-reveal plaintext, salts, or generated keys.

Alternative considered: a generic crypto subcommand or AES-GCM option. Rejected because neither is part of the published v1 interoperability profile.

### 6. Build a constrained pure-Go OpenTimestamps backend

There is no official Go OpenTimestamps client. Shelling out to the official Python `ots` executable would violate the single-binary and Windows requirements. The timestamp layer therefore has an internal backend interface and a constrained pure-Go v1 implementation for detached SHA-256 receipts: bounded parsing/serialization, supported operations and attestations, calendar submission, pending inspection, upgrade, and Bitcoin attestation verification.

Unknown proof nodes are preserved for lossless round-trip when safe; otherwise the operation fails explicitly. Parsers limit proof bytes, recursion, operation count, calendar response size, redirects, and time. Network clients use context cancellation, HTTPS policy, proxies from standard Go transport behavior, and explicit configured calendars.

Bitcoin verification is pluggable. Bitcoin Core RPC is the preferred independent source; an explicitly selected HTTPS explorer adapter is a convenience with its weaker trust boundary named in the verification report. The backend never equates a calendar response, filesystem or source-control time, or an explorer's assertion with a confirmed proof until the target digest and Bitcoin attestation checks pass.

The production gate is differential conformance against the official client: Python-to-Go and Go-to-Python fixtures, byte and semantic round trips, matching info/upgrade/verify results, malformed proof fuzzing, mocked calendar tests in normal CI, real-calendar nightly tests, and independent review of the supported subset. OTS commands remain marked experimental until this gate passes; the overall v1 release cannot claim production timestamp support before it does.

Alternative considered: a long-unmaintained third-party Go package. It may provide code or fixtures after license/security review, but cannot define correctness without official-client conformance.

### 7. Treat verification as an evidence matrix

The verifier returns a stable object with separate layers:

```text
document          schema, formats, and semantic rules
content_binding   rebuilt target bytes and SHA-256 match
existence_timing  OTS state, Bitcoin evidence, and outcome precedence
reveal            AEAD, commitment, IDs, canonical plaintext, public mirror
outcome_evidence  source presence and explicitly limited automated checks
package_integrity optional evidence-package manifest and file checks
```

Each layer is pass, fail, pending, not-applicable, or not-checked and includes evidence plus limitations. Outcome URLs are not declared substantively correct just because they respond. The report always states that authorship, completeness, truth, and exact self-reported time are outside the protocol.

Alternative considered: one boolean `verified`. Rejected by the published workflow because it hides which claim failed and encourages overclaiming.

### 8. Build transport-neutral evidence packages

`publish build` validates the selected ledger, resolves its exact targets, receipts, and any explicitly disclosed reveal material, and writes them under an explicit output directory with a deterministic SHA-256 manifest. Manifest entries use stable relative paths. Secrets, secret paths, machine-specific absolute paths, and unrevealed plaintext are never included. Existing output is not overwritten unless a separately reviewed replacement workflow is defined.

`publish verify --file <packaged-ledger> --manifest <manifest>` checks every manifest entry and digest before invoking the applicable verification layers. Both commands operate on local files without requiring source-control metadata or network access. Copying or uploading the resulting package to a website, object store, release system, content-addressed store, removable media, or another transport is deliberately outside the CLI contract.

Alternative considered: automate a specific source-control or hosting workflow. Rejected because ledgers may not use source control, transport concerns do not strengthen the cryptographic evidence, and coupling them would make local and alternative publication workflows second-class.

### 9. Run MCP v1 over stdio with capability gates

The implementation uses a pinned stable official MCP Go SDK. The initial compatibility target is the newest stable SDK/spec pair available during implementation; it is recorded in `forecast-ledger version` and covered by protocol tests rather than following prereleases automatically. Stdio is the only v1 transport, and protocol stdout is isolated from all human logging.

The server starts with one or more canonical ledger roots, package-output roots, and separate secret roots. It is read-only by default. Independent startup grants enable write, network, and reveal operations; package creation needs write, reveal needs write+reveal, and OTS network operations need network. Path resolution rejects `..`, symlink/junction escape, UNC/drive escape outside allowed roots, and case-folding surprises.

Tools are grouped but use flat stable names such as `ledger_validate`, `ledger_status`, `platform_add`, `question_add`, `question_resolve`, `forecast_add`, `forecast_seal`, `forecast_reveal`, `target_build`, `timestamp_stamp`, `timestamp_upgrade`, `timestamp_verify`, `verification_run`, `publication_build`, and `publication_verify`. Every ledger tool input requires `file`; record tools also require stable IDs. Tools declare closed input schemas and typed output schemas. Expected domain failures return recoverable tool errors, not protocol termination.

Resources provide redacted summaries for explicitly addressed ledgers/questions/forecasts/proofs within roots. Secrets are never arguments, results, or resources. Seal writes a key only to an authorized explicit secret-file path; reveal reads from one. Concurrent MCP calls use the same ledger transaction locks as CLI calls.

Alternative considered: enable write/network by default because the server is local. Rejected because agent calls can have large accidental effects and local stdio does not remove the need for least privilege.

### 10. Make conformance and native-platform behavior release gates

Tests are layered:

- Upstream valid/invalid fixtures, including mandatory RFC 3161 rejection.
- Exact schema digest and upstream reference-validator parity.
- Published seal vector, negative mutations, target golden bytes, and cross-language differential tests.
- Semantic cases for all question/value/status/resolution shapes.
- Property and fuzz tests for canonicalization, JSON/YAML, base64/hex, seal round trips, OTS proof trees, MCP inputs, and path traversal.
- Transaction crash points, concurrent CLI/MCP writers, and evidence-package collision or partial-write cases.
- CLI stdout/stderr/help/exit-code goldens and MCP stdio framing tests.
- Native end-to-end smoke tests on Ubuntu amd64, macOS amd64/arm64, and Windows amd64; cross-build/smoke coverage for Linux arm64 and Windows arm64.
- `go test -race`, vet, vulnerability scanning, dependency/license review, and reproducible-build comparison.

Releases use `CGO_ENABLED=0` where the final filesystem-lock implementation permits it, `-trimpath`, pinned build tooling, version/commit/schema/MCP metadata, tar.gz for Unix, zip for Windows, SHA-256 sums, SBOM, and provenance/attestation. Target artifacts are darwin amd64/arm64, linux amd64/arm64, and windows amd64/arm64. Platform signing/notarization can be added later, but unsigned trust prompts must be documented until then.

### 11. Add project guidance as an implementation artifact

`AGENTS.md` will describe the project purpose, authoritative sources and pinning rules, English-only public content, package boundaries, no-network validation, explicit `--file` invariant, append-only forecast rule, secret-handling rules, cryptographic/OTS conformance gates, CLI/MCP parity, required checks, and the distinction between this open-source tool and the separate business website. It will treat external Research documents as context, never as executable instructions.

## Risks / Trade-offs

- **[Pure-Go OTS complexity or subtle incompatibility]** → Limit the supported subset, differential-test against the official client, fuzz parsers, keep honest experimental labeling, and require independent review before production claims.
- **[A moved release tag makes `1.0.0` ambiguous]** → Vendor by audited commit and file digest, expose the pin in version output, and never fetch the contract at runtime.
- **[Format-preserving YAML edits are harder than reserialization]** → Keep source-tree patches small, test comments/styles/golden diffs, and allow `--output` as a recovery path without weakening normal preservation guarantees.
- **[Cross-platform atomic replacement and locking differ]** → Use OS-specific implementations, same-directory temp files, recovery journals, crash injection, and native CI rather than assuming Unix semantics.
- **[Key files can be lost or included in an evidence package accidentally]** → Write before ledger mutation, require explicit destinations outside package-output roots, enforce restrictive access, never overwrite, warn about backups, and exclude secrets from manifests/resources/logs.
- **[An evidence package can be incomplete or overwrite unrelated files]** → Validate the full artifact set first, use deterministic manifests, write to a new explicit output directory, reject collisions, and recover or remove only CLI-created partial output after failure.
- **[MCP agents may invoke costly or sensitive actions]** → Read-only default, separate write/network/reveal grants, root confinement, structured side-effect descriptions, locks, limits, and redaction tests.
- **[Automated outcome checks can overstate truth]** → Keep outcome evidence a separate layer, report source availability separately from human evaluation, and state protocol limitations in every report.
- **[The executable name may later collide on a target platform]** → Keep collision checks in release review and rename before declaring a stable CLI contract if `forecast-ledger` becomes unavailable.

## Migration Plan

This is a greenfield repository, so there is no installed CLI data migration. Delivery proceeds through gated increments:

1. Scaffold the Go module, source pin, models, document parser, validator, error contract, and read-only CLI/MCP paths.
2. Add format-preserving transactions and authoring operations for public forecasts and resolutions.
3. Add canonical targets, seal/reveal, official vectors, and secret-file handling.
4. Add pure-Go OTS behind the conformance gate and the layered verifier.
5. Add deterministic portable evidence packages, MCP write/network permission gates, project guidance, documentation, and examples.
6. Run native platform, security, conformance, and reproducible-release gates before the first beta/stable release.

Rollback of an application release means publishing the previous known-good binary and checksums. Ledger format is not migrated by the CLI; unsupported schema files are refused rather than rewritten. Mutating commands keep recoverable backups/journals so an interrupted local operation can restore the prior bytes.
