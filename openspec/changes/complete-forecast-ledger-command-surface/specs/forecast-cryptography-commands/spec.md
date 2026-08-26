## Purpose

Defines deterministic target construction and the complete secret-safe sealed forecast lifecycle, including artifact collision behavior, protected key handling, and authenticated reveal.

## ADDED Requirements

### Requirement: Construct the exact forecast target projection
The system SHALL construct `forecast-envelope/v1` from the selected question and forecast using the published typed allowlist and bounded RFC 8785/JCS profile. It SHALL reject floats and values outside I-JSON limits, sort object properties by UTF-16 code units, emit UTF-8 canonical bytes without insignificant whitespace, and calculate SHA-256 over those exact bytes. Integrity metadata and `key_hint` MUST be excluded.

A public forecast target SHALL bind the public value and public explanatory fields. A sealed forecast target SHALL bind the sealed commitment and encryption fields while excluding private plaintext. A revealed forecast SHALL continue to rebuild the original sealed-envelope target, not a new public target, so reveal cannot change the bytes covered by an existing timestamp.

#### Scenario: Cross-platform target identity
- **WHEN** the same forecast is targeted on macOS, Linux, and Windows
- **THEN** canonical bytes and the lowercase SHA-256 digest are byte-for-byte identical

#### Scenario: Revealed target continuity
- **WHEN** a previously targeted sealed forecast is revealed correctly
- **THEN** rebuilding its target produces the exact original sealed target bytes and digest

### Requirement: Build target artifacts deterministically
`forecast-ledger target build` SHALL require `--all` or one question/forecast pair and a real ledger path. The default artifact path SHALL be `proofs/targets/<forecast-id>.json` relative to the ledger directory; forecast IDs are globally unique. The operation SHALL validate the ledger and selected forecast before writing, confine the resolved path to the ledger artifact root, and write through exclusive creation or recoverable replacement.

If the path is absent, build SHALL write the canonical bytes. If it contains identical bytes, build SHALL be idempotent and report `unchanged`. If it differs, build MUST return `conflict` without overwrite. `--all` SHALL precompute every selected path and byte sequence and reject any collision before creating the first artifact.

Target build SHALL NOT change ledger integrity state by itself; `timestamp stamp` records the target metadata together with a retained receipt.

#### Scenario: Build one target
- **WHEN** the selected target does not exist
- **THEN** canonical bytes are written at the deterministic safe path and output includes question ID, forecast ID, relative path, size, and SHA-256

#### Scenario: Different existing target
- **WHEN** the deterministic path already contains different bytes
- **THEN** build returns `conflict`, preserves the existing file and ledger, and reports both expected and actual digests

#### Scenario: All-target preflight failure
- **WHEN** one path in an `--all` build would collide
- **THEN** no target file is created or replaced for any forecast

### Requirement: Check target artifacts without mutation
`forecast-ledger target check` SHALL reconstruct each selected target in memory, read the deterministic artifact, and compare both bytes and SHA-256. When ledger integrity contains target metadata, it SHALL also require exact scope `forecast-envelope/v1`, canonicalization identifier, relative path, algorithm, and digest agreement. It MUST NOT repair, rewrite, or update either ledger or artifact.

#### Scenario: Target bytes changed with same ledger
- **WHEN** an artifact exists but differs from the reconstructed canonical bytes
- **THEN** check returns verification failure with expected/actual digest evidence and performs no mutation

### Requirement: Accept sealed forecast input only through a private channel
`forecast-ledger forecast seal` SHALL require `--file`, `--question`, a globally unique new `--forecast`, `--input <protected-file|->`, and a new `--key-file` path. The private input SHALL contain the same typed value, forecast times, rationale, key factors, and comment fields used by a public forecast, plus optional public note and supersession ID. Those private fields MUST NOT be accepted as scalar argv flags or environment variables.

The selected question SHALL be open, the forecast time SHALL be inside its window, type/value/chronology rules SHALL match public forecast add, and a supersession link SHALL target an earlier forecast in the same question.

#### Scenario: Private plaintext supplied as flag
- **WHEN** a user attempts to pass a raw value, rationale, key, or salt flag to seal
- **THEN** argument parsing rejects it because no such secret-bearing option exists

### Requirement: Seal with the published cryptographic profile
Seal SHALL generate a fresh 32-byte salt, 32-byte key, and 12-byte nonce from the operating-system CSPRNG. It SHALL create the exact published `forecast-seal/v1` canonical plaintext bundle and associated data bound to ledger, question, forecast, and protocol identifiers; compute the required SHA-256 commitment; and encrypt with ChaCha20-Poly1305. It MUST reproduce the pinned upstream vector exactly when deterministic test entropy is injected by conformance tests.

