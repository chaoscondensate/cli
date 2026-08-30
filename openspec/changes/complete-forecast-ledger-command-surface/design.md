> Supersession note (2026-08-30): explicit-TSA-only statements below describe
> the completed v0.4.0 command surface. `add-default-tsa-failover` controls
> the next release's provider and transport model.

## Context

See `proposal.md` for motivation and the seven delta specs for observable behavior. The repository already has a pinned v1 schema and seal fixture, bounded JSON/YAML document trees, typed models, semantic validation, canonicalization foundations, recoverable storage primitives, stable presentation/errors, a hidden urfave preview tree, and working `validate`/`status`. Most product actions still share one unavailable handler; MCP contains only a package scaffold; RFC 3161 and publication contain no production workflow. The older active `build-forecast-ledger-cli-mcp` change contains useful completed foundation tasks but conflicting command specs and is superseded in full by this change.

The exact schema commit and digests remain authoritative. Forecast Ledger v1.2.0 permits zero questions in a ledger and zero forecasts in a question, so `init` may create an empty ledger and `question add` may create a backlog question. The `replace-ots-with-rfc3161` change supersedes this design's earlier v1.0.0 minimum-item assumptions. The CLI must remain useful without source control, hosted services, Python at runtime, or a default ledger. macOS, Linux, and Windows must share the same domain behavior despite different filesystem and credential semantics.

## Goals / Non-Goals

**Goals:**

- Give every CLI/MCP action one shared typed application operation with explicit inputs, results, side-effect description, and stable errors.
- Cover mutable root metadata through `ledger update` while keeping IDs, creation time, questions, and legacy publication metadata outside that patch.
- Keep every mutation valid before and after commit while preserving document presentation and append-only forecast history.
- Make multi-file, secret, network, verification, and package workflows recoverable and testable at each failure boundary.
- Preserve exact v1 target/seal interoperability and constrain the pure-Go RFC 3161 subset behind differential and independent-review gates.
- Unhide actions only when their real implementation, documentation, and acceptance gates pass.

**Non-Goals:**

- Changing the Forecast Ledger schema, authoring its legacy publication object, adding draft/invalid ledgers, or supporting historical v0.5/RFC 3161 protocols.
- Adding source-control operations, hosted publishing, platform imports, scoring, signatures/authorship claims, HTTP MCP, or automatic credential discovery.
- Treating a TSA, block remote source, URL response, file timestamp, or self-reported time as stronger evidence than it provides.
- Replacing the existing schema/document/storage foundations without an independently demonstrated defect.

## Decisions

### 1. Model every action as one transport-neutral operation

Each business action gets a typed request/result contract and one service entry point under `internal/service`, grouped by ledger, platform, question, forecast, target, timestamp, verification, and publication. Requests carry explicit ledger/input paths or bounded already-parsed input, selectors, observation time, dry-run/confirmation, offline/read-only mode, explicit TSA and CA-bundle inputs where applicable, and root context. Results carry a stable operation code, typed public data, warnings, changed safe paths, contacted safe source IDs, trust limitations, and side-effect/recovery state.

CLI actions translate urfave values into requests and present results. MCP handlers translate closed tool schemas into the same requests. Neither adapter performs domain selection, validation, patching, crypto, receipt logic, or package enumeration. `internal/app` remains the lower-level error/contract vocabulary; `internal/service` owns orchestration, avoiding the current storage→app import cycle.

Operation results are converted to explicit public wire DTOs, including schema-compatible tagged-union encodings, before generic redaction or transport serialization. Redaction walks that public representation rather than reflecting over domain structs, so inactive union branches and internal Go field names cannot leak into JSON. Error adapters preserve structured issue collections and safe source locations through every wrapper; presentation chooses human or JSON formatting without replacing the underlying application category.

Alternative considered: implement commands directly in `internal/adapters/cli` and later copy them into MCP. Rejected because it would make parity, locking, cancellation, errors, and secret review diverge.

### 2. Define versioned closed input schemas next to operations

