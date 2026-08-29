## Why

Forecast Ledger schema v1.1.0 makes empty ledgers and questions without forecasts valid, so the CLI and MCP server must stop imposing the older v1.0.0 minimum-item rules. The project has no compatibility obligation yet, which lets us adopt the new contract directly and make empty-state workflows coherent throughout the product.

## What Changes

- **BREAKING** Replace the embedded Forecast Ledger v1.0.0 contract and conformance data with the exact published v1.1.0 release; reject v1.0.0 and every other schema version.
- Let `forecast-ledger init` create a valid ledger without questions when no input document is supplied, while retaining optional input for root metadata, platforms, and an optional initial question.
- Let an initial question and `forecast-ledger question add` omit `initial_forecast`, producing a question with an empty `forecasts` array; retain the existing atomic public and sealed first-forecast paths when it is present.
- Define CLI, MCP, output, dry-run, validation, lifecycle, target, timestamp, verification, and publication behavior for zero questions and zero forecasts.
- Update embedded metadata, generated schemas, help, examples, compatibility statements, notices, tests, and maintained documentation to describe and prove the v1.1.0 behavior.
- Do not add migration, automatic conversion, dual-schema loading, or compatibility logic for old ledger files.

## Capabilities

### New Capabilities

- `forecast-ledger-v1-1-contract`: Exact schema v1.1.0 pinning, validation, conformance, and intentionally breaking version acceptance.
- `empty-ledger-workflows`: Authoring and operating on valid ledgers with no questions and questions with no forecasts across CLI and MCP.

### Modified Capabilities

None. There are no main specifications under `openspec/specs`; this change supplies new delta capabilities and supersedes conflicting v1.0.0 assumptions in active planning artifacts.

## Impact

- Embedded schema assets, typed service inputs, schema-version checks, init and question-add orchestration, CLI flags and presentation, MCP contracts and dispatch, and generated JSON Schemas change.
- Forecast selection, target creation, verification, lifecycle, and publication paths gain explicit empty-set behavior without weakening specific-ID errors or security rules.
- Conformance, service, adapter, generated-contract, documentation, and end-to-end tests gain v1.1.0 and empty-state coverage.
- README, getting-started and command documentation, compatibility and build references, contributor guidance, changelog, attribution, and documentation checks must move together with the contract pin.
