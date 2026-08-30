## Purpose

Ensures YAML ledgers written by the application remain comfortable for people to inspect, review, and edit without sacrificing source-preserving mutations.

## ADDED Requirements

### Requirement: Generated YAML uses block style
Every new `.yaml` or `.yml` ledger and every non-empty mapping or sequence structurally inserted by the application SHALL use YAML block style with consistent two-space nesting, one sequence entry per line, and stable schema-oriented field order. The application MUST NOT serialize a populated question, forecast, platform, source, option, quantile, member, profile, or other record as a flow-style `{...}` mapping or serialize a populated collection as a flow-style `[...]` sequence.

#### Scenario: Added question is expanded
- **WHEN** question add appends a populated question to a YAML ledger
- **THEN** the question begins as an indented `- id:` block and its nested forecast window and other populated structures appear on separate indented lines rather than as one flow-style value

#### Scenario: New ledger is readable
- **WHEN** init creates a YAML ledger containing platforms, a question, or a forecast
- **THEN** every populated structure is emitted in block style with stable indentation and field order

### Requirement: Empty collections remain explicit
Schema-required or intentionally present empty sequences and mappings SHALL remain explicit as `[]` or `{}` because omission or an empty block value would change their YAML type. This exception MUST apply only to genuinely empty collections and MUST NOT permit populated flow-style content.

#### Scenario: Backlog question retains an empty forecast list
- **WHEN** a question is created without forecasts
- **THEN** its readable block mapping contains `forecasts: []` and validation decodes the value as an empty sequence rather than null

#### Scenario: Populated list cannot use the exception
- **WHEN** a later forecast is appended
- **THEN** the populated `forecasts` sequence is represented as block entries rather than an inline bracketed list

### Requirement: Local formatting preserves unrelated source text
A YAML mutation SHALL limit reformatting to the newly created or structurally replaced node and the minimum parent collection syntax needed to represent it. Unrelated comments, scalar quoting, blank lines, key order, line endings, and untouched records SHALL remain byte-preserved under the existing source-preserving transaction model. Existing third-party flow-style nodes outside the mutation target SHALL not trigger whole-document formatting.

#### Scenario: Append beside comments
- **WHEN** a question is appended to a hand-formatted YAML ledger containing comments and unrelated quoted scalars
- **THEN** the new question is block style while unrelated comments and scalar bytes remain unchanged

#### Scenario: Convert an empty parent sequence locally
- **WHEN** an item is appended to a parent represented as `questions: []`
- **THEN** only that empty collection representation and inserted node are changed to the correctly indented block sequence

### Requirement: Every YAML mutation path follows the same style policy
Initialization, public and sealed question creation, platform creation, public and sealed forecast creation, lifecycle source insertion, and every other operation that adds or replaces a populated YAML structure SHALL use the same block-style rendering policy. JSON behavior and canonical cryptographic bytes SHALL remain unchanged.

#### Scenario: Sealed and public records have style parity
- **WHEN** equivalent public and sealed forecast structures are added to YAML ledgers
- **THEN** their public ledger records follow the same block indentation and field-order rules without exposing private material

#### Scenario: JSON remains JSON
- **WHEN** an operation mutates a `.json` ledger
- **THEN** the YAML formatting policy does not alter JSON serialization or validation behavior

### Requirement: Contributor and release policy protects readability
The repository's root contributor instructions SHALL state the permanent MyPC YAML rule that application-authored populated mappings and sequences use readable expanded block style. Formatting fixtures and release checks SHALL cover every structural insertion path and fail if the application emits a populated flow-style node in a YAML ledger.

#### Scenario: A new mutation emits flow style
- **WHEN** a contributor adds a structural YAML mutation that renders a populated mapping or sequence in flow style
- **THEN** the formatting audit fails before release and identifies the mutation fixture