The appended forecast SHALL have visibility `sealed`, integrity `unanchored`, optional safe public note and supersession link, and a commitment containing the published scheme, SHA-256 commitment, `chacha20-poly1305` nonce/ciphertext, and a safe relative key hint. It MUST NOT contain plaintext value, rationale, key factors, or comment.

#### Scenario: Successful seal
- **WHEN** private input is valid and secure key storage succeeds
- **THEN** one sealed forecast is appended and normal, JSON, verbose, and error output contain no salt, key, nonce value, private bundle, or ciphertext plaintext

#### Scenario: Entropy failure
- **WHEN** the operating-system random source cannot return every required byte
- **THEN** seal returns an internal or I/O failure before creating a key file or ledger mutation

### Requirement: Protect key files before ledger publication
The key destination SHALL be explicit, new, outside package-output roots, and not a symlink/junction/reparse-point escape. POSIX creation SHALL use owner-only mode `0600`; Windows creation SHALL apply an owner-only ACL and reject a destination whose protection cannot be established. The file SHALL contain only the documented key-file format and SHALL be flushed before ledger commit.

Seal SHALL write and secure the key first, then commit the ledger. If key creation fails, the ledger remains unchanged. If ledger commit fails after the key is durable, the command SHALL preserve the key, report its safe display path and recovery action without revealing it, and MUST NOT silently delete the only copy.

#### Scenario: Existing key destination
- **WHEN** `--key-file` already exists
- **THEN** seal returns `conflict` without reading, truncating, chmodding, or replacing that entry

#### Scenario: Ledger commit fails after key write
- **WHEN** the key is durable but post-validation or safe replacement fails
- **THEN** the original ledger remains unchanged and output identifies a retained orphan key file using a redacted/safe path

### Requirement: Reveal only after complete authentication
`forecast-ledger forecast reveal` SHALL require approval, a selected sealed forecast, and an explicit protected `--key-file`. Before opening a ledger write transaction it SHALL read the bounded key file, authenticate/decrypt the original ciphertext, verify the commitment hash, associated data, ledger/question/forecast IDs, protocol identifiers, and canonical private bundle, and validate the derived public mirror against the question type and chronology.

Only after every check succeeds SHALL reveal set visibility to `revealed`, add `revealed_at` and the disclosed key required by v1, and derive value/rationale/key factors/comment from authenticated plaintext. It SHALL retain the original commitment hash, nonce, ciphertext, key hint, supersession link, integrity object, target relationship, and public note.

#### Scenario: Wrong key
- **WHEN** the supplied key fails AEAD authentication
- **THEN** reveal returns verification failure and the ledger remains byte-for-byte unchanged

#### Scenario: Bound ID mismatch
- **WHEN** decrypted plaintext authenticates but names a different ledger, question, or forecast
- **THEN** reveal fails before mutation and reports a binding failure without exposing plaintext

#### Scenario: Successful reveal of timestamped forecast
- **WHEN** every reveal check succeeds for a forecast with retained OTS evidence
- **THEN** disclosed mirror fields are added while the original target, receipt, and sealed target digest remain unchanged

### Requirement: Make reveal retry-safe
Revealing an already revealed forecast with a key that exactly matches the stored disclosed key and authenticated bundle SHALL be idempotent and return `unchanged`; a different key SHALL fail verification. A public forecast or a sealed forecast lacking a complete supported commitment SHALL return `conflict` or invalid data rather than being converted.

#### Scenario: Repeat correct reveal
- **WHEN** the same correct key is supplied after successful reveal
- **THEN** no bytes change and the command reports that the forecast is already revealed

### Requirement: Prove cryptographic interoperability and non-disclosure
Release gates SHALL include the pinned positive seal vector, negative mutations for every bound field, wrong-key and tampered-ciphertext cases, Go/Python cross-language seal/reveal fixtures, property tests, fuzzing of bounded key/ciphertext inputs, cross-platform target bytes, crash points across key/artifact/ledger writes, and canary assertions through implemented seal/reveal paths. No crypto command SHALL become visible while its vector, rollback, permission, and redaction gates are incomplete.

#### Scenario: Pinned vector regression
- **WHEN** canonicalization or crypto output differs by one byte from the published vector
- **THEN** conformance and release checks fail before a binary is published

