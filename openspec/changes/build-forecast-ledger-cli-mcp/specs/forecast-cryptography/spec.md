## Purpose

Defines deterministic timestamp targets and interoperable sealed-forecast creation and reveal behavior for Forecast Ledger v1.

## ADDED Requirements

### Requirement: Build canonical forecast targets
The system SHALL build the exact RFC 8785 JCS `forecast-envelope/v1` target for a selected forecast, using the published field profile, I-JSON restrictions, UTF-16 property ordering, and no floating-point values. Integrity metadata SHALL be excluded to avoid recursion, and `key_hint` SHALL be excluded so key location can rotate without changing the target.

#### Scenario: Build a public target
- **WHEN** a user selects a valid public forecast by question and forecast ID
- **THEN** the system writes deterministic canonical bytes and reports their SHA-256 digest and safe relative artifact path

#### Scenario: Rebuild on another platform
- **WHEN** the same ledger and forecast are processed on macOS, Linux, and Windows
- **THEN** every platform produces byte-identical target content and the same SHA-256 digest

### Requirement: Atomic sealed forecast creation
`chaos forecast seal` SHALL accept the private bundle through a secret-safe channel, generate a fresh 32-byte salt, 32-byte key, and 12-byte nonce from the operating-system CSPRNG, construct the published `forecast-seal/v1` plaintext and associated data, calculate the SHA-256 commitment, and encrypt with ChaCha20-Poly1305. It SHALL write the key to an explicit protected destination before appending a sealed record that contains no plaintext value, rationale, key factors, or comment.

#### Scenario: Successful seal
- **WHEN** a user supplies a valid private forecast bundle and a new protected key-file path
- **THEN** the key is stored with restrictive access, the sealed record is appended atomically, and output reports identifiers and safe next steps but no secret material

#### Scenario: Key storage fails
- **WHEN** the generated key cannot be safely written or the destination already exists
- **THEN** the command fails and the ledger remains unchanged

#### Scenario: Attempt to hide a public forecast
- **WHEN** a user asks to seal or hide a forecast that was already stored in plaintext
- **THEN** the system refuses because publication cannot be undone and suggests creating a new sealed forecast if appropriate

### Requirement: Verified reveal
`chaos forecast reveal` SHALL accept the key only through a secret-safe channel and MUST authenticate and decrypt the original ciphertext, verify the commitment and bound IDs, validate the canonical plaintext, and derive the public mirror before changing the ledger. Reveal SHALL retain the original commitment, nonce, ciphertext, and timestamp target relationship.

#### Scenario: Correct reveal key
- **WHEN** the supplied key opens the selected sealed forecast and every binding check succeeds
- **THEN** the record becomes `revealed`, stores the disclosed key and exact decrypted mirror required by v1, and retains the original seal evidence

#### Scenario: Incorrect reveal key
- **WHEN** AEAD authentication or any commitment, ID, or mirror check fails
- **THEN** reveal reports a cryptographic failure and leaves the ledger byte-for-byte unchanged

### Requirement: Published cryptographic conformance
The implementation SHALL reproduce the published `forecast-seal/v1` positive test vector byte-for-byte and SHALL reject its negative mutations. Any protocol or canonicalization change MUST use a new protocol identifier rather than silently changing v1 behavior.

#### Scenario: Official deterministic vector
- **WHEN** the implementation receives the published fixed bundle, salt, key, and nonce
- **THEN** it produces the exact published canonical plaintext, commitment, associated data, and ciphertext

### Requirement: Selective and explicit target generation
Target generation SHALL require stable question and forecast IDs unless the user explicitly passes `--all`. Existing target files MUST NOT be overwritten when bytes differ without an explicit replacement flow and a clear warning that existing receipts may become invalid.

#### Scenario: Target collision
- **WHEN** a target path already exists with bytes different from a newly generated target
- **THEN** the command fails with a conflict and preserves both the ledger and existing target
