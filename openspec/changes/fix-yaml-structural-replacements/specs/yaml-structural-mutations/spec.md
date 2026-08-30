## Purpose

Defines reliable, source-preserving structural mutation of human-readable YAML ledgers without changing domain semantics or weakening atomic writes.

## ADDED Requirements

### Requirement: Valid YAML structural replacements succeed in context
Every valid add, replace, or remove requested by a ledger operation SHALL produce parseable YAML at the addressed location. Replacing a scalar, mapping, or sequence nested beneath a block mapping or block sequence MUST preserve the source collection context needed to indent every emitted line correctly. A valid mutation MUST NOT return `internal` solely because the ledger uses YAML rather than JSON.

#### Scenario: Replace a normalized scalar in a block record
- **WHEN** a question update replaces an existing status or other scalar in a valid block-style YAML question
- **THEN** the operation succeeds, the selected scalar has the requested value, and the resulting ledger passes local validation

#### Scenario: Replace a mapping beneath a sequence item
- **WHEN** annul, resolve, reveal, or timestamp recording replaces a populated mapping inside a question or forecast stored as a block sequence item
- **THEN** every replacement line remains at the correct nesting level and the resulting YAML parses as the intended ledger structure

#### Scenario: Replace a populated sequence
- **WHEN** an allowed operation replaces an existing populated YAML sequence
- **THEN** the replacement remains attached to its original field and all sequence entries decode in their requested order

### Requirement: Structural mutation preserves local YAML presentation
Application-authored populated mappings and sequences SHALL use expanded block style with two-space nesting and stable schema-oriented field order after structural replacement. Genuinely empty mappings and sequences MAY remain explicit as `{}` and `[]`. The mutation SHALL byte-preserve unrelated comments, quoting, key order, blank lines, line endings, and records outside the minimum addressed subtree and required parent syntax.

#### Scenario: Replace beside unrelated source text
- **WHEN** a structural replacement targets one question in a CRLF YAML ledger containing comments, quoted scalars, and another untouched question
- **THEN** the replacement uses valid block style while those unrelated source bytes and CRLF line endings remain unchanged

#### Scenario: Flow input is replaced locally
- **WHEN** the addressed existing value uses YAML flow style and the replacement is populated
- **THEN** only the addressed subtree and minimum parent syntax are converted to expanded block style and unrelated flow-style values remain unchanged

#### Scenario: Empty replacement keeps its type
- **WHEN** a mutation validly replaces a collection with an empty mapping or sequence
- **THEN** the YAML contains explicit `{}` or `[]` at the addressed field and validation does not interpret it as null

### Requirement: YAML and JSON mutations have equivalent domain outcomes
For equivalent valid ledgers and requests, YAML and JSON mutation paths SHALL produce semantically equivalent ledger models, stable application results, cryptographic targets, and side-effect plans. Formatting differences MUST NOT change validation, lifecycle, reveal, timestamp, or publication behavior. Existing transaction rules SHALL keep either format unchanged when prospective parsing or validation fails.

#### Scenario: Lifecycle mutation has format parity
- **WHEN** equivalent YAML and JSON ledgers receive the same valid question status, annul, resolve, or dispute request
- **THEN** both operations succeed with equivalent changed pointers, lifecycle state, and validation outcome

#### Scenario: Reveal has format parity without secret disclosure
- **WHEN** equivalent sealed forecasts in YAML and JSON are revealed with matching protected keys and approval
- **THEN** both ledgers record semantically equivalent revealed forecasts and no private value, key, or salt appears in diagnostics or unintended output

#### Scenario: Deterministic timestamp recording has format parity
- **WHEN** equivalent YAML and JSON forecasts record the same deterministic RFC 3161 fixture and retained trust material
- **THEN** both ledgers record equivalent integrity metadata and the canonical target remains byte-for-byte unchanged

#### Scenario: Failed prospective replacement is atomic
- **WHEN** a replacement cannot be parsed or fails post-mutation validation
- **THEN** the original ledger remains byte-identical, validates successfully, and no temporary sibling or partial ledger state is left behind

### Requirement: Release coverage includes every structural replacement class
Maintained regression coverage SHALL exercise scalar, mapping, and sequence replacements in block and flow collection contexts, including normalized ordered patch values. End-to-end coverage SHALL include question update and lifecycle mutation, platform update, forecast reveal, and deterministic timestamp recording on YAML, with JSON comparison where parity is material. Timestamp coverage MUST use retained deterministic fixtures and MUST NOT depend on a live provider.

#### Scenario: A replacement path loses collection context
- **WHEN** a future change makes any covered YAML replacement invalid, misindented, flow-formatted, semantically different from JSON, or non-atomic
- **THEN** document or end-to-end regression checks fail before release and identify the operation and source context

