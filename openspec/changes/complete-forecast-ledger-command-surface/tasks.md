## 1. Shared operation contracts and change ownership

- [x] 1.1 Add `internal/service` request, result, warning, side-effect, recovery, root/mode/network-profile, and operation interfaces without creating a storage-to-service import cycle.
- [x] 1.2 Define closed typed input models and versioned JSON Schemas for init, root metadata, platform, question, forecast, key-hint repair, resolution, dispute, and publication inputs, including explicit null/removal semantics and deliberate init/question-add type normalization.
- [x] 1.3 Reuse the bounded JSON/YAML parser for operation inputs and add rejection tests for duplicate keys, unknown fields, multiple documents, invalid exact-number forms, excessive depth, and excessive bytes.
- [x] 1.4 Build validated indexes for platform IDs, question IDs, globally unique forecast IDs, per-question order, platform references, and forecast supersession links.
- [x] 1.5 Add injectable observation clock and CSPRNG effect interfaces whose deterministic implementations are available only to tests.
- [x] 1.6 Implement shared dry-run planning only for persistent mutation/resource creation, explicit confirmation, non-interactive approval, cancellation, and timeout policies for CLI and MCP callers; keep read-only network verification controlled by online/offline mode rather than dry-run.
- [x] 1.7 Extend stable public result/error codes and CLI exit mapping for usage, invalid, not-found, conflict, verification, I/O, network-disabled, incomplete, pending, unavailable, and interruption outcomes; do not add a separate permission exit class.
- [x] 1.8 Generate CLI input/result references and MCP schemas from the operation contracts, and fail CI when regeneration changes the tree.
- [x] 1.9 Reject every ledger whose root `schema_version` is not exactly `1.0.0` with code `unsupported_schema_version` and invalid-data exit 3, without fetching, coercing, or guessing a migration.
- [x] 1.10 Mark `build-forecast-ledger-cli-mcp` as fully superseded by this change, map its 31 completed implementation tasks to the replacement requirements, and prevent that older delta from being synced or archived separately.

## 2. Transaction and path safety

- [x] 2.1 Implement minimal source-tree patches for JSON/YAML that preserve format, newline convention, and untouched YAML comments, ordering, and scalar style.
- [x] 2.2 Complete cross-platform ledger locking and add concurrent writer tests on macOS, Linux, and Windows.
- [x] 2.3 Complete recoverable same-directory ledger replacement with prospective decode/schema/semantic validation and fault injection at every write, flush, replace, and cleanup boundary.
- [x] 2.4 Implement a multi-resource side-effect plan and recovery journal that records ownership and rollback class for target, receipt, key, package, and ledger files.
- [x] 2.5 Implement confined path resolution for POSIX and Windows paths, including symlink, junction, reparse-point, drive, UNC, device, traversal, and case-fold collision rejection.
- [x] 2.6 Add same-bytes idempotency and different-bytes conflict helpers for deterministic artifacts without following existing links.
- [x] 2.7 Add crash/retry tests proving that failures preserve the original ledger and never delete or replace unowned files.

## 3. Ledger initialization and root metadata

- [x] 3.1 Implement root forecaster, platform, timezone, ID, and chronology builders for schema version `1.0.0`, requiring at least two members when the forecaster kind is `team`.
- [x] 3.2 Implement the exactly-one initial typed question and exactly-one initial public forecast path with full prospective validation.
- [x] 3.3 Implement the initial sealed forecast path using the exact six-field private mirror input, explicit new `--key-file`, protected-key-first commit ordering, and recovery reporting.
- [x] 3.4 Implement exclusive JSON/YAML ledger creation and reject every pre-existing file, link, junction, reparse point, or directory destination.
- [x] 3.5 Wire `forecast-ledger init` in urfave with leaf-local `--file`, required root identity flags, `--input`, conditional `--key-file`, dry-run, approval, JSON, and stable exit behavior.
- [x] 3.6 Add init schema, semantic, presentation, private-input, rollback, JSON golden, help, completion, and native-platform acceptance tests; unhide `init` only when they pass.
- [x] 3.7 Implement `ledger update` for root title, description, default timezone, and mutable forecaster kind/profile fields, including atomic individual/team member-shape transitions, omission-preserves/null-removes semantics, immutable schema/ledger/creation/forecaster ID/collections/publication, and current-metadata/no-authorship warnings.
- [x] 3.8 Wire and test `forecast-ledger ledger update` with leaf-local `--file` and `--input`, dry-run, approval, JSON, minimal-patch preservation, help, completion, and native-platform acceptance gates.

