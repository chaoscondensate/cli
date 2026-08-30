# RFC 3161 provider qualification and rotation

<!-- doc-metadata
coverage: development
reviewed: 2026-08-30
owner: security
generated: false
security-critical: true
prerequisites: ../how-to/timestamp-forecasts.md
next: releasing.md
-->

This page records the release inputs for built-in RFC 3161 providers. It is not
an endorsement, legal qualification, availability promise, or claim that two
services are organizationally independent. Endpoint reachability alone is not
permission to include a service.

## Current catalog

| Provider | State | Exact endpoint | Transport | Review result |
| --- | --- | --- | --- | --- |
| FreeTSA | admitted | `https://freetsa.org/tsr` | HTTPS | First-party instructions document anonymous RFC 3161 requests over hashes of arbitrary files. |
| DigiCert | unresolved | not shipped | HTTP candidate | The public endpoint and chain are documented, but arbitrary unauthenticated forecast-imprint use is not affirmatively permitted. |
| GlobalSign `r45standard` | excluded | not shipped | HTTP candidate | The reviewed endpoint and subscriber terms are tied to code-signing customers and certificates. |
| Sectigo public code-signing TSA | excluded | not shipped | HTTPS candidate | The reviewed code-signing policy limits the service to the associated code-signing use. |

Review date: 2026-08-30.

The current catalog contains only FreeTSA. Automatic selection therefore makes
at most one provider request. The provider model supports exact compiled HTTPS
and HTTP profiles, but no HTTP provider ships in the current catalog. Custom
TSA input remains public HTTPS-only.

## FreeTSA qualification

Primary sources:

- [service instructions, endpoint, certificate downloads, and rotation notice](https://freetsa.org/index_en.php);
- [FreeTSA certification practice statement](https://www.freetsa.org/freetsa_cps.html);
- [operator CA certificate](https://freetsa.org/files/cacert.pem); and
- [operator TSA certificate](https://freetsa.org/files/tsa.crt).

The service page documents generating a request from a file hash and posting it
without an account, client certificate, API key, or custom header. It lists
SHA-256 as a supported request digest. The instructions invite project and
company users to contact the operator for specific requirements; they do not
publish a numeric request limit. The separate URL-timestamp feature says not to
abuse the service.

FreeTSA is a best-effort provider. The CPS says certificate-status services are
available continuously “if possible.” It publishes no measurable TSA SLA,
formal availability history, redundancy topology, succession plan, or numeric
rate policy. The reviewed material identifies the FreeTSA brand, a personal
contact, and Wuerzburg, Germany, but no legal entity or governing agreement.
No independent TSA audit report or HSM assurance was found in the reviewed
first-party material. These are documentation limits, not proof that a control
does not exist.

FreeTSA says it does not request or save user information. HTTPS protects the
request imprint and nonce from ordinary network observers, but the operator can
still observe the source address, request time, imprint, nonce, and traffic
pattern. The no-logging statement is operator-published and not treated as an
audited guarantee.

## Retained trust material

The exact downloaded files were retrieved on 2026-08-30. Their published file
digests matched the retrieved bytes.

| Role | Source-file SHA-256 | Certificate SHA-256 fingerprint | Validity | Profile |
| --- | --- | --- | --- | --- |
| FreeTSA root CA | `2151b61137ffa86bf664691ba67e7da0b19f98c758e3d228d5d8ebf27e044438` | `a6379e7cecc05faa3cbf076013d745e327bbbaa38c0b9af22469d4701d18aabc` | 2016-03-13 to 2041-03-07 | self-signed RSA 4096, CA true, SHA-512 certificate signature |
| FreeTSA TSA signer | `8bfb0305bb64e2571ca507552ef3245cb1c2fee8728e0ff8689225081ea13467` | `32e841a95cc1164101ffde41298ef2fc75c1c4372ef095e88a6bbd47dfb191fc` | 2026-02-15 to 2040-02-02 | ECDSA P-384, critical timestamping EKU, SHA-512 CA signature |

The embedded bundle is the exact operator root file at
`internal/timestamp/rfc3161/providers/freetsa/ca.pem`. A successful stamp
materializes those bytes beside the ledger. Verification checks the signer
chain at the token's `gen_time`; it never replaces retained bytes with a newer
catalog entry or system trust store.

The service documents a signer transition from its 2016–2026 certificate to
the current 2026–2040 certificate and retains the expired signer for old
evidence. This is useful continuity evidence but not a future SLA.

## Deterministic interoperability fixture

The synthetic target contains no private forecast data. The fixture was
requested once on 2026-08-30 and is checked in so ordinary tests do not contact
FreeTSA.

| Artifact | SHA-256 |
| --- | --- |
| target | `0ac2fd85615071ded99db80f41e33d0ac879af474c7a5b39793c95bfb925fc5f` |
| request | `de76a83e68832390a86ffabd8c8b1227fb44259caa2b5af54e431b72a9f85611` |
| response | `af9cfc38e47957ab71e85ae88baf2b5cbe9acdfcd645f16b46099960717dd4a4` |

The response has a SHA-256 message imprint, a matching nonce, policy
`1.2.3.4.1` (displayed by OpenSSL as `tsa_policy1`), generation time
`2026-08-30T10:02:54Z`, one ESS `SigningCertificate` v1 binding, SHA-512 CMS
digest, and ECDSA-with-SHA-512 signature. OpenSSL 3.6.0 independently reports
`Verification: OK` for the retained target, request, response, root, and signer
certificate.

## Admission and rotation procedure

Before adding or updating a profile:

1. Confirm operator identity, exact endpoint, transport, permitted use, media
   types, rate guidance, availability language, and limitations from current
   first-party material.
2. Retrieve CA and signer material from an operator-controlled source, inspect
   constraints and validity, and pin exact source-file and certificate digests.
3. Submit one non-secret synthetic request, retain the exact response, verify it
   with the candidate Go verifier and pinned OpenSSL, and record ESS, digest,
   signature, policy, and chain profiles.
4. Test the transport profile separately. HTTP requires an exact compiled
   origin/path, no redirects, public-address checks, complete cryptographic
   response authentication, and explicit privacy documentation.
5. Update this page, fixtures, catalog, tests, notices, security guidance, and
   release notes together. Never rewrite trust bytes in existing evidence.

The manual and monthly `scripts/freetsa-canary.sh` workflow submits one public
synthetic target and reports only provider identity, endpoint, safe token
metadata, and artifact digests after local verification. It cannot change the
catalog dynamically and is not part of ordinary CI or release gates.

[Development documentation](index.md)
