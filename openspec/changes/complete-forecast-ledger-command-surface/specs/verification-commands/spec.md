> Supersession note (2026-08-30): generic authoring documents, public
> `--input`, MCP `input`/public `input_file`, v1.2.0 identity, and the old target
> shape below are historical implementation facts. `make-authoring-direct-readable`
> owns their v1.3.0 replacements and takes precedence.

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

Normal human output SHALL render every applicable layer and its state in a stable scannable matrix; it MUST NOT collapse the report to only one overall status line. `--verbose` MAY add safe diagnostic evidence but MUST NOT be required to see the independent layer states. Plain output SHALL expose the same layers in its documented stable field order.

#### Scenario: Mixed evidence states
- **WHEN** document and target checks pass while a receipt remains pending and reveal is inapplicable
- **THEN** the matrix reports those states separately and overall does not claim full verification

#### Scenario: Human report shows every layer
- **WHEN** a user runs layered verification in normal human mode
- **THEN** output shows the overall state and one independently labeled state for every applicable layer without requiring `--json` or `--verbose`

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

### Requirement: Verify existence timing from retained RFC 3161 evidence
For every applicable RFC 3161 entry, verification SHALL use the retained exact target, `.tsq` request, `.tsr` response, and PEM CA bundle. It SHALL verify response success, request/response nonce and message-imprint agreement, target SHA-256 binding, CMS content type and signed attributes, message digest, signature, signing-certificate binding, timestamping EKU, supported algorithms, and certificate chain against the retained trust anchors at `gen_time`. It SHALL parse bounded inputs, reject trailing or ambiguous structures, and perform no TSA, blockchain, system-trust-store, or other timestamp-verification network request.

Existence timing SHALL report local check state, TSA identity, `gen_time`, policy OID, serial number, request/response/CA-bundle safe paths, CA-bundle digest, stored or observed `verified_at`, and whether verified `gen_time` predates a resolved question's `outcome_known_at`. These fields SHALL be present in human, plain, JSON, and MCP evidence whenever safely available; a one-word `verified` state is insufficient. A cryptographically valid late timestamp SHALL fail pre-outcome sufficiency while retaining proof-valid evidence.

Layered verification SHALL try every retained timestamp entry. One complete valid entry is sufficient for the existence-timing cryptographic branch to pass. The layer SHALL fail for cryptographic invalidity only when every completely checked entry fails; an entry that is pending or cannot be checked because retained material is missing SHALL remain visible and prevent an all-branches-failed claim when no other entry passes. `--offline` continues to disable optional outcome-source retrieval but does not weaken or skip the local timestamp checks.

#### Scenario: Pending response remains visible
- **WHEN** one retained timestamp entry is pending and no other complete entry passes
- **THEN** existence timing remains pending or not checked, reports the exact missing or incomplete local check, and performs no network access

#### Scenario: One of two timestamp entries passes
- **WHEN** two retained TSA entries exist and one complete target/request/response/CA chain verifies locally
- **THEN** existence timing passes through that entry while preserving the other entry's independent state and limitations

#### Scenario: Valid late timestamp
- **WHEN** a verified response's `gen_time` is not earlier than the known outcome
- **THEN** proof validity is reported but existence timing for pre-outcome forecasting fails

#### Scenario: Stored labels disagree with local evidence
- **WHEN** stored TSA, policy, serial, generation time, or verification metadata differs from the parsed verified response
- **THEN** existence timing fails without trusting or rewriting the stored label

### Requirement: Verify revealed forecasts end to end
For a revealed forecast, the reveal layer SHALL authenticate/decrypt retained ciphertext using the disclosed v1 key, verify commitment, associated data, protocol and bound IDs, parse the canonical private bundle, compare every public mirror field, and confirm the rebuilt target remains the original sealed target. For sealed and public forecasts, reveal SHALL be `not_applicable` unless a requested check cannot be performed because required evidence is missing, in which case it SHALL be `not_checked` or fail as appropriate. Raw keys and decrypted private values MUST remain absent from output.