## 4. Platform commands

- [x] 4.1 Implement platform kind, URI, account, unique-ID, and reference validation shared by add/update/init.
- [x] 4.2 Implement `platform add` with a closed input, exact map-key insertion, prospective validation, and duplicate conflict reporting.
- [x] 4.3 Implement `platform update` with the mutable-field allowlist, omission-preserves/null-removes behavior, and reference-safe prospective validation.
- [x] 4.4 Implement deterministic `platform list` results sorted by ID with question reference counts, including supported stdin-ledger reads.
- [x] 4.5 Implement redacted `platform show` with the exact selected record and sorted referencing question IDs, including supported stdin-ledger reads.
- [x] 4.6 Implement approved `platform remove` with a no-reference precondition and sorted conflict details.
- [x] 4.7 Wire all five urfave platform actions with exact leaf-local `--file`/`--platform`/`--input` flags, dry-run/approval, JSON envelopes, help, and completion.
- [x] 4.8 Add unit, minimal-patch, stdin, concurrency, error, JSON golden, and native-platform acceptance tests; unhide each platform action independently when its gate passes.

## 5. Question authoring and lifecycle

- [x] 5.1 Implement shared binary, multiple-choice, numeric, and date question builders with init-document versus scalar question-add type normalization, required `forecast_window` and `expected_resolution_at`, type-specific exclusion, unique options/tags, platform-reference, and window chronology rules; reject a duplicate `type` field in question-add `--input`.
- [x] 5.2 Implement `question add` with exactly one initial public or sealed forecast, conditional protected `--key-file`, global ID checks, and one atomic valid commit.
- [x] 5.3 Implement `question update` with the descriptive/unresolved-status allowlist, checks that changed windows still contain every existing forecast, precomputed-target conflict rejection when evidence would stale, and actionable annul-plus-new-question guidance with no override or target rewrite.
- [x] 5.4 Implement deterministic `question list` with type, lifecycle, window, forecast count, expected resolution, and integrity counts.
- [x] 5.5 Implement redacted `question show` with resolution metadata and forecast summaries but no sealed plaintext or revealed key material.
- [x] 5.6 Implement `question resolve` for closed/awaiting or disputed questions with typed outcomes, evidence-source validation, chronology, confirmation, retained forecasts, and explicit notice that resolving a dispute replaces the v1 current-resolution object.
- [x] 5.7 Implement `question annul` for unresolved, resolved, or disputed questions with reason/evidence validation, confirmation for terminal replacement, retained forecasts/evidence, and explicit notice that v1 has no internal resolution history.
- [x] 5.8 Implement `question dispute` only for resolved/annulled questions, replacing the v1 current resolution object while reporting prior status and the lack of internal resolution history.
- [x] 5.9 Wire all seven urfave question actions with exact selectors, closed inputs, conditional key destination, dry-run/approval, JSON envelopes, help, and completion.
- [x] 5.10 Add lifecycle-transition tables, typed-outcome tests, sealed-first-forecast tests, minimal-patch checks, redaction canaries, JSON goldens, and native-platform acceptance tests; unhide each question action independently.

## 6. Public forecast commands

- [x] 6.1 Implement shared forecast value validation for binary basis points, complete multiple-choice distributions, and exact numeric/date point, interval, and quantile forms.
- [x] 6.2 Implement forecast time/window/order validation and append-only supersession checks restricted to an earlier forecast in the same question.
- [x] 6.3 Implement `forecast add` for open questions with globally unique IDs, visibility `public`, integrity `unanchored`, and no accepted commitment/encryption fields.
- [x] 6.4 Implement deterministic `forecast list` in recorded order without collapsing the supersession chain.
- [x] 6.5 Implement `forecast show` for public/revealed records and redacted sealed summaries without decryption or network work.
- [x] 6.6 Wire the three public urfave forecast actions with exact question/forecast selectors, input handling, dry-run, JSON envelopes, help, and completion.
- [x] 6.7 Add boundary, ordering, duplicate-global-ID, supersession, redaction, stdin-read, minimal-patch, JSON golden, and native-platform tests; unhide `forecast add|list|show` independently.

