## Context

See `proposal.md` for motivation and the seven delta specs for observable behavior. The repository already has a pinned v1 schema and seal fixture, bounded JSON/YAML document trees, typed models, semantic validation, canonicalization foundations, recoverable storage primitives, stable presentation/errors, a hidden urfave preview tree, and working `validate`/`status`. Most product actions still share one unavailable handler; MCP contains only a package scaffold; OpenTimestamps and publication contain no production workflow. The older active `build-forecast-ledger-cli-mcp` change contains useful completed foundation tasks but conflicting command specs and is superseded in full by this change.

The exact schema commit and digests remain authoritative. It currently requires at least one question per ledger and at least one forecast per question, which means a schema-valid `init` cannot create an empty shell and `question add` cannot create an empty question. The CLI must remain useful without source control, hosted services, Python at runtime, or a default ledger. macOS, Linux, and Windows must share the same domain behavior despite different filesystem and credential semantics.

## Goals / Non-Goals

**Goals:**

- Give every CLI/MCP action one shared typed application operation with explicit inputs, results, side-effect description, and stable errors.
- Cover mutable root metadata through `ledger update` while keeping IDs, creation identity, questions, and legacy publication metadata outside that patch.
- Keep every mutation valid before and after commit while preserving document presentation and append-only forecast history.
- Make multi-file, secret, network, verification, and package workflows recoverable and testable at each failure boundary.
- Preserve exact v1 target/seal interoperability and constrain the pure-Go OpenTimestamps subset behind differential and independent-review gates.
- Unhide actions only when their real implementation, documentation, and acceptance gates pass.

**Non-Goals:**

- Changing the Forecast Ledger schema, authoring its legacy publication object, adding draft/invalid ledgers, or supporting historical v0.5/RFC 3161 protocols.
- Adding source-control operations, hosted publishing, platform imports, scoring, signatures/authorship claims, HTTP MCP, or automatic credential discovery.
- Treating a calendar, block explorer, URL response, file timestamp, or self-reported time as stronger evidence than it provides.
- Replacing the existing schema/document/storage foundations without an independently demonstrated defect.

## Decisions

### 1. Model every action as one transport-neutral operation

Each business action gets a typed request/result contract and one service entry point under `internal/service`, grouped by ledger, platform, question, forecast, target, timestamp, verification, and publication. Requests carry explicit ledger/input paths or bounded already-parsed input, selectors, observation time, dry-run/confirmation, offline/read-only mode, built-in network-profile selection, and root context. Results carry a stable operation code, typed public data, warnings, changed safe paths, contacted safe source IDs, trust limitations, and side-effect/recovery state.

CLI actions translate urfave values into requests and present results. MCP handlers translate closed tool schemas into the same requests. Neither adapter performs domain selection, validation, patching, crypto, receipt logic, or package enumeration. `internal/app` remains the lower-level error/contract vocabulary; `internal/service` owns orchestration, avoiding the current storage→app import cycle.

Alternative considered: implement commands directly in `internal/adapters/cli` and later copy them into MCP. Rejected because it would make parity, locking, cancellation, errors, and secret review diverge.

### 2. Define versioned closed input schemas next to operations

Complex non-secret and private inputs use bounded JSON/YAML documents decoded into dedicated request types. Each action owns a closed schema with explicit null/removal semantics. The CLI accepts a path or stdin; MCP accepts the equivalent typed object or an authorized private-file reference where raw secret values are prohibited. Scalar flags remain only for identifiers and common unambiguous root fields; the already registered required `question add --type` remains the explicit discriminator and is forbidden inside that command's input document.

JSON/YAML input is parsed by the existing bounded document layer so duplicate keys and resource exhaustion fail consistently. Input schemas and JSON result schemas are exported as generated reference artifacts and used by CLI/MCP parity tests.

Alternative considered: dozens of scalar flags. Rejected because nested typed forecasts, sources, units, options, accounts, and patches become ambiguous, difficult to quote cross-platform, and unsafe for private bundles.

