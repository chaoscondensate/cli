## MODIFIED Requirements

### Requirement: Initialize a ledger without a question
`forecast-ledger init` SHALL require an explicit ledger file and scalar identity flags and SHALL NOT expose `--input` or accept a template or public authoring document. It SHALL create a valid v1.3.0 document containing explicit empty `platforms` and `questions` collections when no question flags are supplied. Every supported non-secret root metadata and platform value SHALL be authorable through flags.

#### Scenario: CLI creates a minimal empty ledger
- **WHEN** a caller runs `forecast-ledger init` with the required `--file`, ledger ID, timezone, forecaster ID, and forecaster name flags and supplies no question fields
- **THEN** the command creates a locally valid v1.3.0 ledger with explicit empty collections and no forecast or key artifact

#### Scenario: Dry-run plans only the empty ledger
- **WHEN** the same direct command is run with `--dry-run`
- **THEN** it validates the prospective empty ledger, reports only the deferred ledger-file effect, and writes no file

#### Scenario: Optional metadata input has no question
- **WHEN** init receives supported root metadata or repeated platform values through flags and no question flags
- **THEN** the command applies those values and creates a ledger with `questions: []` without reading a public input document

### Requirement: Add a question without a forecast
`forecast-ledger question add` SHALL require the explicit ledger, question ID, question type, and type-specific public fields directly through CLI flags or flattened MCP properties. Initial-forecast fields SHALL be optional. Omitting them SHALL append a source-preserving question whose `forecasts` value is an explicit empty array. Both adapters SHALL route the direct request through the same application service and SHALL expose no generic public input document or file alternative.

#### Scenario: Add question to empty JSON ledger
- **WHEN** valid type-specific CLI flags add a question without an initial forecast to an empty JSON ledger
- **THEN** exactly one question is appended, its `forecasts` array is empty, and unrelated source content and formatting remain unchanged

#### Scenario: Add question to YAML ledger
- **WHEN** valid type-specific CLI flags add a question without an initial forecast to a YAML ledger
- **THEN** it is appended through the same source-preserving mutation path as a readable block-style record and the resulting YAML remains valid v1.3.0

#### Scenario: MCP adds a backlog question from direct fields
- **WHEN** `question_add` receives valid top-level question fields without initial-forecast fields
- **THEN** the server appends the question with `forecasts: []` and requires neither a generic input wrapper nor secret capability

#### Scenario: Add with an initial forecast remains atomic
- **WHEN** question-add direct fields include a valid public forecast or sealed forecast metadata plus its purpose-named protected secret reference
- **THEN** the existing first-forecast validation, conditional key-file handling, and atomic commit behavior remain in force

### Requirement: Secret arguments follow forecast presence
CLI `--key-file` and MCP `key_file` SHALL be accepted only when a supplied initial forecast is sealed. They SHALL be rejected as usage errors when the question or initial forecast is absent or when the initial forecast is public. A sealed initial forecast SHALL require the purpose-named CLI `--initial-secret-input` or MCP `initial_secret_input_file` protected reference plus a new protected key destination.

#### Scenario: Key file is rejected for a backlog question
- **WHEN** init or question add omits initial-forecast fields but supplies a key-file path
- **THEN** the operation fails with a usage error before creating or changing any file

#### Scenario: Sealed initial forecast requires secret inputs
- **WHEN** init or question add supplies a sealed initial forecast without its purpose-named protected secret reference or key destination
- **THEN** the operation fails before ledger mutation and does not disclose private forecast content

## ADDED Requirements

### Requirement: CLI and MCP expose equivalent direct creation inputs
CLI init and question add SHALL accept non-secret data only through leaf-local flags. MCP init and question add SHALL accept the corresponding non-secret data only through closed top-level tool properties. Equivalent direct requests SHALL use the same defaults, validation, atomicity, results, and error codes. Protected sealed values SHALL remain outside public direct properties and use only the applicable purpose-named protected reference.

#### Scenario: MCP initializes from top-level fields
- **WHEN** an MCP client invokes init with required identity properties and optional root, platform, or question properties directly on the tool call
- **THEN** the server creates the same ledger as the equivalent CLI flags without `input` or `input_file`

#### Scenario: Sealed creation remains protected
- **WHEN** an MCP client creates a sealed initial forecast
- **THEN** public metadata is supplied directly, private material is resolved from `initial_secret_input_file` inside a configured secret root, and inline private values are rejected

## REMOVED Requirements

### Requirement: CLI and MCP expose equivalent optional inputs
**Reason**: Generic `input` and `input_file` sources are removed in favor of direct CLI flags, direct MCP properties, and purpose-named secret references.

**Migration**: Move public members to their corresponding CLI flags or top-level MCP properties; rename a protected sealed bundle reference to the operation-specific secret property.

#### Scenario: Removed MCP wrappers are rejected
- **WHEN** init or question add receives `input` or `input_file`
- **THEN** the closed MCP tool schema rejects the unknown property without reading a referenced file or changing the ledger
