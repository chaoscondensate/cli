## Purpose

Defines the complete OpenTimestamps evidence lifecycle for exact forecast targets without requiring an external runtime or executable.

## ADDED Requirements

### Requirement: OpenTimestamps is the sole timestamp protocol
The v1 system SHALL create, parse, upgrade, and verify OpenTimestamps receipts and MUST NOT create or accept RFC 3161 or another timestamp protocol as equivalent evidence.

#### Scenario: Unsupported protocol request
- **WHEN** a user requests a timestamp protocol other than OpenTimestamps
- **THEN** the operation fails as unsupported before any file or network change

### Requirement: Stamp an exact target
Timestamp stamping SHALL first build or verify the selected immutable `forecast-envelope/v1` target, submit its digest to explicitly configured OpenTimestamps calendars with bounded timeouts, persist the detached receipt, and only then record matching pending integrity metadata. Partial failure MUST leave a recoverable target/receipt state and MUST NOT claim verified timing.

#### Scenario: Successful pending receipt
- **WHEN** a valid selected target receives an acceptable calendar response
- **THEN** the exact target and `.ots` receipt are stored and the forecast integrity state becomes pending with matching safe relative paths and digest

#### Scenario: Network failure before receipt
- **WHEN** no acceptable calendar response is obtained before timeout
- **THEN** the command reports a network failure, does not mark the forecast pending or verified, and preserves the valid target for retry

### Requirement: Upgrade and verify receipts
The system SHALL upgrade pending receipts, verify confirmed Bitcoin attestations against the exact target, and record the conservative `anchored_before` time and block height only after successful verification. It SHALL never use filesystem, archive, hosting, or source-control timestamps as cryptographic evidence.

#### Scenario: Confirmed receipt
- **WHEN** a pending receipt upgrades to a valid Bitcoin attestation for the exact target
- **THEN** the integrity metadata becomes verified and records the confirmed upper time bound and block height

#### Scenario: Receipt for different bytes
- **WHEN** a receipt is checked against target bytes whose digest does not match the ledger metadata
- **THEN** verification fails and the system does not update the ledger to verified

### Requirement: Report evidence state without overclaiming
Timestamp status SHALL distinguish missing, pending, confirmed-and-verified, failed, and locally inconsistent evidence. Pending SHALL be reported as not yet confirmed rather than as valid historical timing.

#### Scenario: Pending receipt
- **WHEN** the receipt is well formed but contains no confirmed Bitcoin attestation
- **THEN** status reports pending and a machine-readable not-ready result without describing existence timing as verified

### Requirement: Portable OpenTimestamps conformance
The release implementation SHALL operate inside the Go binary and SHALL pass differential and malformed-input conformance tests against the official OpenTimestamps client for all supported receipt operations. Unknown or unsupported operations and attestations SHALL be preserved safely or rejected explicitly, never ignored while claiming success.

#### Scenario: Official-client receipt
- **WHEN** the Go implementation reads a supported receipt produced by the official client
- **THEN** it reports the same digest operations and supported attestation state and can verify it against the same target

#### Scenario: Resource-exhaustion proof
- **WHEN** a receipt exceeds configured size, nesting, operation, or response limits
- **THEN** parsing fails safely without panic, unbounded allocation, or ledger mutation
