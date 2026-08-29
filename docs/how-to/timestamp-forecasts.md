# Timestamp a forecast

<!-- doc-metadata
coverage: v0.4.0
reviewed: 2026-08-29
owner: security
generated: false
security-critical: true
prerequisites: build-targets.md
next: verify-evidence.md
-->

RFC 3161 support is experimental. It binds the exact canonical target bytes to
a signed generation time (`gen_time`) from a timestamp authority (TSA). It does
not prove authorship, completeness, forecast truth, outcome truth, or that the
TSA clock was honest.

## Prepare trust material

Choose a TSA outside this CLI. Obtain the PEM certificate bundle needed to
validate that TSA and retain it beside the ledger, for example
`trust/tsa-ca.pem`. The CLI does not provide a TSA list, download certificates,
or use operating-system roots. Treat the endpoint and bundle as one reviewed
trust choice.

## Stamp

```sh
forecast-ledger timestamp stamp \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --tsa-url https://tsa.example.com/ \
  --ca-bundle trust/tsa-ca.pem
```

Stamp creates a bounded SHA-256 request with a fresh positive nonce and asks
the TSA to include its signing certificate. It makes one HTTPS request and
retains:

- `proofs/targets/f-launch-002.json` — the exact canonical target;
- `proofs/timestamps/f-launch-002/<tsa-id>/request.tsq` — the DER request;
- `proofs/timestamps/f-launch-002/<tsa-id>/response.tsr` — the DER response; and
- the exact ledger-relative PEM CA bundle named by `--ca-bundle`.

`<tsa-id>` is a stable short digest of the normalized TSA URL. The URL must be
public HTTPS without credentials, query, fragment, or a non-default port.
Private, loopback, link-local, reserved, and redirect-to-other-origin
destinations are rejected.

`--dry-run` validates paths and inputs without entropy, network, or writes.
`--offline` fails before a socket because stamp requires the TSA. A transport
failure is a network error, not a failed signature. If a successful response is
retained but complete local verification fails, the entry remains `pending`
with safe reason codes and can be inspected or retried.

Repeat stamp with another TSA URL to keep independent timestamp entries. An
earlier verified entry is preserved. Repeating the same TSA is idempotent only
when retained bytes match exactly; conflicting files are never overwritten.

## Inspect and verify locally

```sh
forecast-ledger timestamp status \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002

forecast-ledger timestamp verify \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002
```

Both commands are local. They read the target, request, response, declared
metadata, and retained CA bundle. Verify checks request/response nonce and
imprint agreement, SHA-256 target binding, CMS signed attributes and signature,
the signing-certificate binding, critical timestamping EKU, certificate chain
at `gen_time`, supported algorithms, and the ledger metadata. It never contacts
the TSA, a blockchain, a Git host, a system trust service, or a revocation
service.

At least one independently verified response passes timing. The layer fails
only after every applicable entry was completely checked and failed. Pending or
uncheckable entries prevent an all-branches failure claim. For a resolved
forecast, a verified `gen_time` must predate `outcome_known_at` to exclude
hindsight.

The retained bundle makes future verification portable, but RFC 3161 alone is
not long-term validation. The CLI performs no CRL/OCSP check, archive timestamp
renewal, RFC 4998 evidence-record maintenance, or system-root fallback. Preserve
the ledger, target, `.tsq`, `.tsr`, and PEM bytes together.

[Verify ledger evidence](verify-evidence.md)
