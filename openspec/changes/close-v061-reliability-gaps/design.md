## Context

See `proposal.md` for motivation and the four delta specs for required behavior. The current presenter converts Go values to generic maps through reflection before `encoding/json`, while MCP marshals service values directly. CLI and MCP then choose outcome literals independently. Ledger replacement recovery is implemented and fault-tested but is callable only as a separate storage function used by tests; normal mutations stop when its journal exists. Outcome-source checking resolves a hostname before handing the original hostname to the default HTTP transport, which resolves again and may use environment proxies. Finally, `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` selects Go 1.26.7 from the tool module when this checkout is entered through an older bootstrap Go, so it cannot load this Go 1.27 project; the same scanner succeeds when explicitly built with Go 1.27.0.

The exact Forecast Ledger v1.3.0 contract, source-preserving ledger bytes, canonical targets, seals, RFC 3161 evidence, and package manifests are outside these defects. The implementation must keep shared behavior below adapters and avoid a new general-purpose framework.

## Goals / Non-Goals

**Goals:**

- Make JSON redaction a transformation of the real public JSON representation, with encoding failures returned rather than silently approximated.
- Give each registered operation one service-owned structured outcome classifier used by CLI and MCP.
- Integrate the existing conservative ledger recovery proof into the writer transaction without releasing the lock between recovery and mutation.
- Share one narrow public-address policy between RFC 3161 and outcome-source transports, while keeping their HTTP methods, headers, redirects, and response profiles separate.
- Make the vulnerability scanner a project tool executed by the project Go toolchain and a fail-closed CI dependency of release snapshots.

**Non-Goals:**

- Migrating every service to the currently unused generic `Operation[Request, Data]` interface or replacing explicit CLI actions with a reflection-based command executor.
- Unifying every plan/commit signature or eliminating all repeated preflight work; concurrency-sensitive revalidation remains where required.
- Splitting `internal/service`, optimizing directory scans or structural patching without profiles, or changing wall-time parsing outside a separately reproduced defect.
- Adding a public recovery command, manual journal editor, HTTP MCP transport, proxy support, or caller-controlled networking knobs.
- Bulk deletion of exported helpers, test convenience wrappers, decorative `String` methods, or empty directories unrelated to the changed paths.

## Decisions

### 1. Marshal first, then redact the decoded JSON tree

Replace the reflection serializer with a fallible shared JSON transformation:

1. Marshal the original public value with `encoding/json`, honoring embedded fields, `omitempty`, `,string`, `json.Marshaler`, and ignored fields.
2. Decode the bytes into `any` with `json.Decoder.UseNumber` so large integers are not rounded.
3. Recursively replace values whose serialized keys match the closed secret-key policy.
4. Marshal the sanitized tree into the CLI or MCP envelope.

Public service result types continue to keep raw artifacts and private bytes unexported or tagged `json:"-"`; contract-audit and canary tests reject any serializable raw byte field. Generic error-detail maps receive an early recursive byte-slice replacement before the JSON pass. Presenter and MCP encoding both use the same transformation and propagate an internal encoding error if the public value cannot be represented.

Alternative considered: repair the reflection walker with `reflect.VisibleFields`. Rejected because it would still have to reproduce field dominance, `omitempty` behavior for non-zero empty collections, tag options, custom marshalers, map-key rules, and future `encoding/json` changes.

### 2. Add a small service-owned outcome classifier, not a new execution framework

Add an `Outcome` contract containing operation name, stable code, safe structured message, failure category, and whether safe data exists. A closed classifier is registered beside each `OperationDefinition`; operation-specific classifiers inspect typed result state such as `Changed`, dry-run, verification state, or `FailureCode`. Registry construction fails if an exposed operation lacks a classifier. The adapters retain only transport work:

- CLI selects human/plain formatting, presents the outcome, and maps a failure category to an exit after presenting safe data.
- MCP builds the same outcome envelope, sets `isError` for a non-empty failure category, and keeps `data` when `HasData` is true.
- Fatal errors that occur before a safe report exists continue through the normal application-error envelope.

This centralizes the exact defects without forcing all 30 operations through the unused generic service interface in one risky patch. The existing `Result[T]`/`OperationFunc` scaffolding is either left intact for a later migration or removed only if the new classifier makes an overlapping unused piece clearly redundant and all contract tests are updated.

Alternative considered: copy corrected literals into both adapters. Rejected because it would repair current examples while preserving the structural cause and leaving no exhaustive parity gate.

### 3. Recover inside `UpdateLedger` while retaining one lock

Move journal inspection after writer-lock acquisition and extract the current recovery proof into an unexported helper that assumes the lock is held. `UpdateLedger` calls it before reading the mutation input. The helper has three outcomes: no journal, recovered, or error. Recovery reuses the mutation's validation callback, validates the journal and sibling name before file access, and preserves the current digest rules:

- Current equals expected: remove only a matching owned sibling if present, flush cleanup, remove and flush the journal.
- Current equals original and sibling equals expected: parse and fully validate the sibling, replace, flush, then remove and flush the journal.
- Anything else: return a typed error and retain all recovery evidence.

