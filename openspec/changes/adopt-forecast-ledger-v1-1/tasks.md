## 1. Pin and embed Forecast Ledger v1.1.0

- [x] 1.1 Vendor the exact schema, four valid examples, invalid-case corpus, forecast-seal vector, license, and provenance from commit `c04c72a178c15cd6cbbdd2e8a7b743d58872a94a` under the v1.1.0 schema directory and verify the recorded archive, schema, fixture, and vector SHA-256 values.
- [x] 1.2 Switch `internal/schema` constants, embed paths, conformance metadata, and comments to the v1.1.0 pin, remove the runtime v1.0.0 vendored contract, and update embed digest tests.
- [x] 1.3 Update local validation's contract ID and schema-version admission to accept only `1.1.0`; add a side-effect-free `unsupported_schema_version` regression for an otherwise valid v1.0.0 ledger.
- [x] 1.4 Update build/version metadata, MCP metadata/resources, and publication manifest expectations to derive and report the v1.1.0 version, commit, and schema digest from the shared constants.
- [x] 1.5 Run conformance against all four pinned valid fixtures, the full invalid-case corpus, and the unchanged forecast-seal vector; assert validation performs no schema network access.

## 2. Make init and initial forecasts optional in services

- [x] 2.1 Change init question and initial-forecast fields to explicit optional values and update the maintained input-schema definitions so present objects retain all current required fields while `question` and `initial_forecast` become optional at their parent levels.
- [x] 2.2 Add a shared service creation classifier and builder flow for ledger-only, question-only, public-initial-forecast, and sealed-initial-forecast shapes without using zero-value nested structs as absence.
- [x] 2.3 Expose a validated question-shell mutation/file path that appends `forecasts: []` with one source-preserving `/questions/-` patch for JSON and YAML ledgers.
- [x] 2.4 Retain the existing public and sealed initial-forecast builders and atomic commit/recovery behavior when an initial forecast is present, routing them through the shared classification result.
- [x] 2.5 Centralize a shape-safe init result containing common counts and effects while omitting question/forecast fields that do not exist; use it for plan and commit results.
- [x] 2.6 Enforce conditional key-file rules before side effects: reject a key path for ledger-only, question-only, and public shapes, and retain protected input, destination separation, and key-first atomicity for sealed shapes.
- [x] 2.7 Add service tests for all four creation shapes, malformed present objects, chronology defaults, team/root metadata, source preservation, dry-run effects, and no nil/slice-index panics.

## 3. Update the urfave CLI surface

- [x] 3.1 Add an init-only optional `--input` flag while keeping `--file/-f`, scalar init flags, and every unrelated command input flag explicitly required on their existing urfave leaf commands.
- [x] 3.2 Update init action routing so absent input creates the empty root, optional question input creates a backlog item, and public/sealed forecast input follows the shared service paths.
- [x] 3.3 Update question-add action routing and success messages so an omitted initial forecast appends only the question while existing public/sealed messages and protected-input checks remain accurate.
- [x] 3.4 Update init/question-add help, examples, human/plain output, JSON output, and help/output goldens to describe optional input and optional initial forecasts without claiming absent records were created.
- [x] 3.5 Add CLI integration tests for JSON and YAML empty init, metadata-only init, question-only init/add, all output modes, `--dry-run`, stdin input when supplied, conditional key errors, and existing public/sealed creation.

## 4. Update the MCP contract and dispatch

- [x] 4.1 Make init accept neither `input` nor `input_file`, keep question add's input source requirement, and preserve explicit file/root confinement in the MCP operation contracts.
- [x] 4.2 Make the inline-public schema restriction conditional on a present `initial_forecast`, allow inline backlog questions, and keep sealed private data restricted to protected `input_file` plus `key_file`.
- [x] 4.3 Route MCP init and question add through the same creation classifier, file services, and shape-safe result builder as CLI, removing direct optional-field and `[0]` access.
- [x] 4.4 Add MCP dispatch and contract tests for empty init, backlog question init/add, dry-run, result omission/counts, inline sealed rejection, protected sealed success, no secret exposure, and recoverable domain errors.

