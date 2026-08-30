## 1. Command and Field Inventory

- [x] 1.1 Enumerate every registered CLI leaf command, identify every current `input`/`input-file` requirement, and classify each leaf as read-only, ordinary authoring, protected-secret authoring, or non-ledger operation.
- [x] 1.2 Map every field in each authoring command's typed request and bounded input schema to a public scalar flag, repeatable collection flag, typed nested group, dedicated child command, protected secret channel, or documented computed/service-only value.
- [x] 1.3 Resolve the final flag names, aliases, enum spellings, repeatability, defaults, and set/clear behavior against the embedded v1.2.0 contract and record them in a maintained command-surface inventory used by tests.
- [x] 1.4 Add the contributor invariant to `AGENTS.md`: every non-secret authoring field needs a complete flag or command route, `--input` cannot be the required ordinary CLI path, secrets stay out of argv, and flag-only acceptance tests and docs are mandatory.

## 2. Shared CLI Input Assembly

- [x] 2.1 Add presence-aware scalar binding helpers that distinguish omitted, explicitly empty, false, zero, and non-zero values without moving domain validation into the CLI adapter.
- [x] 2.2 Add repeatable collection and typed nested-record parsers that preserve order, reject malformed or incomplete values, and never accept embedded JSON/YAML as a substitute for flags.
- [x] 2.3 Add reusable patch helpers for setter versus `--clear-<field>` semantics and reject set/clear conflicts before ledger access.
- [x] 2.4 Add a shared authoring-mode resolver that makes direct flags and optional `--input` mutually exclusive while allowing selectors, output/dry-run controls, and protected destinations in either applicable mode.
- [x] 2.5 Refactor document mode and direct flag mode to construct the same existing transport-neutral typed request before service invocation.
- [x] 2.6 Add focused unit tests for presence semantics, repeated values, typed nested groups, duplicate single-value flags, mixed modes, and set/clear conflicts.

## 3. Ledger and Platform Authoring

- [x] 3.1 Implement flag-only `init` for the minimal v1.2.0 empty ledger, including explicit file, ledger identity, timezone, forecaster identity, operation-clock defaults, dry-run, and no required template/input document.
- [x] 3.2 Add flags for every optional non-secret init root metadata and initial platform field, including repeatable collection behavior where applicable.
- [x] 3.3 Add flag-only optional question and public initial-forecast assembly to init by reusing the question and forecast binders; keep sealed private material in protected input and key files.
- [x] 3.4 Implement complete flag-only `ledger update` with omission/set/clear semantics for every mutable non-secret root field.
- [x] 3.5 Implement `platform add` direct mode so `forecast-ledger platform add --file l.yaml --platform metaculus` succeeds when the embedded contract permits the minimal platform and otherwise names only genuinely contract-required missing flags.
- [x] 3.6 Implement complete flag-only `platform update`, including optional metadata setters and explicit clears, while preserving `platform remove` and read-only command behavior.
- [x] 3.7 Add JSON and YAML source-preservation, dry-run, stable-result, invalid-input, and optional document-mode parity tests for ledger and platform mutations.

## 4. Question Authoring and Lifecycle

- [x] 4.1 Implement shared direct flags for common question fields and all v1.2.0 question-type discriminators.
- [x] 4.2 Implement type-specific flag groups for every supported question type, including required values, options/categories, units/ranges, resolution rules, and other public nested or repeated fields in the inventory.
- [x] 4.3 Implement flag-only `question add` without an initial forecast and verify that it appends an explicit empty forecast list to JSON and YAML ledgers.
- [x] 4.4 Implement optional public or sealed-initial-forecast metadata flags for `question add`, reusing forecast assembly and retaining protected secret and key-file handling.
- [x] 4.5 Implement complete flag-only `question update` with presence-aware setters, collection replacement, and explicit clears for every mutable public field.
- [x] 4.6 Implement complete flag-only `question resolve`, including typed outcome values, known-at time, and repeatable public outcome-source metadata.
- [x] 4.7 Implement complete flag-only `question annul` and `question dispute`, including all public reason/evidence fields and explicit chronology values.
- [x] 4.8 Add per-question-type creation and update tests plus lifecycle parity, missing-type-field, malformed-nested-field, chronology, no-side-effect, and help tests.

## 5. Forecast and Protected Authoring

