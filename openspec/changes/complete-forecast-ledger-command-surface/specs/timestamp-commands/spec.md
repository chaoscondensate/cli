## Purpose

Defines the complete RFC 3161 command lifecycle and portable evidence behavior shared by CLI and MCP for exact Forecast Ledger v1.2.0 forecast targets.

## ADDED Requirements

### Requirement: Use only the v1.2.0 RFC 3161 profile
The application SHALL create and verify only RFC 3161 timestamp requests and responses whose message imprint is SHA-256 over the exact `forecast-envelope/v1` target. It SHALL reject OTS receipts and all other timestamp protocols and SHALL NOT read, upgrade, translate, package, or report them as equivalent evidence.

#### Scenario: OTS receipt is supplied
- **WHEN** a timestamp or publication operation receives an `.ots` receipt or an `opentimestamps` timestamp object
- **THEN** it rejects the unsupported input before mutation or network access and does not invoke an OTS parser

#### Scenario: RFC 3161 uses the exact target
- **WHEN** a timestamp request is created for a selected forecast
- **THEN** its SHA-256 message imprint equals the digest of the exact canonical target bytes recorded by `integrity.target`

### Requirement: Require explicit bounded TSA inputs
`timestamp stamp` and the equivalent MCP operation SHALL require an explicit public HTTPS `tsa_url` and an explicit retained PEM CA-bundle file inside the allowed ledger root. The application SHALL accept no built-in TSA, downloaded endpoint profile, environment-selected endpoint, arbitrary request header, credential-bearing URL, private or link-local destination, cross-origin redirect, or unbounded response. A caller SHALL repeat the operation with a different TSA URL to add redundant independent timestamp evidence.

#### Scenario: Explicit TSA stamp
- **WHEN** a caller supplies a safe TSA URL and an accessible retained CA bundle
- **THEN** the application contacts only that TSA within the configured request timeout and byte limits

#### Scenario: TSA input is absent
- **WHEN** a caller omits either the TSA URL or CA bundle
- **THEN** the operation fails as usage before generating entropy, creating artifacts, mutating the ledger, or opening a socket

#### Scenario: TSA destination is unsafe
- **WHEN** the TSA URL contains credentials, is not public HTTPS, resolves to a prohibited destination, or redirects outside its validated origin
- **THEN** the request is rejected without following the unsafe destination or exposing it in unrestricted error text

### Requirement: Create portable request and response evidence
`timestamp stamp` SHALL validate the ledger, reconstruct or confirm the selected target, create a fresh nonce-bearing RFC 3161 SHA-256 request with certificate inclusion requested, submit the binary `.tsq` using the RFC media types, and retain the exact `.tsq`, `.tsr`, and CA-bundle path. Artifact paths SHALL be safe, relative, deterministic for the forecast and TSA identity, and non-colliding across multiple TSAs. A retry SHALL reuse a complete matching artifact set or fail safely; it MUST NOT overwrite different durable evidence.

#### Scenario: Successful TSA response is retained
- **WHEN** the TSA returns a bounded syntactically valid RFC 3161 response
- **THEN** the exact target, request, response, and CA-bundle reference are retained and the corresponding v1.2.0 timestamp object records their safe relative paths and TSA URL

#### Scenario: Dry-run performs no timestamp effects
- **WHEN** stamp runs with global dry-run
- **THEN** it validates inputs and reports planned target, request, response, and ledger effects without generating a nonce, opening a socket, or writing a file

#### Scenario: Offline stamp is rejected without effects
- **WHEN** stamp is explicitly offline
- **THEN** it returns `network_disabled` with exit `8` before generating a nonce, opening a socket, creating an artifact, or changing the ledger

### Requirement: Verify every RFC 3161 evidence layer locally
Timestamp verification SHALL parse bounded request and response bytes and SHALL separately verify successful TSA status, request nonce and message-imprint binding, target SHA-256 binding, CMS signed attributes and signature, signer certificate timestamping usage, certificate chain against the retained CA bundle at `gen_time`, supported algorithms, and declared `gen_time`, policy OID, serial number, and hash algorithm. Verification SHALL require no TSA, Git host, Bitcoin node, blockchain API, system trust-store mutation, or network request.

#### Scenario: Complete response verifies
- **WHEN** the saved response is valid for the exact saved request and target, its signer chains to the retained CA bundle, and its metadata matches the ledger
- **THEN** existence timing passes with reason `timing.rfc3161_verified` and reports the verified `gen_time`, policy OID, serial number, and safe TSA identity

#### Scenario: Request or target binding fails
- **WHEN** the response nonce or message imprint differs from the saved request or exact target
- **THEN** existence timing fails with a specific binding reason, application category `verification`, and CLI exit `6`

#### Scenario: Signature or trust chain fails
- **WHEN** the CMS signature, timestamping certificate usage, signing algorithm, or retained trust chain is invalid
- **THEN** existence timing fails with a specific cryptographic or trust reason and MUST NOT report the declared `gen_time` as verified timing

#### Scenario: Stored metadata differs
- **WHEN** the cryptographically verified response metadata differs from the ledger's `gen_time`, policy OID, serial number, TSA URL association, or hash algorithm
- **THEN** verification fails rather than silently correcting or trusting the stored fields

