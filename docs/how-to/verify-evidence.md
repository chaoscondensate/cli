# Verify ledger evidence

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: security
generated: false
security-critical: true
prerequisites: timestamp-forecasts.md
next: publish-evidence.md
-->

Run all local checks and allow the built-in Bitcoin observers when timing proof
needs a fresh check:

```sh
forecast-ledger verify --file ledger.yaml
```

Use `--offline` to guarantee that no socket is opened:

```sh
forecast-ledger verify --file ledger.yaml --offline
```

Narrow the report with `--question`; a `--forecast` also requires its question.
Use `--check-sources` only when you explicitly want bounded outcome-source
reachability and stored-digest checks. Reachability does not establish that a
source is authoritative or that its claim is true.

The stable layers are:

- `content_binding`: rebuilds exact target bytes and compares every stored
  target field;
- `existence_timing`: checks retained OTS proof structure and, when online,
  Bitcoin evidence;
- `reveal`: authenticates revealed ciphertext, commitment, associated data,
  canonical private bundle, and public mirror without printing secrets; and
- `outcome_evidence`: checks resolution metadata and optional source bytes.

Each layer returns `pass`, `fail`, `pending`, `not_applicable`, or
`not_checked`, with reason codes, safe evidence, and limitations. Any failure
wins aggregation. Pending work returns overall `pending`; unavailable required
checks return `incomplete`. Exit 0 is reserved for a complete pass. Verification
failure uses exit 6, network failure exit 8, and pending or incomplete exit 9.

Every report states the boundaries: it does not prove authorship, ledger
completeness, calibration, substantive outcome truth, or exact self-reported
time. Network reports also disclose the calendar request-timing/blinded-
commitment and Bitcoin block-height-interest privacy limits.

[Timestamp a forecast](timestamp-forecasts.md) · [Verification claims](../explanation/verification-claims.md)
