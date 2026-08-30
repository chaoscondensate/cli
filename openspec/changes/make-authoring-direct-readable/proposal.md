## Why

Forecast Ledger authoring still exposes generic JSON/YAML input-document paths even though every ordinary public value should be authorable directly. The same surface also requires exact RFC 3339 for planned calendar dates and can emit newly inserted YAML records as dense flow-style values, making common commands harder to use and resulting ledgers harder for people to read. The application must also move its exact embedded contract from Forecast Ledger v1.2.0 to the newly revised v1.3.0 without weakening immutable provenance or conformance.

## What Changes

- **BREAKING** Remove the generic CLI `--input` flag, stdin document mode, and public input-file batch/compatibility mode from every command. Ordinary public data is accepted only through leaf-local CLI flags or dedicated subcommands.
- **BREAKING** Remove generic MCP `input` and `input_file` wrappers. Each tool exposes its public request fields directly as closed, typed top-level properties and rejects the removed wrappers.
- Preserve protected, purpose-named secret channels required by the security boundary. Private forecast values, keys, salts, and credentials remain forbidden in argv and public MCP fields; applicable commands continue to use explicit protected secret-file/stdin references and protected key destinations rather than a generic authoring document.
- **BREAKING** Replace the exact embedded Forecast Ledger v1.2.0 contract with v1.3.0 and accept only v1.3.0 ledgers. Pin the exact upstream commit, release archive digest, schema digest, attribution, compatibility decision, conformance fixtures, and cryptographic vectors together; never fetch a floating tag at build or runtime.
- Add deterministic, non-LLM parsing for a documented allowlist of human calendar-date and timestamp forms on applicable direct CLI time flags. Normalize accepted values to the exact RFC 3339 representation required by Forecast Ledger v1.3.0 before validation or persistence.
- Keep ambiguous numeric dates, relative phrases, locale guessing, system-local timezone inference, and fuzzy typo recovery unsupported. Use the ledger `default_timezone`, or the explicit init timezone, whenever an accepted form omits an offset.
- Make `forecasted_at` optional for public, sealed, and initial forecast creation through both CLI and MCP. When omitted, capture the operation clock once and store that instant with the ledger default-timezone offset, without bypassing forecast-window chronology.
- Require application-authored YAML mappings and non-empty sequences to use readable block style with stable indentation and field order. Preserve unrelated source text and permit `[]` or `{}` only for genuinely empty collections.
- Update active OpenSpec artifacts, contributor guidance, generated schemas, command help, examples, dogfood scripts, documentation, tests, and release audits so current material cannot reintroduce generic input documents or flow-style generated YAML.

## Capabilities

### New Capabilities

- `direct-mcp-authoring`: Closed MCP tools with public authoring fields exposed directly and purpose-named protected references only where secrets require them.
- `friendly-date-authoring`: Deterministic accepted date/time forms, timezone and field-default rules including omitted `forecasted_at`, RFC 3339 normalization, diagnostics, and rejection of ambiguous or relative input.
- `readable-yaml-output`: Human-readable block-style YAML generation and source-preserving local formatting rules for created and mutated ledgers.

### Modified Capabilities

- `cli-flag-authoring`: Replace the optional document mode with an exclusive direct-flag interface for all non-secret authoring and retain only purpose-named protected secret channels.
- `empty-ledger-workflows`: Flatten CLI and MCP creation inputs, remove generic document/file input alternatives, and preserve empty-ledger and backlog-question behavior through direct fields.
- `forecast-ledger-v1-2-contract`: Supersede the version-specific v1.2.0 requirements with the exact immutable v1.3.0 contract, exclusive admission, conformance corpus, and public identity, then rename the maintained capability/title to v1.3.0 during synchronization.

## Impact

- CLI command definitions, authoring-flag builders, completion, help goldens, direct-authoring inventory, tests, fixtures, and dogfood workflows.
- Transport-neutral operation metadata and input-schema use, MCP tool schema generation, tool contracts, dispatch, generated reference schemas, parity tests, and protected-root handling.
- Vendored schema bytes, conformance fixtures, schema/build metadata, version output, MCP initialization, publication manifests, generated examples, compatibility diagnostics, attribution, and exact digest pins.
- Timestamp input normalization around question scheduling and other applicable time flags while retaining the embedded v1.3.0 timestamp contract and strict stored validation unchanged.
- JSON/YAML document patch rendering, insertion style, initialization, formatting fixtures, source-preservation tests, and native-platform write checks.
- `AGENTS.md`, README, getting-started, how-to, reference, security, development baseline, active overlapping OpenSpec artifacts, and release checks. Archived and superseded planning records remain historical and are not runtime or current guidance.
