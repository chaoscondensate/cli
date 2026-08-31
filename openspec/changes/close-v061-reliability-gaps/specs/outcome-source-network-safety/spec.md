## Purpose

Constrains optional outcome-source retrieval so URL validation and the actual network connection enforce the same public-destination policy.

## ADDED Requirements

### Requirement: Permit only bounded public HTTPS outcome sources
Optional outcome-source retrieval SHALL accept only credential-free HTTPS URLs allowed by the closed source profile. Every resolved address MUST be a permitted public unicast destination; private, loopback, link-local, multicast, unspecified, carrier-grade NAT, benchmarking, documentation, and other reserved ranges MUST be rejected before any request bytes are sent. The transport MUST NOT use environment-selected proxies or caller-controlled proxy, header, resolver, or dial settings.

#### Scenario: Source resolves to carrier-grade NAT
- **WHEN** an outcome-source hostname resolves to an address in `100.64.0.0/10`
- **THEN** source checking reports the source as not safely reachable and opens no connection

#### Scenario: Mixed public and prohibited answers
- **WHEN** one DNS answer is public and another answer for the same source is prohibited
- **THEN** the complete destination is rejected rather than selecting only the public answer

### Requirement: Connect only to validated addresses
The transport SHALL connect directly to the exact approved addresses returned by the bounded resolution step while retaining the original hostname for HTTPS certificate verification and the HTTP Host value. It MUST NOT perform a second uncontrolled hostname resolution between policy validation and dialing. Every new connection and redirect SHALL remain subject to the same destination policy.

#### Scenario: DNS answer changes after validation
- **WHEN** a test resolver would return a public address during validation and a loopback or private address on a later lookup
- **THEN** the request either dials only the originally approved public address or is rejected and never connects to the later prohibited address

#### Scenario: HTTPS identity remains the source hostname
- **WHEN** the transport dials an approved numeric address for an outcome-source hostname
- **THEN** TLS certificate verification and the HTTP request continue to authenticate and name the original source hostname

### Requirement: Redirects cannot weaken destination policy
Outcome-source redirects SHALL retain the existing bounded count, preserve the permitted HTTPS and credential rules, and reapply the closed redirect and destination policy before following. A redirect MUST NOT cause a connection to a prohibited address or an origin forbidden by that policy.

#### Scenario: Redirect reaches a private destination
- **WHEN** an otherwise acceptable source redirects to a hostname or address that resolves to a prohibited destination
- **THEN** the redirect is rejected before the prohibited connection is opened

### Requirement: Network-safety tests are deterministic
Destination-policy tests SHALL use injected resolvers and dialers or local deterministic transports and MUST cover IPv4, IPv6, IPv4-mapped IPv6, mixed answers, redirects, rebinding attempts, cancellation, deadlines, and response-size limits without contacting a public service.

#### Scenario: Rebinding regression suite runs offline
- **WHEN** normal tests and release checks exercise outcome-source destination policy
- **THEN** they prove the dialed address set and rejection decisions without depending on public DNS or network availability