The existing exported `RecoverLedger` test seam acquires the lock and delegates to the same helper, preventing two implementations from drifting. The requested mutation continues under the same lock; its normal domain behavior decides whether the recovered operation makes the retry idempotent, unchanged, or conflicting.

Alternative considered: expose `forecast-ledger recover` and `ledger_recover`. Rejected for this defect because it expands both public adapters and still strands unattended callers even though the existing journal contains enough evidence for conservative automatic recovery. Ambiguous state remains an explicit stop rather than an automatic repair.

### 4. Resolve and dial within one constrained transport boundary

Extract only address classification and approved-address resolution into a narrow internal network-policy package shared with the RFC 3161 transport. Outcome retrieval constructs its own `http.Transport` with `Proxy: nil`, a minimum TLS version, bounded timeouts, and a `DialContext` that:

1. takes the hostname for the connection being opened;
2. performs one resolver lookup for that connection;
3. rejects the whole answer set if any address violates the shared public policy, including `100.64.0.0/10` and the existing reserved prefixes;
4. dials only the approved numeric addresses from that lookup.

The URL hostname remains unchanged in the request, so `net/http` supplies it for Host and TLS verification. Redirect checks retain the current maximum and permitted-origin behavior but every redirected connection passes through the same constrained dialer. Resolver, dialer, clock/deadline, and transport seams are internal tests only and cannot be supplied through CLI or MCP.

Alternative considered: validate with `LookupIPAddr` and then call the default client. Rejected because the second resolution is the rebinding gap. Alternative considered: reuse the entire RFC 3161 HTTP client. Rejected because timestamp POST/media-type/no-redirect rules are not the outcome-source GET profile.

### 5. Pin `govulncheck` as a project tool

Use the Go tool dependency mechanism in `go.mod` to pin `golang.org/x/vuln/cmd/govulncheck` at the reviewed version and standardize on `go tool govulncheck ./...`. This causes the scanner to be built in the project module with the Go 1.27 toolchain selected by `go.mod`, works with automatic toolchain selection from an older bootstrap Go, and avoids shell-specific environment assignment. It is development tooling only.

Run the scan once in a dedicated Linux CI job after module verification; make snapshot/release jobs depend on it. Network or advisory-database failure remains a failed gate. Update `AGENTS.md`, `CONTRIBUTING.md`, dependency/release documentation, and dated evidence together. Record both reachable findings and module-only advisories without describing a clean reachable scan as an audit.

Alternative considered: prepend `GOTOOLCHAIN=go1.27.0` to the current `go run` command. It works locally but is shell-specific, duplicates the version already selected by `go.mod`, and leaves the tool outside the project dependency graph.

### 6. Keep cleanup proportional to the fixes

Remove adapter-local outcome branches replaced by the classifier, the reflection-only redaction helpers, and parameters or helpers that become unused because of those edits. Do not turn the review's complete P2/P3/P4 inventory into acceptance scope. In particular, duplicated security-path logic, wall-time ambiguity, and potential performance work require their own focused tests or measurements before modification.

## Risks / Trade-offs

- **[Correcting preview JSON and codes breaks consumers of the defect]** → Call out the exact old/new shapes and codes in changelog, migration notes, generated contracts, and release notes; ship before v1 rather than preserving two incompatible forms.
- **[Marshal/decode redaction could change large numbers]** → Decode with `UseNumber` and compare sanitized output against direct JSON tokens in property tests.
- **[A result type accidentally exposes raw bytes as base64]** → Keep bytes out of public result schemas, audit registered result types, and run canary tests through both adapters.
- **[Automatic recovery completes a prior mutation before the retry runs]** → Retain one lock, validate exact digests and prospective bytes, then let normal idempotency/conflict behavior report the recovered current state; document this sequence.
- **[Shared address policy changes RFC 3161 behavior]** → Move the existing RFC 3161 prefix set without weakening it and run all timestamp transport, provider, redirect, and deterministic fixture tests unchanged.
- **[Direct numeric dialing reduces resolver failover behavior]** → Try approved answers in deterministic resolver order within the operation deadline and return only bounded safe network errors.
- **[The vulnerability database is unavailable]** → Fail closed in CI and distinguish infrastructure failure from a vulnerability finding; do not reuse a stale clean claim.
- **[Broad outcome parity work grows unexpectedly]** → Use the current operation registry as the exhaustive boundary and avoid changing request parsing, domain services, or human formatter text unless required by a shared outcome.

## Migration Plan

1. Add redaction and outcome parity regression tests that reproduce v0.6.1 before changing behavior.
2. Land the shared JSON sanitizer and outcome classifiers, regenerate references, and update CLI/MCP fixtures and migration notes together.
3. Integrate lock-scoped recovery and run deterministic storage faults plus native filesystem CI.
4. Introduce the shared address policy and constrained outcome transport, then rerun RFC 3161 and offline verification suites.
5. Pin the project tool, add the CI dependency, run a complete scan, and update dated documentation with the actual result.
6. Release as a pre-v1 compatibility correction. Rollback is a code rollback only; no ledger, evidence, key, or package migration is needed. Do not roll back by restoring the unsafe network transport or documenting manual journal deletion.
