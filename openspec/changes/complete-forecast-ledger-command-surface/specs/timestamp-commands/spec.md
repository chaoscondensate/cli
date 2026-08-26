## Purpose

Defines the complete pure-Go OpenTimestamps command lifecycle for creating, upgrading, inspecting, and independently verifying receipts bound to exact forecast target bytes.

## ADDED Requirements

### Requirement: Use only the supported OpenTimestamps profile
All timestamp commands SHALL accept and produce only the bounded detached SHA-256 OpenTimestamps proof subset approved for Forecast Ledger v1. RFC 3161 and other timestamp protocols MUST be rejected as unsupported and MUST NOT be converted or reported as equivalent evidence. Core timestamp behavior SHALL require no Python process or external executable.

#### Scenario: RFC 3161 artifact supplied
- **WHEN** a timestamp command receives an RFC 3161 request or response
- **THEN** it returns invalid/unsupported data before writing a file or ledger state

### Requirement: Require explicit forecast and network endpoints
`timestamp stamp`, `upgrade`, `status`, and `verify` SHALL require a real `--file`, `--question`, and `--forecast`; they MUST NOT accept `--file -`. Network commands SHALL contact only endpoints explicitly named for that invocation. Stamp and upgrade SHALL require one or more repeatable `--calendar <https-url>` allowlist values and SHALL reject receipt calendar hints not present in that allowlist. No timestamp command SHALL read a hidden default calendar, proxy credential, Bitcoin credential, or endpoint from project config or environment.

#### Scenario: Upgrade with unapproved calendar hint
- **WHEN** a receipt contains a calendar URL absent from all supplied `--calendar` values
- **THEN** upgrade does not contact that URL and reports the branch as not checked

### Requirement: Stamp the exact deterministic target
`forecast-ledger timestamp stamp` SHALL validate the ledger, reconstruct the selected `forecast-envelope/v1` target, and preflight deterministic paths `proofs/targets/<forecast-id>.json` and `proofs/receipts/<forecast-id>.json.ots`. It SHALL accept repeatable calendars, bounded `--timeout`, and `--min-success` from 1 through the number of calendars. It SHALL submit the exact target SHA-256 digest concurrently within limits and accept success only when the configured number of valid calendar branches is obtained.

The command SHALL combine accepted branches into one deterministic detached receipt, exclusively create or confirm-identical target and receipt artifacts, then atomically change forecast integrity from `unanchored` to `pending`. The pending object SHALL contain scope `forecast-envelope/v1`, the approved canonicalization identifier, relative target path, SHA-256 digest, and exactly one OpenTimestamps entry with the deterministic receipt path and state `pending`. A calendar response or local file timestamp MUST NOT create a verified claim.

#### Scenario: Successful pending stamp
- **WHEN** the minimum number of approved calendars return valid bounded branches
- **THEN** durable target/receipt artifacts exist and matching pending ledger metadata is committed atomically

#### Scenario: Insufficient calendar success
- **WHEN** fewer than `--min-success` approved calendars return valid responses
- **THEN** stamp returns network failure, does not mark the ledger pending or verified, and reports any retained target artifact available for safe retry

#### Scenario: Repeat identical stamp
- **WHEN** target, receipt, and matching pending metadata already exist
- **THEN** stamp is idempotent and does not submit duplicate requests unless `--restamp` is introduced by a separately reviewed change

### Requirement: Recover multi-file stamp failures
Before the first write, stamp SHALL validate all resolved paths and collisions. Different existing target or receipt bytes SHALL return `conflict`. If interruption occurs after creating artifacts but before ledger commit, the original ledger SHALL remain coherent and a recovery journal SHALL identify only CLI-created paths. A retry SHALL either complete the matching pending mutation or safely report the retained artifacts; it MUST NOT delete unrelated or pre-existing files.

#### Scenario: Receipt durable and ledger replacement interrupted
- **WHEN** the process stops after flushing the receipt but before ledger commit
- **THEN** recovery preserves the original ledger and enables a retry to associate only the matching target and receipt

### Requirement: Upgrade pending receipts without losing proof data
`forecast-ledger timestamp upgrade` SHALL require pending matching ledger metadata and artifacts, verify target bytes/digest first, parse the bounded receipt, and query only approved calendars for missing attestations. It SHALL merge valid responses deterministically and preserve safe unknown proof nodes byte-for-byte. It SHALL replace the receipt recoverably only when the new receipt is a semantic superset; otherwise it SHALL report `unchanged` or failure.

Upgrade SHALL leave integrity `pending` until independent Bitcoin verification succeeds. It MUST NOT set `anchored_before`, block height, `verified_at`, or `verified` merely because a calendar returns a Bitcoin attestation.