### 3. Initialize and add questions with their first forecast atomically

The pinned schema's non-empty arrays are treated as a hard contract. Init parses root identity plus exactly one initial question/forecast pair and exclusively creates one fully valid file. Question add parses one question plus exactly one public or sealed forecast and appends them in one transaction. A sealed first forecast requires an explicit new key destination and treats the whole structured input as private; it reuses the normal protected-key-first transaction. There is no invalid “draft ledger” state and no placeholder forecast. Additional records use the normal add/seal actions after the initial valid commit.

Shared builders construct the same typed question/forecast records used by later add/seal operations, including explicit clock injection for deterministic tests. This avoids special init-only semantics.

Alternative considered: create empty arrays and allow later commands to repair the ledger. Rejected because every subsequent command starts by validating the current ledger and the created file would violate the authoritative schema.

### 4. Use selector indexes and immutable-field policies before patching

After full validation, an operation builds indexes for platform IDs, question IDs, globally unique forecast IDs, per-question forecast order, references, and supersession links. Selection returns stable not-found/conflict errors before patching. Update policies use explicit allowlists: root display/forecaster authoring metadata, platform value fields, and question descriptive/unresolved lifecycle fields are constrained; IDs, creation identity, question type-defining fields, forecasts, and resolution fields remain immutable outside dedicated actions.

Forecast mutations are append-only. No operation receives a mutable forecast pointer for replacement; add/seal constructs a new record and appends it after all uniqueness, chronology, type, and supersession checks. Imported failed integrity is terminal and recovery appends a new forecast revision. Resolution is the schema-mandated current object rather than append-only history: explicit resolve/annul/dispute transitions may replace it with confirmation, report prior status, and never imply that old resolution bytes remain inside v1.

Alternative considered: generic JSON Patch. Rejected because callers could bypass lifecycle/immutability rules and because externally supplied pointers are too permissive for a security-sensitive ledger.

### 5. Reuse one mutation transaction with prospective validation

Single-ledger mutations use the existing lock and recovery transaction: resolve safe path, lock, parse/validate, build indexes, apply minimal source-tree patch to a copy, re-decode/full-validate, write a same-directory temporary sibling, flush, safe-replace, and clean/recover the journal. Operations return the before/after document digest and changed JSON pointers for audit-friendly output.

Dry-run executes through prospective validation but substitutes a recorder for file/secret/network effects. Its result explicitly distinguishes checks performed now from entropy, remote, race, and commit checks deferred to execution.

Alternative considered: mutate typed structs and reserialize. Rejected because it destroys YAML presentation and broadens the changed surface.

### 6. Add a multi-resource transaction coordinator for artifacts and secrets

Target, timestamp, seal, and publication operations first construct a side-effect plan with canonical resolved targets, preconditions, ownership, and rollback class. Writes use exclusive create or recoverable replacement, flush content and parent directories where supported, and journal only CLI-created paths. The ledger is committed only after required external artifacts are durable.

Secrets use a stricter coordinator: protected key creation is exclusive and committed before the ledger, but a durable key is never auto-deleted after a later ledger failure because it may be the only recovery copy. Public target/receipt/package staging may be cleaned when the journal proves the CLI created it. Every failure returns a typed recovery state.

Alternative considered: pretend several filesystem writes are atomic. Rejected because no cross-platform filesystem transaction spans key, target, receipt, package, and ledger files.

### 7. Fix deterministic artifact profiles

Targets use `proofs/targets/<forecast-id>.json`; receipts use `proofs/receipts/<forecast-id>.json.ots`. Forecast IDs are globally unique, so these paths are deterministic without question names. Resolvers join relative paths against the ledger directory, reject case-folding collisions and escape forms, and never follow an in-root link for creation.