### Requirement: Keep the RFC 3161 command surface minimal
The CLI SHALL expose `timestamp stamp`, `timestamp status`, and `timestamp verify` with leaf-level `--file`, `--question`, and `--forecast`. Stamp SHALL additionally expose `--tsa-url`, `--ca-bundle`, and `--offline`; status and verify SHALL be local. MCP SHALL expose equivalent transport-neutral operations and fields subject to configured ledger roots, server-wide write mode, and server-wide offline mode. No surface SHALL expose `timestamp upgrade`, calendars, calendar thresholds, OTS profiles, explorers, Bitcoin Core, Bitcoin credentials, block heights, or blockchain-specific online rechecks.

#### Scenario: Help contains only RFC 3161 controls
- **WHEN** a user inspects CLI help, MCP tools, generated input schemas, and command examples
- **THEN** the timestamp surface contains the RFC 3161 operations and inputs and contains none of the removed OTS or Bitcoin controls

#### Scenario: Local status does not mutate
- **WHEN** timestamp status reads a retained RFC 3161 artifact set
- **THEN** it reports local request, response, trust-material, and metadata state without a network request or ledger mutation

### Requirement: Commit only honest RFC 3161 state
The application SHALL record timestamp `state: verified` and integrity `status: verified` only after all required local RFC 3161 checks pass. A retained response not yet successfully verified SHALL remain `pending`; a network error or malformed response SHALL NOT become verified timing. Adding another pending TSA response SHALL NOT erase an already verified response, and integrity SHALL remain verified when at least one retained timestamp independently verifies.

#### Scenario: Stamp completes verification
- **WHEN** a TSA returns a response that passes every required local check
- **THEN** stamp atomically retains the evidence and records its verified metadata without requiring a later upgrade operation

#### Scenario: Response is retained but not verified
- **WHEN** a bounded response and request are safely retained but local verification cannot establish valid timing
- **THEN** the result preserves a recoverable pending report and does not claim verified timing

#### Scenario: Second TSA remains independent
- **WHEN** a forecast already has one verified timestamp and stamp is repeated for a different TSA
- **THEN** the new request and response use distinct artifact paths and the earlier verified evidence remains unchanged

### Requirement: Verify multiple TSAs conservatively
Existence timing SHALL pass when at least one retained RFC 3161 timestamp independently verifies. It SHALL fail only when every applicable timestamp is completely checked and every one fails. If none verifies and at least one timestamp is pending or cannot be completely checked, the layer SHALL remain pending or not checked instead of claiming cryptographic failure. For a resolved forecast, at least one verified `gen_time` MUST predate `outcome_known_at` for the timing layer to exclude hindsight.

#### Scenario: One of two TSAs verifies
- **WHEN** one retained TSA response fails and a second independently verifies
- **THEN** existence timing passes using the verified response while preserving the failed branch observation

#### Scenario: Timestamp follows the known outcome
- **WHEN** every otherwise valid RFC 3161 response has `gen_time` at or after `outcome_known_at`
- **THEN** the timing layer fails to exclude hindsight and does not claim that the forecast predated the outcome

### Requirement: Package complete RFC 3161 evidence
Publication build SHALL include every referenced exact target, `.tsq` request, `.tsr` response, and retained CA bundle under distinct transport-neutral manifest roles. Publication verify SHALL recompute package-file digests and run the same local RFC 3161 checks against packaged bytes by default. It SHALL NOT require or offer a Bitcoin, blockchain, calendar, TSA, or other online recheck to establish packaged timing.

#### Scenario: Build packages a verified forecast
- **WHEN** publication build selects a forecast with RFC 3161 evidence
- **THEN** the package contains the ledger, exact target, every referenced request and response, every referenced CA bundle, and manifest digests for all files

#### Scenario: Package verifies without network
- **WHEN** publication verify receives a complete untampered RFC 3161 package on a machine without network access
- **THEN** it verifies manifest integrity, target binding, timestamp signatures and trust chains, and declared metadata from packaged evidence alone

#### Scenario: Required timestamp artifact is absent
- **WHEN** a referenced request, response, target, or CA bundle is missing or differs from its manifest digest
- **THEN** package verification fails the affected layer and does not fall back to a remote service

### Requirement: Conformance is independent and bounded
The RFC 3161 parser and verifier SHALL enforce explicit byte, certificate, chain-depth, attribute, and algorithm bounds; reject malformed, ambiguous, trailing, unsupported, or adversarial ASN.1/CMS input; and pass deterministic interoperability fixtures created and verified by an independent RFC 3161 implementation. Normal tests and releases SHALL not depend on a live public TSA.

#### Scenario: OpenSSL interoperability fixtures pass
- **WHEN** the conformance suite exchanges SHA-256 requests and responses with pinned OpenSSL-generated fixtures
- **THEN** request binding, response parsing, signature verification, metadata extraction, and negative mutations agree for the supported profile

#### Scenario: Malformed response is bounded
- **WHEN** a response exceeds limits or contains malformed, ambiguous, unsupported, or trailing ASN.1/CMS data
- **THEN** it is rejected within bounded resources without panic, partial success, or unsafe diagnostic output