#### Scenario: No upgrade available
- **WHEN** every approved calendar reports no new branch
- **THEN** the command returns pending/not-ready, leaves receipt and ledger unchanged, and provides a retry-safe result

#### Scenario: Downgrade response
- **WHEN** a response would discard an existing proof branch or supported attestation
- **THEN** upgrade rejects it and preserves the original receipt

### Requirement: Report timestamp status locally
`forecast-ledger timestamp status` SHALL perform no network request and no mutation. It SHALL report one of `missing`, `unanchored`, `pending`, `confirmed_unverified`, `verified`, `failed`, or `inconsistent`, together with target/receipt presence, digest agreement, supported attestations, recorded ledger state, and safe next commands. It SHALL distinguish receipt syntax/attestation presence from independent Bitcoin verification.

#### Scenario: Bitcoin attestation not independently checked
- **WHEN** a well-formed receipt contains a supported Bitcoin attestation but the ledger is still pending
- **THEN** status reports `confirmed_unverified`, not verified existence timing

#### Scenario: Missing pending receipt
- **WHEN** ledger metadata is pending but the receipt path is absent
- **THEN** status reports `inconsistent` and returns verification failure without modifying integrity

### Requirement: Verify receipts through one explicit Bitcoin source
`forecast-ledger timestamp verify` SHALL reconstruct and compare the target before proof verification and require exactly one explicit Bitcoin source:

- `--bitcoin-source core` with explicit RPC URL and a protected `--bitcoin-auth-file`; or
- `--bitcoin-source explorer` with an explicit HTTPS base URL whose weaker trust boundary is reported.

Credentials MUST NOT be accepted in argv, URLs, or environment variables. The verifier SHALL validate proof operations and the confirmed transaction/block commitment against the exact target using bounded requests, response sizes, redirects, and chain data. It SHALL derive the protocol's conservative `anchored_before` value and block height only from a successfully verified attestation.

#### Scenario: Exact proof verification
- **WHEN** the selected source independently confirms a supported attestation for the reconstructed target
- **THEN** verification reports the confirmed block evidence and conservative time bound

#### Scenario: Receipt for different bytes
- **WHEN** the receipt is valid for a digest different from the reconstructed target
- **THEN** verification fails cryptographically and performs no ledger transition

#### Scenario: Explorer verification
- **WHEN** an HTTPS explorer is explicitly selected and confirms the proof
- **THEN** output names that source and states that the result trusts the explorer rather than an independently operated node

### Requirement: Commit verified integrity only after all checks
After successful proof verification, `timestamp verify` SHALL atomically change matching pending integrity to `verified`, retain target and all receipt references, set the OpenTimestamps entry to `confirmed`, store the verified conservative `anchored_before` and Bitcoin block height, and set `verified_at` to the exact command time or explicit override. It MUST NOT mutate on network failure, unavailable source, unsupported proof branch, target mismatch, or invalid proof.

If the question is resolved and `anchored_before` is not earlier than `outcome_known_at`, the cryptographic timestamp MAY still become verified, but output SHALL warn that it is insufficient to exclude hindsight; layered verification SHALL fail the pre-outcome evidence claim.

#### Scenario: Late but valid timestamp
- **WHEN** Bitcoin evidence is valid but its conservative bound is after the known outcome
- **THEN** integrity records cryptographic verification while output explicitly marks pre-outcome sufficiency as failed

### Requirement: Keep pending and failure semantics honest
Pending/not yet confirmed SHALL return exit `9` when it is the primary result. Network unavailability SHALL return exit `8`; malformed or mismatched proof evidence SHALL return exit `6`; unsafe/missing local files SHALL use the corresponding I/O/not-found categories. A verification failure MUST NOT automatically destroy a previously verified record or rewrite integrity to `failed`; persistent failure-state mutation requires retained evidence and an explicit separately specified repair/review action.

#### Scenario: Temporary Bitcoin source outage
- **WHEN** the selected source times out
- **THEN** verify returns network failure and leaves pending or verified metadata unchanged

### Requirement: Pass official-client and adversarial conformance gates
Timestamp commands MUST remain unavailable or explicitly experimental until Python-to-Go and Go-to-Python receipts round-trip, supported info/upgrade/verify results match the official client, mocked calendar and Bitcoin-source tests pass, malformed/oversized/deep proof fuzzing is clean, redirect/timeout/SSRF controls pass, native-platform recovery tests pass, real-calendar nightly tests are stable, and the supported subset receives independent review.

#### Scenario: Unsupported proof operation
- **WHEN** a receipt contains an operation outside the reviewed subset
- **THEN** it is preserved safely for round-trip or rejected explicitly, never ignored while claiming successful verification

