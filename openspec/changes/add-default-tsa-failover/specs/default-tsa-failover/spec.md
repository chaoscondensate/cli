## Purpose

Defines a usable-by-default RFC 3161 stamping path with qualified built-in
provider profiles, transport-policy enforcement, retained trust material,
interoperable local verification, and portable package behavior shared by CLI
and MCP.

## ADDED Requirements

### Requirement: Stamp automatically with the qualified built-in catalog
When neither a built-in provider nor custom TSA inputs are supplied,
`timestamp stamp` and `timestamp_stamp` SHALL select `auto` and try every
provider in the released built-in catalog in its fixed order. Each attempt SHALL
use a fresh nonce-bearing request and SHALL stop after the first response passes
the complete local RFC 3161 verification profile. The current released catalog
SHALL contain only `freetsa`, so automatic mode currently makes at most one
provider request. Later releases SHALL add or reorder providers only after the
same qualification and provenance gates pass.

#### Scenario: Current automatic stamp verifies
- **WHEN** stamp is invoked without TSA arguments and the FreeTSA response passes every local verification check
- **THEN** the operation commits the FreeTSA evidence and makes exactly one TSA request

#### Scenario: A later catalog contains more than one provider
- **WHEN** a released catalog has multiple qualified entries and an earlier attempt fails with a fallback-eligible provider error
- **THEN** automatic mode tries the next catalog entry with a fresh request within the caller's overall deadline

#### Scenario: A local error prevents provider attempts
- **WHEN** stamp cannot validate or confine the ledger, target, destination, or local trust artifact
- **THEN** it fails before opening a provider connection and performs no mutation

### Requirement: Keep automatic selection transactional and honestly reported
Automatic mode SHALL commit only a locally verified successful provider's
target, request, response, materialized CA bundle, and ledger update. It SHALL
not retain requests, responses, CA files, or ledger entries from failed
attempts. When no provider verifies, the operation SHALL leave the ledger and
artifact tree unchanged and return an ordered, bounded attempt summary with
safe provider IDs and stable reason codes. If every attempt is unavailable it
SHALL return `network` with CLI exit `8`; if at least one received response is
cryptographically or structurally invalid it SHALL return `verification` with
CLI exit `6`. The result SHALL never claim pending or verified timing without
retained evidence.

#### Scenario: Current built-in provider is unavailable
- **WHEN** the FreeTSA automatic attempt times out or returns a provider-availability failure
- **THEN** stamp returns `network`, reports the safe provider outcome, and leaves no timestamp mutation or artifact

#### Scenario: A built-in provider returns an invalid token
- **WHEN** a built-in provider response fails local verification and no later catalog provider verifies
- **THEN** stamp returns `verification`, preserves the specific verification reason in the attempt summary, and commits no evidence

#### Scenario: Failed provider bodies remain private
- **WHEN** automatic selection reports one or more unsuccessful attempts
- **THEN** normal, plain, JSON, verbose, and MCP output contains no raw request, response, certificate, resolved IP address, or unrestricted remote error text

### Requirement: Ship exact qualified provider profiles and materialize trust
The current built-in catalog SHALL contain only the stable ID `freetsa` at the
exact endpoint `https://freetsa.org/tsr`. The profile SHALL contain an exact,
source-attributed PEM CA bundle pinned by digest in the released source. A
successful built-in stamp SHALL write the exact embedded bundle to a
deterministic ledger-relative `trust/` path and record that path and endpoint
in the v1.2.0 timestamp object. Later verification and publication SHALL use
the retained bytes only, with no system root, catalog lookup, certificate
download, or network request.

#### Scenario: Built-in evidence verifies after the catalog changes
- **WHEN** a package created with a built-in provider is verified by a later binary whose catalog or embedded bundle has changed
- **THEN** verification uses the package's retained CA bytes and recorded TSA URL rather than the later catalog

#### Scenario: Embedded trust material collides with different bytes
- **WHEN** the deterministic trust path already contains bytes that differ from the selected profile bundle
- **THEN** stamp fails as a local conflict before contacting any provider and does not overwrite the file

### Requirement: Enforce HTTPS and HTTP as explicit built-in transport profiles
Every built-in provider profile SHALL declare one exact normalized HTTPS or HTTP
endpoint. HTTPS SHALL be the default transport policy. A built-in HTTP profile
SHALL be releasable only when first-party operator material publishes that
exact endpoint, the provider passes the usage-policy gate, and the compiled
profile explicitly opts into HTTP. The client SHALL apply public-destination
resolution, request and response bounds, deadline limits, RFC media types,
nonce and message-imprint binding, CMS signature verification, retained-chain
verification, and a no-redirect policy to every built-in transport. The current
catalog SHALL contain no HTTP profile. Custom TSA URLs SHALL remain public
HTTPS without credentials, query, fragment, prohibited destinations, or
cross-origin redirects; no caller-controlled input SHALL enable HTTP.

