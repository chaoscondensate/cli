## 1. Shared operation contracts and change ownership

- [ ] 1.1 Add `internal/service` request, result, warning, side-effect, recovery, root/mode/network-profile, and operation interfaces without creating a storage-to-service import cycle.
- [ ] 1.2 Define closed typed input models and versioned JSON Schemas for init, root metadata, platform, question, forecast, resolution, dispute, and publication inputs, including explicit null/removal semantics.
- [ ] 1.3 Reuse the bounded JSON/YAML parser for operation inputs and add rejection tests for duplicate keys, unknown fields, multiple documents, invalid exact-number forms, excessive depth, and excessive bytes.
- [ ] 1.4 Build validated indexes for platform IDs, question IDs, globally unique forecast IDs, per-question order, platform references, and forecast supersession links.
- [ ] 1.5 Add injectable observation clock and CSPRNG effect interfaces whose deterministic implementations are available only to tests.
- [ ] 1.6 Implement shared dry-run planning, explicit confirmation, non-interactive approval, cancellation, and timeout policies for CLI and MCP callers.
- [ ] 1.7 Extend stable public result/error codes and CLI exit mapping for usage, invalid, not-found, conflict, verification, I/O, network-disabled, incomplete, pending, unavailable, and interruption outcomes; do not add a separate permission exit class.
- [ ] 1.8 Generate CLI input/result references and MCP schemas from the operation contracts, and fail CI when regeneration changes the tree.
- [ ] 1.9 Reject every ledger whose root `schema_version` is not exactly `1.0.0` with code `unsupported_schema_version` and invalid-data exit 3, without fetching, coercing, or guessing a migration.
- [ ] 1.10 Mark `build-forecast-ledger-cli-mcp` as fully superseded by this change, map its 31 completed implementation tasks to the replacement requirements, and prevent that older delta from being synced or archived separately.

## 2. Transaction and path safety

- [ ] 2.1 Implement minimal source-tree patches for JSON/YAML that preserve format, newline convention, and untouched YAML comments, ordering, and scalar style.
- [ ] 2.2 Complete cross-platform ledger locking and add concurrent writer tests on macOS, Linux, and Windows.
- [ ] 2.3 Complete recoverable same-directory ledger replacement with prospective decode/schema/semantic validation and fault injection at every write, flush, replace, and cleanup boundary.
- [ ] 2.4 Implement a multi-resource side-effect plan and recovery journal that records ownership and rollback class for target, receipt, key, package, and ledger files.
- [ ] 2.5 Implement confined path resolution for POSIX and Windows paths, including symlink, junction, reparse-point, drive, UNC, device, traversal, and case-fold collision rejection.
- [ ] 2.6 Add same-bytes idempotency and different-bytes conflict helpers for deterministic artifacts without following existing links.
- [ ] 2.7 Add crash/retry tests proving that failures preserve the original ledger and never delete or replace unowned files.

## 3. Ledger initialization and root metadata

- [ ] 3.1 Implement root forecaster, platform, timezone, ID, and chronology builders for schema version `1.0.0`, requiring at least two members when the forecaster kind is `team`.
- [ ] 3.2 Implement the exactly-one initial typed question and exactly-one initial public forecast path with full prospective validation.
- [ ] 3.3 Implement the initial sealed forecast path using the exact six-field private mirror input, explicit new `--key-file`, protected-key-first commit ordering, and recovery reporting.
- [ ] 3.4 Implement exclusive JSON/YAML ledger creation and reject every pre-existing file, link, junction, reparse point, or directory destination.
- [ ] 3.5 Wire `forecast-ledger init` in urfave with leaf-local `--file`, required root identity flags, `--input`, conditional `--key-file`, dry-run, approval, JSON, and stable exit behavior.
- [ ] 3.6 Add init schema, semantic, presentation, private-input, rollback, JSON golden, help, completion, and native-platform acceptance tests; unhide `init` only when they pass.
- [ ] 3.7 Implement `ledger update` for the root title, description, default timezone, and mutable forecaster profile fields, with omission-preserves/null-removes semantics for optional values and immutable schema, ledger, creation, forecaster identity/kind, collection, and imported-publication fields.
- [ ] 3.8 Wire and test `forecast-ledger ledger update` with leaf-local `--file` and `--input`, dry-run, approval, JSON, minimal-patch preservation, help, completion, and native-platform acceptance gates.

