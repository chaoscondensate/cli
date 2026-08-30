# CLI Flag Authoring Specification

## Purpose

Defines a complete, usable command-line authoring interface for non-secret ledger data without requiring users to prepare JSON or YAML fragments.

## Requirements

### Requirement: Every non-secret authoring field has a CLI representation
Every `forecast-ledger` command that creates or changes ledger data SHALL provide flags or a dedicated subcommand for every non-secret field accepted by that operation's typed application request. The direct CLI route SHALL NOT require `--input`, an input document, shell-generated JSON/YAML, or manual editing of the ledger. This requirement SHALL cover `init`, `ledger update`, `platform add|update`, `question add|update|resolve|annul|dispute`, `forecast add`, non-secret metadata for `forecast seal|reveal`, and `forecast key-hint update`, plus any future authoring command.

#### Scenario: Add a minimal platform from flags
- **WHEN** a user runs `forecast-ledger platform add --file l.yaml --platform metaculus --name Metaculus --kind scoring_platform` against a valid ledger
- **THEN** the command appends a valid platform with ID `metaculus` and does not report `Required flag "input" not set`

#### Scenario: Complete a rich mutation from flags
- **WHEN** a user supplies every applicable non-secret value for a mutation through its documented flags
- **THEN** the command constructs the transport-neutral typed application request and produces the validated ledger state

#### Scenario: Audit catches incomplete flag coverage
- **WHEN** a command input schema gains a non-secret authorable field without a corresponding flag or dedicated subcommand representation
- **THEN** the maintained command-surface test fails before release

### Requirement: Flag shapes preserve typed intent
Scalar fields SHALL use named long flags with types and validation consistent with the application request. Collection fields SHALL use repeatable flags whose order is preserved where order is meaningful. Nested public records SHALL use documented, repeatable field groups or dedicated child commands; they MUST NOT require inline JSON or YAML hidden inside a string flag. Patch commands SHALL distinguish omission from setting a zero value and SHALL provide explicit `--clear-<field>` operations for optional values or collections that can be removed. The CLI SHALL reject duplicate single-value flags, incomplete groups, mutually exclusive set/clear flags, and ambiguous combinations as usage errors before mutation.

#### Scenario: Repeatable values create a collection
- **WHEN** a user repeats a documented collection flag with valid values
- **THEN** the command preserves the values in documented order and validates the resulting typed collection

#### Scenario: Optional field is explicitly cleared
- **WHEN** a patch command receives `--clear-<field>` for a clearable optional field and no setter for that field
- **THEN** the field is removed through the normal source-preserving mutation path

#### Scenario: Nested data never disguises a document
- **WHEN** an operation authors a nested non-secret record
- **THEN** help exposes its individual fields or a dedicated child command and does not instruct the user to pass JSON/YAML text as one argument

### Requirement: Generic public input documents are unavailable
An authoring command MUST NOT expose generic `--input`, public file/stdin document decoding, or any batch/compatibility path that requires a caller-authored JSON/YAML document. Public authoring SHALL use ordinary leaf-local flags or dedicated subcommands only. Purpose-named protected secret channels remain allowed where raw private material cannot safely enter argv.

#### Scenario: Removed generic flag is rejected
- **WHEN** a user supplies `--input` to any command
- **THEN** argument parsing reports an unknown flag before reading the referenced path or changing any file

### Requirement: Required fields and type variants are explained at the command
Each authoring leaf command SHALL validate the fields required for its selected record type after parsing flags and SHALL return a stable usage error that names the missing flags. Help SHALL show type-specific requirements, valid enum values, repeatability, clear operations, defaults, and at least one copyable flag-only example. A group help page SHALL NOT make a leaf-only authoring flag appear required.

#### Scenario: Missing type-specific field is actionable
- **WHEN** a user selects a question or forecast type but omits a field required for that type
- **THEN** the command fails before mutation and names the missing command-line flag rather than asking for an input document

#### Scenario: Leaf help is sufficient to author data
- **WHEN** a user reads help for any authoring leaf command
- **THEN** the user can identify all required and optional flags and complete the operation without consulting an input-file schema

### Requirement: Secrets never move into argv
Private forecast values, encryption keys, salts, credentials, and other secret material SHALL NOT be accepted through command arguments or environment variables. Commands handling sealed or revealed data SHALL expose all non-secret metadata through flags but SHALL continue to read secret bundles from protected files or stdin and write keys only to protected files. Help and errors SHALL explain this exception without echoing secret content.

#### Scenario: Seal separates public flags from private input
- **WHEN** a user creates a sealed forecast
- **THEN** IDs and public metadata are accepted as flags while the private forecast bundle is read from a protected file or stdin and no secret appears in argv, logs, diagnostics, JSON, or normal stdout

#### Scenario: Secret-shaped flag is unavailable
- **WHEN** a user inspects completion or help for a command that consumes private material
- **THEN** no flag accepts the raw private value, key, salt, or credential

### Requirement: Direct adapters share application behavior
CLI flags and flattened MCP properties SHALL converge on the same transport-neutral typed application request before validation, locking, cryptography, or mutation. They SHALL produce the same stable success result, error codes, dry-run effects, source-preserving write behavior, and atomicity for equivalent values. Adapter parsing logic MUST NOT implement a second domain mutation path.

#### Scenario: Equivalent adapters have parity
- **WHEN** equivalent non-secret values are supplied once by CLI flags and once by direct MCP properties against equivalent ledgers
- **THEN** normalized results and resulting ledger data are semantically identical

#### Scenario: Invalid flag input has no side effect
- **WHEN** direct flags produce an invalid typed request
- **THEN** the command fails through the normal validation path before any ledger, key, target, receipt, or journal effect

### Requirement: Contributor policy prevents input-file-only regressions
Maintained contributor guidance SHALL state that new or changed CLI authoring commands MUST provide complete direct flags or dedicated subcommands for every non-secret field, MUST NOT add generic public document modes, and MUST include direct acceptance tests and documentation. Review and release checks SHALL inventory every authoring leaf command and fail when ordinary use requires JSON/YAML preparation.

#### Scenario: A future authoring command is reviewed
- **WHEN** a contributor adds or changes an authoring leaf command
- **THEN** the change includes direct-flag coverage, help and a copyable flag-only test or is rejected by the command-surface audit
