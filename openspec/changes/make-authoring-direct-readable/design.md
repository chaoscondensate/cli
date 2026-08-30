## Context

See `proposal.md` for motivation. Public authoring currently has two adapter routes: CLI flags or a bounded JSON/YAML document, while MCP wraps the same operation schemas under `input` and sometimes resolves `input_file`. `internal/service` already owns typed requests and domain validation, but registry `InputTransport`, generated MCP schemas, manual tool contracts, dispatch, docs, and tests encode the wrapper modes independently.

The current worktree still pins Forecast Ledger v1.2.0, while this change must consume the revised v1.3.0 source as one exact vendored release. The v1.3.0 commit, archive digest, schema digest, attribution, and fixtures are not inferred from the version string and must be present before implementation can mark the contract work complete. Timestamp validation remains intentionally strict under the new embedded contract. CLI adapters currently cast flag strings directly to `ledger.Timestamp`, and several actions format `Clock.Now()` before the ledger and its `default_timezone` are available. Forecast request types require `forecasted_at` even though `recorded_at` already defaults from the operation clock.

Structural patches preserve unrelated source bytes. YAML additions normally use `yaml.Marshal`, but additions to flow parents and values that expose JSON-oriented marshal behavior can retain or introduce populated flow-style fragments. The application therefore needs a local fragment policy rather than a whole-document formatter.

## Goals / Non-Goals

**Goals:**

- Make direct CLI flags and flattened MCP properties the only public authoring routes while retaining shared service behavior.
- Replace the v1.2.0 embed with exact, reproducible v1.3.0 bytes and one consistent public contract identity.
- Keep secret material out of argv and public MCP properties through explicit, operation-specific protected references.
- Normalize bounded human CLI times only after the correct ledger/init timezone is known, while leaving stored and machine-facing timestamps strict.
- Derive omitted forecast times from one operation-clock observation across public, sealed, and initial creation.
- Render every application-authored populated YAML structure in readable block style without reformatting unrelated source bytes.
- Make active contracts, generated references, contributor policy, docs, fixtures, and release checks agree with the new surface.

**Non-Goals:**

- No generic replacement batch-import flag, hidden compatibility alias, environment-variable input, or JSON/YAML document encoded inside one string flag.
- No raw secret values in CLI arguments or MCP public request properties.
- No relative-time language, fuzzy parsing, locale inference, timezone abbreviations, or third-party/LLM date parser.
- No hand editing of vendored v1.3.0 schema/fixture bytes, floating-tag fetch, automatic v1.2.0 conversion, or dual-version runtime.
- No intentional change to canonical target/seal bytes or RFC 3161 behavior beyond changes required by the exact published v1.3.0 contract and vectors.
- No whole-file YAML formatter and no rewrite of archived or superseded OpenSpec history.

## Decisions

### 1. Adopt v1.3.0 as one exact immutable source set

Implementation will obtain the published v1.3.0 release asset from one exact full commit, calculate and review the release/archive and schema SHA-256 values, and copy the schema, license/attribution, and conformance fixtures unchanged into versioned vendor/testdata directories. `internal/schema`, build information, release checks, and public metadata will use compile-time constants from that reviewed source record; no build or runtime step fetches upstream.

The application remains exclusive-version: v1.3.0 is accepted and v1.2.0 is rejected without mutation, artifact, secret, or network effects. There is no implicit upgrader. The maintained capability title/purpose/path will be moved from its version-specific v1.2.0 name to v1.3.0 when the delta is synchronized so current specs do not retain a misleading identity.

The complete upstream v1.3.0 valid/invalid corpus and cryptographic vectors are release gates. If the revised schema changes field shapes used by authoring, input contracts, target/seal bytes, or fixtures, those differences are implemented from the exact source before the direct-interface work is declared compatible. Vendored bytes are never edited to fit existing code.

Alternative considered: change only `Version` and directory names. This was rejected because it would detach the binary identity from immutable provenance and could silently retain obsolete fixtures or vectors.

### 2. Remove generic input transports at the operation contract

Public operation definitions will describe the shared typed request and direct fields, not a transport choice. The generic inline/public-file variants of `InputTransport`, `optionalInputFlag`, `directOrDocument`, public `DecodeOperationInput` call sites, and document-mode dispatch branches will be removed. Typed request structs and closed schema definitions remain useful internal contracts; removing side-loaded documents does not mean replacing typed services with adapter-specific logic.