#### Scenario: Current built-in HTTPS provider responds
- **WHEN** the exact FreeTSA HTTPS endpoint returns a timestamp response
- **THEN** the response is accepted only after the complete local cryptographic checks pass

#### Scenario: A qualified built-in HTTP profile is released later
- **WHEN** automatic or named selection constructs an HTTP request from that exact compiled provider profile
- **THEN** the client may send it only to the profile's exact origin and path, follows no redirect, and authenticates the response cryptographically

#### Scenario: Custom HTTP TSA is supplied
- **WHEN** a caller supplies an `http://` URL through the custom TSA input
- **THEN** the operation rejects it before entropy, file creation, ledger mutation, or network access

#### Scenario: Built-in endpoint redirects
- **WHEN** a built-in HTTP or HTTPS endpoint returns any redirect
- **THEN** the client rejects that attempt without following the redirect and applies normal automatic-selection rules

### Requirement: Preserve explicit provider and custom TSA control
CLI SHALL expose optional `--tsa-provider` with current values `auto` or
`freetsa`; MCP SHALL expose the equivalent optional `tsa_provider`. Omitting
all TSA selection inputs SHALL mean `auto`. Selecting `freetsa` SHALL contact
only that provider and materialize its embedded CA bundle. The existing custom
`tsa_url` and `ca_bundle` inputs SHALL remain an all-or-nothing pair and SHALL
be mutually exclusive with `tsa_provider`. Explicit custom stamping SHALL
retain its existing single-provider pending and retry semantics. A provider ID
not present in the released catalog SHALL be rejected as usage.

#### Scenario: Caller selects FreeTSA
- **WHEN** a caller selects provider `freetsa`
- **THEN** only the exact FreeTSA profile is used and no other provider is contacted

#### Scenario: Caller supplies a custom TSA pair
- **WHEN** a caller supplies both a valid custom TSA URL and retained CA-bundle path
- **THEN** the operation uses only those inputs and does not consult or materialize a built-in profile

#### Scenario: Caller mixes provider and custom inputs
- **WHEN** a caller supplies `tsa_provider` together with either custom TSA field, or supplies only one custom field
- **THEN** the operation fails as usage before entropy, writes, ledger mutation, or network access

#### Scenario: Automatic mode is dry-run or offline
- **WHEN** automatic stamp runs with dry-run or server-wide/CLI offline mode
- **THEN** dry-run reports the ordered current catalog plan without entropy, writes, or sockets, while offline returns `network_disabled` before attempting a provider

### Requirement: Verify interoperable ESS and strong signer profiles
RFC 3161 verification SHALL keep the forecast message imprint fixed to SHA-256
and SHALL accept an otherwise valid token that binds its signer through either
ESS `SigningCertificate` or `SigningCertificateV2`. ESS v1's SHA-1 certificate
hash SHALL be used only as the signed certificate identifier; SHA-1 SHALL remain
forbidden for the message imprint, CMS signer digest, signature algorithm, and
certificate signatures. The verifier SHALL require an unambiguous signer and
certificate match, validate every present ESS binding, and reject missing,
duplicate, conflicting, malformed, or non-matching bindings. CMS signer digests
SHA-256, SHA-384, and SHA-512 SHALL be accepted with the existing strong key,
signature, EKU, chain, nonce, and target-binding checks.

#### Scenario: Valid SigningCertificate v1 response verifies
- **WHEN** a bounded token has one valid ESS v1 binding, a strong CMS signature, and a valid retained trust chain for the exact request and target
- **THEN** local verification passes even though the token has no SigningCertificateV2 attribute

#### Scenario: Valid SigningCertificateV2 response verifies
- **WHEN** a bounded token has one valid ESS v2 binding and passes every other supported-profile check
- **THEN** local verification continues to pass

#### Scenario: ESS versions identify different certificates
- **WHEN** a token contains v1 and v2 signer bindings that do not identify the same unique certificate
- **THEN** verification fails the signer-certificate profile and does not claim verified timing

