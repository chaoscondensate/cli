# Verification Outcome Semantics Specification

## Purpose

Define conservative, transport-neutral verification outcomes so no command or aggregate claims more than the evidence checks actually established.

## Requirements

### Requirement: Separate observation acquisition from proof comparison
The system SHALL treat obtaining a Bitcoin block observation and comparing an OpenTimestamps proof with that observation as separate steps. Failure to obtain a complete observation MUST NOT be reported as proof failure. A verification failure MAY be reported only after a complete accepted observation was obtained and the selected proof did not match it.

For a receipt with multiple Bitcoin attestations, verification SHALL pass if any supported attestation verifies. It SHALL report a cryptographic failure only if every candidate was checked against a complete observation and every comparison failed. If no candidate verifies and any candidate could not be observed, the result SHALL remain not checked rather than claiming that all proof branches failed.

#### Scenario: One required public source is unavailable
- **WHEN** `timestamp verify` has a well-formed bound receipt and one required source in the built-in Bitcoin profile times out, refuses the request, or returns an unavailable response
- **THEN** the timing check is `not_checked` with reason `timing.source_unavailable`, the application category is `network`, the CLI exits 8, and the ledger is unchanged

#### Scenario: Public sources cannot form one accepted observation
- **WHEN** the configured sources respond but their block observations disagree or cannot pass the observer's structural checks
- **THEN** the timing check is `not_checked` with a safe observation-inconclusive reason and MUST NOT claim that the receipt failed cryptographic verification

#### Scenario: Proof mismatches a complete observation
- **WHEN** a complete accepted block observation is obtained for every supported candidate but no evaluated attestation matches its observed Bitcoin commitment
- **THEN** the timing check is `fail` with reason `timing.bitcoin_mismatch`, the application category is `verification`, and the CLI exits 6

#### Scenario: Later proof branch verifies
- **WHEN** an earlier Bitcoin attestation cannot be observed or does not match but a later supported attestation verifies against a complete accepted observation
- **THEN** the timing check passes using the verified branch and does not emit an overall source-unavailable or mismatch failure

### Requirement: Preserve safe partial timestamp reports
Expected non-success outcomes after a timestamp receipt has been safely loaded and evaluated SHALL preserve a transport-neutral report. The report MUST include the selected question and forecast IDs, target and receipt identity, local receipt state, timing-check state and reason codes, Bitcoin height when known, network profile, bounded request summary, and safe affected source IDs when available. It MUST NOT include endpoint URLs, raw remote bodies, credentials, private forecast data, or unrestricted underlying error text.

CLI human, plain, and JSON modes SHALL write this primary report to stdout before returning its stable non-zero application exit. MCP SHALL return the same public fields as a recoverable tool outcome. Failures that occur before a safe report exists SHALL continue to use the ordinary application error contract.

#### Scenario: JSON source-outage report
- **WHEN** `forecast-ledger --json timestamp verify` cannot complete the built-in Bitcoin observation after safely evaluating a confirmed receipt
- **THEN** stdout contains one stable JSON result with `not_checked`, `timing.source_unavailable`, network-profile and request-summary data, stderr contains no duplicate JSON result, and the process exits 8

#### Scenario: Source identity is safe to disclose
- **WHEN** one named source in the built-in profile is unavailable
- **THEN** the report identifies it only by its stable public source ID and does not expose its URL or raw transport error

#### Scenario: MCP observation failure
- **WHEN** the MCP `timestamp_verify` tool encounters the same source outage
- **THEN** it returns a recoverable structured tool outcome with fields and classification equivalent to the CLI result and keeps the MCP session alive

### Requirement: Reserve pass for applicable evidence
An aggregate verification result SHALL be `pass` only when at least one forecast-evidence layer in the selected scope is applicable, every applicable layer completed, and none is pending, not checked, or failed. Document validity, manifest integrity, and package-file digest checks SHALL remain independently visible but SHALL NOT by themselves make an evidence aggregate pass.

When the selected scope contains no forecasts or every forecast-evidence layer is `not_applicable`, the aggregate SHALL be `no_evidence`. This state SHALL use application category `incomplete` and CLI exit 9, while preserving all independently established document, manifest, and file observations. The state does not claim failure, corruption, or missing promised evidence.

Aggregate precedence SHALL be `fail`, then `incomplete` caused by unavailable or unperformed applicable checks, then `pending`, then `no_evidence`, then `pass`. A layer that passes SHALL count as applicable; `not_applicable` SHALL not.

#### Scenario: Empty ledger verification
- **WHEN** layered verification selects a valid ledger containing no forecasts
- **THEN** it performs no network requests, reports the document result and `overall no_evidence`, and exits 9

#### Scenario: Empty question selection
- **WHEN** layered verification selects a valid question containing no forecasts
- **THEN** it reports `overall no_evidence` rather than `pass` and exits 9

#### Scenario: Forecast with no applicable evidence
- **WHEN** every reported forecast-evidence layer is `not_applicable`
- **THEN** layered or package verification reports `overall no_evidence` and exits 9

#### Scenario: Package bytes pass but evidence does not apply
- **WHEN** a package manifest, listed files, and packaged ledger pass their integrity checks but the selected ledger has no applicable forecast-evidence layer
- **THEN** package verification preserves those passing package observations, reports `overall no_evidence`, and does not describe the forecast evidence as verified

#### Scenario: At least one applicable layer passes
- **WHEN** at least one forecast-evidence layer is applicable and passes, all other applicable layers pass, and remaining layers are `not_applicable`
- **THEN** the aggregate is `pass` and exits 0

### Requirement: Keep adapters and public guidance aligned
CLI and MCP adapters SHALL derive outcome names, reason codes, result fields, and application categories from the same service results. Human, plain, JSON, generated schemas, help, and maintained verification, timestamp, publication, security, MCP, error/exit, and documentation-baseline material SHALL describe `no_evidence` and observation failures consistently.

Regression tests MUST use deterministic local observers and HTTP fixtures; release verification MUST NOT depend on current availability of public Bitcoin services.

#### Scenario: CLI and MCP parity fixture
- **WHEN** the same deterministic observer fixture is exercised through CLI and MCP
- **THEN** both adapters expose equivalent state, reason, safe source identity, network profile, request counts, and application category

#### Scenario: Public-source outage regression
- **WHEN** the v0.3.0 one-source-outage scenario is replayed against a local fixture
- **THEN** tests prove that it cannot regress to `verification`, exit 6, or the message `Bitcoin evidence did not verify`
