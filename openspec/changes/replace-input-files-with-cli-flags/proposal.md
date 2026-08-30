## Why

The CLI currently requires users to author a separate syntactically valid JSON or YAML fragment for many ordinary mutations, so basic workflows such as `platform add` cannot be completed from command-line arguments. This makes the shipped command surface impractical and contradicts the expectation that a CLI can author every non-secret ledger value directly.

## What Changes

- Make direct flags the complete, documented authoring interface for every non-secret field accepted by `init` and every ledger mutation command; a user can complete each normal workflow without creating an input document.
- Make `--input` optional wherever retained as an advanced batch-input convenience, define flag mode and input-document mode as mutually exclusive, and reject mixed sources instead of silently choosing a value.
- Add repeated flags and explicit clear/unset operations where ledger fields are collections, nested values, or optional patch fields, while preserving stable IDs and the v1.2.0 data contract.
- Keep secrets out of argv: sealed plaintext, keys, salts, and credentials continue to use protected files or stdin even though their non-secret metadata is available through flags.
- Make `forecast-ledger init` create a valid empty ledger from scalar flags alone, without a template or input file.
- Improve human `forecast-ledger version` output with readable labels and restrained terminal color; `--json`, non-TTY output, `NO_COLOR`, `TERM=dumb`, and `--no-color` remain stable and undecorated.
- Add a durable contributor rule and command-surface audit so future CLI commands cannot make a structured input file mandatory when the same non-secret data can be expressed safely as flags.
- Update CLI help, generated schemas where applicable, README and command documentation so every authoring command has a copyable flag-only example and optional input-file behavior is clearly secondary.

## Capabilities

### New Capabilities

- `cli-flag-authoring`: Complete flag-based creation and mutation of all non-secret Forecast Ledger data, including collection/nested-field encoding, input precedence, conflict handling, and secret boundaries.
- `cli-version-presentation`: Readable, optionally colored human version information with stable plain and JSON behavior.

### Modified Capabilities

- `empty-ledger-workflows`: Remove the remaining requirement for question input documents and require flag-only init and question creation workflows.

## Impact

This affects the urfave CLI command definitions and help, transport-neutral request construction, mutation input validation, presentation/color policy, CLI acceptance tests, generated command schemas or fixtures, contributor guidance, README, getting-started and command reference material. Shared application services, MCP request contracts, the embedded v1.2.0 schema, source-preserving storage, stable error codes, and secret-handling guarantees remain authoritative; MCP does not need to imitate CLI flags.