## 4. Platform commands

- [ ] 4.1 Implement platform kind, URI, account, unique-ID, and reference validation shared by add/update/init.
- [ ] 4.2 Implement `platform add` with a closed input, exact map-key insertion, prospective validation, and duplicate conflict reporting.
- [ ] 4.3 Implement `platform update` with the mutable-field allowlist, omission-preserves/null-removes behavior, and reference-safe prospective validation.
- [ ] 4.4 Implement deterministic `platform list` results sorted by ID with question reference counts, including supported stdin-ledger reads.
- [ ] 4.5 Implement redacted `platform show` with the exact selected record and sorted referencing question IDs, including supported stdin-ledger reads.
- [ ] 4.6 Implement approved `platform remove` with a no-reference precondition and sorted conflict details.
- [ ] 4.7 Wire all five urfave platform actions with exact leaf-local `--file`/`--platform`/`--input` flags, dry-run/approval, JSON envelopes, help, and completion.
- [ ] 4.8 Add unit, minimal-patch, stdin, concurrency, error, JSON golden, and native-platform acceptance tests; unhide each platform action independently when its gate passes.

## 5. Question authoring and lifecycle

- [ ] 5.1 Implement binary, multiple-choice, numeric, and date question builders with required scalar `--type`, required `forecast_window` and `expected_resolution_at`, type-specific exclusion, unique options/tags, platform-reference, and window chronology rules; reject a duplicate `type` field in `--input`.
- [ ] 5.2 Implement `question add` with exactly one initial public or sealed forecast, conditional protected `--key-file`, global ID checks, and one atomic valid commit.
- [ ] 5.3 Implement `question update` with the descriptive/unresolved-status allowlist, checks that changed windows still contain every existing forecast, and precomputed-target conflict rejection when a target-covered change would stale any retained target or integrity artifact.
- [ ] 5.4 Implement deterministic `question list` with type, lifecycle, window, forecast count, expected resolution, and integrity counts.
- [ ] 5.5 Implement redacted `question show` with resolution metadata and forecast summaries but no sealed plaintext or revealed key material.
- [ ] 5.6 Implement `question resolve` for closed/awaiting or disputed questions with typed outcomes, evidence-source validation, chronology, confirmation, retained forecasts, and explicit notice that resolving a dispute replaces the v1 current-resolution object.
- [ ] 5.7 Implement `question annul` for unresolved, resolved, or disputed questions with reason/evidence validation, confirmation for terminal replacement, retained forecasts/evidence, and explicit notice that v1 has no internal resolution history.
- [ ] 5.8 Implement `question dispute` only for resolved/annulled questions, replacing the v1 current resolution object while reporting prior status and the lack of internal resolution history.
- [ ] 5.9 Wire all seven urfave question actions with exact selectors, closed inputs, conditional key destination, dry-run/approval, JSON envelopes, help, and completion.
- [ ] 5.10 Add lifecycle-transition tables, typed-outcome tests, sealed-first-forecast tests, minimal-patch checks, redaction canaries, JSON goldens, and native-platform acceptance tests; unhide each question action independently.

## 6. Public forecast commands