CLI leaf builders will be the sole public CLI constructors and will continue to map into those request structs. All non-secret nested data will use existing repeated flags or dedicated subcommands. If a nested record cannot be expressed clearly, the command surface must gain explicit fields rather than falling back to JSON/YAML.

Secret reads remain a separate transport concern with semantic names. CLI uses `--secret-input` or `--initial-secret-input`; MCP uses `secret_input_file` or `initial_secret_input_file`. These paths are checked as protected inputs, confined to secret roots where applicable, and never generalized as public request documents.

Alternatives considered:

- Keeping `--input` as a deprecated alias was rejected because it leaves the duplicate behavior and regression path in place.
- Adding a dedicated `import` command is outside this change. A future import workflow would need its own explicit contract, provenance, security, and compatibility design.
- Moving private values into flags or inline MCP was rejected by the existing secret-handling invariant.

### 3. Flatten MCP schemas from the same closed request definitions

The MCP generator will merge the selected operation schema's root properties and required set into the tool's top-level properties alongside selectors and controls. Nested public objects and arrays remain typed schema values; only the generic wrapper disappears. Operation-specific fields such as IDs, `dry_run`, and `confirm` remain top-level and name collisions become generator errors.

The service registry becomes the single inventory for generated schemas and runtime contracts. Hand-maintained `Allowed`/`Required` lists will either be generated from that inventory or checked against it so schema, dispatch, and docs cannot drift. Dispatch will decode the complete top-level tool argument map into the existing request type after removing only selectors and controls known to the operation.

Protected references are not merged from public schemas. They are explicit registry fields with a secret role, root class, requiredness, and public-safe description. This retains root confinement and enables generated tool contracts without reviving `input_file`.

Alternative considered: retain an `input` object but forbid files. This was rejected because the user-facing contract is direct MCP properties and the wrapper continues to obscure field discovery.

### 4. Normalize human CLI times inside service orchestration

A bounded authoring-time parser will accept only the layouts named in `friendly-date-authoring`. It will be standard-library based and return a canonical timestamp plus metadata indicating the applied layout, timezone, and field default. Strict `ParseTimestamp` remains unchanged and is always run on the normalized result.

Raw CLI time strings must reach the service boundary without first becoming trusted `ledger.Timestamp` values. For file-backed operations, normalization occurs after `LoadAndValidateLedger` supplies `default_timezone` and before the domain builder runs under the mutation plan/transaction. Init normalization uses the explicit timezone in `InitRootRequest`. This avoids adapter-side ledger reads, system-timezone inference, and time-of-check/time-of-use disagreement.

MCP properties remain strict RFC 3339. They pass through the same normalization entry point as the canonical case but cannot use the human-only layouts because their schema rejects them first.

For wall times without offsets, the parser loads the IANA location, constructs the candidate, and verifies the requested wall components round-trip exactly. It also checks for an alternative valid offset around a transition; skipped or repeated wall times are rejected. Date-only defaults are applied by semantic field policy, not by a global parser default.

Alternatives considered:

- Host-local timezone was rejected as non-reproducible.
- UTC for every omitted offset was rejected because the ledger already declares its human scheduling timezone.
- A fuzzy parser dependency was rejected because it expands grammar silently and makes compatibility difficult to pin.
- A separate `--closes-on` flag was not selected because the requested interface is the existing timestamp flags; field-specific date-only semantics make that use deterministic.

### 5. Capture and format one operation instant

Forecast input types will represent `forecasted_at` as optional until defaults are applied. Each authoring action captures `Clock.Now()` once as a `time.Time` or equivalent observation before planning/commit. After the ledger timezone is known, the service formats that same instant with `observed.In(location).Format(time.RFC3339)` and uses it for omitted `forecasted_at` and omitted `recorded_at`.

Explicit values override only their own fields. Thus an explicit historical `forecasted_at` can coexist with defaulted current `recorded_at`, while both omitted fields are exactly equal as instants and strings. Initial public/sealed forecasts and ordinary public/sealed appends share one default helper before target, commitment, chronology, and patch construction.

The default never changes the question window. An observation after close or before open returns the existing inclusive chronology error, and planning must detect it before any key, journal, or ledger effect.

Alternative considered: capture separately in adapters and crypto services. This was rejected because calls could cross a second boundary and produce unequal defaults or different offsets.

### 6. Render populated YAML fragments from ordered block nodes

