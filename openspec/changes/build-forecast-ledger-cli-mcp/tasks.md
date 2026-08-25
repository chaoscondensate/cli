## 1. Project Foundation and Guidance

- [x] 1.1 Check `forecast-ledger` executable/package-name collisions on macOS, Linux, Windows, Homebrew, Scoop, and Winget; confirm before exposing stable commands
- [x] 1.2 Create the `github.com/chaoscondensate/cli` Go module, `cmd/forecast-ledger` entrypoint, internal package skeleton, and reproducible developer build commands
- [x] 1.3 Select and pin reviewed versions of `urfave/cli/v3`, the stable official MCP Go SDK, JSON Schema/YAML, crypto, locking, and other dependencies; record license and vulnerability review results
- [x] 1.4 Vendor the v1.0.0 schema, upstream valid/invalid fixtures, reference attribution, and seal vector from commit `e409463d702888fefd253b32f21b9b2f864aabed`; assert the schema and release digests in tests
- [x] 1.5 Implement build/version metadata for binary version, source revision, schema version/commit/digest, Go version, and supported MCP protocol
- [x] 1.6 Write `AGENTS.md` with project context, source precedence, English-only public content, package boundaries, explicit `--file`, append-only history, secret rules, conformance gates, required checks, and open-source/business positioning

## 2. Document Model and Validation

- [x] 2.1 Define typed Forecast Ledger v1 models for identities, platforms, questions, all four forecast value types, lifecycle states, integrity, OTS metadata, sealed commitments, and resolutions
- [x] 2.2 Implement bounded JSON parsing with duplicate-key, invalid Unicode, float, and unsafe-integer rejection plus precise document locations
- [x] 2.3 Implement bounded YAML parsing with duplicate-key, alias/depth/size checks and a retained source tree for comments, order, style, and newline convention
- [x] 2.4 Embed and run the pinned Draft 2020-12 schema and format assertions without runtime network resolution
- [x] 2.5 Port upstream semantic checks for IDs, references, timezones, chronology, status/resolution rules, value/type agreement, probabilities, quantiles, target digests, and revealed mirrors
- [x] 2.6 Add source-tree patching that changes selected nodes while preserving untouched JSON/YAML data and YAML presentation
- [x] 2.7 Run all upstream valid and invalid fixtures through the Go validator, including explicit rejection of the RFC 3161 negative fixture
- [x] 2.8 Add parity tests against the upstream Python validator and fuzz/resource-limit tests for JSON and YAML readers

## 3. Storage, Paths, Transactions, and Errors

- [x] 3.1 Define stable domain error codes/details and CLI exit-code mappings for usage, invalid data, not found, conflict, verification, I/O, network, pending, internal, and interrupt outcomes
- [x] 3.2 Implement explicit ledger/artifact path resolution, safe relative-path handling, symlink/junction checks, Windows drive/UNC rules, and root confinement helpers
- [x] 3.3 Implement cross-platform exclusive ledger locks shared by CLI and MCP operations, with deterministic conflict reporting
- [x] 3.4 Implement parse-validate-patch-validate transactions with same-directory temporary files, flush, OS-specific safe replacement, and recovery journals
- [x] 3.5 Add crash-injection, permission, existing-output, concurrent-writer, path-traversal, case-folding, and Windows replacement tests

## 4. CLI Contract and Presentation

- [x] 4.1 Build the urfave command tree for init, validate, status, platform, question, forecast, target, timestamp, verify, publish, MCP, completion, and version
- [x] 4.2 Require `--file/-f` on every ledger operation and stable question/forecast selectors on every record operation; allow `--file -` only for eligible read-only commands
- [x] 4.3 Implement examples-first English help, typo guidance, full subcommand help, and shell completion for bash, zsh, fish, and PowerShell
- [x] 4.4 Implement human, plain, quiet, verbose, and stable JSON presenters with strict stdout/stderr separation, TTY detection, `NO_COLOR`, and secret redaction
- [x] 4.5 Implement `--no-input`, `--dry-run`, timeouts, context cancellation, and prompt rules for multi-file or network actions
- [x] 4.6 Add golden tests for help, stdout/stderr, JSON schemas, errors, exit codes, non-TTY output, cancellation, and accidental secret disclosure