- [ ] 6.1 Implement shared forecast value validation for binary basis points, complete multiple-choice distributions, and exact numeric/date point, interval, and quantile forms.
- [ ] 6.2 Implement forecast time/window/order validation and append-only supersession checks restricted to an earlier forecast in the same question.
- [ ] 6.3 Implement `forecast add` for open questions with globally unique IDs, visibility `public`, integrity `unanchored`, and no accepted commitment/encryption fields.
- [ ] 6.4 Implement deterministic `forecast list` in recorded order without collapsing the supersession chain.
- [ ] 6.5 Implement `forecast show` for public/revealed records and redacted sealed summaries without decryption or network work.
- [ ] 6.6 Wire the three public urfave forecast actions with exact question/forecast selectors, input handling, dry-run, JSON envelopes, help, and completion.
- [ ] 6.7 Add boundary, ordering, duplicate-global-ID, supersession, redaction, stdin-read, minimal-patch, JSON golden, and native-platform tests; unhide `forecast add|list|show` independently.

## 7. Deterministic target commands

- [ ] 7.1 Implement typed `forecast-envelope/v1` projections whose root is exactly `schema`, `ledger_id`, `question`, and `forecast`, with the specified question/type fields and public, sealed, and revealed-as-originally-sealed forecast allowlists and no leaked integrity, key, resolution, or unrelated ledger fields.
- [ ] 7.2 Complete bounded RFC 8785/JCS behavior and cross-language fixtures for UTF-16 property order, I-JSON limits, exact UTF-8 bytes, and SHA-256 digests.
- [ ] 7.3 Implement deterministic target path resolution at `proofs/targets/<forecast-id>.json` and metadata construction without mutating ledger integrity.
- [ ] 7.4 Implement `target build` for one selector and `--all`, with complete preflight, exclusive/identical/conflict behavior, recovery, and dry-run.
- [ ] 7.5 Implement non-mutating `target check` against reconstructed bytes, digest, scope, canonicalization, path, and recorded integrity metadata.
- [ ] 7.6 Wire both urfave target actions with mutually exclusive `--all` versus question/forecast selectors, real-file enforcement, JSON output, help, and completion.
- [ ] 7.7 Add pinned, cross-platform, altered-byte, collision, all-or-none, cancellation, recovery, and JSON golden tests; unhide `target build|check` independently.

## 8. Sealed forecast lifecycle

- [ ] 8.1 Implement the exact closed canonical `forecast-seal/v1` plaintext containing only `bundle`, `forecast_id`, `question_id`, `salt`, and `schema`, and associated data containing only `scheme`, `question_id`, `forecast_id`, and `commitment_sha256`; explicitly do not bind `ledger_id`, `public_note`, `supersedes`, or `key_hint` into the seal.
- [ ] 8.2 Implement ChaCha20-Poly1305 sealing with fresh 32-byte salt/key and 12-byte nonce from the OS CSPRNG and reproduce the pinned upstream vector under test entropy.
- [ ] 8.3 Implement exact `forecast-key/v1` bytes as a closed JCS object containing only `schema`, `question_id`, `forecast_id`, and lowercase `key_hex`, followed by one LF, with exclusive protected creation, POSIX `0600` checks, and fail-closed Windows owner-only ACL checks.
- [ ] 8.4 Implement `forecast seal` private-file/stdin input requiring `forecasted_at`, `recorded_at`, `value`, `rationale`, `key_factors`, and `comment` after defaults, open-question/supersession validation, protected-key-first append of a new sealed forecast, no in-place hiding or resealing, and retained-orphan recovery.
- [ ] 8.5 Implement full reveal authentication for AEAD, commitment, exact associated data, question/forecast IDs, protocol, canonical bundle, typed mirror, and original sealed target continuity before mutation, with no nonexistent ledger-ID seal check.
- [ ] 8.6 Implement `forecast reveal` confirmation, all required revealed fields (`value`, `rationale`, `key_factors`, `comment`, and retained `commitment`), retained ciphertext/integrity evidence, and correct-key idempotency.
- [ ] 8.7 Wire `forecast seal|reveal` in urfave with exact selectors, private `--input`, explicit `--key-file`, dry-run/approval, secret-safe JSON, help, and completion.
- [ ] 8.8 Add positive plaintext/AAD/target/key byte vectors, cross-ledger and bound-versus-unbound field cases, cross-language, wrong-key, tampered-field, fuzz, property, rollback, key-protection, output-writer-failure, secret-canary, and native-platform tests; unhide seal/reveal independently only after their gates pass.