#### Scenario: Edited revealed mirror
- **WHEN** a public revealed field differs from authenticated plaintext
- **THEN** reveal fails even if the ciphertext and timestamp receipt are otherwise valid

### Requirement: Observe outcome evidence without asserting truth
For resolved questions, outcome evidence SHALL validate resolution type, chronology, non-empty sources, optional content digests, and source metadata. By default it SHALL make no request to outcome URLs; automatic RFC 3161 RFC 3161 checks do not enable arbitrary source retrieval. With explicit `--check-sources`, it MAY perform bounded retrieval of public HTTPS URLs to report reachability, final approved URL, response time, and digest match, but MUST reject private/link-local/reserved destinations and MUST NOT infer that a reachable page is authoritative or that the outcome is substantively true. `--offline` SHALL override `--check-sources`. Annulled/disputed states SHALL report their recorded reason/evidence without converting them to resolved truth.

#### Scenario: Reachable outcome URL
- **WHEN** an explicitly checked source returns successfully and matches its stored digest
- **THEN** the layer reports source availability and byte match while retaining the limitation that substantive correctness was not established

### Requirement: State protocol limitations in every report
Every human and JSON report SHALL explicitly state that Forecast Ledger v1 does not by itself prove authorship, completeness of the ledger or forecast set, truth or calibration of a forecast, exactness of self-reported times, TSA clock honesty, current certificate revocation status, long-term legal validity, or substantive correctness of outcome evidence. Filesystem, archive, hosting, source-control, and external-anchor timestamps MUST NOT be promoted to cryptographic existence evidence. A stamp result SHALL state that the selected TSA learns request timing and the target digest commitment. The report MUST NOT imply network privacy or anonymity.

#### Scenario: Every technical layer passes
- **WHEN** all applicable automated checks pass
- **THEN** overall reports verified technical evidence and still includes every required limitation

#### Scenario: Timestamp trust disclosure
- **WHEN** existence timing passes against a retained CA bundle
- **THEN** human, JSON, and MCP results identify the retained trust input and state that signature/chain validity does not establish TSA clock honesty or long-term legal validity

### Requirement: Apply deterministic overall status and exit precedence
Overall SHALL be `fail` when any required applicable layer fails, `pending` when none fail and at least one is pending, `incomplete` when none fail/pending but a requested required layer is not checked, including request-budget exhaustion, and `pass` only when every requested applicable layer passes. Document invalidity SHALL use exit `3`; cryptographic/evidence failure exit `6`; pending or incomplete exit `9`; required automatic or explicitly selected network failure exit `8`; only a complete pass exits `0`. JSON output SHALL still contain the full safely available matrix for non-zero evidence outcomes. Presenting a successful result envelope or human report before returning a pending/incomplete outcome MUST NOT erase or remap that outcome to internal failure; exit selection SHALL use the report's final application category in every output mode.

#### Scenario: Failure plus pending
- **WHEN** content binding fails and another receipt is pending
- **THEN** overall is fail and exit `6`, while the pending layer remains visible in the matrix

#### Scenario: Required layer not checked
- **WHEN** no layer fails or remains pending but an applicable requested layer cannot be checked
- **THEN** overall is incomplete and the command exits `9`, never `0`

#### Scenario: Pending report is presented successfully
- **WHEN** the complete available human, plain, or JSON matrix is written successfully and overall remains pending
- **THEN** the command exits `9`, not `1`, while preserving the emitted report

### Requirement: Produce deterministic verification results
Given identical local bytes and an explicit observation time, repeated verification SHALL produce semantically identical JSON ordered by ledger question/forecast order and stable layer order. Volatile duration/progress fields MUST stay outside the stable result or be clearly separated as diagnostics.

#### Scenario: Repeat offline verification
- **WHEN** the same package is verified twice offline
- **THEN** normalized JSON evidence, states, codes, paths, and limitations are identical