## 5. Ledger Authoring Operations

- [ ] 5.1 Implement `forecast-ledger init` for new JSON/YAML ledgers with explicit identity, ledger ID, timezone, and no-overwrite behavior
- [x] 5.2 Implement `forecast-ledger validate` and read-only `forecast-ledger status` with local checks only and structured summaries
- [ ] 5.3 Implement platform add, update, list, show, and safe remove operations with reference-conflict checks
- [ ] 5.4 Implement question add, constrained update, list, show, and typed resolve/annul/dispute transitions with evidence and chronology checks
- [ ] 5.5 Implement public forecast add/list/show with all four value types, stable IDs, input-file/stdin support, and append-only supersession links
- [ ] 5.6 Add end-to-end authoring tests for JSON/YAML equivalence, unchanged historical forecasts, invalid transitions, duplicate IDs, format preservation, and rollback on failed post-validation

## 6. Canonical Targets and Sealed Forecasts

- [ ] 6.1 Implement bounded RFC 8785/JCS canonicalization with UTF-16 property ordering and published I-JSON/no-float restrictions
- [ ] 6.2 Implement typed public/sealed/revealed `forecast-envelope/v1` projections that exclude integrity and key hints exactly as specified
- [ ] 6.3 Implement target build/check commands with explicit selectors or `--all`, deterministic safe paths, SHA-256 reporting, and collision protection
- [ ] 6.4 Implement protected secret-file creation and reading with no overwrite, POSIX owner-only mode, Windows owner ACL, location checks, and complete output/log redaction
- [ ] 6.5 Implement atomic `forecast-ledger forecast seal` with OS CSPRNG salt/key/nonce, SHA-256 commitment, exact AAD, ChaCha20-Poly1305, and key-write-before-ledger semantics
- [ ] 6.6 Implement `forecast-ledger forecast reveal` with authenticate/decrypt/commitment/ID/canonical checks before mutation and retained original seal evidence
- [ ] 6.7 Reproduce the upstream seal vector and target bytes exactly; add negative, cross-language, cross-platform, property, and fuzz tests

## 7. Pure-Go OpenTimestamps

- [ ] 7.1 Define the timestamp backend interface and implement bounded detached-proof parsing/serialization for the supported SHA-256 OTS operations and attestations
- [ ] 7.2 Implement configured-calendar stamping with timeouts, response limits, explicit success policy, recoverable target/receipt writes, and pending metadata updates
- [ ] 7.3 Implement pending-receipt inspection and calendar upgrade without losing safe unknown proof nodes
- [ ] 7.4 Implement Bitcoin Core RPC verification and an explicitly selected HTTPS explorer adapter that reports its weaker trust boundary
- [ ] 7.5 Implement timestamp verify/status commands and ledger transitions among unanchored, pending, verified, and failed with exact target checks
- [ ] 7.6 Build official-client differential fixtures for Python-to-Go and Go-to-Python round trips and matching info/upgrade/verify results
- [ ] 7.7 Add malformed, oversized, deeply nested, timeout, redirect, calendar, and proof fuzz tests plus mocked CI and real-calendar nightly suites
- [ ] 7.8 Complete an independent review of the supported OTS subset and remove experimental labeling only after every conformance gate passes

## 8. Layered Verification

- [ ] 8.1 Define the stable evidence-matrix result model with pass, fail, pending, not-applicable, and not-checked states plus limitations
- [ ] 8.2 Implement document and content-binding layers that validate the ledger and rebuild/compare target bytes and digests
- [ ] 8.3 Implement existence-timing checks for receipt state, Bitcoin verification source, conservative upper bound, and precedence over `outcome_known_at`
- [ ] 8.4 Implement revealed-forecast checks for AEAD, commitment, bound IDs, canonical plaintext, public mirror, and unchanged sealed target
- [ ] 8.5 Implement outcome-evidence observations without converting URL availability, filesystem or source-control timestamps, or account control into stronger claims
- [ ] 8.6 Implement human and JSON verification reports that always state authorship, completeness, truth, and self-reported-time limitations
- [ ] 8.7 Add end-to-end tests for mixed pass/fail/pending layers, late anchors, tampered mirrors, missing artifacts, offline packages, and unavailable sources