## 7. Deterministic target commands

- [x] 7.1 Implement typed `forecast-envelope/v1` projections whose root is exactly `schema`, `ledger_id`, `question`, and `forecast`, with the specified question/type fields and public, sealed, and revealed-as-originally-sealed forecast allowlists and no leaked integrity, key, resolution, or unrelated ledger fields.
- [x] 7.2 Complete bounded RFC 8785/JCS behavior and cross-language fixtures for UTF-16 property order, I-JSON limits, exact UTF-8 bytes, and SHA-256 digests.
- [x] 7.3 Implement deterministic target path resolution at `proofs/targets/<forecast-id>.json` and metadata construction without mutating ledger integrity.
- [x] 7.4 Implement `target build` for one selector and `--all`, with complete preflight, exclusive/identical/conflict behavior, recovery, and dry-run.
- [x] 7.5 Implement non-mutating `target check` against reconstructed bytes, digest, scope, canonicalization, path, and recorded integrity metadata.
- [x] 7.6 Wire both urfave target actions with mutually exclusive `--all` versus question/forecast selectors, real-file enforcement, JSON output, help, and completion.
- [x] 7.7 Add pinned, cross-platform, altered-byte, collision, all-or-none, cancellation, recovery, and JSON golden tests; unhide `target build|check` independently.

## 8. Sealed forecast lifecycle

- [x] 8.1 Implement the exact closed canonical `forecast-seal/v1` plaintext containing only `bundle`, `forecast_id`, `question_id`, `salt`, and `schema`, and associated data containing only `scheme`, `question_id`, `forecast_id`, and `commitment_sha256`; explicitly do not bind `ledger_id`, `public_note`, `supersedes`, or `key_hint` into the seal.
- [x] 8.2 Implement ChaCha20-Poly1305 sealing with fresh 32-byte salt/key and 12-byte nonce from the OS CSPRNG and reproduce the pinned upstream vector under test entropy.
- [x] 8.3 Implement exact `forecast-key/v1` bytes as a closed JCS object containing only `schema`, `question_id`, `forecast_id`, and lowercase `key_hex`, followed by one LF, with exclusive protected creation, POSIX `0600` checks, and fail-closed Windows owner-only ACL checks.
- [x] 8.4 Implement `forecast seal` private-file/stdin input requiring `forecasted_at`, `recorded_at`, `value`, `rationale`, `key_factors`, and `comment` after defaults, open-question/supersession validation, protected-key-first append of a new sealed forecast, no in-place hiding or resealing, and retained-orphan recovery.
- [x] 8.5 Implement full reveal authentication for AEAD, commitment, exact associated data, question/forecast IDs, protocol, canonical bundle, typed mirror, and original sealed target continuity before mutation, with no nonexistent ledger-ID seal check.
- [x] 8.6 Implement `forecast reveal` confirmation, all required revealed fields (`value`, `rationale`, `key_factors`, `comment`, and retained `commitment`), retained ciphertext/integrity evidence, and correct-key idempotency.
- [x] 8.7 Wire `forecast seal|reveal` in urfave with exact selectors, private `--input`, explicit `--key-file`, dry-run/approval, secret-safe JSON, help, and completion.
- [x] 8.8 Add positive plaintext/AAD/target/key byte vectors, cross-ledger and bound-versus-unbound field cases, cross-language, wrong-key, tampered-field, fuzz, property, rollback, key-protection, output-writer-failure, secret-canary, and native-platform tests; unhide seal/reveal independently only after their gates pass.
- [x] 8.9 Implement `forecast key-hint update` for sealed/revealed records using the closed path-free `scheme:opaque` grammar, idempotent single-field patching, explicit support for imported failed integrity, and proof that seal/target/receipt/integrity bytes remain unchanged.
- [x] 8.10 Wire CLI and MCP key-hint update with exact selectors and scalar hint, dry-run, JSON/help/completion, package-repair guidance, invalid path/credential cases, and native-platform acceptance tests.