The document package will own a YAML fragment renderer that converts application patch values into ordered `yaml.Node` trees, recursively clears `FlowStyle` on every populated mapping/sequence, preserves explicit empty `[]`/`{}` nodes, and encodes with two-space indentation and the document's newline convention. Ordered patch values must retain schema field order without falling back to their JSON marshaler.

Block-parent additions insert only the new block fragment. Adding to an empty flow collection such as `questions: []` replaces that collection locally with a block collection. Adding to or structurally replacing an existing populated flow collection re-renders that addressed parent/subtree in block style; unrelated siblings and document bytes stay untouched. Scalar-only replacements continue to use the current byte-splice path and preserve scalar style where safe.

This policy applies below `internal/document`, so init, platforms, questions, forecasts, lifecycle sources, and future structural patches cannot select different YAML styles. JSON rendering and canonical cryptographic serialization remain separate.

Alternatives considered:

- Running a full YAML marshal after every mutation was rejected because it loses comments, quoting, order, line endings, and reviewable diffs.
- Preserving a populated flow parent during additions was rejected because it would keep generating the unreadable form this change forbids.

### 7. Treat active guidance as one consistency boundary

Implementation starts by reconciling the active `complete-forecast-ledger-command-surface` artifacts and current main specs with this superseding decision. Runtime code, generators, generated references, help, completion, examples, scripts, current docs, and tests then change together. Root `AGENTS.md` receives permanent direct-only authoring and MyPC YAML block-style rules.

A release audit searches active/current paths for forbidden generic input surfaces and parses produced YAML fixtures to detect populated flow-style nodes. Archived and superseded OpenSpec directories are excluded because they are immutable historical records, not product guidance.

## Risks / Trade-offs

- **[Breaking removal disrupts existing batch scripts]** → Provide a migration table from each request member to its flag or MCP property, update dogfood coverage, and call out that no compatibility alias remains.
- **[The v1.3.0 source set is incomplete or unpublished when apply starts]** → Do not guess pins or use a floating tag; leave the contract tasks incomplete until the exact commit, release asset, digests, attribution, fixtures, and vectors are available together.
- **[v1.3.0 changes cryptographic or authoring shapes]** → Treat its published contract and vectors as higher precedence, update affected services/tests/docs together, and do not preserve v1.2.0 behavior through a local schema edit.
- **[Flattened MCP fields collide with selectors or controls]** → Make schema generation fail on duplicate property names and cover every operation with generated/manual contract parity tests.
- **[Large nested CLI records become verbose]** → Prefer repeatable typed flag groups or dedicated child commands; do not trade discoverability for hidden document syntax.
- **[Timezone database differences affect future offsets]** → Persist the normalized explicit offset immediately, retain the IANA zone as context, reject transition ambiguity, and test representative zones without changing stored validation.
- **[Default current forecast time surprises backdating workflows]** → Explicit `--forecasted-at`/MCP `forecasted_at` remains available; results expose the derived value and normal chronology rejects an invalid current instant.
- **[Local flow-parent conversion can alter formatting inside the touched subtree]** → Bound replacement to the addressed collection, preserve unrelated bytes, and add comment/order/newline fixtures around every structural path.
- **[Ordered values accidentally use JSON marshaling in YAML]** → Build YAML nodes from semantic ordered values and assert both key order and absence of populated `FlowStyle` in decoded output.

## Migration Plan

1. Update active OpenSpec/current contributor contracts, including the root `AGENTS.md` permanent rules, before changing exposed interfaces.
2. Vendor and pin the exact v1.3.0 contract, provenance, corpus, and vectors; make all build/public metadata agree and reject v1.2.0.
3. Introduce the direct operation-field inventory and purpose-named secret roles; flatten MCP generation and dispatch, then remove generic MCP wrappers.
4. Remove CLI document-mode flags, decoders, branches, fixtures, and examples after every public field has direct coverage.
5. Add authoring-time normalization and one-observation forecast defaults while retaining strict v1.3.0 stored validation.
6. Centralize block-style YAML structural rendering and run source-preservation/conformance tests.
7. Regenerate help, completion, MCP/request references, update dogfood and public docs, and add release audits.
8. Run documentation checks and the full Go verification/security suite. Release notes mark the v1.3.0-only, direct-only, flattened-MCP surface as breaking.

Rollback is a normal source rollback before release. After release, do not restore `--input` or wrapper aliases as an emergency compatibility path; any future import feature requires a separate accepted change. Ledger files remain compatible because canonical stored data and schema version do not change.