Complex non-secret and private inputs use bounded JSON/YAML documents decoded into dedicated request types. Each action owns a closed schema with explicit null/removal semantics. The CLI accepts a path or stdin; MCP accepts the equivalent typed object or an authorized private-file reference where raw secret values are prohibited. Scalar flags remain only for identifiers and common unambiguous root fields; the already registered required `question add --type` remains the explicit discriminator and is forbidden inside that command's input document. Init instead owns a complete nested initial-question document whose schema includes `type`; both adapters normalize type into the same builder request before validation, so the input schemas intentionally differ without forking domain behavior.

JSON/YAML input is parsed by the existing bounded document layer so duplicate keys and resource exhaustion fail consistently. Input schemas and JSON result schemas are exported as generated reference artifacts and used by CLI/MCP parity tests.

The YAML adapter performs one schema-directed normalization before typed decoding: a YAML `!!timestamp` scalar is converted to its RFC 3339 lexical string only when the operation schema declares that exact field as a timestamp. Other YAML tags retain normal closed-schema type checking. Maintained examples still quote timestamps to avoid parser-dependent behavior outside this binary.

Alternative considered: dozens of scalar flags. Rejected because nested typed forecasts, sources, units, options, accounts, and patches become ambiguous, difficult to quote cross-platform, and unsafe for private bundles.

### 3. Support all valid initial creation shapes

Init parses root identity and optional input, then exclusively creates one valid file as a ledger only, ledger plus question, or ledger plus question and public/sealed forecast. Question add likewise appends a question with or without one initial forecast. A sealed initial forecast requires an explicit new key destination and treats the whole structured input as private; it reuses the normal protected-key-first transaction. Empty arrays are schema-valid workflow states, not placeholders or partially valid documents.

Shared builders construct the same typed question and optional forecast records used by later add/seal operations, including explicit clock injection for deterministic tests. This avoids special init-only semantics. When a forecast is supplied, question and forecast creation and any protected-key effect remain atomic.

The earlier v1.0.0 design rejected empty arrays. Forecast Ledger v1.2.0 deliberately removes those minimum-item constraints, so the newer `replace-ots-with-rfc3161` contract supersedes that decision.

### 4. Use selector indexes and immutable-field policies before patching

After full validation, an operation builds indexes for platform IDs, question IDs, globally unique forecast IDs, per-question forecast order, references, and supersession links. Selection returns stable not-found/conflict errors before patching. Update policies use explicit allowlists: root display/current-forecaster metadata, platform value fields, and question descriptive/unresolved lifecycle fields are constrained; IDs, creation time, question type-defining fields, forecasts, and resolution fields remain immutable outside dedicated actions. Forecaster kind changes atomically with member shape because v1 already treats name and membership as mutable current metadata and carries neither historical identity snapshots nor authorship proof.

Forecast prediction mutations are append-only. No operation receives a mutable forecast pointer for replacement; add/seal constructs a new record and appends it after all uniqueness, chronology, type, and supersession checks. The one narrow metadata exception is `forecast key-hint update`: it patches only the non-authoritative location-independent hint that both seal and target exclude, allowing imported path-like hints to be normalized without invalidating evidence. Imported failed integrity is terminal except for that non-evidentiary hint repair, and prediction recovery appends a new forecast revision. Resolution is the schema-mandated current object rather than append-only history: explicit resolve/annul/dispute transitions may replace it with confirmation, report prior status, and never imply that old resolution bytes remain inside v1.

Once a target exists, target-covered question wording/timing is frozen because replacement would make retained evidence ambiguous. The supported product workflow is to annul the old question and create a new ID, optionally recording the predecessor in notes; there is no override or target rewrite.

Alternative considered: generic JSON Patch. Rejected because callers could bypass lifecycle/immutability rules and because externally supplied pointers are too permissive for a security-sensitive ledger.

### 5. Reuse one mutation transaction with prospective validation

Single-ledger mutations use the existing lock and recovery transaction: resolve safe path, lock, parse/validate, build indexes, apply minimal source-tree patch to a copy, re-decode/full-validate, write a same-directory temporary sibling, flush, safe-replace, and clean/recover the journal. Operations return the before/after document digest and changed JSON pointers for audit-friendly output.

