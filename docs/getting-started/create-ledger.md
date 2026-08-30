# Create a ledger

<!-- doc-metadata
coverage: v0.5.1
reviewed: 2026-08-30
owner: interface
generated: false
security-critical: true
prerequisites: install.md
next: ../how-to/index.md
-->

`forecast-ledger init` creates one new JSON or YAML ledger. It never overwrites
an existing file and makes no network request. Forecast Ledger schema v1.2.0
allows an empty question list, so no input document is required.

Create an empty ledger first:

```sh
forecast-ledger init \
  --file ledger.yaml \
  --ledger-id my-forecasts \
  --timezone Europe/London \
  --forecaster-id me \
  --forecaster-name "My Name"
```

Add a backlog question and, later, its first forecast directly from flags:

```sh
forecast-ledger question add \
  --file ledger.yaml \
  --question q-example \
  --type binary \
  --title "Will the named event happen by the deadline?" \
  --resolution-criteria "Resolve from the named public source." \
  --created-at 2026-08-26T12:00:00+01:00 \
  --closes-at 2026-12-31T23:59:59Z \
  --expected-resolution-at 2027-01-15T12:00:00Z
forecast-ledger forecast add \
  --file ledger.yaml \
  --question q-example \
  --forecast f-example-001 \
  --forecasted-at 2026-09-01T09:00:00+01:00 \
  --value-kind binary \
  --probability-bp 6500
```

The forecast input format is shown in [Manage public forecasts](../how-to/manage-public-forecasts.md).
The first forecast has no implicit `supersedes_forecast_id`. If that field is
supplied, it must name an existing forecast in the same question.

Use `--dry-run` to validate all supplied flags and destinations without
writing. `--input` remains an optional batch mode for a closed JSON or YAML
document and is mutually exclusive with document-mapped authoring flags;
`--input -` reads it from stdin. The ledger destination itself never accepts
`-`.

Init observes the local operation clock once. An omitted ledger, supplied
question, or supplied initial-forecast time uses that observation where the
schema allows a default.
An explicit ledger `created_at` is never copied into an omitted question
`created_at` or initial `recorded_at`; those remain independent facts. If you
need reproducible historical import times, provide each timestamp explicitly.
Equality at inclusive forecast-window and recorded-time boundaries is valid.

For a sealed first forecast, keep only the private bundle in an owner-only file:

```yaml
value:
  kind: binary
  probability_bp: 6500
rationale: Private reasoning.
key_factors:
  - A private observation.
comment: Private working note.
```

Supply public question and forecast metadata as flags, plus the protected input
and an unused key destination:

```sh
forecast-ledger init \
  --file ledger.yaml \
  --ledger-id my-forecasts \
  --timezone Europe/London \
  --forecaster-id me \
  --forecaster-name "My Name" \
  --question q-example \
  --question-type binary \
  --question-title "Will the named event happen?" \
  --question-resolution-criteria "Resolve from the named public source." \
  --question-closes-at 2026-12-31T23:59:59Z \
  --question-expected-resolution-at 2027-01-15T12:00:00Z \
  --initial-forecast f-example-001 \
  --initial-visibility sealed \
  --initial-forecasted-at 2026-09-01T09:00:00+01:00 \
  --initial-secret-input private-forecast.yaml \
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

Set or clear only the fields to change:

```sh
forecast-ledger ledger update \
  --file ledger.yaml \
  --title "Updated ledger title" \
  --clear-description \
  --forecaster-name "Current display name"
```

Omitted fields stay unchanged. `--clear-*` removes only optional title,
description, contact, profiles, or members. A switch to a team must
set `kind: team` and at least two unique members in the same patch; a switch to
an individual must set `kind: individual` and `members: null` together. Ledger,
forecaster, question, and forecast IDs are immutable here. Forecast Ledger v1
stores only current forecaster metadata, has no internal identity history, and
does not prove authorship.

[Getting started index](index.md) · [Documentation index](../index.md)