## 5. Prove downstream empty-state behavior

- [x] 5.1 Add JSON/YAML tests showing validate, status, platform operations, question list/show/update, forecast list, and question resolve/annul/dispute handle zero questions or zero forecasts with the specified counts and empty outputs.
- [x] 5.2 Add public and sealed forecast tests showing the first later forecast on a backlog question has no implicit supersedes link and an explicit missing superseded ID is rejected without mutation.
- [x] 5.3 Short-circuit target plan/commit/check for an empty `--all` selection before directory inspection, directory creation, or journal creation; assert empty results and a byte-for-byte unchanged working directory.
- [x] 5.4 Add selector-free verification tests for empty ledgers and backlog questions that assert document pass, overall pass, empty forecast results, preserved limitations, and zero observer/HTTP requests.
- [x] 5.5 Add specific-selector negative tests across forecast show, target, timestamp, and reveal paths to retain `not_found` and prove no file or network side effect when a question has no forecasts; prove separately that seal can create the first forecast.
- [x] 5.6 Add deterministic publication build/verify tests for an empty evidence set, asserting ledger-plus-manifest contents, v1.1.0 pin, empty evidence, `complete`/`pass` results, completeness limitation, and zero requests.

## 6. Regenerate interfaces and update maintained information

- [x] 6.1 Run `go generate ./internal/service`, inspect generated input/MCP schemas and fixtures, and update their deterministic goldens so optionality and conditional secret rules match runtime behavior.
- [x] 6.2 Update `AGENTS.md`, `CONTRIBUTING.md`, `REUSE.toml`, `THIRD_PARTY_NOTICES.md`, build/development documentation, and compatibility metadata with the exact v1.1.0 commit, paths, digests, attribution, and no-migration policy.
- [x] 6.3 Rewrite the README and getting-started creation flow to show empty init, optional backlog question creation, and a first later forecast, while keeping a correct protected sealed example and linking the broader project at `https://chaoscondensate.com/`.
- [x] 6.4 Update command reference, question/forecast how-to pages, MCP descriptions, validation/verification explanations, and navigation/metadata so empty collections, selector behavior, publication semantics, and applicable-check `pass` wording are explicit.
- [x] 6.5 Add a changelog/release-note entry identifying the v1.1.0 cutover as breaking and stating that v1.0.0 files are rejected with no migration command.
- [x] 6.6 Reconcile non-historical conflicting v1.0.0 and mandatory-initial-forecast assertions in the active `complete-forecast-ledger-command-surface` OpenSpec artifacts, recording this change as the superseding decision without editing archived history.
- [x] 6.7 Run documentation checks and executable snippet tests, including the empty-first CLI sequence, and fix stale help, schema, link, platform, or package references.

## 7. Complete regression and release-readiness checks

- [x] 7.1 Run focused schema, validation, service, CLI, MCP, target, verification, publication, presentation, and doccheck tests and resolve every new or changed golden deliberately.
- [x] 7.2 Run `gofmt -w cmd internal`, `go mod verify`, `go test ./...`, and `go vet ./...` with the repository-pinned toolchain.
- [x] 7.3 Run `govulncheck` and cross-build the binary for supported macOS, Linux, and Windows targets; record any native filesystem checks that require the release matrix rather than claiming cross-build equivalence.
- [x] 7.4 Search maintained code, generated artifacts, docs, and active planning files for stale v1.0.0 pins or claims that every ledger/question requires a forecast; review intentional historical or dependency-version matches separately.
- [x] 7.5 Run `openspec validate adopt-forecast-ledger-v1-1 --strict` after implementation updates and confirm every requirement scenario has corresponding automated coverage or an explicit platform-matrix verification entry.