Both current and prospective validation receive the same confined artifact filesystem rooted at the selected ledger directory. This allows semantic checks to resolve already referenced targets and receipts after stamping instead of accidentally treating their presence as invalid authoring state. Missing, unsafe, or tampered retained evidence still fails according to its real validation category.

Structural patch rendering is presentation-aware. It derives indentation, newline, collection layout, and JSON/YAML fragment style from the insertion/replacement context, while retaining untouched source slices verbatim. Canonical/JCS serialization is used only where an evidence, manifest, key, or cryptographic profile requires exact bytes; it is not reused to render ledger fragments.

Dry-run is attached to persistent mutation or resource creation, executes through prospective validation, and substitutes a recorder for file/secret/network effects used by that operation. Its result explicitly distinguishes checks performed now from entropy, remote, race, and commit checks deferred to execution. Pure verification does not gain a dry-run mode; RFC 3161 verification is local, while general `--offline` controls only optional outcome-source retrieval.

Alternative considered: mutate typed structs and reserialize. Rejected because it destroys YAML presentation and broadens the changed surface.

### 6. Add a multi-resource transaction coordinator for artifacts and secrets

Target, timestamp, seal, and publication operations first construct a side-effect plan with canonical resolved targets, preconditions, ownership, and rollback class. Writes use exclusive create or recoverable replacement, flush content and parent directories where supported, and journal only CLI-created paths. The ledger is committed only after required external artifacts are durable.

Secrets use a stricter coordinator: protected key creation is exclusive and committed before the ledger, but a durable key is never auto-deleted after a later ledger failure because it may be the only recovery copy. Public target/receipt/package staging may be cleaned when the journal proves the CLI created it. Every failure returns a typed recovery state.

Alternative considered: pretend several filesystem writes are atomic. Rejected because no cross-platform filesystem transaction spans key, target, receipt, package, and ledger files.

### 7. Fix deterministic artifact profiles

Targets use `proofs/targets/<forecast-id>.json`; receipts use `proofs/timestamps/<forecast-id>.tsr`. Forecast IDs are globally unique, so these paths are deterministic without question names. Resolvers join relative paths against the ledger directory, reject case-folding collisions and escape forms, and never follow an in-root link for creation.

Target build computes all bytes and collisions before `--all` writes. Same bytes are idempotent; different bytes conflict. Target build does not mutate integrity. Stamp records target metadata only when a valid receipt is durable. A question update precomputes old/new targets and refuses target-covered changes when a matching artifact or integrity reference already exists.

Package layout and a versioned canonical manifest are fixed by the publication spec. Manifest entries use only the closed `ledger`, `forecast_target`, and `rfc3161_receipt` roles, forward-slash paths, and sorted stable records without time/machine/source-control fields.

Alternative considered: user-selected arbitrary artifact paths. Rejected for the primary workflow because it weakens reproducibility and makes cross-ledger/package verification harder; future import/migration can be separately specified.

### 8. Isolate the exact v1 crypto profile

`internal/canonical` remains a bounded JCS implementation. A typed projection module constructs the exact spec-listed public, sealed, and revealed-as-originally-sealed envelopes; generic maps are not accepted at the crypto boundary. A checked-in fixture pins public/sealed/revealed target bytes and digests across Go and the reference implementation.

Seal/reveal uses only the pinned protocol constants and `crypto/rand`/ChaCha20-Poly1305/SHA-256. Typed closed structs generate the exact fixture plaintext `{schema, question_id, forecast_id, salt, bundle}` and AAD `{scheme, question_id, forecast_id, commitment_sha256}`. No ledger ID is passed to seal/reveal because the authoritative fixture and implemented AAD do not bind it; the separately built target binds ledger ID. Test-only deterministic entropy and fixture key hint are injected behind unexported test interfaces and cannot be configured by production CLI/MCP.

Secret files use exact JCS `forecast-key/v1` bytes plus LF and OS-specific secure creation: mode/ownership checks on POSIX and owner-only ACL code on Windows. Absolute secret paths are retained only in private operation state and never written to `key_hint`; production uses the non-authoritative logical hint `forecast-key:<forecast-id>`. A dedicated hint-update operation accepts only the closed path-free `scheme:opaque` profile and changes no crypto/evidence bytes. Reveal checks key-file IDs before AEAD and derives all schema-required mirror fields. Moving a key needs only a new explicit `--key-file` argument, not a ledger mutation; hint update exists to repair imported public metadata, not to discover or move secrets.