## 9. Pure-Go OpenTimestamps core

- [ ] 9.1 Implement bounded detached SHA-256 receipt parsing for the reviewed OTS operation and attestation subset without Python or subprocesses.
- [ ] 9.2 Implement a lossless proof tree that deterministically serializes supported nodes and safely preserves or explicitly rejects unknown nodes.
- [ ] 9.3 Implement proof-operation evaluation, attestation extraction, semantic-superset comparison, deterministic branch merge, and receipt binding checks.
- [ ] 9.4 Implement immutable `opentimestamps-public-v1` metadata with four pinned submission endpoints, accepted receipt identities, fixed two-of-four success, two public Bitcoin APIs, source IDs/trust text, help/version exposure, and no runtime download or extension.
- [ ] 9.5 Implement optional CLI Bitcoin Core verification with protected auth-file reading, bounded RPC requests, precedence over the public profile, and independently operated trust reporting.
- [ ] 9.6 Implement dual-public-source verification against mempool.space and Blockstream with required hash/header agreement, local header/proof-of-work/attestation checks, bounded requests, and mandatory third-party trust limitations.
- [ ] 9.7 Add official Python client round-trip/info/upgrade/verify differential fixtures for every supported operation and attestation.
- [ ] 9.8 Add malformed, oversized, excessive-depth, unsupported-operation, redirect, timeout, and parser/evaluator fuzz tests.
- [ ] 9.9 Add profile identity, mocked 2-of-4 calendar, dual-source agreement/disagreement/outage, Bitcoin Core, real-calendar nightly, and tracked independent-review gates before stable OTS availability.

## 10. Timestamp commands

- [ ] 10.1 Implement `timestamp stamp` preflight, concurrent built-in four-calendar submission, fixed two-response success, deterministic receipt merge, durable target/receipt writes, and pending ledger transition.
- [ ] 10.2 Implement stamp retry and recovery for target/receipt/ledger interruption without duplicate implicit requests or deletion of unowned artifacts.
- [ ] 10.3 Implement `timestamp upgrade` for matching pending evidence, built-in-profile calendar identities only, semantic-superset receipt replacement, and pending/not-ready results without premature verification.
- [ ] 10.4 Implement local-only `timestamp status` states `unanchored`, `pending`, `confirmed_unverified`, `verified`, `failed`, and `inconsistent` with safe next actions; classify missing artifacts referenced by pending or verified evidence as `inconsistent`.
- [ ] 10.5 Implement zero-config `timestamp verify` through the dual-public-source profile plus optional CLI Bitcoin Core override after exact target/receipt checks, deriving conservative `anchored_before` and block height.
- [ ] 10.6 Implement atomic pending-to-verified ledger transition, retained proof references, byte-for-byte preservation of every imported `external_anchor`, explicit verification time, and the valid-but-too-late warning; treat imported `failed` integrity as terminal and require a new forecast revision for recovery.
- [ ] 10.7 Wire all four urfave timestamp actions with exact selectors, zero-config built-in sources, `--offline`, optional Core URL/protected auth options, dry-run, timeout, JSON results, honest exits, help, and completion.
- [ ] 10.8 Add idempotency, offline-no-socket, pending/network/crypto exit, public-source agreement/disagreement, late-anchor, recovery, source-trust, credential-redaction, cancellation, JSON golden, and native-platform acceptance tests; unhide actions only after applicable OTS gates pass.

## 11. Layered verification

