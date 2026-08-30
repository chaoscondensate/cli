# Seal and reveal forecasts

<!-- doc-metadata
coverage: v0.5.1
reviewed: 2026-08-30
owner: interface
generated: false
security-critical: true
prerequisites: manage-public-forecasts.md
next: build-targets.md
-->

A sealed forecast publishes an encrypted commitment while keeping its value and
explanation private until an approved reveal. Seal always appends a new forecast
ID; it never hides or reseals an existing public record.

Prepare the private input in an owner-only file, or supply it on stdin:

```yaml
forecasted_at: "2026-09-01T09:00:00+01:00"
recorded_at: "2026-09-01T09:01:00+01:00"
value:
  kind: binary
  probability_bp: 6500
rationale: Private reasoning.
key_factors:
  - A private observation.
comment: Private working note.
supersedes_forecast_id: f-launch-001
```

```sh
chmod 600 private-forecast.yaml
forecast-ledger forecast seal \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --input private-forecast.yaml \
  --key-file f-launch-002.key
```

The key destination must be new. On POSIX it is created with mode `0600`; on
Windows it receives an owner-only ACL. The key is made durable before the
ledger is updated. If the later ledger update fails, the only key copy is
retained and recovery output identifies its safe display name. `--dry-run`
validates input and destinations without generating a salt, key, or nonce.

The `forecast-seal/v1` plaintext authenticates exactly the question ID,
forecast ID, random salt, forecast and recorded times, typed value, rationale,
key factors, and comment. Associated data also authenticates the scheme and
commitment digest. It does not bind the ledger ID, public note, supersession,
key hint, forecaster, or question wording. A separate `forecast-envelope/v1`
target binds the listed public context. After reveal publishes the key, anyone
with the retained ciphertext can recover the private bundle.

Reveal only with explicit approval:

```sh
forecast-ledger forecast reveal \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --key-file f-launch-002.key \
  --yes
```

Reveal verifies the protected key file, AEAD, commitment digest, protocol,
IDs, exact canonical plaintext, typed mirror, and original sealed target before
changing the ledger. It publishes the value, rationale, factors, comment, and
key required by schema v1 while retaining ciphertext, commitment, public fields,
integrity, target, and timestamp evidence. Normal and JSON results never print
the key or private fields. Repeating reveal with the correct key is unchanged.
Use `--revealed-at` for an explicit reproducible RFC 3339 time.

The key hint is public, non-authoritative metadata; it never locates or reads a
key. Repair it without changing seal or target bytes:

```sh
forecast-ledger forecast key-hint update \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --key-hint forecast-key:f-launch-002
```

Hints use a path-free `scheme:opaque` form. File paths, URLs with authority,
credentials, slashes, backslashes, queries, fragments, and `file:` are rejected.
For `forecast-key`, the opaque value must be the selected forecast ID.

[How-to index](index.md) · [Documentation index](../index.md)