Alternative considered: general crypto algorithms or configurable canonicalization. Rejected because interoperability and review depend on one exact published profile.

### 9. Implement a constrained RFC 3161 backend with explicit trust inputs

`internal/timestamp/rfc3161` owns bounded request/response parsing, SHA-256 request construction, CSPRNG nonces, CMS signature and signed-attribute checks, X.509 timestamping-EKU and chain verification, supported-algorithm policy, TSA HTTP submission, and normalized errors. Service-level timestamp orchestration owns target, request, response, CA-bundle, and ledger state transitions. Dependency types do not cross the internal boundary.

Stamp requires one explicit public HTTPS TSA URL and one retained PEM CA bundle inside the permitted ledger root. It sends `application/timestamp-query`, accepts only a bounded timestamp-reply, follows no cross-origin redirect, and rejects credentials, private/link-local/reserved destinations, trailing data, weak algorithms, and unsupported critical structures. The request contains SHA-256 over the exact target, a fresh nonce, and `certReq=true`; it does not add a second blinding construction beyond the RFC request nonce.

Stamp performs network work outside the ledger lock, then re-reads and revalidates the selected immutable forecast under the lock before committing the target, `.tsq`, `.tsr`, retained CA path, and source-preserving ledger patch. Status parses local evidence only. Verify rechecks the complete local chain against the retained CA bundle at `gen_time`; layered and package verification use the same local verifier and require no TSA or other network service. The feature remains experimental until OpenSSL differential fixtures, ASN.1/CMS malformed-input and fuzz coverage, real-TSA interoperability, native recovery, and independent-review gates pass.

Alternatives considered: shell out to OpenSSL, use system roots, embed a default TSA/trust profile, or expose a general authenticated HTTP client. These were rejected to preserve single-binary distribution, reproducible retained trust, explicit provider choice, and a narrow network surface. OpenSSL remains a development-time conformance oracle only.

### 10. Represent verification as composable layer results

Each verifier returns `{state, reason_codes, evidence, limitations}` for one layer. A deterministic aggregator applies dependency and exit precedence without erasing partial evidence. Ledger and package verification reconstruct the exact target and verify retained request, response, and CA-bundle bytes locally. Missing or unreadable retained evidence yields explicit not-checked or failure layers rather than an invented pass. Outcome-URL retrieval stays explicitly requested and separate from timestamp verification. Report builders consume the same model for human, JSON, MCP, and package verification.

Human, plain, and JSON presenters all iterate the same ordered layer collection; the human presenter prints the matrix in normal mode rather than reducing it to an overall sentence. Presentation completion is separate from process exit selection: after writing the available report, the adapter returns the original typed pending/incomplete/failure category through an unwrap-compatible path so the central exit mapper cannot turn it into `internal`.

Document failure blocks dependent layers. Content binding precedes RFC 3161 proof checks. Reveal authenticates before comparing the mirror. Outcome source checks distinguish metadata/digest/reachability from truth. Limitations are data in every result, not presentation-only prose that adapters can omit.

Alternative considered: one `verified` boolean. Rejected because it conflates distinct claims and cannot express pending/not-checked evidence honestly.

### 11. Build packages from an allowlisted evidence graph

Publication derives a graph rooted at the exact ledger: each ledger reference resolves to one of three closed public artifact roles. Only graph nodes permitted by the package profile are copied. Key/secret/private/temp/lock/journal roles are impossible to add through the normal API. Before creating output, all source bytes are read through bounded/confined handles, verified, hashed, and mapped to collision-free destinations. Unsafe imported key hints fail with a repair command rather than making the ledger permanently unpackageable.

The graph is reference-driven, not directory-driven. Deterministic targets created independently but not referenced by the selected ledger are deliberately excluded; the builder never scans `proofs` to infer membership. Documentation explains this so users do not mistake every locally built target for published evidence.