## 9. Pure-Go OpenTimestamps core

- [x] 9.1 Implement bounded detached SHA-256 receipt parsing and construction for the reviewed OTS operation/attestation subset, including exact append-16-byte-nonce then SHA-256 blinding, without Python or subprocesses.
- [x] 9.2 Implement a lossless proof tree that deterministically serializes supported nodes and safely preserves or explicitly rejects unknown nodes.
- [x] 9.3 Implement proof-operation evaluation, attestation extraction, semantic-superset comparison, deterministic branch merge, and receipt binding checks.
- [x] 9.4 Implement immutable `opentimestamps-public-v1` metadata with four pinned submission endpoints, accepted returned identities, fixed two-of-four success, two public Bitcoin APIs, exact height/request/concurrency budgets, source IDs/trust/privacy text, help/version exposure, and no runtime download or extension.
- [x] 9.5 Implement optional CLI Bitcoin Core verification with protected auth-file reading, bounded RPC requests, precedence over the public profile, and independently operated trust reporting.
- [x] 9.6 Implement dual-public-source verification against mempool.space and Blockstream with required hash/header agreement, invocation-wide `(source,height)` deduplication/singleflight, 32-height/128-request/4-concurrency limits, local header/proof-of-work/attestation checks, and mandatory third-party trust/privacy limitations.
- [ ] 9.7 Add official Python client round-trip/info/stamp/upgrade/verify differential fixtures for nonce blinding and every supported operation and attestation.
- [x] 9.8 Add malformed, oversized, excessive-depth, unsupported-operation, redirect, timeout, and parser/evaluator fuzz tests.
- [ ] 9.9 Add profile/returned-identity, pool-remapping, mocked 2-of-4/custom-calendar, dual-source agreement/disagreement/outage/rate-limit/budget, privacy, Bitcoin Core, real-calendar liveness nightly, and tracked independent-review gates before stable OTS availability.

## 10. Timestamp commands

- [x] 10.1 Implement `timestamp stamp` preflight, fresh 16-byte nonce blinding before calendar submission, concurrent built-in four-calendar/fixed two-response default plus validated explicit CLI custom mode, deterministic branch merge for one nonce, durable target/receipt writes, retry reuse, and pending ledger transition.
- [x] 10.2 Implement stamp retry and recovery for target/receipt/ledger interruption without duplicate implicit requests or deletion of unowned artifacts.
- [x] 10.3 Implement `timestamp upgrade` for matching pending evidence using built-in accepted identities by default or explicit safe CLI custom calendars, semantic-superset receipt replacement, custom trust labeling, and pending/not-ready results without premature verification.
- [x] 10.4 Implement local-only `timestamp status` states `unanchored`, `pending`, `confirmed_unverified`, `verified`, `failed`, and `inconsistent` with safe next actions; classify missing artifacts referenced by pending or verified evidence as `inconsistent`.
- [x] 10.5 Implement zero-config `timestamp verify` through the dual-public-source profile plus optional CLI Bitcoin Core override after exact target/receipt checks, deriving conservative `anchored_before` and block height.
- [x] 10.6 Implement atomic pending-to-verified ledger transition, retained proof references, byte-for-byte preservation of every imported `external_anchor`, explicit verification time, and the valid-but-too-late warning; treat imported `failed` integrity as terminal and require a new forecast revision for recovery.
- [x] 10.7 Wire all four urfave timestamp actions with exact selectors, zero-config built-in sources, CLI-only repeatable custom calendars/threshold for stamp and upgrade, `--offline`, optional Core URL/protected auth for verification, dry-run only on stamp/upgrade/verify mutations, timeout, JSON results, honest exits, help, and completion.
- [x] 10.8 Add nonce/privacy, idempotency, offline-no-socket, pending/network/crypto/budget exit, public-source agreement/disagreement/deduplication, custom-calendar safety, late-anchor, recovery, source-trust, credential-redaction, cancellation, JSON golden, and native-platform acceptance tests; unhide actions only after applicable OTS gates pass.

