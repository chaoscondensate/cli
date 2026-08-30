## 1. Reconcile Contracts and Permanent Contributor Rules

- [x] 1.1 Update the active `complete-forecast-ledger-command-surface` proposal, design, delta specs, and remaining tasks so this change explicitly supersedes every generic `--input`, MCP `input`, and public `input_file` clause without altering archived history.
- [x] 1.2 Update current main-spec wording that permits optional public input documents so direct CLI flags, flattened MCP properties, and purpose-named secret references are the only active contract.
- [x] 1.3 Update root `AGENTS.md` with permanent rules forbidding generic public authoring documents and requiring MyPC YAML populated mappings/sequences to use readable expanded block style, with explicit `[]`/`{}` exceptions for empty collections.
- [x] 1.4 Add an active-surface audit with a documented exclusion for archived/superseded OpenSpec history; make it fail on runtime, generator, current docs, help, script, fixture, or test use of generic `--input`, MCP `input`, or public `input_file`.
- [x] 1.5 Obtain the exact Forecast Ledger v1.3.0 full commit, published release asset, release-archive SHA-256, schema SHA-256, license/attribution, compatibility decision, conformance corpus, and cryptographic vectors; do not proceed from a floating tag or incomplete source set.
- [x] 1.6 Vendor the exact v1.3.0 schema and published fixtures into versioned directories without hand edits, update `SOURCE.md`, and add byte/digest tests that reproduce every declared pin.
- [x] 1.7 Update `internal/schema`, build information, validation admission, CLI version output, MCP initialization/resources, publication manifests, generated examples, package metadata, and release checks to one v1.3.0 identity while rejecting v1.2.0 without side effects.
- [x] 1.8 Run the complete v1.3.0 valid/invalid corpus and reproduce every published target/seal vector byte-for-byte; update affected application behavior from the exact contract rather than changing vendored bytes.
- [x] 1.9 Rename the maintained version-specific contract capability/title/purpose from v1.2.0 to v1.3.0 during spec synchronization and remove stale current-navigation links without rewriting archived history.

## 2. Make CLI Authoring Direct-Only

- [x] 2.1 Re-audit every mutation request against the maintained command-surface inventory and add any missing leaf-local flags, repeatable typed groups, clear operations, and copyable examples before removing document mode.
- [x] 2.2 Remove `optionalInputFlag`, `directOrDocument`, public path/stdin document decoding, mixed-mode checks, and generic `--input` branches from all CLI commands and completion metadata.
- [x] 2.3 Delete or retire public input-document fixtures and decoder tests that no longer serve an internal protocol, while retaining bounded parsers only where another accepted non-public contract still requires them.
- [x] 2.4 Standardize sealed workflows on `--secret-input` and `--initial-secret-input`, preserve protected file/stdin checks, and prove private values, keys, salts, credentials, and absolute secret paths never enter argv or output.
- [x] 2.5 Update help goldens and flag-only acceptance tests for init, root/platform/question lifecycle, public forecast, seal/reveal metadata, and key-hint operations; assert `--input` is an unknown flag with no file or mutation effect.

## 3. Flatten MCP Authoring Requests

- [x] 3.1 Refactor operation registry metadata to describe one direct public request field inventory plus explicit selector/control and purpose-named secret-reference roles instead of generic public input transports.
- [x] 3.2 Make MCP schema generation flatten each closed public request schema into top-level tool properties, preserve nested typed objects/arrays and patch omission/clear semantics, and fail on selector/control name collisions.
- [x] 3.3 Derive or validate runtime `Allowed`/`Required` contracts from the same registry so generated schema, server validation, and dispatch cannot disagree.
- [x] 3.4 Update dispatch to decode flattened arguments into the shared typed requests after removing only registered selectors and controls; remove `input`/`input_file` wrapper branches.
- [x] 3.5 Rename protected sealed references to `secret_input_file` and `initial_secret_input_file`, retain `key_file`, enforce secret-root confinement, and add non-echoing negative tests for attempted inline secrets.
- [x] 3.6 Regenerate MCP tool schemas and current MCP references, add every-operation contract parity coverage, and assert generic wrappers and undocumented public properties are rejected before filesystem access.

## 4. Add Deterministic Human Time Authoring

- [x] 4.1 Implement a standard-library bounded parser for exact RFC 3339, ISO dates, English short/full month dates, and allowed 24-hour timestamp forms with structured normalization metadata and field-specific policies.
- [x] 4.2 Resolve offset-free CLI values only after the ledger or init timezone is known; apply start-of-day to optional `opens_at`, end-of-day to `expected_resolution_at`, and keep date-only input invalid for evidentiary timestamps.
- [x] 4.3 Detect and reject invalid dates, slash/two-digit/relative/fuzzy forms, timezone abbreviations, trailing content, unsupported precision, and skipped or repeated IANA wall times without an explicit numeric offset.
- [x] 4.4 Keep raw CLI authoring-time values untrusted until the ledger/init timezone is known, normalize them before constructing the typed service request and running strict `ParseTimestamp`/chronology, and keep embedded ledger validation and direct MCP timestamp schemas exact RFC 3339.
- [x] 4.5 Expose normalized exact values in dry-run and structured success data whenever a non-canonical human CLI form was supplied, without decorating `--json` stdout.
- [x] 4.6 Add table, property, and boundary tests for every accepted layout and field policy, leap years, invalid calendar values, positive/negative offsets, DST gaps/folds, non-hour transitions, init timezone use, and host-timezone independence.