Verification parses and validates the manifest before trusting paths or roles, then hashes all entries and checks for unexpected regular files before ledger/evidence verification. Package operations never inspect source-control metadata or upload data.

Alternative considered: recursively copy the ledger directory. Rejected because it can include keys, drafts, unrelated ledgers, credentials, and nondeterministic files.

### 12. Run MCP as a thin root-confined adapter

The official pinned MCP SDK owns framing and negotiation. A server builder registers closed tool/resource schemas generated from the same operation types only when each tool's availability gate passes. Each operation definition carries a static `read-only` or `mutating` effect plus required root classes; server mode is separate runtime metadata and never rewrites that effect. Startup canonicalizes repeatable named ledger/output/secret roots and selects full versus optional read-only mode and online versus optional offline mode. Read-only startup omits every mutating tool from registration, so discovery and direct-call behavior match the general startup-disabled rule. Tool middleware performs schema validation, root-class checks, root resolution, limits, context timeout, and error conversion before calling services. Root failures retain the stable root class and safe configured root ID while absolute paths remain redacted.

The default server exposes applicable read/write operations without general grants; missing output/secret roots fail only operations that need those roots. Timestamp stamp accepts explicit TSA and CA-bundle fields and is disabled by server-wide offline mode; timestamp status and verify stay local. Reveal is the sole capability exception: it is absent unless startup explicitly includes `--allow-reveal`, because a secret root confines files but does not express consent for irreversible publication. Request `confirm: true` remains required but is not an authorization boundary. A held ledger writer lock produces the same immediate conflict in CLI and MCP; the server does not queue writers.

Protocol stdout is passed directly to the SDK; all application diagnostics use an injected stderr logger. Resource URIs contain a root ID plus encoded relative path, never an absolute path. Apart from the explicit public `tsa_url` on timestamp stamp, tool schemas contain no remote source, proxy, credential, or general HTTP configuration. Real-process tests treat any non-protocol stdout byte as a failure.

Alternatives considered: split secret paths into read/write root classes, retain separate write/network/reveal grants, or have MCP handlers invoke CLI subprocesses. Split roots were rejected because seal itself needs protected input reads plus key writes and the resulting policy is hard to explain; broad grants were rejected because roots plus server-wide modes already constrain ordinary effects. A single reveal gate is retained for the asymmetric disclosure risk. Subprocess handlers were rejected because they duplicate serialization, prevent structured cancellation/errors, and leak process-boundary details.

### 13. Use per-command availability gates

The unavailable handler and hidden flag are removed one action at a time only when operation code, unit/integration tests, CLI JSON/help goldens, dry-run/rollback/cancellation checks, MCP parity where the tool exists, documentation, and native-platform gates applicable to that action pass. Groups become visible when they contain at least one implemented child; help labels experimental RFC 3161 behavior honestly.

This allows reviewable increments without again advertising scaffolding as a finished application. Hidden preview flag definitions are corrected before each command is exposed: artifact-dependent reads lose stdin, timestamp verify gains mutation/dry-run semantics, question type remains scalar, root flags become repeatable, general MCP grants disappear, and the reveal gate remains. The final release gate asserts that no planned leaf uses the unavailable handler and every documented/MCP action maps to a service; incomplete or startup-disabled MCP tools are absent from discovery rather than returning application `unavailable`.

Alternative considered: unhide the complete tree after the first implementation slice. Rejected because it recreates the readiness mismatch that triggered this change.

### 14. Treat generated reference and conformance as build outputs

Operation definitions drive CLI/MCP input schemas, JSON result schemas, command reference tables, and parity fixtures. CI runs clean-tree regeneration checks. Tests are layered: pure builders/indexes; document patches; transaction fault injection; crypto/target vectors; RFC 3161 differential/fuzz; verification matrices; package determinism; CLI goldens; MCP in-memory/real-process; and native filesystem/ACL/replacement tests.

Maintained documentation examples are executable fixtures. CI parses every JSON/YAML input example, including quoted timestamps, and runs representative commands against temporary ledgers so prose cannot advertise syntax that the typed input layer rejects. A dogfooding lifecycle fixture additionally exercises authoring before and after retained timestamp evidence with realistic multi-record formatting.

