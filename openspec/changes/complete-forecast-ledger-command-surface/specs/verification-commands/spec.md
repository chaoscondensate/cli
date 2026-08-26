## Purpose

Defines a layered, non-mutating verification command that reports exactly which ledger, content, timing, reveal, outcome, and package claims are supported without collapsing them into a misleading boolean.

## ADDED Requirements

### Requirement: Select verification scope explicitly
`forecast-ledger verify` SHALL require a real `--file` path and SHALL verify every question/forecast by default. `--question` SHALL narrow to one question; `--forecast` SHALL require `--question` and narrow to one forecast. A missing selector SHALL return `not_found`; duplicate IDs SHALL already be invalid data. The command MUST NOT mutate the ledger, target, receipt, key, or package.

#### Scenario: Verify one forecast
- **WHEN** a user passes one existing question and forecast ID
- **THEN** the report contains only ledger-wide prerequisites and layers applicable to that exact forecast

### Requirement: Return a stable evidence matrix
Human and JSON output SHALL contain overall status plus independently named layers: `document`, `content_binding`, `existence_timing`, `reveal`, `outcome_evidence`, and optional `package_integrity`. Each layer SHALL use exactly one state from `pass`, `fail`, `pending`, `not_applicable`, or `not_checked` and include stable reason codes, evidence summaries, limitations, and safe paths/identifiers. Overall success MUST NOT convert `not_checked` into `pass`.

#### Scenario: Mixed evidence states
- **WHEN** document and target checks pass while a receipt remains pending and reveal is inapplicable
- **THEN** the matrix reports those states separately and overall does not claim full verification

### Requirement: Verify document validity first
The document layer SHALL run bounded parsing, exact embedded schema validation, format checks, domain decoding, semantic checks, and artifact-path confinement. A document failure SHALL prevent dependent layers from claiming pass, but the report MAY continue with safely computable independent observations. Source values and secrets MUST NOT be echoed in issues.

#### Scenario: Invalid multiple-choice forecast
- **WHEN** probabilities do not cover options or sum to 10,000 basis points
- **THEN** document fails with exact locations and dependent content/timing/reveal layers are not checked

### Requirement: Verify content binding before timestamp evidence
For every forecast with integrity target metadata, content binding SHALL rebuild the correct public or sealed/revealed `forecast-envelope/v1`, compare canonical bytes with the retained artifact, recompute SHA-256, and compare scope, canonicalization, relative path, algorithm, and digest. Existence timing MUST NOT pass unless content binding passes for the exact bytes referenced by the receipt.

#### Scenario: Ledger changed after timestamping
- **WHEN** forecast fields rebuild to bytes different from the retained target
- **THEN** content binding fails even if the receipt remains a valid proof for the old artifact

### Requirement: Verify existence timing with explicit network policy
Verification SHALL be offline by default. Offline mode SHALL parse and inspect retained receipts but report confirmed attestations as `not_checked` or pending unless ledger metadata already contains internally consistent previously verified evidence. A user MAY provide the same explicit Bitcoin source options as `timestamp verify`; only then may the command perform bounded network checks and independently pass existence timing.

Existence timing SHALL report target/receipt binding, proof validity, Bitcoin source and trust boundary, block evidence, conservative `anchored_before`, and whether that bound predates a resolved question's `outcome_known_at`. A cryptographically valid late anchor SHALL fail pre-outcome sufficiency while retaining proof-valid evidence.

#### Scenario: Offline pending proof
- **WHEN** a receipt is pending and no Bitcoin source is selected
- **THEN** existence timing is pending with no network access

#### Scenario: Valid late anchor
- **WHEN** a verified proof's conservative bound is not earlier than the known outcome
- **THEN** proof validity is reported but existence timing for pre-outcome forecasting fails

### Requirement: Verify revealed forecasts end to end
For a revealed forecast, the reveal layer SHALL authenticate/decrypt retained ciphertext using the disclosed v1 key, verify commitment, associated data, protocol and bound IDs, parse the canonical private bundle, compare every public mirror field, and confirm the rebuilt target remains the original sealed target. For sealed and public forecasts, reveal SHALL be `not_applicable` unless a requested check cannot be performed because required evidence is missing, in which case it SHALL be `not_checked` or fail as appropriate. Raw keys and decrypted private values MUST remain absent from output.

#### Scenario: Edited revealed mirror
- **WHEN** a public revealed field differs from authenticated plaintext
- **THEN** reveal fails even if the ciphertext and timestamp receipt are otherwise valid

### Requirement: Observe outcome evidence without asserting truth
For resolved questions, outcome evidence SHALL validate resolution type, chronology, non-empty sources, optional content digests, and source metadata. By default it SHALL make no network request. With explicit `--check-sources`, it MAY perform bounded retrieval to report reachability, final approved URL, response time, and digest match, but MUST NOT infer that a reachable page is authoritative or that the outcome is substantively true. Annulled/disputed states SHALL report their recorded reason/evidence without converting them to resolved truth.

#### Scenario: Reachable outcome URL
- **WHEN** an explicitly checked source returns successfully and matches its stored digest
- **THEN** the layer reports source availability and byte match while retaining the limitation that substantive correctness was not established

### Requirement: State protocol limitations in every report
Every human and JSON report SHALL explicitly state that Forecast Ledger v1 does not by itself prove authorship, completeness of the ledger or forecast set, truth or calibration of a forecast, exactness of self-reported times, or substantive correctness of outcome evidence. Filesystem, archive, hosting, source-control, and external-anchor timestamps MUST NOT be promoted to cryptographic existence evidence.

#### Scenario: Every technical layer passes
- **WHEN** all applicable automated checks pass
- **THEN** overall reports verified technical evidence and still includes every required limitation

### Requirement: Apply deterministic overall status and exit precedence
Overall SHALL be `fail` when any required applicable layer fails, `pending` when none fail and at least one is pending, `incomplete` when none fail/pending but a requested required layer is not checked, and `pass` only when every requested applicable layer passes. Document invalidity SHALL use exit `3`; cryptographic/evidence failure exit `6`; pending exit `9`; explicitly requested network failure exit `8`; otherwise a complete pass exits `0`. JSON output SHALL still contain the full safely available matrix for non-zero evidence outcomes.

#### Scenario: Failure plus pending
- **WHEN** content binding fails and another receipt is pending
- **THEN** overall is fail and exit `6`, while the pending layer remains visible in the matrix

### Requirement: Produce deterministic verification results
Given identical local bytes, explicit source responses, and an explicit observation time, repeated verification SHALL produce semantically identical JSON ordered by ledger question/forecast order and stable layer order. Volatile duration/progress fields MUST stay outside the stable result or be clearly separated as diagnostics.

#### Scenario: Repeat offline verification
- **WHEN** the same package is verified twice offline
- **THEN** normalized JSON evidence, states, codes, paths, and limitations are identical

