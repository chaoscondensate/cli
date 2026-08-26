## Purpose

Defines deterministic target construction and the complete secret-safe sealed forecast lifecycle, including artifact collision behavior, protected key handling, and authenticated reveal.

## ADDED Requirements

### Requirement: Construct the exact forecast target projection
The system SHALL construct a closed `forecast-envelope/v1` object with exactly four root properties: `schema` equal to `forecast-envelope/v1`, `ledger_id`, `question`, and `forecast`. The projection SHALL use this complete allowlist:

| Object | Included fields |
| --- | --- |
| `question` for every forecast | `id`, `title`, `type`, `resolution_criteria`, `created_at`, `forecast_window`, `expected_resolution_at`; `options` only for multiple-choice; `unit` only for numeric |
| `forecast` common | `id`, `forecasted_at`, `recorded_at`, normalized `visibility`; include `public_note` and `supersedes_forecast_id` only when present |
| public forecast extension | `value`; include `rationale`, `key_factors`, and `comment` exactly when present |
| sealed forecast extension | `commitment` containing exactly `scheme`, `commitment_hash`, and `encryption` (`algorithm`, `nonce`, `ciphertext`) |
| revealed forecast extension | the same sealed extension and normalized `visibility: sealed`, rebuilding the forecast as it existed before reveal |

Every other ledger, question, forecast, resolution, integrity, publication, platform, forecaster, and presentation field MUST be excluded. In particular, `question.status`, `platform_refs`, `tags`, `notes`, `forecasts`, and `resolution`; forecast `integrity`; commitment `key_hint`, `revealed_at`, and `revealed_key`; and the revealed public mirror MUST be excluded. The target therefore binds the ledger ID and listed question meaning, but does not prove forecaster authorship or bind omitted mutable metadata.

The complete closed projection SHALL be encoded with the bounded RFC 8785/JCS profile. It SHALL reject floats and values outside I-JSON limits, sort object properties by UTF-16 code units, emit UTF-8 canonical bytes without insignificant whitespace, and calculate SHA-256 over those exact bytes.

A public forecast target SHALL bind the public value and public explanatory fields. A sealed forecast target SHALL bind the sealed commitment and encryption fields while excluding private plaintext. A revealed forecast SHALL continue to rebuild the original sealed-envelope target, not a new public target, so reveal cannot change the bytes covered by an existing timestamp.

#### Scenario: Cross-platform target identity
- **WHEN** the same forecast is targeted on macOS, Linux, and Windows
- **THEN** canonical bytes and the lowercase SHA-256 digest are byte-for-byte identical

#### Scenario: Revealed target continuity
- **WHEN** a previously targeted sealed forecast is revealed correctly
- **THEN** rebuilding its target produces the exact original sealed target bytes and digest

#### Scenario: Target changes across ledgers
- **WHEN** the same valid sealed commitment is placed under the same question and forecast IDs in a ledger with a different `ledger_id`
- **THEN** seal authentication can still succeed but the `forecast-envelope/v1` bytes and digest differ

#### Scenario: Excluded status change
- **WHEN** only question status or resolution changes
- **THEN** the forecast target bytes remain unchanged because lifecycle outcome is not part of the prediction target

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

After input parsing and time defaulting, `forecasted_at`, `recorded_at`, `value`, `rationale`, `key_factors`, and `comment` SHALL all be present because the revealed schema requires all six mirror fields. `rationale` and `comment` MAY be empty strings where the schema permits; `key_factors` MAY be an empty array but every included item MUST be non-empty. `public_note` and `supersedes_forecast_id` remain public record fields outside the private bundle.

The selected question SHALL be open, the forecast time SHALL be inside its window, type/value/chronology rules SHALL match public forecast add, and a supersession link SHALL target an earlier forecast in the same question. Seal MUST create a new append-only forecast and MUST NOT accept an existing public, sealed, or revealed forecast ID for in-place hiding or resealing.

#### Scenario: Private plaintext supplied as flag
- **WHEN** a user attempts to pass a raw value, rationale, key, or salt flag to seal
- **THEN** argument parsing rejects it because no such secret-bearing option exists

#### Scenario: Missing reveal mirror field
- **WHEN** private input omits rationale, key factors, or comment after defaults are applied
- **THEN** seal rejects the bundle before entropy or file creation because a future schema-valid reveal would be impossible

#### Scenario: Attempt to hide a public forecast
- **WHEN** seal receives a forecast ID already used by a public forecast
- **THEN** it returns conflict and does not encrypt, create a key, or change the published record

### Requirement: Seal with the published cryptographic profile
Seal SHALL generate a fresh 32-byte salt, 32-byte key, and 12-byte nonce from the operating-system CSPRNG. It SHALL create this exact closed canonical plaintext object:

