## Purpose

Defines stable, secret-safe structured results and equivalent outcome classification for every public CLI and MCP operation.

## ADDED Requirements

### Requirement: Redaction preserves the public JSON contract
Structured output SHALL have the same field names, embedding behavior, omission behavior, scalar encodings, and collection shape as direct JSON serialization of the public result type. Redaction MUST recursively replace secret values without inventing Go field or type names, exposing ignored fields, restoring omitted fields, or bypassing a type's public JSON representation. Raw secret values and raw byte slices MUST NOT appear in normal CLI or MCP output.

#### Scenario: Embedded timestamp result remains flat
- **WHEN** timestamp verification serializes a result that embeds the common timestamp artifact fields
- **THEN** `question_id`, `forecast_id`, `state`, and the other embedded fields remain direct members of `data`, `verification` remains their sibling, and no `TimestampArtifactResult` member appears

#### Scenario: Omitted and string-tagged fields retain JSON semantics
- **WHEN** a result contains an empty value omitted by its JSON contract or a field with a JSON scalar-encoding option
- **THEN** redacted output omits or encodes that field exactly as direct JSON serialization would unless the field's value is replaced because its JSON key is secret

#### Scenario: Custom public JSON is still redacted
- **WHEN** a result type supplies a custom public JSON representation containing a nested secret-named member
- **THEN** the custom representation's shape is preserved and the nested secret value is replaced before output

### Requirement: One service outcome drives both adapters
The transport-neutral service layer SHALL classify each completed operation as successful, planned, unchanged, partially failed, pending, or failed and SHALL supply its stable outcome code, safe message, typed public data, and application category. CLI and MCP adapters MUST use that classification instead of maintaining independent outcome tables. Equivalent calls SHALL return the same outcome code and public data; transport envelopes, interactive approval representation, human/plain formatting, and CLI exit status MAY differ.

#### Scenario: Verified stamp uses one code
- **WHEN** equivalent CLI and MCP timestamp-stamp calls retain and locally verify the same deterministic RFC 3161 response
- **THEN** both structured results use `timestamp.verified` and equivalent typed public data

#### Scenario: Publication dry-run is a plan
- **WHEN** equivalent CLI and MCP publication-build calls use dry-run and no package files are written
- **THEN** both structured results use `publication.build.planned`, state that no files were written, and do not use `publication.built`

#### Scenario: Idempotent mutation is unchanged
- **WHEN** a mutation completes successfully with no changed ledger bytes, such as an already-applied metadata, platform, question, key-hint, or reveal update
- **THEN** both adapters return the operation's stable `*.unchanged` outcome instead of an `*.updated` or `*.revealed` claim

### Requirement: Safe reports survive non-zero outcomes
When an operation completes enough bounded checks to construct a safe typed report, a non-zero application category SHALL accompany that report rather than replace it. CLI JSON SHALL write the single structured result to stdout before returning the mapped non-zero exit, and MCP SHALL return the equivalent structured tool outcome as a recoverable tool error without terminating the session. Failures before any safe report exists SHALL continue to use the ordinary application error contract.

#### Scenario: Target mismatch remains inspectable
- **WHEN** target check completes an ordered selection containing one or more mismatched or missing referenced targets
- **THEN** CLI and MCP both return `target.failed` with the complete safe target report and verification category; CLI exits `6` and the MCP session remains usable

#### Scenario: Timestamp partial report remains inspectable
- **WHEN** timestamp stamp or verify produces a safe pending, unavailable, or cryptographic-failure report
- **THEN** both adapters preserve equivalent state, reason codes, public artifact identity, request summary, outcome code, and application category

### Requirement: Result parity is a maintained contract
Generated result references and regression tests SHALL cover every registered operation's success, dry-run where supported, unchanged where possible, and safe partial-failure outcomes. Adding or changing an operation result field, code, or message MUST update the shared classifier, generated references, CLI fixtures, MCP fixtures, and parity matrix together.

#### Scenario: Adapter-local result literal drifts
- **WHEN** an adapter introduces an outcome code or structured message that is not produced by the shared result contract
- **THEN** contract generation or parity tests fail before release
