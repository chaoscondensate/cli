# Empty Ledger Workflows Specification

## Purpose

Defines predictable authoring and command behavior when a valid ledger has no questions or when a valid question has no forecasts.

## Requirements

### Requirement: Initialize a ledger without a question
`forecast-ledger init` SHALL require an explicit ledger file and scalar identity flags and SHALL NOT accept generic `--input` or a template document. It SHALL create a valid v1.3.0 document containing explicit empty `platforms` and `questions` collections. Every supported non-secret root metadata and platform value SHALL be authorable through flags.

#### Scenario: CLI creates a minimal empty ledger
- **WHEN** a caller runs `forecast-ledger init` with the required `--file`, ledger ID, timezone, forecaster ID, and forecaster name flags
- **THEN** the command creates a locally valid v1.3.0 ledger with explicit empty collections and no forecast or key artifact

#### Scenario: Dry-run plans only the empty ledger
- **WHEN** the same flag-only command is run with `--dry-run`
- **THEN** it validates the prospective empty ledger, reports only the deferred ledger-file effect, and writes no file

#### Scenario: Optional metadata input has no question
- **WHEN** init receives supported root metadata or repeated platform values through flags and no question flags
- **THEN** the command applies those values and creates a ledger with `questions: []` without reading an input document

### Requirement: Optionally initialize one question and forecast
`forecast-ledger init` MAY create one question and an optional initial forecast from the same direct flags exposed by the corresponding question-add and forecast-add workflows. The question SHALL contain all fields required by its type. When forecast flags are absent, the created question SHALL contain `forecasts: []`. When public forecast flags are present, existing atomic creation rules SHALL apply. A sealed initial forecast SHALL keep its private bundle and key destination in protected file or stdin channels while accepting all non-secret metadata through flags.

#### Scenario: Init creates a question backlog item
- **WHEN** valid init flags describe a question and omit initial-forecast flags
- **THEN** the new ledger contains that open question with an explicit empty `forecasts` array

#### Scenario: Init retains public first-forecast behavior
- **WHEN** valid init flags describe a question and a public initial forecast
- **THEN** the ledger and public forecast are created atomically under the existing validation and chronology rules

#### Scenario: Init retains sealed first-forecast behavior
- **WHEN** init flags describe a question and sealed forecast metadata while protected input and a new key-file destination supply secret material
- **THEN** the ledger, commitment, and protected key file are created under the existing atomic secret-handling rules

### Requirement: Add a question without a forecast
`forecast-ledger question add` SHALL require the explicit ledger, question ID, question type, and the type-specific question flags and SHALL NOT accept generic `--input`. Initial-forecast flags SHALL be optional. Omitting them SHALL append a source-preserving question whose `forecasts` value is an explicit empty array. MCP SHALL use flattened direct properties and route through the same application service.

#### Scenario: Add question to empty JSON ledger
- **WHEN** valid type-specific flags add a question without an initial forecast to an empty JSON ledger
- **THEN** exactly one question is appended, its `forecasts` array is empty, and unrelated source content and formatting remain unchanged

#### Scenario: Add question to YAML ledger
- **WHEN** valid type-specific flags add a question without an initial forecast to a YAML ledger
- **THEN** it is appended through the same source-preserving mutation path and the resulting YAML remains valid v1.3.0

#### Scenario: Add with an initial forecast remains atomic
- **WHEN** question-add flags include a valid public forecast or sealed forecast metadata plus protected secret inputs
- **THEN** the existing first-forecast validation, conditional key-file handling, and atomic commit behavior remain in force

### Requirement: Secret arguments follow forecast presence
`--key-file` and MCP `key_file` SHALL be accepted only when a supplied initial forecast is sealed. They SHALL be rejected as usage errors when the question or initial forecast is absent or when the initial forecast is public. A sealed initial forecast SHALL still require protected private input and a new protected key destination.

#### Scenario: Key file is rejected for a backlog question
- **WHEN** init or question add omits `initial_forecast` but supplies a key-file path
- **THEN** the operation fails with a usage error before creating or changing any file

#### Scenario: Sealed initial forecast requires secret inputs
- **WHEN** init or question add supplies a sealed initial forecast without the required protected input or key destination
- **THEN** the operation fails before ledger mutation and does not disclose private forecast content

### Requirement: Init results represent every creation shape
CLI and MCP init results SHALL always include `ledger_id`, `schema_version`, `question_count`, `forecast_count`, `effects`, and `recovery`. They SHALL include `question_id` only when an initial question exists, and SHALL include `forecast_id` and `visibility` only when an initial forecast exists. Human and plain success text SHALL NOT claim that a question or forecast was created when it was absent.

#### Scenario: Result for an empty ledger is index-safe
- **WHEN** init creates or plans a ledger without questions
- **THEN** the result reports zero question and forecast counts, omits question and forecast identifiers, and completes without an internal error

#### Scenario: Result for a backlog question omits forecast fields
- **WHEN** init creates one question without a forecast
- **THEN** the result reports one question and zero forecasts, includes the question ID, and omits forecast ID and visibility

### Requirement: CLI and MCP expose equivalent direct requests
The CLI and MCP server SHALL route empty-ledger and backlog-question creation through the same application services. CLI init and question add SHALL accept complete non-secret data from flags. MCP init and question add SHALL expose the corresponding non-secret request fields as top-level tool properties without generic wrappers or public file references. Sealed private forecast data SHALL continue to require purpose-named protected secret references and `key_file` inside configured roots.

#### Scenario: MCP initializes without input
- **WHEN** an MCP client invokes init with the required file and scalar identity properties
- **THEN** the server creates the same empty ledger as the equivalent CLI command