- `schema`: `forecast-seal/v1`;
- `question_id`: the selected question ID;
- `forecast_id`: the new forecast ID;
- `salt`: the 32-byte salt as 64 lowercase hexadecimal characters;
- `bundle`: exactly `forecasted_at`, `recorded_at`, `value`, `rationale`, `key_factors`, and `comment`.

The commitment value SHALL be SHA-256 of the RFC 8785/JCS canonical UTF-8 plaintext bytes. ChaCha20-Poly1305 associated data SHALL be the RFC 8785/JCS canonical encoding of exactly `scheme: forecast-seal/v1`, `question_id`, `forecast_id`, and `commitment_sha256` containing that lowercase digest. The encryption output SHALL use the fresh 32-byte key and 12-byte nonce. Neither plaintext nor AAD contains `ledger_id`, `public_note`, `supersedes_forecast_id`, or `key_hint`. The implementation MUST reproduce the pinned upstream plaintext, commitment, nonce/ciphertext, and complete commitment vector exactly when deterministic test entropy and fixture key hint are injected by conformance tests.

The appended forecast SHALL have visibility `sealed`, integrity `unanchored`, optional safe public note and supersession link, and a commitment containing the published scheme, SHA-256 commitment, `chacha20-poly1305` nonce/ciphertext, and a safe logical key hint. Production CLI creation SHALL use `forecast-key:<forecast-id>` unless a separately reviewed future profile adds another safe logical form. `key_hint` MUST NOT contain the `--key-file` path, an absolute path, a file URI, a secret-root name, or credential material. The sealed forecast MUST NOT contain plaintext value, rationale, key factors, or comment.

#### Scenario: Successful seal
- **WHEN** private input is valid and secure key storage succeeds
- **THEN** one sealed forecast is appended and normal, JSON, verbose, and error output contain no salt, key, nonce value, private bundle, or ciphertext plaintext

#### Scenario: Entropy failure
- **WHEN** the operating-system random source cannot return every required byte
- **THEN** seal returns an internal or I/O failure before creating a key file or ledger mutation

### Requirement: State the seal profile's exact security boundaries
Documentation and every machine-readable crypto profile description SHALL state that `forecast-seal/v1` authenticates the exact question ID, forecast ID, salt, private bundle, scheme, and commitment digest described above. It does not authenticate `ledger_id`, `public_note`, `supersedes_forecast_id`, `key_hint`, forecaster identity, question wording, or any other ledger metadata. A sealed commitment can therefore authenticate after transfer to another ledger when its question and forecast IDs are unchanged; a `forecast-envelope/v1` target created in that ledger separately binds the ledger ID and listed question fields.

After reveal publishes the key, anyone with the retained ciphertext can recover the salt and private bundle. `public_note` and `supersedes_forecast_id` are not protected by the seal commitment but are included in the sealed target if target evidence is created. `key_hint` is deliberately excluded from both seal authentication and target bytes and is non-authoritative; reveal always requires an explicit protected key file, so moving a key file does not require changing the ledger.

#### Scenario: Public metadata tampered before targeting
- **WHEN** only `public_note`, `supersedes_forecast_id`, or `key_hint` changes on an otherwise authentic sealed record before a target exists
- **THEN** seal authentication alone does not detect that change and verification reports the documented limitation

#### Scenario: Public metadata tampered after targeting
- **WHEN** `public_note` or `supersedes_forecast_id` changes after a matching target is retained
- **THEN** content-binding verification fails even though seal authentication can still succeed

#### Scenario: Key file moved intentionally
- **WHEN** an operator securely moves a valid key file and later supplies its new explicit path to reveal
- **THEN** reveal uses the key file's bound question/forecast IDs, requires no ledger or `key_hint` mutation, and the logical hint remains non-authoritative

### Requirement: Protect key files before ledger publication
The key destination SHALL be explicit, new, outside package-output roots, and not a symlink/junction/reparse-point escape. POSIX creation SHALL use owner-only mode `0600`; Windows creation SHALL apply an owner-only ACL and reject a destination whose protection cannot be established. The file SHALL contain only the documented key-file format and SHALL be flushed before ledger commit.

The `forecast-key/v1` file bytes SHALL be RFC 8785/JCS canonical UTF-8 for a closed object containing exactly `schema: forecast-key/v1`, `question_id`, `forecast_id`, and `key_hex` as 64 lowercase hexadecimal characters, followed by exactly one LF byte. It SHALL contain no ledger ID, salt, nonce, ciphertext, key hint, timestamp, path, or additional field. On read, schema, IDs, hex form, and decoded 32-byte length MUST match the selected operation before decryption begins.

Seal SHALL write and secure the key first, then commit the ledger. If key creation fails, the ledger remains unchanged. If ledger commit fails after the key is durable, the command SHALL preserve the key, report its safe display path and recovery action without revealing it, and MUST NOT silently delete the only copy.

