# Verification claims and evidence terms

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: security
generated: false
security-critical: true
prerequisites: ../getting-started/index.md
next: ../security/index.md
-->

Applies to: current unreleased Forecast Ledger CLI source (Preview).
Last substantive review: 2026-08-26.
Owner: security and interface owners

This page defines the strongest conclusion the documentation may draw from a
result. The current source implements sealing, timestamps, layered
verification, reveal checks, outcome-evidence checks, and publication. Their
availability does not expand the evidence claims below. OpenTimestamps remains
experimental and unaudited.

## Result states

Each checked layer reports one of these states:

- **pass** — the named check completed and its exact condition holds;
- **fail** — the named check completed and its exact condition does not hold;
- **pending** — evidence exists but cannot yet support pass or fail;
- **not applicable** — the layer does not apply to the selected material; or
- **not checked** — the tool did not run that layer or lacked required input.

Do not collapse these states into one boolean named `verified`.

## Approved terms

### Valid

A valid ledger passed the named parsing, embedded-schema, format, and semantic
checks. State which checks ran when the distinction matters.

Validity does not establish who wrote the ledger, whether records were omitted,
whether a forecast is true, or whether any timestamp is exact.

### Sealed

A sealed forecast has ciphertext and associated metadata produced according to
the named sealing profile, and its private forecast plaintext is absent from the
sealed ledger field. This term does not guarantee safe key custody, secure input
handling outside the tool, authorship, or timing.

### Timestamped

Timestamped means that a named timestamp receipt is associated with the exact
target bytes or digest. Always state whether the receipt is pending, upgraded,
failed, or verified. A pending receipt is not verified existence timing.

### Anchored

Anchored means that the receipt's cryptographic path was verified against the
named external trust source, such as a specific Bitcoin block header accepted
under the documented OpenTimestamps policy. It does not mean the ledger is
complete or its content is true.

### Verified

Verified is always qualified by a layer and subject, for example:

- “schema and semantic validity passed for `ledger.yaml`”;
- “content binding passed for forecast `f-001`”; or
- “existence timing is pending for the selected target.”

Avoid “the forecast is verified.” That phrase hides which conclusion was
checked and may imply truth or authorship.

### Revealed

Revealed means that disclosed forecast material was checked against the named
sealed commitment and that the reveal relationship passed. Reveal does not
replace or delete the original commitment evidence, and it does not establish
who created either value.

### Published

Published means material was made available through a named transport or
location. Publication may improve availability, but it adds no content-binding,
timing, authorship, completeness, or truth claim by itself. Git commits, hosted
file dates, release dates, and archive metadata are not substitutes for a
cryptographic timestamp.

### Evidence

Evidence is inspectable material relevant to a bounded claim. Examples include
ledger bytes, canonical targets, digests, sealed commitments, OTS receipts,
reveal inputs, outcome records, manifests, and release attestations. Evidence
must be named and interpreted through the check that consumes it.

### Proof

Use proof only with a named protocol and conclusion, such as “a valid
OpenTimestamps path proving that these target bytes existed no later than the
verified anchoring bound.” Do not use proof as a synonym for confidence,
validity, publication, or a passing group of unrelated checks.

### Authorship

Authorship identifies who created or approved material. Forecast Ledger v1 does
not provide a digital authorship signature. An ID, account name, repository
commit, file owner, or possession of a reveal key is not sufficient authorship
proof.

### Completeness

Completeness would establish that no relevant question, forecast, revision, or
evidence was omitted. A valid append-only history makes recorded revisions
inspectable but cannot prove that every real-world forecast was recorded.

### Truth

Truth concerns whether a forecast, resolution, or factual statement matches
reality. Structural validation, cryptographic binding, timestamps, reveal, and
publication do not determine truth.

### Outcome evidence

Outcome evidence is material supplied to support a recorded resolution. The CLI
may validate its structure, digest, availability observation, or relationship
to a ledger record. A responding URL, saved file, or passing format check does
not establish the source's authority or substantive correctness.

## Verification layers

Keep these conclusions separate in human and JSON results:

1. **Structural and semantic validity** — the ledger can be parsed and follows
   the embedded contract and project semantic rules.
2. **Content binding** — the selected ledger content matches the exact target,
   digest, commitment, or package manifest being checked.
3. **Existence timing** — a receipt supports a bounded “existed no later than”
   claim under its named trust source.
4. **Reveal validity** — disclosed material matches the sealed commitment.
5. **Outcome evidence** — recorded resolution evidence passes only the checks
   actually performed on it.

Authorship, completeness, truth, exact self-reported creation time, and
substantive outcome-source correctness remain outside these layers.

## Writing examples

Preferred:

> The ledger passed embedded-schema and semantic validation. No authorship,
> completeness, truth, or existence-timing check was performed.

Not allowed:

> The forecast is proven valid and authentic.

The second sentence combines an unnamed validity result with an unsupported
authorship claim.

[Explanation index](index.md) · [Documentation index](../index.md)
