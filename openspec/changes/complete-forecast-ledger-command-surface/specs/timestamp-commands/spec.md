## Purpose

Defines the complete pure-Go OpenTimestamps command lifecycle for creating, upgrading, inspecting, and verifying receipts bound to exact forecast target bytes with zero-config public sources or optional independent Bitcoin Core.

## ADDED Requirements

### Requirement: Use only the supported OpenTimestamps profile
All timestamp commands SHALL accept and produce only the bounded detached SHA-256 OpenTimestamps proof subset approved for Forecast Ledger v1. RFC 3161 and other timestamp protocols MUST be rejected as unsupported and MUST NOT be converted or reported as equivalent evidence. Core timestamp behavior SHALL require no Python process or external executable.

#### Scenario: RFC 3161 artifact supplied
- **WHEN** a timestamp command receives an RFC 3161 request or response
- **THEN** it returns invalid/unsupported data before writing a file or ledger state

### Requirement: Use a visible versioned public network profile by default
`timestamp stamp`, `upgrade`, `status`, and `verify` SHALL require a real `--file`, `--question`, and `--forecast`; they MUST NOT accept `--file -`. Network-capable timestamp actions SHALL require no calendar, explorer, account, API-key, or network-permission setup for their normal public operation. They SHALL use the immutable profile `opentimestamps-public-v1` embedded in the released binary and SHALL report its profile ID and contacted source IDs in human and JSON results.

The profile SHALL submit stamps to exactly these four HTTPS endpoints with required success `2`:

- `https://a.pool.opentimestamps.org`
- `https://b.pool.opentimestamps.org`
- `https://a.pool.eternitywall.com`
- `https://ots.btc.catallaxy.com`

The profile SHALL allow upgrades only from those submission endpoints and their pinned receipt calendar identities for Alice, Bob, Finney, and Catallaxy. It SHALL verify Bitcoin block observations by querying both `https://mempool.space/api` and `https://blockstream.info/api` and requiring agreement as specified below. The profile MUST NOT be downloaded, remotely replaced, extended from a receipt, project config, environment variable, redirect, or MCP request; changing its endpoints or trust policy requires a reviewed binary release and a new profile ID.

Every network-capable action SHALL support `--offline`. Offline mode MUST open no socket. `timestamp stamp` and `upgrade` SHALL return `network_disabled`/exit `8` before side effects; `timestamp verify` SHALL perform all local checks and return pending or not-checked timing without changing integrity. `timestamp status` is always local. Standard Go proxy behavior MAY be used but credentials MUST NOT be discovered from project files or emitted.

#### Scenario: Zero-config public stamp
- **WHEN** a user invokes stamp without network-source flags
- **THEN** the command uses `opentimestamps-public-v1`, contacts only its four pinned calendar endpoints, and identifies the profile and responding source IDs

#### Scenario: Upgrade with calendar outside the built-in profile
- **WHEN** a receipt contains a calendar URL outside the built-in profile
- **THEN** upgrade does not contact that URL and reports the branch as not checked

#### Scenario: Explicit offline stamp
- **WHEN** a user invokes stamp with `--offline`
- **THEN** it returns network-disabled before target, receipt, or ledger mutation and opens no socket

### Requirement: Stamp the exact deterministic target
`forecast-ledger timestamp stamp` SHALL validate the ledger, reconstruct the selected `forecast-envelope/v1` target, and preflight deterministic paths `proofs/targets/<forecast-id>.json` and `proofs/receipts/<forecast-id>.json.ots`. It SHALL accept a bounded `--timeout`, submit the exact target SHA-256 digest concurrently to the four built-in profile calendars, and accept success only when at least two valid calendar branches are obtained. Users and MCP callers MUST NOT be required or permitted to weaken that threshold or supply arbitrary calendar URLs in v1.

The command SHALL combine accepted branches into one deterministic detached receipt, exclusively create or confirm-identical target and receipt artifacts, then atomically change forecast integrity from `unanchored` to `pending`. The pending object SHALL contain scope `forecast-envelope/v1`, the approved canonicalization identifier, relative target path, SHA-256 digest, and exactly one OpenTimestamps entry with the deterministic receipt path and state `pending`. A calendar response or local file timestamp MUST NOT create a verified claim.

#### Scenario: Successful pending stamp
- **WHEN** the minimum number of built-in-profile calendars return valid bounded branches
- **THEN** durable target/receipt artifacts exist and matching pending ledger metadata is committed atomically

#### Scenario: Insufficient calendar success
- **WHEN** fewer than two built-in profile calendars return valid responses
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
`forecast-ledger timestamp upgrade` SHALL require pending matching ledger metadata and artifacts, verify target bytes/digest first, parse the bounded receipt, and query only calendar identities included in the built-in profile for missing attestations. It SHALL merge valid responses deterministically and preserve safe unknown proof nodes byte-for-byte. It SHALL replace the receipt recoverably only when the new receipt is a semantic superset; otherwise it SHALL report `unchanged` or failure.

Upgrade SHALL leave integrity `pending` until Bitcoin block verification succeeds through the built-in public profile or optional Bitcoin Core mode. It MUST NOT set `anchored_before`, block height, `verified_at`, or `verified` merely because a calendar returns a Bitcoin attestation.

#### Scenario: No upgrade available
- **WHEN** every matching built-in-profile calendar reports no new branch
- **THEN** the command returns pending/not-ready, leaves receipt and ledger unchanged, and provides a retry-safe result

#### Scenario: Downgrade response
- **WHEN** a response would discard an existing proof branch or supported attestation
- **THEN** upgrade rejects it and preserves the original receipt