Target build computes all bytes and collisions before `--all` writes. Same bytes are idempotent; different bytes conflict. Target build does not mutate integrity. Stamp records target metadata only when a valid receipt is durable. A question update precomputes old/new targets and refuses target-covered changes when a matching artifact or integrity reference already exists.

Package layout and a versioned canonical manifest are fixed by the publication spec. Manifest entries use forward-slash paths and sorted stable records without time/machine/source-control fields.

Alternative considered: user-selected arbitrary artifact paths. Rejected for the primary workflow because it weakens reproducibility and makes cross-ledger/package verification harder; future import/migration can be separately specified.

### 8. Isolate the exact v1 crypto profile

`internal/canonical` remains a bounded JCS implementation. A typed projection module constructs the exact spec-listed public, sealed, and revealed-as-originally-sealed envelopes; generic maps are not accepted at the crypto boundary. A checked-in fixture pins public/sealed/revealed target bytes and digests across Go and the reference implementation.

Seal/reveal uses only the pinned protocol constants and `crypto/rand`/ChaCha20-Poly1305/SHA-256. Typed closed structs generate the exact fixture plaintext `{schema, question_id, forecast_id, salt, bundle}` and AAD `{scheme, question_id, forecast_id, commitment_sha256}`. No ledger ID is passed to seal/reveal because the authoritative fixture and implemented AAD do not bind it; the separately built target binds ledger ID. Test-only deterministic entropy and fixture key hint are injected behind unexported test interfaces and cannot be configured by production CLI/MCP.

Secret files use exact JCS `forecast-key/v1` bytes plus LF and OS-specific secure creation: mode/ownership checks on POSIX and owner-only ACL code on Windows. Absolute secret paths are retained only in private operation state and never written to `key_hint`; production uses the non-authoritative logical hint `forecast-key:<forecast-id>`. Reveal checks key-file IDs before AEAD and derives all schema-required mirror fields. Moving a key needs only a new explicit `--key-file` argument, not a ledger mutation.

Alternative considered: general crypto algorithms or configurable canonicalization. Rejected because interoperability and review depend on one exact published profile.

### 9. Implement a constrained OpenTimestamps backend and embedded network profile

`internal/timestamp/ots` owns bounded detached-proof parsing, lossless representation, supported operations/attestations, deterministic merge/serialization, calendar client, and verification requests. `internal/timestamp` (or service-level timestamp orchestration) owns target/receipt/ledger state transitions. An immutable `opentimestamps-public-v1` profile embedded in the binary owns the four submission URLs, their accepted receipt calendar identities, two-of-four policy, the mempool.space and Blockstream Bitcoin APIs, source IDs, trust text, and bounded client policy. Its ID and sources are visible in version/help/results, but it is never downloaded or mutated at runtime; endpoint changes require a reviewed release and a new profile ID.

Default Bitcoin verification queries both public APIs and requires identical height/hash/header observations before locally checking the header, encoded target, OTS operations, and attestation. This reduces single-service error but still trusts third parties for canonical-chain selection, which every result states. Optional CLI-only Bitcoin Core verification implements the same evidence-source interface with protected credential-file reading and a stronger independently operated boundary. MCP never accepts endpoint URLs and uses only the embedded profile or offline mode. Unknown proof nodes are preserved if lossless/safe or cause an explicit unsupported result. The feature remains experimental until official-client differential, profile, dual-source, real-calendar, malformed-input, native recovery, and independent-review gates pass.

Alternatives considered: require every user to configure calendars/explorers, or shell out to Python `ots`. Explicit per-user endpoints were rejected because they burden the normal workflow, create CLI/MCP drift, and allow unsafe URL input; the binary still offers `--offline` and optional CLI Bitcoin Core. A subprocess was rejected because it violates single-binary distribution and creates version, Windows, cancellation, and error-parity problems.

### 10. Represent verification as composable layer results