## 11. Layered verification

- [x] 11.1 Implement the stable layer result model with `pass`, `fail`, `pending`, `not_applicable`, and `not_checked`, deterministic aggregation, reason codes, evidence, and limitations; return overall `incomplete` with exit 9 whenever required work remains and reserve exit 0 for a complete pass.
- [x] 11.2 Implement document verification with bounded parse, embedded schema, format, domain, semantic, and artifact-path checks plus dependency blocking.
- [x] 11.3 Implement content-binding verification by rebuilding and comparing exact public or original sealed targets and every recorded metadata field.
- [x] 11.4 Implement automatic built-in-profile existence-timing verification with invocation-wide height deduplication and exact request budgets, explicit offline behavior, and optional CLI Bitcoin Core replacement, including proof validity versus pre-outcome sufficiency and incomplete/budget handling.
- [x] 11.5 Implement revealed-forecast authentication and mirror comparison without exposing decrypted fields or stored keys.
- [x] 11.6 Implement outcome metadata/digest checks and optional bounded `--check-sources` reachability without asserting authority or substantive truth.
- [x] 11.7 Include authorship, completeness, calibration, self-reported-time, outcome-truth, calendar request-timing/blinded-commitment, and Bitcoin block-height-interest limitations in every human, JSON, and MCP report.
- [x] 11.8 Wire read-only `forecast-ledger verify` with all/default and narrowed selectors, real-file enforcement, automatic built-in Bitcoin sources, `--offline`, optional Core, separate `--check-sources`, no dry-run, deterministic JSON, request summaries, and documented exit precedence.
- [x] 11.9 Add mixed-matrix, dependency, edited-ledger, late-proof, edited-reveal, source, offline-no-network, determinism, JSON golden, and native-platform tests; unhide `verify` only when complete.

## 12. Publication commands

- [x] 12.1 Define the closed versioned canonical publication manifest with exactly `ledger`, `forecast_target`, and `opentimestamps_receipt` entry roles plus deterministic path, ordering, digest, and schema-pin rules.
- [x] 12.2 Implement the allowlisted evidence graph rooted at the exact complete ledger and every referenced target/OTS receipt, with no source-control or recursive directory discovery.
- [x] 12.3 Implement secret/private/temp/lock/journal exclusion, protected-root checks, package secret-canary scanning, strict shared `scheme:opaque` key-hint validation, and actionable key-hint repair without copying or recording an actual key path.
- [x] 12.4 Implement `publish build` full source verification, collision preflight, new-root creation, safe file copying, manifest-last commit, dry-run, and interruption recovery.
- [x] 12.5 Implement `publish verify` manifest-first confinement, exact role checks, selected-path not-found exit 4 versus missing-listed-entry verification exit 6, unexpected-file rejection, then document/content/reveal/offline timestamp verification.
- [x] 12.6 Implement optional `--online` package timing checks through the shared budgeted/deduplicated built-in public observer plus optional Core replacement, including privacy/request summaries and without coupling package integrity to source availability.
- [x] 12.7 Wire both urfave publication actions with explicit `--file`, `--output` or `--manifest`, dry-run only for build, explicit offline-default versus `--online` verify behavior, JSON identity/matrix output, help, and completion.
- [x] 12.8 Add standalone source-control-free, byte-determinism, missing/tampered/extra file, traversal/link/case collision, adjacent-key, pending-proof, interruption, removable-copy, JSON golden, and native-platform tests; unhide both actions independently.

## 13. MCP runtime and parity