The pinned Python validator parity job compares case verdicts and normalized issue codes/locations against Go rather than merely running the Python harness alone.

Alternative considered: hand-maintain separate reference pages and schema copies. Rejected because drift would be likely across 30 actions and MCP equivalents.

### 15. Make boundary results explicit and consistent across creation, inspection, and presentation

One operation observation time is captured before building new records. Whenever an authored forecast omits `recorded_at`, including the initial forecast accepted by `init`, that operation time supplies the value. If `created_at` is also omitted, the same captured time may supply both fields; an explicit or defaulted `created_at` is otherwise independent and is never copied into `recorded_at`. Builders retain the origin of a defaulted field long enough for safe diagnostics to say that the value came from the operation clock.

Validation locations are modeled as a pointer plus an optional valid one-based source span. A presenter never manufactures line or column zero. Semantic issues against structured input resolve the pointer through the parsed source tree when possible; otherwise human output omits the position and JSON omits the span while retaining the pointer and code. Temporal messages describe the actual inclusive relation, using `must not be before` or `must not be after` rather than implying that equality is rejected.

Source-tree insertions use a declarative semantic field order shared with full-document initialization and generated input/result references. The patcher preserves untouched source bytes and local layout but does not alphabetically sort a newly inserted question, forecast, resolution, or nested business object. Canonical property ordering remains confined to evidence and cryptographic formats.

Target checking produces one result per selected forecast. A forecast with neither retained target metadata nor a deterministic target artifact is `not_applicable` with reason `content.no_retained_target`, its stable IDs, safe expected relative path, and a `target build` next action. `--all` continues after that result and reports every selected forecast; retained-but-missing, unreadable, or mismatched evidence remains an error. A report containing only `pass` and `not_applicable` results succeeds without claiming that absent targets were checked.

Read-only forecast presentation maps retained integrity into an explicit safe public DTO. Confirmed RFC 3161 evidence includes request/response state, TSA identity, `gen_time`, policy OID, serial number, retained CA-bundle path, and `verified_at` in human, plain, JSON, and MCP results. Output distinguishes parsed stored metadata from a complete fresh local cryptographic verification.

MCP root configuration is represented internally by root class, safe route ID, originating flag, and canonical private path. Startup and tool errors expose only the first three. A missing root names the responsible route; an overlap names both conflicting class/ID descriptors, never their absolute paths.

Ledger locking remains fail-fast. A second writer receives immediate conflict and is never queued by the application. `--timeout` bounds work after an operation begins and bounded network/wait effects; it is not a ledger-lock wait duration. CLI callers that intentionally contend must serialize work or implement bounded retry/backoff, and maintained help and documentation state this contract.

Alternatives considered: retain init-specific clock semantics, emit placeholder coordinates, treat absent targets as filesystem failures, queue lock contenders, or require another network verification to see stored block evidence. Rejected because each hides an observable distinction already available to the application and makes ordinary automation or diagnosis less reliable.

## Risks / Trade-offs

