## MODIFIED Requirements

### Requirement: Initialize a ledger without a question
`forecast-ledger init` SHALL require an explicit ledger file and scalar identity flags but SHALL NOT require `--input` or a template document. With no input, it SHALL create a valid v1.2.0 document containing explicit empty `platforms` and `questions` collections. Every supported non-secret root metadata and platform value SHALL be authorable through flags. An optional input document MAY remain available as a mutually exclusive batch-input mode.

#### Scenario: CLI creates a minimal empty ledger
- **WHEN** a caller runs `forecast-ledger init` with the required `--file`, ledger ID, timezone, forecaster ID, and forecaster name flags and omits `--input`
- **THEN** the command creates a locally valid v1.2.0 ledger with explicit empty collections and no forecast or key artifact

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
`forecast-ledger question add` SHALL require the explicit ledger, question ID, question type, and the type-specific question flags, but SHALL NOT require `--input`. Initial-forecast flags SHALL be optional. Omitting them SHALL append a source-preserving question whose `forecasts` value is an explicit empty array. MCP SHALL continue to use its typed object or configured file input and SHALL route through the same application service.

#### Scenario: Add question to empty JSON ledger
- **WHEN** valid type-specific flags add a question without an initial forecast to an empty JSON ledger
- **THEN** exactly one question is appended, its `forecasts` array is empty, and unrelated source content and formatting remain unchanged

#### Scenario: Add question to YAML ledger
- **WHEN** valid type-specific flags add a question without an initial forecast to a YAML ledger
- **THEN** it is appended through the same source-preserving mutation path and the resulting YAML remains valid v1.2.0

#### Scenario: Add with an initial forecast remains atomic
- **WHEN** question-add flags include a valid public forecast or sealed forecast metadata plus protected secret inputs
- **THEN** the existing first-forecast validation, conditional key-file handling, and atomic commit behavior remain in force

### Requirement: CLI and MCP expose equivalent optional inputs
The CLI and MCP server SHALL route empty-ledger and backlog-question creation through the same application services. CLI init and question add SHALL accept complete non-secret data from flags without an input document. MCP init SHALL permit both `input` and `input_file` to be absent, while MCP question add SHALL accept exactly one supported typed question-input source. If inline MCP input includes an initial forecast, only a public forecast SHALL be allowed; sealed private forecast data SHALL continue to require a protected `input_file` and `key_file` inside configured roots.

#### Scenario: MCP initializes without input
- **WHEN** an MCP client invokes init with the required file and scalar identity properties but without `input` or `input_file`
- **THEN** the server creates the same empty ledger as the equivalent flag-only CLI command

#### Scenario: MCP adds an inline backlog question
- **WHEN** an MCP client supplies valid inline question input without `initial_forecast`
- **THEN** the server appends the question with `forecasts: []` and does not require secret capability

#### Scenario: MCP sealed input remains file-only
- **WHEN** an MCP client supplies a sealed initial forecast inline
- **THEN** the server rejects it and directs the caller to the protected file-based input path

### Requirement: Generated contracts and documentation teach empty-first use
Generated input schemas, MCP tool schemas, CLI help, README, getting-started material, command reference, examples, compatibility notes, and changelog SHALL agree that CLI init and question creation do not require input documents or initial forecasts. CLI examples SHALL show complete flag-only empty-first, platform-add, question-add, and later forecast-add workflows, while protected sealed examples SHALL keep private data out of argv.

#### Scenario: Generated schemas match runtime behavior
- **WHEN** maintained command contracts are regenerated and checked
- **THEN** CLI init and question add have no required `input` flag, MCP preserves its typed request contract, and conditional sealed-input restrictions remain represented

#### Scenario: A new user follows the empty-first documentation
- **WHEN** a user copies the documented init, platform-add, question-add-without-forecast, and later forecast-add commands
- **THEN** each command uses real flags, requires no prepared JSON/YAML fragment, and produces a valid v1.2.0 ledger at every step