- [x] 13.1 Add the pinned MCP SDK and implement `mcp serve` stdio initialization, version negotiation, root/mode and built-in-profile metadata, clean stdout, stderr diagnostics, graceful EOF, and shutdown.
- [x] 13.2 Implement startup canonicalization and overlap validation for named ledger, output, and secret roots with no inferred default ledger, plus contradiction checks for reveal without a secret root or reveal combined with read-only mode.
- [x] 13.3 Implement default full online operation plus optional whole-server `--read-only` and `--offline` modes, required root-class checks before secret/network/lock/file effects, no general write/network grants, and default-off `--allow-reveal` as the sole capability boundary.
- [x] 13.4 Generate and register closed tools for ledger init/update/validate/status and all platform actions over the shared service operations.
- [x] 13.5 Generate and register closed tools for all question and forecast actions, including key-hint repair, private protected-file references, explicit confirmation fields, and conditional reveal registration only when startup enables it.
- [x] 13.6 Generate and register closed tools for target, timestamp, layered verification, and publication actions with matching dry-run/offline behavior, built-in network profile use, and no endpoint URL inputs.
- [x] 13.7 Implement versioned redacted `forecast-ledger://` resources for addressed ledgers, questions, forecasts, artifact summaries, reports, and manifests.
- [x] 13.8 Implement stable recoverable tool failures, protocol-error separation, request cancellation/timeouts, immediate deterministic conflict for a second writer on the same ledger, cross-ledger concurrency, and global resource limits.
- [x] 13.9 Wire urfave `mcp serve` with repeatable ledger/output/secret roots, optional `--read-only`/`--offline`/`--allow-reveal`, removal of general allow-write/allow-network flags, limits, help, completion, and no protocol-breaking banner.
- [ ] 13.10 Register only completed and startup-enabled tools in MCP discovery, including omission of every mutating tool in read-only mode; return unknown-tool for absent/disabled direct calls without side effects; publish static per-tool effects separately from current server mode and safe root-class diagnostics; and add in-memory and real-process tests for incremental, reveal-conditional, and read-only discovery, every available schema/tool, CLI parity, root/mode contradictions, full/read-only and online/offline modes, pinned-endpoint/budget confinement, root races/traversal, secret canaries, framing purity, cancellation, immediate writer conflicts, malformed/oversized requests, and prior supported protocol compatibility; unhide `mcp serve` only when complete.

## 14. Command surface and user documentation

- [x] 14.1 Add a command-availability manifest that proves every visible leaf has a real service action and complete acceptance gate, and fails if any planned leaf still uses the unavailable handler at final release.
- [x] 14.2 Add full English `--help` text and examples for every command using explicit `--file`, simple terminology, safe placeholders, stdin rules, built-in/custom network-profile disclosure, online/offline/Core choices, mutation-only dry-run, approvals, exits, `--plain`/`--quiet` output rules, and no secrets.
- [x] 14.3 Generate and review command reference pages covering every flag, closed input shape, output envelope, business precondition, mutation, artifact, recovery, and network behavior.
- [x] 14.4 Document the complete public/sealed/reveal and key-hint repair, target/timestamp/verify, zero-config plus advanced custom calendar, standalone publication, and MCP root/read-only/offline/reveal workflows in README and project documentation.
- [x] 14.5 Document schema v1 initial-question/forecast constraints, init versus question-add type inputs, global ID uniqueness, append-only revisions, current forecaster metadata, frozen targeted-question recovery, dispute replacement semantics, OTS nonce/trust/liveness/privacy/request-budget boundaries, online-default versus package-offline-default behavior, and all verification limitations.
- [x] 14.6 Update `AGENTS.md` with rules requiring code changes to update README, command references, examples, schemas, OpenSpec artifacts, and release notes when observable behavior changes, and reconcile its stale MCP wording with root confinement, optional read-only/offline modes, no general grants, and the sole explicit reveal gate.
- [x] 14.7 Mention Chaos Condensate and `https://chaoscondensate.com/` in appropriate README, project-context, and documentation locations without characterizing the organization, practice, or website as non-commercial.
- [x] 14.8 Add docs link, example, command/flag, generated-reference, secret-canary, simple-English lint, and clean-tree checks to CI.

## 15. Cross-platform conformance and release readiness

