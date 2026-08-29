## Context

See `proposal.md` for motivation and the two delta specs for observable behavior. The binary currently embeds the exact v1.0.0 schema and fixtures, rejects every other version, and repeats that pin in validation, build metadata, publication manifests, tests, contributor guidance, and public documentation. Its Go model already represents questions and forecasts as slices, and most read paths naturally handle zero-length slices, but authoring inputs, generated input schemas, CLI/MCP dispatch, and init presentation assume an initial question and forecast always exist.

The upstream v1.1.0 change is deliberately narrow: root `questions` and per-question `forecasts` remain required arrays, but their `minItems: 1` constraints are gone. The exact upstream source is commit `c04c72a178c15cd6cbbdd2e8a7b743d58872a94a`. No external users, persisted data, or compatibility promise constrain this repository, so the implementation can be a clean cutover rather than a migration.

## Goals / Non-Goals

**Goals:**

- Keep one embedded schema identity and one validation path across CLI, MCP, publication, and tests.
- Represent absence explicitly at service boundaries so adapters never infer it from zero-value nested structs or index empty slices.
- Preserve all security and atomicity rules when an initial forecast is supplied.
- Make empty aggregate operations truly side-effect-free and keep specific selectors strict.
- Regenerate and verify every maintained machine-readable and human-readable interface that derives from the changed inputs.

**Non-Goals:**

- Reading, migrating, converting, or rewriting v1.0.0 ledgers.
- Supporting more than one schema version in a binary.
- Making `questions` or `forecasts` optional/null in stored documents; valid empty states use explicit arrays.
- Changing forecast cryptography, target canonicalization, OpenTimestamps, lifecycle rules, or evidence claims.
- Adding a migration command, configuration switch, feature flag, or network schema resolver.

## Decisions

### 1. Replace the contract as one immutable vendored unit

Vendor the upstream v1.1.0 schema, four valid examples, invalid-case corpus, forecast-seal vector, license, and provenance under a versioned `v1.1.0` directory. Update the schema constants, embedded paths, contract identifier, conformance digest table, publication pin, attribution, and build metadata in the same implementation change. Remove the runtime v1.0.0 vendored contract rather than retaining a dormant second schema.

This keeps the existing single-schema architecture and makes accidental fallback impossible. Keeping both versions was rejected because it would add compatibility behavior the user explicitly does not need. Fetching the floating `v1.1.0` tag or live schema was rejected because reproducibility depends on the exact commit and digests.

### 2. Model optional creation stages with pointers, not zero values

Change init input to hold an optional question and change both initial-question and question-add inputs to hold an optional initial forecast. The generated JSON Schemas will remove `question` and `initial_forecast` from the relevant `required` arrays while leaving every required field inside a present object unchanged.

Use a shared service-level creation classifier with four shapes: ledger only, ledger plus question, ledger plus public forecast, and ledger plus sealed forecast. Builders first create and validate the root, optionally build the question shell, then optionally add a forecast. CLI and MCP adapters inspect the classified shape only to enforce transport-specific protected-input rules and select dry-run/commit behavior; they do not dereference optional nested inputs independently.

Pointers provide an unambiguous distinction between “not supplied” and a malformed supplied object. Treating an all-zero nested struct as absence was rejected because it would blur omission with invalid input and could bypass validation. Removing the existing convenience path for initializing one question was rejected because v1.1.0 adds flexibility rather than forbidding the old valid workflow.

### 3. Keep urfave leaf flags explicit and make only init input optional

Introduce an optional form of the existing `--input` leaf flag for `init`; all other commands retain their current required flags. When the flag is absent, the adapter passes an empty init input without opening stdin or a path. When it is present, including `--input -`, the existing bounded decoder and protection checks apply.

This is directly supported by urfave: the leaf command continues to define `--file/-f` and the scalar flags as required, while the init-only input flag omits `Required: true`. Making the generic input helper optional was rejected because it would silently weaken unrelated commands.

### 4. Centralize conditional key handling around the creation shape

The service classifier determines whether the request contains no forecast, a public forecast, or a sealed forecast before any destination is created. A key path is valid only for the sealed shape. Sealed input continues to require a protected input file or stdin where allowed, a confined new key path, ledger/key path separation, key-first atomic commit, and retained-secret recovery semantics. Ledger-only and question-only paths use the normal single-ledger commit and never initialize cryptographic effects.

For MCP, absence of both `input` and `input_file` is valid only for init. Question add still requires exactly one input source. Inline backlog questions are safe; the schema restriction on inline initial-forecast visibility must be conditional on `initial_forecast` being present, while sealed initial forecasts remain file-only. A separate “allow secrets inline” exception was rejected because schema flexibility does not change the secret boundary.

### 5. Return a shape-safe init result

Replace positional construction of the init response with a typed or equivalently centralized result builder. Counts and common fields are always populated; question and forecast fields use omission semantics based on actual slice lengths. CLI and MCP use the same result builder for planned and committed operations.