### Requirement: Report timestamp status locally
`forecast-ledger timestamp status` SHALL perform no network request and no mutation. It SHALL report one of `unanchored`, `pending`, `confirmed_unverified`, `verified`, `failed`, or `inconsistent`, together with target/receipt presence, digest agreement, supported attestations, recorded ledger state, and safe next commands. Because forecast integrity is schema-required, there is no separate `missing` integrity state: absent artifacts referenced by pending/verified metadata are `inconsistent`, while an unanchored forecast is `unanchored`. Status SHALL distinguish receipt syntax/attestation presence from a previously recorded Bitcoin verification and SHALL state that v1 does not retain whether the prior source was public or independently operated.

#### Scenario: Bitcoin attestation not checked against block data
- **WHEN** a well-formed receipt contains a supported Bitcoin attestation but the ledger is still pending
- **THEN** status reports `confirmed_unverified`, not verified existence timing

#### Scenario: Missing pending receipt
- **WHEN** ledger metadata is pending but the receipt path is absent
- **THEN** status reports `inconsistent` and returns verification failure without modifying integrity

### Requirement: Verify receipts automatically with an optional independent source
`forecast-ledger timestamp verify` SHALL reconstruct and compare the target before proof verification. By default it SHALL query both public Bitcoin APIs in `opentimestamps-public-v1` for the attested height, require identical block hash and raw 80-byte header observations, validate each response binding, header hash, encoded proof-of-work target, proof operations, and the attestation against the exact target, and report that canonical-chain selection still trusts two third-party public services. One unavailable, malformed, disagreeing, or invalid public response MUST prevent a verified transition; safely established partial observations SHALL remain visible.

An advanced user MAY instead select Bitcoin Core with `--bitcoin-core <rpc-url>` and protected `--bitcoin-auth-file <path>`. Core mode SHALL contact only that explicit RPC URL, SHALL be reported as independently operated verification, and SHALL take precedence over the public profile for that invocation. Credentials MUST NOT be accepted inline in argv URLs, ordinary environment variables, output, or diagnostics. No arbitrary public explorer URL flag SHALL exist in v1.

The verifier SHALL use bounded requests, response sizes, redirects, and chain data. It SHALL derive the protocol's conservative `anchored_before` value and block height only from a successfully verified attestation. Results SHALL include source mode, profile ID or safe Core identity, agreement state, trust limitation, and exact observed block evidence. Because the v1 ledger does not persist verification-source identity, later offline reports MUST state that the prior source is not retained and that the receipt can be reverified.

#### Scenario: Exact proof verification
- **WHEN** both built-in public Bitcoin APIs agree and the locally checked header/proof confirms a supported attestation for the reconstructed target
- **THEN** verification reports the confirmed block evidence and conservative time bound

#### Scenario: Receipt for different bytes
- **WHEN** the receipt is valid for a digest different from the reconstructed target
- **THEN** verification fails cryptographically and performs no ledger transition

#### Scenario: Public sources disagree
- **WHEN** the two built-in public APIs return different block hashes or headers for the attested height
- **THEN** verification returns evidence failure, preserves ledger state, and reports both safe source IDs without selecting either response

#### Scenario: Bitcoin Core override
- **WHEN** valid Core connection and protected authentication options are supplied
- **THEN** the command uses Core instead of public APIs and reports the stronger independently operated source boundary

### Requirement: Commit verified integrity only after all checks
After successful proof verification, `timestamp verify` SHALL atomically change matching pending integrity to `verified`, retain target, every receipt reference, and every `external_anchor` byte-for-byte, set the OpenTimestamps entry to `confirmed`, store the verified conservative `anchored_before` and Bitcoin block height, and set `verified_at` to the exact command time or explicit override. It MUST NOT mutate on network failure, unavailable source, unsupported proof branch, target mismatch, invalid proof, or imported failed integrity.

If the question is resolved and `anchored_before` is not earlier than `outcome_known_at`, the cryptographic timestamp MAY still become verified, but output SHALL warn that it is insufficient to exclude hindsight; layered verification SHALL fail the pre-outcome evidence claim.

#### Scenario: Late but valid timestamp
- **WHEN** Bitcoin evidence is valid but its conservative bound is after the known outcome
- **THEN** integrity records cryptographic verification while output explicitly marks pre-outcome sufficiency as failed

### Requirement: Keep pending and failure semantics honest
Pending/not yet confirmed SHALL return exit `9` when it is the primary result. Network unavailability SHALL return exit `8`; malformed or mismatched proof evidence SHALL return exit `6`; unsafe/missing local files SHALL use the corresponding I/O/not-found categories. A verification failure MUST NOT automatically destroy a previously verified record or rewrite integrity to `failed`; persistent failure-state mutation requires retained evidence and an explicit separately specified repair/review action.

#### Scenario: Temporary Bitcoin source outage
- **WHEN** either required public source or the selected Core source times out
- **THEN** verify returns network failure and leaves pending or verified metadata unchanged

### Requirement: Pass official-client and adversarial conformance gates
Timestamp commands MUST remain unavailable or explicitly experimental until Python-to-Go and Go-to-Python receipts round-trip, supported info/upgrade/verify results match the official client, the built-in profile endpoints and identities have conformance fixtures, dual-public-source and Bitcoin Core tests pass, malformed/oversized/deep proof fuzzing is clean, redirect/timeout/SSRF controls pass, native-platform recovery tests pass, real-calendar nightly tests are stable, and the supported subset receives independent review.

#### Scenario: Unsupported proof operation
- **WHEN** a receipt contains an operation outside the reviewed subset
- **THEN** it is preserved safely for round-trip or rejected explicitly, never ignored while claiming successful verification
