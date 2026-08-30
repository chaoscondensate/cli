## Why

`forecast-ledger` 0.6.0 cannot replace an existing nested mapping or sequence in a YAML ledger: the generated replacement loses its source indentation, reparsing fails, and the operation returns `internal`. This blocks question lifecycle changes, annulment, sealed forecast reveal, timestamp recording, and other core workflows on the documented default format even though equivalent JSON operations succeed.

## What Changes

- Make structural YAML replacement preserve the addressed node's indentation and collection context while continuing to emit populated application-authored values in expanded block style.
- Require add, replace, and remove patches to leave unrelated YAML bytes unchanged and to produce a parseable, valid prospective ledger before commit.
- Restore YAML/JSON behavioral parity for every service mutation that replaces an existing structure, including question and platform mutation, forecast reveal, and deterministic timestamp recording.
- Add block-to-block and collection-context regression coverage at document, service, CLI, and MCP-facing shared-service levels; retain atomic rollback and stable error handling for genuine failures.
- Review current user and release documentation, add a regression/release note where maintained, and keep examples on `ledger.yaml` only after the end-to-end replacement matrix passes.

## Capabilities

### New Capabilities

- `yaml-structural-mutations`: Source-preserving, block-style-safe add, replace, and remove behavior for nested YAML mappings and sequences, including parity and atomicity across ledger mutation workflows.

### Modified Capabilities

None.

## Impact

- Affects `internal/document` structural patch rendering and its unit/fuzz coverage.
- Affects shared service mutation paths for questions, platforms, reveal, and RFC 3161 timestamp state, plus CLI/MCP parity and transaction tests.
- Does not change the Forecast Ledger v1.3.0 schema, command flags, MCP request properties, JSON serialization, canonical cryptographic targets, or public error-code contract.
- Extends the readable YAML work planned by `make-authoring-direct-readable`; that completed but unarchived change remains the source of the general block-style policy, while this change specifies the missing structural-mutation correctness contract.