- [x] 5.1 Implement shared direct flags for common forecast metadata and all public v1.2.0 forecast-type discriminators.
- [x] 5.2 Implement type-specific flag groups for every public forecast value shape, confidence/weight metadata, supersedes links, timestamps, rationale, references, and other non-secret fields in the inventory.
- [x] 5.3 Implement complete flag-only `forecast add` for the first and later public forecasts while preserving append-only history and supersedes validation.
- [x] 5.4 Refactor `forecast seal` so every non-secret selector and metadata field is available through flags while plaintext, keys, salts, and credentials remain limited to protected files or stdin as specified.
- [x] 5.5 Implement direct non-secret flags for `forecast reveal` and `forecast key-hint update` while preserving protected key input and no-secret-output guarantees.
- [x] 5.6 Add per-forecast-type tests for flag-only public creation, first-forecast behavior, revisions, sealed creation atomicity, reveal/key-hint behavior, dry-run, source preservation, and optional document-mode parity.
- [x] 5.7 Add negative tests proving private values, raw keys, salts, and credentials have no argv or environment flag and never appear in help, completion, errors, logs, JSON, or normal stdout.

## 6. Whole-Surface Regression Guard

- [x] 6.1 Add an inventory test that fails for any registered ordinary authoring leaf with a required `input` flag or without a classified route for every non-secret authorable request field.
- [x] 6.2 Add an inventory test that fails if a secret-classified request field is exposed as an argv/environment flag or if a computed/service-only field lacks an explicit rationale.
- [x] 6.3 Add table-driven CLI acceptance tests invoking every authoring leaf without `--input` and asserting that no path returns `Required flag "input" not set`.
- [x] 6.4 Add direct-versus-document semantic parity tests for every retained optional input schema, including stable results, error categories, dry-run effects, and resulting normalized ledger data.
- [x] 6.5 Regenerate and verify maintained CLI/MCP operation schemas, fixtures, help snapshots, and completion data so CLI input optionality changes do not unintentionally alter MCP typed-input or secret-root contracts.

## 7. Version Presentation

- [x] 7.1 Add a compact human version renderer covering release, build/source, embedded schema, Go, and MCP metadata with one consistent placeholder for unavailable optional values.
- [x] 7.2 Route version styling through the shared presentation color policy and restrict color to labels or small accents.
- [x] 7.3 Preserve the established `version --json` field names/types and stable plain output, with diagnostics on stderr and no network access.
- [x] 7.4 Add TTY, redirected, `--no-color`, `NO_COLOR`, `TERM=dumb`, plain, JSON, release-build, and development-build tests, including assertions that disabled modes contain no ANSI escapes.

## 8. Help and Public Documentation

- [x] 8.1 Rewrite every authoring leaf help page to describe all direct flags, type-specific requirements, repeated/nested syntax, defaults, set/clear behavior, protected-secret exceptions, and at least one copyable flag-only example.
- [x] 8.2 Update README and getting-started material with a valid flag-only sequence for init, minimal platform add, backlog question add, first public forecast add, and a protected sealed workflow.
- [x] 8.3 Update the canonical CLI command reference and relevant how-to/explanation pages so input documents are clearly optional batch/compatibility mode and mixed modes are documented as errors.
- [x] 8.4 Review security documentation for argv/process-list exposure of public values and protected handling of private forecasts, keys, salts, and credentials; update warnings without weakening existing guarantees.
- [x] 8.5 Review compatibility, changelog, documentation baseline, generated examples, and navigation; update every affected page and record any intentionally unchanged area in the implementation handoff.
- [x] 8.6 Run `go test ./internal/doccheck` and exercise every maintained shell example against the built CLI on representative JSON and YAML ledgers.

## 9. Verification and Handoff

- [x] 9.1 Run focused CLI, service-parity, presentation, source-preservation, atomic-write, and secret-boundary tests while implementing each command group.
- [x] 9.2 Run `gofmt -w cmd internal`, `go mod verify`, `go test ./...`, and `go vet ./...` with the repository's selected toolchain.
- [x] 9.3 Run `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` and record any environment-only limitation separately from product failures.
- [x] 9.4 Verify native command parsing and filesystem behavior on macOS, Linux, and Windows rather than treating cross-builds as runtime evidence.
- [x] 9.5 Perform a final command-surface inventory diff against the registered CLI tree and do not complete the change until every ordinary authoring field has a tested direct route and no public documentation teaches an input-file-only happy path.
