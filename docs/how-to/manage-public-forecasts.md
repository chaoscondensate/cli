# Manage public forecasts

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/create-ledger.md
next: ../reference/index.md
-->

Public forecasts are append-only records under one question. Adding a revision
does not edit or hide the earlier forecast. Every command explicitly selects the
ledger and question; forecast IDs must be unique across the entire ledger.

Create `forecast.yaml` with a value that matches the question type:

```yaml
forecasted_at: "2026-09-01T09:00:00+01:00"
recorded_at: "2026-09-01T09:01:00+01:00"
value:
  kind: binary
  probability_bp: 6500
rationale: Evidence moved slightly in favor of the outcome.
key_factors:
  - The latest reported measurement increased.
supersedes_forecast_id: f-launch-001
```

Probability uses integer basis points: `6500` means 65%. Multiple-choice input
must include every option exactly once and total 10,000. Numeric values use
exact decimal strings. Numeric and date values can contain a point, interval,
quantiles, or a supported combination documented by the schema.

Append the record:

```sh
forecast-ledger forecast add \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --input forecast.yaml
```

The question must be open. `forecasted_at` must be inside its forecast window
and must not follow `recorded_at`. If `recorded_at` is omitted, the command
captures it once from the local clock and returns the exact stored value. This
self-reported time is not cryptographic proof.

Use `--dry-run` to parse and validate the prospective append without writing.
The result cannot guarantee that another process will not change the ledger
before a later real command.

List the complete revision history in recorded order or inspect one record:

```sh
forecast-ledger forecast list --file ledger.yaml --question q-launch
forecast-ledger forecast show \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002
```

Both read commands accept `--file -`. They do not contact the network or change
integrity data. A sealed forecast is shown only as a redacted summary, and a
revealed forecast never prints its stored revealed key. Normal human and plain
list rows include a type-aware public value summary; show includes all public
business fields. Because stdin contains only ledger bytes, these commands do
not resolve or check sibling target and receipt files.

Quote timestamps in maintained YAML input as shown above. The CLI also accepts
a YAML timestamp scalar in a known timestamp field, but rejects that implicit
type in ordinary text fields.

[How-to index](index.md) · [Documentation index](../index.md)
