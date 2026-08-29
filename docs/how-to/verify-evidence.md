# Verify ledger evidence

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-29
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
`not_checked`, with reason codes, safe evidence, and limitations. Aggregation
uses this precedence: any completed failure wins; otherwise a required
`not_checked` layer is incomplete; otherwise pending evidence is pending;
otherwise an empty or all-not-applicable selection is `no_evidence`; otherwise
the applicable evidence passes. Exit 0 is reserved for a complete pass with at
least one applicable forecast-evidence layer. Verification failure uses exit 6,
network source failure exit 8, and pending, incomplete, or `no_evidence` exit 9.

An empty ledger returns `no_evidence` with an empty forecast report and zero
network requests. The document layer can still pass, but it is not counted as
forecast evidence. A question-only selector over an empty forecast set has the
same result; a selector naming a nonexistent forecast returns `not_found`.

When fresh Bitcoin observation cannot be acquired, existence timing is
`not_checked`, not failed. A source outage uses `timing.source_unavailable` and
network exit 8. `timing.bitcoin_mismatch` is reserved for comparison against a
complete observation and uses verification exit 6.

Human, plain, JSON, and MCP reports expose safely available timing evidence,
including the receipt and target paths, binding/proof state, Bitcoin block
height, conservative `anchored_before`, and `verified_at`. Offline reports can
reuse internally consistent confirmed metadata already stored in the ledger;
they label it `stored_verification`, set `freshly_checked` to false, and explain
that v1 retained no prior Bitcoin source identity. Displaying this data never
opens a network connection.

Every report states the boundaries: it does not prove authorship, ledger
completeness, calibration, substantive outcome truth, or exact self-reported
time. Network reports also disclose the calendar request-timing/blinded-
commitment and Bitcoin block-height-interest privacy limits.

[Timestamp a forecast](timestamp-forecasts.md) · [Verification claims](../explanation/verification-claims.md)