- [ ] 15.1 Add CLI black-box parity tests for every command in human, JSON, and plain modes, both valid `--json` placements, `--quiet`, `--no-color`, `TERM=dumb`, stdin eligibility, exact schema-version refusal, and stable exit codes.
- [x] 15.2 Add macOS, Linux, and Windows CI matrices for locking, replacement, path confinement, key permissions/ACLs, target/package determinism, cancellation, and recovery.
- [x] 15.3 Add race tests and bounded fuzz targets for parsers, selectors, canonicalization, crypto inputs, OTS proofs, manifests, MCP messages, and path normalization.
- [ ] 15.4 Add Go/Python schema validator parity using normalized verdicts, issue codes, and locations for shared fixtures.
- [ ] 15.5 Run pinned target/seal/key-hint vectors, nonce-blinded OTS differential and profile-liveness suites, budgeted layered verification matrices, publication determinism/role/exit tests, and reveal-conditional real-process MCP tests as release-blocking checks.
- [x] 15.6 Verify GoReleaser cross-builds and package smoke tests exercise the complete visible command/help surface on macOS, Linux, and Windows artifacts.
- [x] 15.7 Update changelog and release notes with newly available commands, correction of hidden preview flags and stdin behavior before those commands become stable, experimental OTS status, migration/no-migration statement, recovery behavior, and security/trust limitations.
- [ ] 15.8 Run `go test`, race, fuzz smoke, lint, generated-file, OpenSpec strict validation, docs, package, native smoke, and section 16 dogfooding-regression gates; record evidence that all 30 required actions and every advertised/startup-enabled MCP parity tool are implemented before marking the change complete.

## 16. Dogfooding corrections

- [ ] 16.1 Pass the confined ledger-relative artifact filesystem through current and prospective semantic validation for every authoring mutation, then add lifecycle regressions for `init → add → target/stamp → add/update → close → resolve`, retained valid evidence, and missing/tampered evidence classification.
- [ ] 16.2 Normalize YAML `!!timestamp` scalars only for timestamp-typed input fields, retain strict type rejection elsewhere, quote timestamps in every maintained YAML example, and execute those examples as CI fixtures.
- [ ] 16.3 Preserve ordered structured parse/schema/semantic issues through service and CLI error wrapping, including safe codes, JSON pointers, known line/column positions, unknown-field identity, human rendering, JSON details, and private-value redaction tests.
- [ ] 16.4 Render inserted and replaced source-tree fragments in the surrounding JSON/YAML indentation, newline, and expanded/compact style while retaining untouched bytes; add realistic 30-plus-forecast readability, maximum-line, and minimal-diff regressions proving canonical JSON is limited to evidence/crypto artifacts.
- [ ] 16.5 Preserve typed pending/incomplete application errors after report presentation so layered `verify` and `publish verify` exit `9` rather than `1`; cover human, plain, JSON, quiet/error-writer, and mixed evidence outcomes.
- [ ] 16.6 Convert domain results to documented public wire DTOs before redaction and serialization, then add compatibility goldens proving forecast-value and integrity tagged unions contain no PascalCase Go branches, inactive null siblings, or reflection metadata.
- [ ] 16.7 Give protected private input and key-file checks distinct argument roles and safe diagnostics; add POSIX and Windows cases proving an unsafe `--input` is never mislabeled as `--key-file` and neither path leaks protected absolute locations.
- [ ] 16.8 Expand human `question show`, `forecast show`, and concise list rows to include the required business fields and type-aware values, and render the complete ordered verification evidence matrix in normal human/plain output without requiring verbose mode; add redaction and golden tests.
- [ ] 16.9 Implement static MCP tool-effect metadata independently of server mode, omit mutating tools from read-only registration, return unknown-tool for their direct calls, and preserve safe `ledger`/`output`/`secret` root-class identifiers in recoverable failures without absolute-path disclosure.
- [ ] 16.10 Update README, help, command references, tutorials, and generated examples to explain stdin's inability to resolve sibling artifacts, reference-driven publication versus adjacent standalone targets, read-only MCP discovery, human detail/report output, and the quoted-YAML timestamp convention.
- [ ] 16.11 Add a reproducible black-box dogfooding suite based on a multi-question/multi-forecast ledger that exercises all authoring, seal/reveal, target/timestamp, layered verification, publication, and MCP mode paths while asserting stable exits, readable files, no secret leakage, and unchanged retained evidence.
- [ ] 16.12 Run the dogfooding suite on packaged macOS, Linux, and Windows binaries together with docs examples and public JSON compatibility fixtures, record the release evidence, and keep the final gate open until every legitimate dogfooding regression above passes.