This prevents empty-slice panics and makes JSON accurately describe all four creation shapes. Returning empty-string IDs was rejected because clients could confuse them with real values and because the schema already defines non-empty IDs.

### 6. Add a question-shell mutation path beside existing forecast paths

Expose the existing question-shell builder through a question-add mutation that appends the validated question with `forecasts: []` and emits one `/questions/-` source patch. If an initial forecast exists, retain the existing public or sealed builders and atomic file services. Initial ledger construction uses the same shell logic, preventing CLI/MCP and init/question-add validation drift.

This reuses current type-specific question validation and the existing empty-slice initialization. Constructing raw maps in adapters was rejected because it would duplicate validation and source-preserving patch behavior.

### 7. Define empty aggregate work before allocating external effects

Selection remains strict for a specific question/forecast: existing selectors return `not_found`. Aggregate selection (`target ... --all` and selector-free verify) may return zero items successfully. Target planning and commit must return immediately after selection when the artifact slice is empty, before inspecting or creating `proofs`, `proofs/targets`, or a resource journal. Verification may construct an observer/profile object but must not make a request when there are no forecast layers to check; tests use a counting observer to prove this.

Publication is different from target generation: even with no forecasts, the ledger and manifest are meaningful portable artifacts. Its current collection loop already naturally yields those two files, so the implementation should preserve that behavior and add explicit tests and output assertions. Treating empty publication as an error was rejected because a signed or transported question backlog is a valid v1.1.0 use case.

### 8. Preserve lifecycle and history semantics

Question lifecycle operations continue to validate the selected question and resolution data, not forecast count. Forecast add/seal continue to append history; on an empty forecast array there is no implicit supersedes link. If a caller explicitly names `supersedes_forecast_id`, normal same-question existence checks still apply. No command fabricates a placeholder forecast merely to satisfy downstream logic.

### 9. Regenerate contracts from one input-schema source and reconcile planning text

Update the maintained service input-schema source first, then run the repository generator for standalone input schemas and MCP operation schemas. Update CLI help/goldens, MCP contract tests, README, guides, reference, build/provenance documentation, third-party notices, changelog, and `AGENTS.md` together. Documentation examples use an empty-first sequence followed by question add and forecast add, with a separate sealed example.

The active `complete-forecast-ledger-command-surface` change contains accepted v1.0.0 and mandatory-initial-forecast assumptions. During apply, reconcile only its live planning artifacts that conflict with this change so subsequent work does not reintroduce the old contract. Superseded archived artifacts remain historical records and are not rewritten.

### 10. Test a small state-and-adapter matrix

Use the published fixtures for contract conformance and repository-created fixtures for behavior. Cover JSON and YAML across three states: zero questions, one question/zero forecasts, and one question/one forecast. Exercise CLI service paths, MCP dispatch, human/plain/JSON presentation, dry-run and commit, first later public/sealed forecast, lifecycle changes, specific-selector failures, empty aggregate targets/verification, and empty publication build/verify. Retain existing security, recovery, invalid-case, and cryptographic-vector suites unchanged except for the schema pin.

## Risks / Trade-offs

- [A nil optional value is dereferenced in one adapter or presenter] → Route through one creation classifier and one result builder; add empty-state CLI and MCP regression tests.
- [A zero-target operation still creates directories or a journal] → Short-circuit after validated selection and before filesystem preflight; assert the entire ledger directory is unchanged.
- [An empty verification `pass` is read as proof of completeness] → Preserve the standard completeness limitation, expose an empty forecast list and document-layer evidence, and document that pass covers applicable checks only.
- [Conditional MCP schemas accidentally permit sealed material inline] → Generate restrictions for a present forecast, retain protected-file tests, and add negative inline-sealed cases.
- [Schema identity drifts across metadata, publication, docs, or notices] → Keep one constants package, pin digest tests, regenerate goldens, and search for stale v1.0.0 identifiers before completion.
- [Breaking cutover surprises a developer with an old local file] → Make the changelog and compatibility docs explicit; return `unsupported_schema_version` with the one supported version. Do not add conversion code.
- [Editing another active change obscures history] → Reconcile only current conflicting assertions and record the superseding change name; leave superseded/archived artifacts untouched.

## Migration Plan

There is no ledger-data migration. Apply the change atomically in source: vendor and verify v1.1.0, switch the single schema pin, update optional service inputs and creation paths, add empty-operation guards, regenerate interfaces, update documentation, then run conformance and the full test suite. Release the result as a breaking pre-1.0 application update with v1.1.0 called out in release notes.

Rollback is a source/binary rollback to the previous release. The previous binary will reject newly created v1.1.0 ledgers, and the new binary will reject v1.0.0 ledgers; neither direction rewrites data. Because the project currently has no active-user or stored-data compatibility requirement, no automated rollback converter is provided.
