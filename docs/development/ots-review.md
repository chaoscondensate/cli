# OpenTimestamps release review

<!-- doc-metadata
coverage: v0.3.1
reviewed: 2026-08-29
owner: security
generated: false
security-critical: true
prerequisites: ../../internal/timestamp/ots, ../how-to/timestamp-forecasts.md
next: documentation-baseline.md
-->

OpenTimestamps remains experimental until every gate below has recorded
evidence. Passing the normal unit tests is not an independent review.

- [x] Bounded pure-Go parser, serializer, evaluator, merge, and binding tests.
- [x] Fixed four-calendar threshold and returned-identity tests.
- [x] Dual public Bitcoin agreement, disagreement, outage, deduplication, and
  request-budget tests.
- [x] Mocked authenticated Bitcoin Core verification.
- [x] Scheduled fixed-profile calendar liveness workflow.
- [ ] Official Python client differential fixtures for every supported
  operation, attestation, stamp, upgrade, info, and verify result.
- [ ] Stable liveness history across the release observation window.
- [ ] Independent cryptographic and parser review with reviewer, commit, scope,
  findings, and disposition recorded here.

Do not remove the experimental label or describe OTS as audited while any item
is open.

[Development documentation](index.md) · [Timestamp workflow](../how-to/timestamp-forecasts.md)
