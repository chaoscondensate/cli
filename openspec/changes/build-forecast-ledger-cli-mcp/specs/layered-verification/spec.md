## Purpose

Defines independent verification as separate evidence claims so users are never given a misleading single “verified” answer.

## ADDED Requirements

### Requirement: Verify independent evidence layers
Verification SHALL report structural and semantic validity, target content binding, OpenTimestamps existence timing, sealed reveal validity, and outcome-evidence status as separate named results with evidence and failure reasons.

#### Scenario: Valid data with pending timestamp
- **WHEN** a ledger and target are valid but the OpenTimestamps receipt is still pending
- **THEN** data and content-binding layers pass, existence timing reports pending, and the overall report does not claim full verification

#### Scenario: Valid timestamp with false outcome evidence
- **WHEN** target timing verifies but supplied outcome sources do not establish the stated resolution
- **THEN** existence timing passes while outcome evidence remains unverified or failed

### Requirement: Rebuild targets before proof verification
The verifier SHALL reconstruct the selected forecast target from ledger data and compare its bytes and digest with the retained artifact and ledger metadata before checking the receipt.

#### Scenario: Ledger statement changed after anchoring
- **WHEN** the forecast fields no longer rebuild to the retained target bytes
- **THEN** content binding fails even if the retained target's receipt itself is a valid OpenTimestamps proof

### Requirement: Enforce precedence over known outcomes
For a resolved forecast, existence timing SHALL pass only when the confirmed `anchored_before` bound predates `resolution.outcome_known_at`. A later timestamp MUST be reported as insufficient to exclude hindsight.

#### Scenario: Late timestamp
- **WHEN** a valid receipt anchors the forecast after the recorded outcome became knowable
- **THEN** the proof is reported cryptographically valid but insufficient for pre-outcome verification

### Requirement: Verify revealed forecasts end to end
For a revealed forecast, verification SHALL authenticate and decrypt the retained ciphertext, verify the commitment and bound IDs, compare the decrypted bundle with the public mirror, and ensure the rebuilt target is identical to the originally timestamped sealed envelope.

#### Scenario: Edited revealed mirror
- **WHEN** a public mirror differs from the authenticated decrypted bundle
- **THEN** reveal validity fails even if the ciphertext and timestamp receipt are intact

### Requirement: State security boundaries
Every human and JSON verification report SHALL state that v1 evidence does not by itself prove authorship, ledger completeness, truth of a forecast, exact self-reported forecast time, or the substantive correctness of an outcome source.

#### Scenario: Fully passing technical checks
- **WHEN** all automated layers pass
- **THEN** the result identifies the proven claims and does not add authorship, completeness, or truth claims

### Requirement: Build a portable evidence package
The system SHALL build a local evidence package from an explicitly selected ledger, its exact target artifacts, receipts, and any explicitly selected disclosed reveal material. The package SHALL include a deterministic manifest of stable relative paths and SHA-256 digests, MUST exclude keys, credentials, secret paths, and unrevealed sealed plaintext, and MUST NOT require a source-control repository, hosting service, or network access.

#### Scenario: Build from a standalone ledger
- **WHEN** a user runs `chaos publish build --file ledger.yaml --output evidence` for a ledger stored outside source control
- **THEN** the command creates a complete local evidence package without reading source-control metadata or contacting a remote service

#### Scenario: Repeat package build
- **WHEN** the same evidence set is built on two supported platforms
- **THEN** the manifest bytes and digest are identical and contain no secret or machine-specific path

### Requirement: Verify a retained package locally
Given an evidence package, `chaos publish verify` SHALL require an explicit `--file` for the ledger inside the package and an explicit `--manifest` for the package manifest. It SHALL check every manifest path and digest before running the applicable content, reveal, and existence layers. Package verification SHALL work from local files without requiring the original authoring location, a source-control repository, or a hosting service.

#### Scenario: Offline package verification
- **WHEN** a verifier runs `chaos publish verify --file evidence/ledger.yaml --manifest evidence/manifest.json` and has the Bitcoin verification source required by OpenTimestamps
- **THEN** manifest integrity plus the applicable content, reveal, and existence layers can be checked without contacting the package's original publication location
