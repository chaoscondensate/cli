## Purpose

Defines creation, validation, inspection, and safe append-only maintenance of Forecast Ledger v1 JSON and YAML documents.

## ADDED Requirements

### Requirement: Create a valid ledger
The system SHALL create a JSON or YAML Forecast Ledger v1 document at the explicit `--file` path from supplied forecaster identity, ledger ID, timezone, and creation time. It SHALL refuse to overwrite an existing file unless the user explicitly selects a safe replacement flow.

#### Scenario: Initialize a YAML ledger
- **WHEN** a user supplies a new `.yaml` path and all required root identity fields
- **THEN** the system writes a schema-valid v1 ledger with an empty platform registry and question list

### Requirement: Offline structural and semantic validation
Validation SHALL use the embedded pinned v1.0.0 contract without network `$ref` resolution and SHALL enforce JSON Schema Draft 2020-12 formats plus all published cross-record semantic rules, including duplicate-key rejection, IDs and references, IANA timezone validity, lifecycle chronology, question/forecast type agreement, probability coverage and sums, quantile ordering, integrity target digests, and revealed seal consistency.

#### Scenario: Structurally valid but semantically invalid ledger
- **WHEN** a multiple-choice forecast omits an option or its basis points do not sum to 10,000
- **THEN** validation fails with one or more precise errors identifying the forecast and offending values

#### Scenario: Unsupported timestamp protocol
- **WHEN** a ledger contains an RFC 3161 timestamp entry or any timestamp type other than OpenTimestamps
- **THEN** validation rejects the entry as unsupported by Forecast Ledger v1

#### Scenario: Offline validation
- **WHEN** a valid ledger is checked without network access
- **THEN** structural, format, semantic, target-digest, and revealed-bundle checks complete without downloading schema files

### Requirement: Exact schema-version handling
The v1 implementation SHALL accept only the exact schema versions it embeds and declares supported. It MUST reject an unknown or changed contract rather than silently coercing, upgrading, or validating against a floating remote tag.

#### Scenario: Future schema version
- **WHEN** a ledger declares an unsupported schema version
- **THEN** the command fails with an unsupported-version error and identifies the supported versions

### Requirement: Manage platforms and questions
The system SHALL list, show, add, and update platform registry entries and SHALL list, show, add, and transition questions using stable IDs and the typed fields required by the published contract. Removal SHALL be refused when a record is referenced or when it would erase historical evidence.

#### Scenario: Referenced platform removal
- **WHEN** a user tries to remove a platform referenced by a question
- **THEN** the operation fails with a conflict and lists the referencing question IDs

#### Scenario: Add a numeric question
- **WHEN** a user supplies a unique ID, exact unit, resolution criteria, forecast window, and expected resolution time for a numeric question
- **THEN** the system appends a valid numeric question without changing existing questions

### Requirement: Append public forecasts and updates
The system SHALL append public forecast records with stable forecast IDs and typed quantitative values. A revised forecast SHALL be a new record with `supersedes_forecast_id`; the system MUST NOT overwrite or delete an earlier forecast statement.

#### Scenario: Update a forecast
- **WHEN** a user records a changed probability and identifies the prior forecast
- **THEN** a new forecast is appended with a unique ID and a valid supersession link while the prior record remains unchanged

#### Scenario: Attempt to reuse a forecast ID
- **WHEN** a new forecast uses an existing forecast ID
- **THEN** the operation fails without changing the ledger

### Requirement: Resolve questions with evidence
The system SHALL record resolved, annulled, or disputed outcomes only when the resulting status, typed outcome, `outcome_known_at`, `recorded_at`, and evidence sources satisfy the v1 contract. It SHALL keep derived scores and rankings out of the ledger.

#### Scenario: Resolve a binary question
- **WHEN** a user supplies a boolean outcome, valid chronology, and at least one evidence source
- **THEN** the question becomes resolved and retains all forecasts unchanged

#### Scenario: Resolution without evidence
- **WHEN** a user attempts to mark a question resolved without an evidence source
- **THEN** the operation fails validation and leaves the question unchanged

### Requirement: Inspect without mutating
The system SHALL provide ledger, platform, question, and forecast list/show/status views with stable selectors and both human and JSON output. Inspection MUST NOT perform network verification or change integrity metadata.

#### Scenario: Inspect a forecast
- **WHEN** a user supplies `--file`, `--question-id`, and `--forecast-id`
- **THEN** the system returns that exact forecast and a derived summary without changing the ledger or contacting a remote service

