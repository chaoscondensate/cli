# Manage public forecasts

<!-- doc-metadata
coverage: v0.5.2
reviewed: 2026-08-30
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/create-ledger.md
next: ../reference/index.md
-->

Public forecasts are append-only records under one question. Adding a revision
does not edit or hide the earlier forecast. Every command explicitly selects the
ledger and question; forecast IDs must be unique across the entire ledger.

A question may have no forecasts. `forecast add` creates its first public
forecast normally, without adding an implicit `supersedes_forecast_id`. Supply
that field only for a later revision, and only with an existing forecast ID from
the same question.

Probability uses integer basis points: `6500` means 65%. Multiple-choice input
uses repeated `--choice-probability option-id,basis-points` values that include
every option exactly once and total 10,000. Numeric values use exact decimal
strings. Numeric and date values can use `--point`, `--interval
lower,upper,credibility-bp`, repeated `--quantile probability-bp,value`, or a
supported combination.
Omit `supersedes_forecast_id` when this is the question's first forecast.

Append the record:

```sh
forecast-ledger forecast add \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002 \
  --forecasted-at "1 Sep 2026 09:00" \
  --recorded-at "1 Sep 2026 09:01" \
  --value-kind binary \
  --probability-bp 6500 \
  --rationale "Evidence moved slightly in favor of the outcome." \
  --key-factor "The latest reported measurement increased." \
  --supersedes-forecast f-launch-001
```

The question must be open. `forecasted_at` must not precede an optional window
opening and must not follow `recorded_at`. If `forecasted_at` is omitted, the
command captures one current instant, formats it in the ledger timezone, and
uses it for both omitted forecast and record times. This self-reported time is
not cryptographic proof.

Timestamp flags accept exact RFC 3339 or deterministic local forms such as
`2030-08-10 14:05`, `10 Aug 2030 14:05`, and `August 10 2030`. Local forms use
the ledger `default_timezone`. Forecast and evidence timestamps require a time;
date-only input is rejected. Skipped or repeated DST wall times require an
explicit numeric offset.

Use `--dry-run` to parse and validate the prospective append without writing.
The result cannot guarantee that another process will not change the ledger
before a later real command.

Public forecast authoring is flag-only; there is no generic request-document
mode.

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
business fields. Show also includes the stored integrity projection: target
metadata and RFC 3161 entry paths, states, generation times, and authority
identities when present. This is retained ledger data, not a fresh verification;
the command opens no network connection and labels stored verified evidence
accordingly. Because stdin contains only ledger bytes, these commands do not
resolve or check sibling target or timestamp files.

For a question with no forecasts, `forecast list` returns an empty list.
Selecting any forecast with `forecast show` returns `not_found`; it never
creates a placeholder or performs another side effect.

[How-to index](index.md) · [Documentation index](../index.md)
