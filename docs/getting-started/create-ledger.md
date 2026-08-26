# Create a ledger

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: true
prerequisites: install.md
next: ../how-to/index.md
-->

`forecast-ledger init` creates one new JSON or YAML ledger. It never overwrites
an existing file and makes no network request. Forecast Ledger schema v1.0.0
requires one initial question and one initial forecast, so init cannot create an
empty draft.

Create `initial-question.yaml`:

```yaml
created_at: 2026-08-26T12:00:00+01:00
question:
  id: q-example
  title: Will the named event happen by the deadline?
  type: binary
  resolution_criteria: Resolve from the named public source.
  forecast_window:
    closes_at: 2026-12-31T23:59:59Z
  expected_resolution_at: 2027-01-15T12:00:00Z
  initial_forecast:
    id: f-example-001
    visibility: public
    forecasted_at: 2026-08-26T12:00:00+01:00
    value:
      kind: binary
      probability_bp: 6000
```

Then create the ledger:

```sh
forecast-ledger init \
  --file ledger.yaml \
  --ledger-id my-forecasts \
  --timezone Europe/London \
  --forecaster-id me \
  --forecaster-name "My Name" \
  --input initial-question.yaml
```

Use `--dry-run` to validate the input and both destinations without writing.
The input may be JSON or YAML; `--input -` reads it from stdin. The ledger
destination itself never accepts `-`.

For a sealed initial forecast, set `visibility: sealed`, include `rationale`,
`key_factors`, and `comment`, and add an explicit unused key destination:

```sh
forecast-ledger init \
  --file ledger.yaml \
  --ledger-id my-forecasts \
  --timezone Europe/London \
  --forecaster-id me \
  --forecaster-name "My Name" \
  --input private-initial-question.yaml \
  --key-file f-example-001.key
```

Treat the complete sealed input as private. The command creates the protected
key first with owner-only access, then creates the public ledger. It never puts
the key path or private forecast fields in the ledger or normal output. If the
ledger cannot be created after the key is durable, the key is retained and the
error returns a safe recovery instruction. Keep the key outside publication
packages and source repositories.

Initialization supports `binary`, `multiple_choice`, `numeric`, and `date`
questions. IDs are global stable slugs, probabilities use integer basis points,
numeric values use exact decimal strings, and timestamps require RFC 3339 with
seconds and an explicit offset.

## Update current metadata

Create a closed JSON or YAML patch containing only the fields to change:

```yaml
title: Updated ledger title
description: null
forecaster:
  name: Current display name
```

Apply it with:

```sh
forecast-ledger ledger update --file ledger.yaml --input metadata-patch.yaml
```

Omitted fields stay unchanged. `null` removes only optional title,
description, contact, profiles, or individual members. A switch to a team must
set `kind: team` and at least two unique members in the same patch; a switch to
an individual must set `kind: individual` and `members: null` together. Ledger,
forecaster, question, and forecast IDs are immutable here. Forecast Ledger v1
stores only current forecaster metadata, has no internal identity history, and
does not prove authorship.

[Getting started index](index.md) · [Documentation index](../index.md)
