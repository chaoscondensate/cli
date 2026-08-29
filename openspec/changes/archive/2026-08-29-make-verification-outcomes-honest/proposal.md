## Why

The v0.3.0 dogfooding run showed that verification summaries can make stronger claims than the checks support: `timestamp verify` reports a broken proof when a required Bitcoin source is unavailable, while layered verification classifies the same event as not checked, and a package with no applicable forecast-evidence layer reports `pass`. These outcomes can mislead both people and automation at the exact trust boundary the CLI is meant to make explicit.

## What Changes

- Introduce one shared outcome policy for timestamp, layered-ledger, and package verification so aggregate states and exit categories cannot be stronger than their underlying observations.
- Distinguish inability to obtain a Bitcoin observation from failure of a proof against a successfully obtained observation. Source unavailability returns a safe, structured not-checked result and the network exit category; a cryptographic mismatch after successful observation remains a verification failure.
- Return the safely available timestamp result, built-in network profile, bounded request summary, reason code, height, and safe affected source IDs when online observation cannot complete, without mutating the ledger or exposing endpoint or credential data.
- Add an explicit aggregate state for a valid selection or package with no applicable forecast-evidence checks. It is not reported as `pass`; it uses a non-success evidence exit category while retaining independently established document, manifest, and file-integrity observations.
- Apply the same no-evidence rule to empty ledgers, question-only selections, forecasts whose evidence layers are all `not_applicable`, and portable packages containing those forms.
- **BREAKING (preview):** source outages move from `verification`/exit 6 to `network`/exit 8 for `timestamp verify`, and no-evidence reports move from `pass`/exit 0 to the new aggregate state with the existing incomplete/pending exit family.
- Update CLI/MCP result schemas, human/plain/JSON presentation, security and verification guidance, generated reference material, and v0.3.0 dogfooding regression coverage.

## Capabilities

### New Capabilities

- `verification-outcome-semantics`: Cross-command rules for classifying observation failures, aggregating applicable evidence, preserving partial reports, and mapping results to stable CLI/MCP outputs and exit categories.

### Modified Capabilities

None. The command capabilities are still defined by the active `complete-forecast-ledger-command-surface` change and have not been archived into main specs; this change adds the cross-command outcome contract that their implementations must follow.

## Impact

- Affects `internal/timestamp/ots`, `internal/service` timestamp and verification orchestration, presentation, CLI and MCP adapters, generated public result contracts, and tests.
- Changes externally observable result states and exit codes for unavailable Bitcoin observations and selections with no applicable forecast evidence; ledger, target, receipt, and package bytes do not change.
- Requires documentation impact review across timestamp, verification, publication, MCP, security, error/exit, generated reference, and documentation-baseline material.
- Adds no dependency, network source, protocol, schema-version migration, hosted publication behavior, or broader evidence claim.
