## MODIFIED Requirements

### Requirement: Separate observation acquisition from proof comparison
The system SHALL treat obtaining an RFC 3161 response during stamp and locally verifying that response as separate results. A TSA timeout, refusal, unavailable response, or bounded transport failure MUST NOT be reported as cryptographic proof failure. A verification failure MAY be reported only after the saved request, response, target, and trust material were completely evaluated and a specific binding, signature, chain, algorithm, metadata, or chronology check failed.

For evidence with multiple TSA responses, verification SHALL pass if any applicable response verifies. It SHALL report an overall timing failure only if every applicable response was completely checked and every response failed. If no response verifies and any response is still pending or could not be completely checked, the result SHALL remain pending or not checked rather than claiming that all proof branches failed.

#### Scenario: One required public source is unavailable
- **WHEN** `timestamp stamp` has valid local inputs but the explicitly selected TSA times out, refuses the request, or returns an unavailable response
- **THEN** the timing check is `not_checked` with reason `timing.tsa_unavailable`, the application category is `network`, the CLI exits `8`, and no verified timing is recorded

#### Scenario: Public sources cannot form one accepted observation
- **WHEN** required local request, target, response, or retained trust material is unavailable or unreadable after a safe partial report can be constructed
- **THEN** the timing check is `not_checked` with a specific incomplete-evidence reason and MUST NOT claim cryptographic mismatch

#### Scenario: Proof mismatches a complete observation
- **WHEN** a complete local evaluation establishes that the response does not bind the request or target, has an invalid signature or chain, uses an unsupported algorithm, or disagrees with declared metadata
- **THEN** the timing check is `fail` with a specific RFC 3161 reason, the application category is `verification`, and the CLI exits `6`

#### Scenario: Later proof branch verifies
- **WHEN** an earlier retained TSA response cannot be checked or fails but a later independent response verifies completely
- **THEN** the timing check passes using the verified response and does not emit an overall unavailable or mismatch failure

### Requirement: Preserve safe partial timestamp reports
Expected non-success outcomes after RFC 3161 evidence has been safely loaded and evaluated SHALL preserve a transport-neutral report. The report MUST include the selected question and forecast IDs, target identity, request and response identities, local timestamp state, timing-check state and reason codes, verified `gen_time` when known, policy OID and serial number when safely parsed, and a bounded request summary when a TSA was contacted. It MUST NOT include raw remote bodies, credentials, private forecast data, unrestricted certificate content, or unrestricted underlying error text.

CLI human, plain, and JSON modes SHALL write this primary report to stdout before returning its stable non-zero application exit. MCP SHALL return the same public fields as a recoverable tool outcome. Failures that occur before a safe report exists SHALL continue to use the ordinary application error contract.

#### Scenario: JSON source-outage report
- **WHEN** `forecast-ledger --json timestamp stamp` cannot obtain a response from the selected TSA after safely building a report
- **THEN** stdout contains one stable JSON result with `not_checked`, `timing.tsa_unavailable`, and bounded request-summary data, stderr contains no duplicate JSON result, and the process exits `8`

#### Scenario: Source identity is safe to disclose
- **WHEN** a TSA request or retained response produces a safe partial report
- **THEN** the report identifies the TSA and failed check only through ledger-recorded public identity and safe artifact paths without emitting raw ASN.1, certificate dumps, remote bodies, credentials, or private forecast content

#### Scenario: MCP observation failure
- **WHEN** the MCP `timestamp_verify` tool encounters the same complete cryptographic failure
- **THEN** it returns a recoverable structured tool outcome with fields and classification equivalent to the CLI result and keeps the MCP session alive

### Requirement: Keep adapters and public guidance aligned
CLI and MCP adapters SHALL derive outcome names, reason codes, result fields, and application categories from the same service results. Human, plain, JSON, generated schemas, help, and maintained verification, timestamp, publication, security, MCP, error/exit, and documentation-baseline material SHALL describe `no_evidence`, TSA network failures, pending evidence, and RFC 3161 verification failures consistently.

Regression tests MUST use deterministic local TSA and cryptographic fixtures; release verification MUST NOT depend on current availability of a public TSA, Bitcoin service, calendar, explorer, or blockchain node.

#### Scenario: CLI and MCP parity fixture
- **WHEN** the same deterministic RFC 3161 fixture is exercised through CLI and MCP
- **THEN** both adapters expose equivalent state, reasons, safe TSA and artifact identity, request counts, verified metadata, and application category

#### Scenario: Public-source outage regression
- **WHEN** generated contracts, tests, help, documentation, and result schemas are checked after the cutover
- **THEN** none contains Bitcoin-observation outcomes, OTS profile fields, block heights, calendar source IDs, or the message `Bitcoin evidence did not verify`