Each verifier returns `{state, reason_codes, evidence, limitations}` for one layer. A deterministic aggregator applies dependency and exit precedence without erasing partial evidence. Ledger verification supplies the embedded Bitcoin observer automatically unless offline or replaced by CLI Bitcoin Core; package verification remains offline unless its online mode is requested. Outcome-URL retrieval stays explicitly requested and separate from the pinned protocol profile. Report builders consume the same model for human, JSON, MCP, and package verification.

Document failure blocks dependent layers. Content binding precedes OTS proof checks. Reveal authenticates before comparing the mirror. Outcome source checks distinguish metadata/digest/reachability from truth. Limitations are data in every result, not presentation-only prose that adapters can omit.

Alternative considered: one `verified` boolean. Rejected because it conflates distinct claims and cannot express pending/not-checked evidence honestly.

### 11. Build packages from an allowlisted evidence graph

Publication derives a graph rooted at the exact ledger: each ledger reference resolves to a typed public artifact role. Only graph nodes permitted by the package profile are copied. Key/secret/private/temp/lock/journal roles are impossible to add through the normal API. Before creating output, all source bytes are read through bounded/confined handles, verified, hashed, and mapped to collision-free destinations.

Verification parses and validates the manifest before trusting paths or roles, then hashes all entries and checks for unexpected regular files before ledger/evidence verification. Package operations never inspect source-control metadata or upload data.

Alternative considered: recursively copy the ledger directory. Rejected because it can include keys, drafts, unrelated ledgers, credentials, and nondeterministic files.

### 12. Run MCP as a thin root-confined adapter

The official pinned MCP SDK owns framing and negotiation. A server builder registers closed tool/resource schemas generated from the same operation types only when each tool's availability gate passes. Startup canonicalizes repeatable named ledger/output/secret roots and selects full versus optional read-only mode and online versus optional offline mode. Tool middleware performs schema validation, root-class checks, mode checks, root resolution, limits, context timeout, and error conversion before calling services. The default server exposes applicable write and built-in-network operations without separate grants; missing output/secret roots fail only operations that need those roots. A held ledger writer lock produces the same immediate conflict in CLI and MCP; the server does not queue writers.

Protocol stdout is passed directly to the SDK; all application diagnostics use an injected stderr logger. Resource URIs contain a root ID plus encoded relative path, never an absolute path. Tool schemas contain no calendar, explorer, proxy, or Bitcoin endpoint URL. Real-process tests treat any non-protocol stdout byte as a failure.

Alternative considered: MCP handlers invoke CLI subprocesses. Rejected because it duplicates serialization, prevents structured cancellation/errors, and leaks process-boundary details.

### 13. Use per-command availability gates

The unavailable handler and hidden flag are removed one action at a time only when operation code, unit/integration tests, CLI JSON/help goldens, dry-run/rollback/cancellation checks, MCP parity where the tool exists, documentation, and native-platform gates applicable to that action pass. Groups become visible when they contain at least one implemented child; help labels experimental OTS behavior honestly.

This allows reviewable increments without again advertising scaffolding as a finished application. Hidden preview flag definitions are corrected before each command is exposed: artifact-dependent reads lose stdin, timestamp verify gains mutation/dry-run semantics, question type remains scalar, and root flags become repeatable. The final release gate asserts that no planned leaf uses the unavailable handler and every documented/MCP action maps to a service; incomplete MCP tools are absent from discovery rather than returning application `unavailable`.

Alternative considered: unhide the complete tree after the first implementation slice. Rejected because it recreates the readiness mismatch that triggered this change.

### 14. Treat generated reference and conformance as build outputs

Operation definitions drive CLI/MCP input schemas, JSON result schemas, command reference tables, and parity fixtures. CI runs clean-tree regeneration checks. Tests are layered: pure builders/indexes; document patches; transaction fault injection; crypto/target vectors; OTS differential/fuzz; verification matrices; package determinism; CLI goldens; MCP in-memory/real-process; and native filesystem/ACL/replacement tests.