#### Scenario: MCP adds an inline backlog question
- **WHEN** an MCP client supplies valid direct question properties without `initial_forecast`
- **THEN** the server appends the question with `forecasts: []` and does not require secret capability

#### Scenario: MCP sealed input remains file-only
- **WHEN** an MCP client attempts to supply raw sealed initial forecast material inline
- **THEN** the server rejects it and directs the caller to the purpose-named protected secret reference

### Requirement: Empty collections are valid read and lifecycle states
Validation, status, platform operations, question list and show, forecast list, and question update, resolve, annul, and dispute SHALL accept valid ledgers with empty collections whenever their selected record exists. List and status outputs SHALL report empty collections and zero counts without indexing assumptions. Existing question lifecycle and resolution-value rules SHALL apply even when a question has no forecasts.

#### Scenario: Empty ledger read commands succeed
- **WHEN** validate, status, question list, or platform list reads a valid ledger with no questions
- **THEN** the command succeeds and reports an empty collection or zero count in human, plain, and JSON modes

#### Scenario: Forecast list on backlog question succeeds
- **WHEN** forecast list selects an existing question whose `forecasts` array is empty
- **THEN** it succeeds with an empty list and the human result says `No forecasts`

#### Scenario: Resolve a question without forecasts
- **WHEN** an existing forecast-free question satisfies the normal resolution lifecycle and resolution-value rules
- **THEN** resolve, annul, or dispute changes its lifecycle state without adding or requiring a forecast

### Requirement: The first later forecast starts history
Public add and sealed forecast commands SHALL accept an existing open question with no forecasts. The first forecast SHALL NOT gain an implicit `supersedes_forecast_id`; an explicitly supplied superseded ID SHALL still have to identify an existing forecast in that question.

#### Scenario: First public forecast is appended
- **WHEN** forecast add targets a question with no forecasts and omits `supersedes_forecast_id`
- **THEN** it appends the first forecast with no supersedes link

#### Scenario: Missing superseded forecast is rejected
- **WHEN** the first forecast names a superseded forecast ID that does not exist
- **THEN** the operation fails without changing the ledger

### Requirement: Aggregate forecast operations handle an empty selection
Selector-free verification and explicit `--all` target operations SHALL handle a valid selection containing zero forecasts without fabricating forecast evidence. They SHALL return empty forecast or target arrays, make no network request, and create no target files, proof directories, resource journal, or other artifact. Verification SHALL report `no_evidence` with application category `incomplete` and exit 9; target operations SHALL remain successful empty operations. An operation selecting a specific absent question or forecast SHALL retain the stable `not_found` error.

#### Scenario: Verify empty ledger locally
- **WHEN** verify runs on a valid empty ledger without a specific forecast selector
- **THEN** the document layer passes, `forecasts` is empty, overall is `no_evidence`, the application category is `incomplete`, request counts are zero, and no network request occurs

#### Scenario: Build all targets for no forecasts
- **WHEN** target build runs with `--all` on a ledger containing no forecasts
- **THEN** it succeeds with `targets: []`, reports no effects, and creates no `proofs` path or journal

#### Scenario: Check all targets for no forecasts
- **WHEN** target check runs with `--all` on a ledger containing no forecasts
- **THEN** it succeeds with `targets: []` and no failure code

#### Scenario: Specific existing forecast remains not found
- **WHEN** target, timestamp, forecast show, or reveal selects a forecast in an existing question whose forecast list is empty
- **THEN** it fails with stable code `not_found` and performs no write or network request

#### Scenario: Seal creates the first forecast
- **WHEN** forecast seal targets an existing open question whose forecast list is empty and supplies a new forecast ID and valid protected inputs
- **THEN** it creates the first sealed forecast without a supersedes link under the normal atomic secret-handling rules

### Requirement: Publication supports empty evidence sets
Publication build SHALL accept a valid empty ledger and create a deterministic package containing the ledger and manifest with no forecast target or receipt entries. Publication verify SHALL validate that package, preserve its manifest and file-integrity observations, report an empty evidence list and zero network requests, and return overall `no_evidence` with application category `incomplete` and exit 9. It SHALL NOT claim forecast evidence or forecast-set completeness.

#### Scenario: Build package from empty ledger
- **WHEN** publication build receives a valid ledger with no forecasts
- **THEN** it creates only the ledger entry and manifest, records the v1.3.0 schema pin, and reports evidence state `complete`

#### Scenario: Verify empty package
- **WHEN** publication verify checks that package
- **THEN** it returns `evidence: []`, overall `no_evidence`, application category `incomplete`, zero requests, and the standard limitation that forecast-set completeness is not proved

### Requirement: Generated contracts and documentation teach empty-first use
Generated request schemas, MCP tool schemas, CLI help, README, getting-started material, command reference, examples, compatibility notes, and changelog SHALL agree that CLI init and question creation reject generic input documents and do not require initial forecasts. CLI examples SHALL show complete flag-only empty-first, platform-add, question-add, and later forecast-add workflows, while protected sealed examples SHALL keep private data out of argv.

#### Scenario: Generated schemas match runtime behavior
- **WHEN** maintained schemas and MCP contracts are regenerated and checked
- **THEN** CLI exposes no generic public document flag, MCP exposes no generic wrapper or public file property, and conditional sealed-input restrictions remain represented

#### Scenario: A new user follows the empty-first documentation
- **WHEN** a user copies the documented init, platform-add, question-add-without-forecast, and later forecast-add commands
- **THEN** each command uses real flags, requires no prepared JSON/YAML fragment, and produces a valid v1.3.0 ledger at every step
