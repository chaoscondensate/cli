## Why

The repository currently exposes only validation, status, version, and completion while the command surface that creates, maintains, seals, timestamps, verifies, packages, and serves forecast ledgers remains unavailable. The complete product needs one reviewable contract that defines every command's inputs, business rules, side effects, outputs, failure modes, and CLI/MCP parity before implementation resumes.

## What Changes

- Complete 30 command actions: ledger initialization and root-metadata update; platform, question, forecast, and key-hint management; target generation; seal and reveal; OpenTimestamps lifecycle operations; layered verification; evidence-package build and verification; and MCP stdio service.
- Define exact leaf-local flags, typed input-file schemas, selector rules, JSON result shapes, exit categories, dry-run behavior, confirmation rules, network boundaries, and cross-platform path behavior.
- Enforce the published Forecast Ledger v1.0.0 model: globally unique stable IDs, append-only forecast history, typed question/value/resolution agreement, lifecycle chronology, exact schema validation, and format-preserving transactions.
- Define the complete byte contract for `forecast-envelope/v1`, the pinned `forecast-seal/v1` canonical plaintext and AAD, and a new `forecast-key/v1` protected key file; implement deterministic targets, atomic sealing, authenticated reveal, repairable non-authoritative key hints, explicit non-claims, and byte-for-byte vector conformance.
- Implement the supported pure-Go OpenTimestamps subset with nonce-blinded calendar commitments, a versioned zero-config public calendar/Bitcoin-source profile, an explicit CLI-only custom-calendar escape hatch, bounded and deduplicated network observations, recoverable artifacts, `--offline`, optional Bitcoin Core verification, and honest pending/verified/failure reporting without silently persisting `failed` integrity.
- Implement layered verification and deterministic local evidence packages without requiring source control or hosted publication; package verification remains locally useful while ordinary ledger verification can use the same visible built-in network profile automatically.
- Start an MCP stdio server that exposes the same application operations through closed schemas and explicit ledger/output/secret roots, with optional read-only/offline modes, no general write/network grants, and one explicit default-off `--allow-reveal` boundary for irreversible disclosure.
- **BREAKING (preview surface)** Correct the currently registered hidden urfave scaffold before commands become available: real-file requirements replace invalid stdin support for artifact-dependent checks, timestamp verify becomes mutating/dry-runnable, `question add` keeps its required scalar `--type`, MCP roots become repeatable, and the existing reveal gate remains as the sole capability flag.
- Replace preview placeholders only when each CLI action and corresponding MCP tool passes its command-specific acceptance, rollback, redaction, conformance, documentation, and native-platform tests; incomplete MCP tools are absent from discovery.

## Capabilities

### New Capabilities

- `cli-command-execution`: Shared execution contract for every command, including flags, selectors, typed inputs, output envelopes, dry-run, confirmation, errors, and availability.
- `ledger-authoring-commands`: Business behavior for `init`, `ledger update`, all platform and question operations, and public forecast add/list/show operations.
- `forecast-cryptography-commands`: Business behavior for target build/check, atomic forecast seal/reveal, and safe key-hint repair operations.
- `timestamp-commands`: Business behavior for OpenTimestamps stamp, upgrade, status, and verify operations.
- `verification-commands`: Layered local verification behavior for `forecast-ledger verify`.
- `publication-commands`: Deterministic local evidence-package build and verification behavior.
- `mcp-command-runtime`: MCP stdio startup, roots, read-only/offline modes, built-in network-profile use, tool/resource surface, errors, redaction, and parity with CLI services.

### Modified Capabilities

None. The repository has no archived main capability specs yet. This change supersedes the entire active `build-forecast-ledger-cli-mcp` planning change, incorporates the usable foundations already implemented there, and replaces its conflicting/underspecified command contracts. The older change MUST NOT be synced or archived into main specs separately.

## Impact

- Connects the registered urfave command tree to shared application services and removes the `unavailable` preview behavior command by command.
- Changes hidden preview flags/help and their goldens before those actions are advertised; no available implemented command loses supported behavior.
- Extends application/domain, storage transaction, canonicalization, cryptography, OpenTimestamps, verification, publication, presentation, and MCP adapter packages.
- Adds protected secret-file handling, deterministic artifact naming/manifests, typed CLI input documents, versioned built-in and explicit custom network modes, bounded request budgets, conditional MCP reveal discovery, and native platform behavior.
- Adds upstream Python/Go validator parity, byte-level seal/target/key fixtures, official OpenTimestamps differential tests, CLI/MCP parity tests, crash/rollback tests, fuzzing, and release gates.
- Updates README, command reference, tutorials, security guidance, MCP setup, evidence limitations, and implementation status as each command becomes available.
