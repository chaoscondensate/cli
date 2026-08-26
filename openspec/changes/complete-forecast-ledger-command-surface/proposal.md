## Why

The repository currently exposes only validation, status, version, and completion while the command surface that creates, maintains, seals, timestamps, verifies, packages, and serves forecast ledgers remains unavailable. The complete product needs one reviewable contract that defines every command's inputs, business rules, side effects, outputs, failure modes, and CLI/MCP parity before implementation resumes.

## What Changes

- Complete all 28 planned command actions: ledger initialization; platform, question, and forecast management; target generation; seal and reveal; OpenTimestamps lifecycle operations; layered verification; evidence-package build and verification; and MCP stdio service.
- Define exact leaf-local flags, typed input-file schemas, selector rules, JSON result shapes, exit categories, dry-run behavior, confirmation rules, network boundaries, and cross-platform path behavior.
- Enforce the published Forecast Ledger v1.0.0 model: globally unique stable IDs, append-only forecast history, typed question/value/resolution agreement, lifecycle chronology, exact schema validation, and format-preserving transactions.
- Implement deterministic `forecast-envelope/v1` targets, atomic `forecast-seal/v1` sealing, authenticated reveal, protected key files, and byte-for-byte upstream vector conformance.
- Implement the supported pure-Go OpenTimestamps subset with explicit calendars and Bitcoin verification sources, bounded network behavior, recoverable artifacts, and honest pending/verified/failed states.
- Implement layered verification and deterministic local evidence packages without requiring source control, hosted publication, or implicit network access.
- Start an MCP stdio server that exposes the same application operations through closed schemas, explicit roots, read-only defaults, and separate write, network, and reveal grants.
- Replace preview placeholders only when each command passes its command-specific acceptance, rollback, redaction, conformance, and native-platform tests.

## Capabilities

### New Capabilities

- `cli-command-execution`: Shared execution contract for every command, including flags, selectors, typed inputs, output envelopes, dry-run, confirmation, errors, and availability.
- `ledger-authoring-commands`: Business behavior for `init`, all platform and question operations, and public forecast add/list/show operations.
- `forecast-cryptography-commands`: Business behavior for target build/check and atomic forecast seal/reveal operations.
- `timestamp-commands`: Business behavior for OpenTimestamps stamp, upgrade, status, and verify operations.
- `verification-commands`: Layered local verification behavior for `forecast-ledger verify`.
- `publication-commands`: Deterministic local evidence-package build and verification behavior.
- `mcp-command-runtime`: MCP stdio startup, roots, grants, tool/resource surface, errors, redaction, and parity with CLI services.

### Modified Capabilities

None. The repository has no archived main capability specs yet; this change adds detailed implementation contracts while preserving the accepted v1 direction from the active foundational change.

## Impact

- Connects the registered urfave command tree to shared application services and removes the `unavailable` preview behavior command by command.
- Extends application/domain, storage transaction, canonicalization, cryptography, OpenTimestamps, verification, publication, presentation, and MCP adapter packages.
- Adds protected secret-file handling, deterministic artifact naming/manifests, typed CLI input documents, MCP schemas, and native platform behavior.
- Adds upstream Python/Go validator parity, crypto and target vector conformance, official OpenTimestamps differential tests, CLI/MCP parity tests, crash/rollback tests, fuzzing, and release gates.
- Updates README, command reference, tutorials, security guidance, MCP setup, evidence limitations, and implementation status as each command becomes available.