#### Scenario: Existing key destination
- **WHEN** `--key-file` already exists
- **THEN** seal returns `conflict` without reading, truncating, chmodding, or replacing that entry

#### Scenario: Key file bound to another forecast
- **WHEN** reveal receives a valid `forecast-key/v1` file naming another question or forecast
- **THEN** it returns verification failure before decryption and does not expose the key or mutate the ledger

#### Scenario: Ledger commit fails after key write
- **WHEN** the key is durable but post-validation or safe replacement fails
- **THEN** the original ledger remains unchanged and output identifies a retained orphan key file using a redacted/safe path

### Requirement: Repair non-authoritative key hints safely
`forecast-ledger forecast key-hint update` SHALL require `--file`, `--question`, `--forecast`, and scalar `--key-hint`. It SHALL select a sealed or revealed forecast with a supported commitment and change only `commitment.key_hint`; it MUST NOT read, move, create, discover, validate, or disclose any actual key file. The operation SHALL remain available for imported `unanchored`, `pending`, `verified`, or `failed` integrity because the hint is outside seal authentication and `forecast-envelope/v1`; every target, receipt, disclosed key, commitment digest, ciphertext, and integrity byte SHALL remain unchanged. Identical input SHALL be idempotent.

A package-safe v1 hint SHALL match the closed ASCII form `scheme:opaque`: scheme MUST match `[a-z][a-z0-9+.-]*`, MUST NOT equal `file`, and opaque MUST match `[A-Za-z0-9._~+-]+`. Slashes, backslashes, additional colons, percent encoding, whitespace, authority/user-info syntax, query, fragment, drive/UNC/device syntax, and empty components are forbidden. For scheme `forecast-key`, opaque MUST equal the selected forecast ID. CLI seal creation SHALL continue to use `forecast-key:<forecast-id>`; other safe logical schemes are explicit public metadata and MUST NOT trigger automatic key discovery.

#### Scenario: Normalize an imported path hint
- **WHEN** a schema-valid imported sealed forecast has `key_hint: keys/f-001.key` and the user updates it to `forecast-key:f-001`
- **THEN** only the hint changes, the forecast becomes package-safe, and every existing cryptographic target and receipt still verifies byte-for-byte

#### Scenario: Reject a file-like logical hint
- **WHEN** an update supplies `file:secret.key`, `secret-manager://item`, `C:\\keys\\f.key`, or an opaque value containing path or credential syntax
- **THEN** the command returns invalid data before mutation and identifies the required non-location `scheme:opaque` form

#### Scenario: Repair a failed imported record
- **WHEN** an imported forecast has terminal failed integrity but only its unsafe key hint needs normalization
- **THEN** key-hint update may change that non-evidentiary hint while preserving the complete failed integrity record

### Requirement: Reveal only after complete authentication
`forecast-ledger forecast reveal` SHALL require approval, a selected sealed forecast, and an explicit protected `--key-file`. Before opening a ledger write transaction it SHALL read the bounded key file, authenticate/decrypt the original ciphertext, verify the commitment hash, exact associated data, question/forecast IDs, protocol identifiers, and exact canonical private bundle, and validate the six derived public mirror fields against the question type and chronology. It MUST NOT claim or attempt a ledger-ID binding that does not exist in `forecast-seal/v1`.

Only after every check succeeds SHALL reveal set visibility to `revealed`, add `revealed_at` and the disclosed key required by v1, and derive value/rationale/key factors/comment from authenticated plaintext. It SHALL retain the original commitment hash, nonce, ciphertext, key hint, supersession link, integrity object, target relationship, and public note.

#### Scenario: Wrong key
- **WHEN** the supplied key fails AEAD authentication
- **THEN** reveal returns verification failure and the ledger remains byte-for-byte unchanged

#### Scenario: Bound ID mismatch
- **WHEN** decrypted plaintext authenticates but names a different question or forecast
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
Release gates SHALL include the pinned positive seal vector, a checked-in cross-language `forecast-envelope/v1` vector for public/sealed/revealed continuity, exact `forecast-key/v1` bytes, negative mutations for every actually bound field, explicit non-binding tests for ledger ID/public note/supersession/key hint, wrong-key and tampered-ciphertext cases, Go/Python cross-language seal/reveal fixtures, property tests, fuzzing of bounded key/ciphertext inputs, cross-platform target bytes, crash points across key/artifact/ledger writes, and canary assertions through implemented seal/reveal paths. No crypto command SHALL become visible while its vector, rollback, native key-protection, and redaction gates are incomplete.

#### Scenario: Pinned vector regression
- **WHEN** canonicalization or crypto output differs by one byte from the published vector
- **THEN** conformance and release checks fail before a binary is published