## 5. Default Forecast Time from One Observation

- [x] 5.1 Make `forecasted_at` optional in public, sealed, and initial forecast request contracts, CLI help/required-flag logic, and flattened MCP schemas while keeping the stored ledger field required.
- [x] 5.2 Capture the operation clock exactly once per authoring action and format that instant with the loaded ledger `default_timezone`, or init timezone, before applying any omitted forecast/record time defaults.
- [x] 5.3 Share one default helper across public append, sealed append, init initial forecast, and question-add initial forecast so omitted `forecasted_at` and `recorded_at` become the same instant and explicit values override independently.
- [x] 5.4 Preserve inclusive opening/order validation and prove an omitted time before `opens_at` fails before ledger, key, target, journal, or entropy effects rather than being clamped.
- [x] 5.5 Add fixed-clock CLI/service/MCP parity tests for public, sealed, and initial defaults, explicit backdating, non-UTC ledger offsets, both-times-omitted equality, dry-run, and atomic failure.

## 6. Enforce Readable Block-Style YAML

- [x] 6.1 Add one ordered YAML fragment renderer that recursively clears flow style for populated mappings/sequences, preserves explicit empty `[]`/`{}`, uses two-space nesting, stable field order, and the source newline convention.
- [x] 6.2 Route YAML structural additions and subtree replacements through the block renderer without using JSON-marshaler output for ordered patch values; leave JSON and scalar-splice paths unchanged.
- [x] 6.3 Convert an addressed empty flow parent to a local block collection on first insertion and locally re-render an addressed populated flow parent when required, without formatting unrelated siblings or the whole document.
- [x] 6.4 Add document-level fixtures for questions, platforms, public/sealed forecasts, options, quantiles, members, profiles, lifecycle sources, nested collections, comments, quoting, key order, LF/CRLF, and repeated additions.
- [x] 6.5 Add end-to-end init and mutation tests proving every application-authored populated YAML node is block style, `forecasts: []` remains an empty sequence, untouched bytes remain preserved, and cryptographic targets still match the exact v1.3.0 vectors.
- [x] 6.6 Add a release formatting audit that parses produced YAML nodes and fails on populated flow style instead of relying only on line-text matching.

## 7. Update User Guidance and Executable Examples

- [x] 7.1 Remove generic public input-document instructions and calls from README, getting-started, how-to, CLI/MCP reference, generated navigation, examples, help text, and maintained dogfood scripts.
- [x] 7.2 Document direct flag and flattened MCP migrations for every affected operation, including repeatable nested fields and explicit clear behavior, without publishing unavailable or aspirational syntax.
- [x] 7.3 Document supported human date forms, field-specific date-only defaults, timezone selection, DST ambiguity errors, exact MCP timestamps, and optional `forecasted_at` current-time behavior with copyable examples.
- [x] 7.4 Update security guidance to distinguish forbidden public document modes from retained purpose-named protected secret channels and to explain why raw sealed values remain unavailable in argv and MCP public properties.
- [x] 7.5 Document the human-readable YAML guarantee and empty-collection exception, update development/release baselines and breaking-change notes, and rename any public generated "input schema" references whose name falsely implies side-loaded files.
- [x] 7.6 Run and extend `internal/doccheck` so current docs, examples, links, command names, flags, MCP properties, and generated artifact names match the implemented surface.
- [x] 7.7 Update every current schema/version/digest reference, compatibility warning, fixture name, installation artifact, and publication example to the exact v1.3.0 pins; keep planned or unavailable migration behavior visibly absent.

## 8. Complete Verification and Release Gates

- [x] 8.1 Regenerate checked-in help, completion, MCP schemas, request-schema references, and other derived artifacts; verify generation is clean and deterministic on a second run.
- [x] 8.2 Run focused CLI/MCP parity, secret-redaction, source-preservation, YAML style, timestamp normalization/default, and contract-audit tests with negative cases and race-safe fixed clocks.
- [x] 8.3 Run the exact v1.3.0 schema, format, semantic, upstream conformance, target, seal, and publication identity suites fully offline and verify no build/runtime path resolves a remote schema or floating tag.
- [x] 8.4 Exercise native YAML file creation and mutation behavior on macOS, Linux, and Windows, including LF/CRLF preservation, atomic replacement, locks, protected secret paths, and interrupt recovery where affected.
- [x] 8.5 Run `gofmt -w cmd internal tools`, `go mod verify`, `go test ./...`, `go vet ./...`, and `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`; record any environment-limited native checks explicitly.
- [x] 8.6 Perform a final documentation-impact and repository search review proving active code/current guidance identify only the exact v1.3.0 contract, contain no generic public input mode, expose every public authoring field directly, and render every generated populated YAML structure in readable block style.
