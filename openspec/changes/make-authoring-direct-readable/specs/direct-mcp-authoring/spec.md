## Purpose

Defines a direct, closed MCP authoring surface that exposes public values as typed tool properties while keeping private material behind purpose-named protected references.

## ADDED Requirements

### Requirement: Public MCP authoring fields are top-level properties
Every MCP tool that creates or changes ledger data SHALL expose each non-secret request field as a documented top-level property with the same type, requiredness, enum, collection ordering, null/clear semantics, and validation meaning as the shared application request. Tool schemas MUST NOT expose generic `input` or `input_file` properties and MUST remain closed to unknown fields.

#### Scenario: Add a platform directly
- **WHEN** `platform_add` receives `file`, `platform`, `name`, `kind`, and any optional account properties directly
- **THEN** it constructs and validates the shared platform-create request without a nested input object

#### Scenario: Removed wrapper is rejected
- **WHEN** any authoring tool receives an `input` or public `input_file` property
- **THEN** schema validation rejects the call before dispatch or filesystem access

### Requirement: Nested public values stay typed
Nested public records and collections SHALL use normal MCP object and array properties, not JSON/YAML strings and not side-loaded documents. Patch tools SHALL preserve the distinction between omission, explicit zero values, and supported clear operations. Generated schemas and maintained MCP reference material SHALL enumerate the complete direct property surface.

#### Scenario: Resolve with typed sources
- **WHEN** `question_resolve` receives an outcome and an array of typed evidence-source objects
- **THEN** source ordering and optional members are preserved in the shared resolution request without parsing document text

#### Scenario: Patch clears an optional field
- **WHEN** an MCP patch call uses its documented explicit clear representation
- **THEN** the service removes only that field and omission continues to preserve existing state

### Requirement: Secret references are purpose-named and confined
MCP tools SHALL NOT accept raw private forecast values, keys, salts, credentials, or other secret bundles in public properties. A tool that requires secret material SHALL expose only an operation-specific protected reference such as `secret_input_file` or `initial_secret_input_file`, plus applicable protected destination properties such as `key_file`. Every reference SHALL remain confined to the configured secret root and SHALL not appear in results, errors, logs, or resources as an absolute path.

#### Scenario: Seal resolves protected material
- **WHEN** `forecast_seal` receives public forecast metadata directly plus `secret_input_file` and `key_file`
- **THEN** the server resolves both inside configured roots, keeps secret contents out of protocol output, and uses the shared atomic seal service

#### Scenario: Inline secret is rejected
- **WHEN** a caller attempts to supply raw private value, key, salt, or credential fields
- **THEN** the closed schema rejects the unsupported properties without echoing their values

### Requirement: CLI and MCP direct requests remain behaviorally equivalent
Equivalent direct CLI and MCP authoring requests SHALL call the same transport-neutral service and produce equivalent validation, stable error codes, dry-run plans, mutations, cryptographic effects, and normalized ledger values. Adapter-specific parsing SHALL not duplicate domain rules.

#### Scenario: Equivalent direct question additions
- **WHEN** equivalent question data is supplied through CLI flags and flattened MCP properties against equivalent ledgers
- **THEN** both requests create semantically identical question records and report the same application operation and outcome

### Requirement: Contract audits prevent wrapper regression
Generated MCP schemas, registry metadata, manual tool contracts, dispatch tests, and maintained references SHALL be derived or checked against one direct-field inventory. Release checks SHALL fail if an authoring tool regains `input` or public `input_file`, omits a non-secret request field, or accepts an undocumented property.

#### Scenario: Generator adds a generic wrapper
- **WHEN** generated or handwritten MCP schema code exposes a forbidden generic wrapper
- **THEN** the contract audit fails before release and names the affected operation