- [ ] 11.1 Implement the stable layer result model with `pass`, `fail`, `pending`, `not_applicable`, and `not_checked`, deterministic aggregation, reason codes, evidence, and limitations; return overall `incomplete` with exit 9 whenever required work remains and reserve exit 0 for a complete pass.
- [ ] 11.2 Implement document verification with bounded parse, embedded schema, format, domain, semantic, and artifact-path checks plus dependency blocking.
- [ ] 11.3 Implement content-binding verification by rebuilding and comparing exact public or original sealed targets and every recorded metadata field.
- [ ] 11.4 Implement automatic built-in-profile existence-timing verification, explicit offline behavior, and optional CLI Bitcoin Core replacement, including proof validity versus pre-outcome sufficiency.
- [ ] 11.5 Implement revealed-forecast authentication and mirror comparison without exposing decrypted fields or stored keys.
- [ ] 11.6 Implement outcome metadata/digest checks and optional bounded `--check-sources` reachability without asserting authority or substantive truth.
- [ ] 11.7 Include authorship, completeness, calibration, self-reported-time, and outcome-truth limitations in every human, JSON, and MCP report.
- [ ] 11.8 Wire `forecast-ledger verify` with all/default and narrowed selectors, real-file enforcement, automatic built-in Bitcoin sources, `--offline`, optional Core, separate `--check-sources`, deterministic JSON, and documented exit precedence.
- [ ] 11.9 Add mixed-matrix, dependency, edited-ledger, late-proof, edited-reveal, source, offline-no-network, determinism, JSON golden, and native-platform tests; unhide `verify` only when complete.

## 12. Publication commands

- [ ] 12.1 Define the closed versioned canonical publication manifest and deterministic path, role, ordering, digest, and schema-pin rules.
- [ ] 12.2 Implement the allowlisted evidence graph rooted at the exact complete ledger and every referenced target/OTS receipt, with no source-control or recursive directory discovery.
- [ ] 12.3 Implement secret/private/temp/lock/journal exclusion, protected-root checks, absolute-path checks, package secret-canary scanning, and rejection of path-like `key_hint` values; preserve safe logical hints such as `forecast-key:<forecast-id>` without copying or recording the actual key path.
- [ ] 12.4 Implement `publish build` full source verification, collision preflight, new-root creation, safe file copying, manifest-last commit, dry-run, and interruption recovery.
- [ ] 12.5 Implement `publish verify` manifest-first confinement, listed file size/digest/role checks, unexpected-file rejection, then document/content/reveal/offline timestamp verification.
- [ ] 12.6 Implement optional `--online` package timing checks through the built-in public profile plus optional Core replacement, without coupling package integrity to source availability.
- [ ] 12.7 Wire both urfave publication actions with explicit `--file`, `--output` or `--manifest`, dry-run, JSON identity/matrix output, help, and completion.
- [ ] 12.8 Add standalone source-control-free, byte-determinism, missing/tampered/extra file, traversal/link/case collision, adjacent-key, pending-proof, interruption, removable-copy, JSON golden, and native-platform tests; unhide both actions independently.

## 13. MCP runtime and parity

- [ ] 13.1 Add the pinned MCP SDK and implement `mcp serve` stdio initialization, version negotiation, root/mode and built-in-profile metadata, clean stdout, stderr diagnostics, graceful EOF, and shutdown.
- [ ] 13.2 Implement startup canonicalization and overlap validation for named ledger, output, and secret roots with no inferred default ledger.
- [ ] 13.3 Implement default full online operation plus optional whole-server `--read-only` and `--offline` modes, with required root-class checks before secret, network, lock, or file effects and no per-capability grants.
- [ ] 13.4 Generate and register closed tools for ledger init/update/validate/status and all platform actions over the shared service operations.
- [ ] 13.5 Generate and register closed tools for all question and forecast actions, including private protected-file references and explicit confirmation fields.
- [ ] 13.6 Generate and register closed tools for target, timestamp, layered verification, and publication actions with matching dry-run/offline behavior, built-in network profile use, and no endpoint URL inputs.
- [ ] 13.7 Implement versioned redacted `forecast-ledger://` resources for addressed ledgers, questions, forecasts, artifact summaries, reports, and manifests.
- [ ] 13.8 Implement stable recoverable tool failures, protocol-error separation, request cancellation/timeouts, immediate deterministic conflict for a second writer on the same ledger, cross-ledger concurrency, and global resource limits.
- [ ] 13.9 Wire urfave `mcp serve` with repeatable ledger/output/secret roots, optional `--read-only`/`--offline`, limits, help, completion, and no protocol-breaking banner.
- [ ] 13.10 Register only completed tools in MCP discovery, return unknown-tool for direct calls to absent tools without side effects, and add in-memory and real-process tests for incremental discovery, every available schema/tool, CLI parity, root-class combinations, full/read-only and online/offline modes, pinned-endpoint confinement, root races/traversal, secret canaries, framing purity, cancellation, immediate writer conflicts, malformed/oversized requests, and prior supported protocol compatibility; unhide `mcp serve` only when complete.