## 9. Portable Evidence Packages

- [ ] 9.1 Implement deterministic publication manifests with safe relative paths, SHA-256 digests, stable bytes, and exclusion of secrets and unpublished plaintext
- [ ] 9.2 Implement `publish build --file <ledger> --output <directory>` with ledger/artifact validation, exact target/receipt selection, explicit disclosed reveal material, safe relative paths, and no overwrite
- [ ] 9.3 Implement `publish verify --file <packaged-ledger> --manifest <manifest>` with complete manifest path/digest checks followed by the applicable layered-verification checks
- [ ] 9.4 Add tests for standalone ledgers, missing/mismatched receipts, path escapes, secret and unrevealed-plaintext exclusion, output collisions, recoverable partial writes, offline verification, and deterministic manifests across platforms

## 10. MCP Server

- [ ] 10.1 Pin a stable official MCP Go SDK/spec pair and implement `forecast-ledger mcp serve` over stdio with protocol-only stdout and version negotiation tests
- [ ] 10.2 Implement read-only defaults, canonical ledger/package-output/secret roots, and separate write, network, and reveal startup grants
- [ ] 10.3 Define closed typed schemas and handlers for validation, status, inspection, list/show, target check, timestamp status, layered verification, and evidence-package verification tools
- [ ] 10.4 Add write-gated tools for ledger initialization, platform/question operations, public forecasts, resolution, target build, and evidence-package creation
- [ ] 10.5 Add separately gated seal/reveal and timestamp network tools using protected secret-file references only
- [ ] 10.6 Implement redacted ledger/question/forecast/proof/evidence-package resources confined to allowed roots, with no secret resources
- [ ] 10.7 Map expected domain failures to recoverable structured tool errors while reserving protocol errors for malformed MCP/session failures
- [ ] 10.8 Add in-memory and real-process stdio tests for tool schemas, parity with CLI errors/results, permissions, traversal, redaction, concurrent calls, cancellation, and stdout framing

## 11. Documentation and Examples

Public documentation delivery is owned by the
`establish-open-source-product-documentation` change. These completed handoff
items prevent the same deliverables from being tracked in two checklists; they
do not assert that the documentation itself is complete.

- [x] 11.1 Hand off README and workflow quick starts to documentation tasks 4.1–4.8, 5.3–5.4, and 6.1–6.3
- [x] 11.2 Hand off CLI reference, JSON, and exit-code documentation to tasks 7.1–7.2 and 7.5–7.6
- [x] 11.3 Hand off security and evidence-limit documentation to tasks 3.5, 4.4, 8.2–8.3, and 8.6
- [x] 11.4 Hand off MCP setup and safety documentation to tasks 6.4 and 7.3
- [x] 11.5 Hand off schema-update and conformance documentation to tasks 6.7, 7.5, 8.1, and 8.6
- [x] 11.6 Hand off checked examples and tutorials to tasks 5.3–5.4, 6.1–6.6, and 9.1–9.6

## 12. CI, Security, and Releases

- [ ] 12.1 Configure CI for formatting, unit/integration tests, race detection, vet, vulnerability scanning, dependency/license checks, and schema/crypto conformance
- [ ] 12.2 Add native end-to-end jobs for Ubuntu amd64, macOS amd64/arm64, and Windows amd64 plus cross-build/smoke coverage for Linux arm64 and Windows arm64
- [ ] 12.3 Add scheduled fuzz, real-calendar OTS, differential official-client, and longer crash/concurrency test workflows
- [ ] 12.4 Configure pinned reproducible release builds with `-trimpath`, version metadata, Unix tarballs, Windows zip files, and all six declared OS/architecture targets
- [ ] 12.5 Generate SHA-256 checksums, SBOMs, provenance/attestations, dependency metadata, and documented signature/notarization status for each release
- [ ] 12.6 Run clean-tree release gates for validation parity, crypto vectors, OTS conformance/review, MCP protocol, native smoke tests, reproducible builds, and checksum verification before the first stable release