- **[The scope is large and cross-cutting]** → Deliver in dependency-ordered vertical slices and keep unavailable gates per action; do not merge placeholder completion.
- **[Flexible empty states drift across adapters]** → Classify the four creation shapes once in shared services and test CLI/MCP parity for each applicable shape.
- **[Init and question-add obtain type from different transport fields]** → Generate distinct closed schemas, normalize both into one builder request, and test identical domain results.
- **[The schema contains legacy publication and external/failed states outside this authoring workflow]** → Preserve valid imported values, never infer source-control behavior, keep external anchors distinct, treat failed forecast integrity as terminal, and recover through an append-only revision.
- **[Resolution dispute replaces the one v1 resolution object]** → Require confirmation, report prior status, preserve forecasts/evidence, and document that v1 does not provide internal resolution history.
- **[Protected Windows key ACLs are easy to get wrong]** → Isolate native code, test on Windows runners, fail closed when owner-only protection cannot be proven, and keep seal unavailable until native tests pass.
- **[Multi-file operations cannot be truly atomic]** → Journal explicit ownership/state, order durable writes conservatively, make retries idempotent, and return recovery state rather than claiming rollback that did not occur.
- **[Pure-Go RFC 3161/CMS handling may accept behavior broader than the product profile]** → Constrain and bound the subset, reject unsupported structures and algorithms, differential-test checked-in fixtures with OpenSSL, fuzz malformed inputs, run real-TSA tests, and require independent review.
- **[A selected TSA may be unavailable or dishonest]** → Require the user to choose it explicitly, make retries safe, preserve verified metadata without overstating clock truth, and allow separate entries from independently selected TSAs.
- **[A retained CA bundle may expire or omit future revocation evidence]** → Verify the chain at `gen_time`, report the exact retained trust input and its limitation, and make no long-term-validation or legal-validity claim.
- **[Target evidence freezes question wording and timing]** → Reject ambiguous rewrites and document annul-plus-new-question as the supported correction path.
- **[Package enumeration can leak secrets]** → Build from a typed allowlisted graph, reject extra roles/paths, scan canaries, and test with adjacent key/private files.
- **[MCP expands side-effect reach and reveal is irreversible]** → Confine every file class to explicit roots, provide optional whole-server read-only/offline modes, validate the explicit timestamp-stamp TSA URL, keep reveal default-off behind its sole explicit gate plus confirmation, enforce limits, and share application transactions.
- **[Two active OpenSpec changes overlap]** → Treat this change as the sole replacement contract, map already completed foundation evidence into its tasks, and retire the older change without syncing or archiving its delta specs.
- **[Stable result schemas can freeze mistakes]** → Version persisted/transport schemas, keep human prose flexible, and require compatibility review for field removal or semantic change.
- **[Presentation-aware patches can still produce large diffs on unfamiliar source styles]** → Preserve untouched slices, constrain style inference to the local container, and gate realistic expanded JSON/YAML ledgers with diff/readability assertions.
- **[Hiding mutating MCP tools changes discovery across server modes]** → Publish static effect metadata, document mode-dependent discovery, and test `tools/list` plus direct-call unknown-tool behavior for every mode.
- **[Changing init's omitted `recorded_at` source changes newly created bytes]** → Treat it as an explicit patch-release behavior correction, report the stored value, add fixed-clock compatibility fixtures, and do not rewrite existing ledgers.
- **[`not_applicable` target results could be mistaken for retained evidence]** → Keep the state and `content.no_retained_target` reason explicit in every output mode, provide the safe build command, and never aggregate it into a claim that a target passed.

## Migration Plan

1. Freeze `build-forecast-ledger-cli-mcp` as superseded, map its 31 completed foundation tasks to equivalent tasks/evidence here, and ensure its delta specs will never be independently synced or archived.
2. Add shared operation request/result contracts, typed input schemas, selector indexes, clocks/effect recorders, exact crypto artifact profiles, and generated references; correct hidden preview flags/help without changing available commands.
3. Implement schema-valid init, root metadata update, and platform actions; unhide each action only after its gates pass.
4. Implement question lifecycle and public forecast append-only actions, including initial-forecast atomicity, forecaster-kind transitions, frozen-question guidance, and disputed replacement transitions.
5. Implement target build/check, protected key files, seal/reveal, safe key-hint repair, and exact vector/cross-platform gates.
6. Implement the constrained RFC 3161 backend, explicit TSA/CA-bundle stamping, complete local verification, and timestamp commands; keep experimental labeling through conformance and review.
7. Implement layered verification and deterministic package build/verify.
8. Implement MCP roots, read-only/offline/reveal modes, incrementally discovered tools, and resources over the same operations and pass real-process parity/framing tests.
9. Run full native, race, fuzz, validator/crypto/RFC 3161 conformance, documentation-generation, package, and release gates; assert no product leaf remains unavailable.
10. Ship the v0.2.2 dogfooding corrections in a new patch release after CLI/MCP parity, executable documentation, and packaged macOS/Linux/Windows regressions pass; no existing ledger or evidence bytes require migration.

Existing v1 ledgers require no data migration. New actions refuse unsupported schema versions rather than rewriting them. Rollback of an implementation slice re-hides its command/tool and restores the previous known-good binary; it does not downgrade or rewrite ledgers. Any newly persisted artifact/profile schema receives an explicit version and compatibility test before release.