## 14. Command surface and user documentation

- [ ] 14.1 Add a command-availability manifest that proves every visible leaf has a real service action and complete acceptance gate, and fails if any planned leaf still uses the unavailable handler at final release.
- [ ] 14.2 Add full English `--help` text and examples for every command using explicit `--file`, simple terminology, safe placeholders, stdin rules, built-in network-profile disclosure, offline/Core choices, dry-run, approvals, exits, `--plain`/`--quiet` output rules, and no secrets.
- [ ] 14.3 Generate and review command reference pages covering every flag, closed input shape, output envelope, business precondition, mutation, artifact, recovery, and network behavior.
- [ ] 14.4 Document the complete public/sealed/reveal, target/timestamp/verify, zero-config network profile, standalone publication, and MCP root/read-only/offline workflows in README and project documentation.
- [ ] 14.5 Document schema v1 initial-question/forecast constraints, global ID uniqueness, append-only revisions, dispute replacement semantics, OTS trust boundaries, and all verification limitations.
- [ ] 14.6 Update `AGENTS.md` with rules requiring code changes to update README, command references, examples, schemas, OpenSpec artifacts, and release notes when observable behavior changes, and reconcile its stale MCP read-only/grant wording with the full-by-default, root-confined, optional read-only/offline design.
- [ ] 14.7 Mention Chaos Condensate and `https://chaoscondensate.com/` in appropriate README, project-context, and documentation locations without characterizing the organization, practice, or website as non-commercial.
- [ ] 14.8 Add docs link, example, command/flag, generated-reference, secret-canary, simple-English lint, and clean-tree checks to CI.

## 15. Cross-platform conformance and release readiness

- [ ] 15.1 Add CLI black-box parity tests for every command in human, JSON, and plain modes, both valid `--json` placements, `--quiet`, `--no-color`, `TERM=dumb`, stdin eligibility, exact schema-version refusal, and stable exit codes.
- [ ] 15.2 Add macOS, Linux, and Windows CI matrices for locking, replacement, path confinement, key permissions/ACLs, target/package determinism, cancellation, and recovery.
- [ ] 15.3 Add race tests and bounded fuzz targets for parsers, selectors, canonicalization, crypto inputs, OTS proofs, manifests, MCP messages, and path normalization.
- [ ] 15.4 Add Go/Python schema validator parity using normalized verdicts, issue codes, and locations for shared fixtures.
- [ ] 15.5 Run pinned target/seal vectors, OTS differential suites, layered verification matrices, publication determinism, and real-process MCP tests as release-blocking checks.
- [ ] 15.6 Verify GoReleaser cross-builds and package smoke tests exercise the complete visible command/help surface on macOS, Linux, and Windows artifacts.
- [ ] 15.7 Update changelog and release notes with newly available commands, correction of hidden preview flags and stdin behavior before those commands become stable, experimental OTS status, migration/no-migration statement, recovery behavior, and security/trust limitations.
- [ ] 15.8 Run `go test`, race, fuzz smoke, lint, generated-file, OpenSpec strict validation, docs, package, and native smoke gates; record evidence that all 29 requested actions and every advertised MCP parity tool are implemented before marking the change complete.