#### Scenario: Strong SHA-512 CMS signer verifies a SHA-256 imprint
- **WHEN** the RFC 3161 message imprint is SHA-256 and an otherwise valid TSA token uses a supported SHA-512 CMS signer digest
- **THEN** verification passes without treating the two digest roles as inconsistent

### Requirement: Report the actual RFC 3161 failure layer
The verifier SHALL distinguish request status, policy, nonce, message-imprint,
target, ESS signer binding, algorithm, CMS signature, certificate profile, and
retained trust-chain failures with stable reason codes. Failure to understand or
validate an ESS signer attribute SHALL not be reported as
`rfc3161.binding_mismatch` when the request imprint and nonce match.

#### Scenario: ESS v1 is malformed but request binding matches
- **WHEN** nonce, policy, and message imprint match but the ESS v1 certificate identifier is malformed
- **THEN** verification reports a signer-certificate profile reason rather than `rfc3161.binding_mismatch`

#### Scenario: Nonce differs
- **WHEN** the signed response nonce differs from the request nonce
- **THEN** verification reports the specific request-binding reason and does not continue as a valid signer-profile case

#### Scenario: CMS signer digest is weak
- **WHEN** a response uses SHA-1 or another unsupported CMS signer digest
- **THEN** verification reports `rfc3161.algorithm_unsupported` and does not claim verified timing

### Requirement: Verify publication artifacts from the package root
Publication build SHALL continue to place the byte-exact ledger below
`ledger/` and its referenced `proofs/` and `trust/` files at their stable paths
from the package root. Publication verify in both CLI and MCP SHALL perform
schema and semantic preflight, manifest verification, and evidence verification
with ledger artifact paths resolved against that package root, not the
directory containing the packaged ledger. The package-aware root SHALL remain
confined to the manifest directory and SHALL not weaken path traversal,
symlink, collision, unexpected-file, or digest checks.

#### Scenario: Timestamped package verifies in its emitted layout
- **WHEN** publish build emits `ledger/<file>`, `proofs/...`, `trust/...`, and `manifest.json` for a timestamped forecast
- **THEN** the unmodified package verifies through both the CLI and MCP package-verification surfaces

#### Scenario: Artifact exists only beside the package ledger
- **WHEN** a referenced proof is moved under `ledger/` and no longer exists at its manifest path from the package root
- **THEN** package verification rejects the modified package rather than changing its artifact-root convention

#### Scenario: Package path escapes the manifest directory
- **WHEN** a ledger or manifest path attempts to traverse or follow a link outside the package root
- **THEN** package verification rejects it before reading the external target

### Requirement: Removed commands remain usage errors under help
An absent or removed CLI subcommand SHALL produce a bounded usage diagnostic and
CLI exit `2` whether or not the invocation includes `--help`. Command
discovery or help lookup failure SHALL never be presented as an internal error
or exit `3`.

#### Scenario: Removed timestamp upgrade requests help
- **WHEN** a caller invokes `forecast-ledger timestamp upgrade --help`
- **THEN** the CLI reports that `upgrade` is not a timestamp command and exits `2` without printing `internal error`

#### Scenario: Removed timestamp upgrade runs without help
- **WHEN** a caller invokes `forecast-ledger timestamp upgrade`
- **THEN** it returns the same error category and exit code as the help form

### Requirement: Qualify and maintain every built-in provider
A release SHALL include a built-in provider only when checked-in provenance
identifies operator documentation for the exact endpoint and CA material,
affirmatively substantiates unauthenticated timestamping of arbitrary forecast
imprints, pins all incorporated bytes and digests, and demonstrates the
supported response profile through deterministic fixtures and an independent
RFC 3161 verifier. A successful anonymous response SHALL not prove permission.
Provider order SHALL not be described as an SLA, endorsement, legal
qualification, organizational independence, or guarantee of free or permanent
service. Normal tests and releases SHALL not require a live TSA.

#### Scenario: Provider terms do not cover the product use
- **WHEN** official provider material restricts the TSA to an incompatible use or does not affirmatively substantiate the built-in use
- **THEN** that provider is excluded from the released catalog even if its endpoint accepts a request

#### Scenario: Provider certificate profile rotates
- **WHEN** a health canary observes new signer or chain material for a built-in provider
- **THEN** maintainers qualify and release an updated pinned profile while existing retained evidence remains locally verifiable from its saved bundle

#### Scenario: Live provider is unavailable during normal tests
- **WHEN** unit, conformance, documentation, snapshot, or release tests run without public TSA access
- **THEN** they use deterministic checked-in fixtures and do not fail solely because a live provider is unavailable