The pinned Python validator parity job compares case verdicts and normalized issue codes/locations against Go rather than merely running the Python harness alone.

Alternative considered: hand-maintain separate reference pages and schema copies. Rejected because drift would be likely across 29 actions and MCP equivalents.

## Risks / Trade-offs

- **[The scope is large and cross-cutting]** → Deliver in dependency-ordered vertical slices and keep unavailable gates per action; do not merge placeholder completion.
- **[Schema non-empty arrays make init less minimal]** → Require an initial question/forecast explicitly and reuse normal builders so the first file is valid.
- **[The schema contains legacy publication and external/failed states outside this authoring workflow]** → Preserve valid imported values, never infer source-control behavior, keep external anchors distinct, treat failed forecast integrity as terminal, and recover through an append-only revision.
- **[Resolution dispute replaces the one v1 resolution object]** → Require confirmation, report prior status, preserve forecasts/evidence, and document that v1 does not provide internal resolution history.
- **[Protected Windows key ACLs are easy to get wrong]** → Isolate native code, test on Windows runners, fail closed when owner-only protection cannot be proven, and keep seal unavailable until native tests pass.
- **[Multi-file operations cannot be truly atomic]** → Journal explicit ownership/state, order durable writes conservatively, make retries idempotent, and return recovery state rather than claiming rollback that did not occur.
- **[Pure-Go OTS may diverge from the official client]** → Constrain the subset, preserve/reject unknown nodes explicitly, differential-test every supported operation, fuzz, run real-calendar tests, and require independent review.
- **[Built-in public Bitcoin APIs may fail, disagree, or change]** → Pin both in a versioned release profile, require agreement, fail without a state transition on disagreement/outage, expose source IDs, and state that third-party canonical-chain selection is weaker than optional Bitcoin Core.
- **[Package enumeration can leak secrets]** → Build from a typed allowlisted graph, reject extra roles/paths, scan canaries, and test with adjacent key/private files.
- **[MCP expands side-effect reach]** → Confine every file class to explicit roots, provide optional whole-server read-only/offline modes, reject arbitrary protocol endpoint URLs, retain explicit confirmation fields, enforce limits, and share application transactions.
- **[Two active OpenSpec changes overlap]** → Treat this change as the sole replacement contract, map already completed foundation evidence into its tasks, and retire the older change without syncing or archiving its delta specs.
- **[Stable result schemas can freeze mistakes]** → Version persisted/transport schemas, keep human prose flexible, and require compatibility review for field removal or semantic change.

## Migration Plan

1. Freeze `build-forecast-ledger-cli-mcp` as superseded, map its 31 completed foundation tasks to equivalent tasks/evidence here, and ensure its delta specs will never be independently synced or archived.
2. Add shared operation request/result contracts, typed input schemas, selector indexes, clocks/effect recorders, exact crypto artifact profiles, and generated references; correct hidden preview flags/help without changing available commands.
3. Implement schema-valid init, root metadata update, and platform actions; unhide each action only after its gates pass.
4. Implement question lifecycle and public forecast append-only actions, including initial-forecast atomicity and disputed replacement transitions.
5. Implement target build/check, protected key files, seal/reveal, and exact vector/cross-platform gates.
6. Implement the constrained OTS backend and timestamp commands; keep experimental labeling through conformance and review.
7. Implement layered verification and deterministic package build/verify.
8. Implement MCP roots, read-only/offline modes, incrementally discovered tools, and resources over the same operations and pass real-process parity/framing tests.
9. Run full native, race, fuzz, validator/crypto/OTS conformance, documentation-generation, package, and release gates; assert no product leaf remains unavailable.

Existing v1 ledgers require no data migration. New actions refuse unsupported schema versions rather than rewriting them. Rollback of an implementation slice re-hides its command/tool and restores the previous known-good binary; it does not downgrade or rewrite ledgers. Any newly persisted artifact/profile schema receives an explicit version and compatibility test before release.
