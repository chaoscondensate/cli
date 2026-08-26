# Manage questions and resolutions

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/create-ledger.md
next: manage-public-forecasts.md
-->

Forecast Ledger v1 requires every question to contain at least one forecast.
`question add` therefore creates the typed question and its first public or
sealed forecast in one operation.

The question type is a required scalar flag. Do not repeat `type` inside the
closed input document:

```sh
forecast-ledger question add \
  --file ledger.yaml \
  --question q-launch \
  --type binary \
  --input question.yaml
```

The supported types are `binary`, `multiple_choice`, `numeric`, and `date`.
Multiple-choice questions require at least two unique options. Numeric questions
require a unit. Binary and date questions accept neither options nor a unit.
Every referenced platform must already exist, and question and forecast IDs
must be unique in their respective ledger-wide namespaces.

When `initial_forecast.visibility` is `sealed`, the entire input is private and
the command requires a new protected `--key-file`. A public first forecast must
not supply that flag. The protected key is created before the ledger update; if
the update fails, the key is retained and recovery output tells you what to do.
Dry-run validates the destination and prospective shape without generating a
real key, salt, or nonce.

Update only the allowed fields with a closed patch:

```yaml
status: closed
forecast_window:
  closes_at: "2026-10-01T22:00:00+01:00"
notes: Ready for resolution review.
```

```sh
forecast-ledger question update \
  --file ledger.yaml \
  --question q-launch \
  --input question-patch.yaml
```

An updated window must still contain every existing forecast. Question ID,
type, creation time, options, unit, forecasts, and terminal resolution cannot be
edited here. If target evidence exists, target-covered wording and timing are
frozen. Annul the old question and create a new ID instead of rewriting the
evidence; record the predecessor ID in notes if useful.

List or inspect questions locally:

```sh
forecast-ledger question list --file ledger.yaml
forecast-ledger question show --file ledger.yaml --question q-launch
```

Both accept `--file -`. Show returns public question metadata, the current
resolution, and redacted forecast summaries. It never decrypts a sealed
forecast or prints a revealed key. Normal human and plain output includes the
question's title, type, lifecycle, resolution rules and times, type-specific
options or unit, and its forecast summaries. Stdin supplies only ledger bytes,
so these reads do not resolve sibling evidence artifacts.

Resolution is an explicit, approved claim. First set an unresolved question to
`closed` or `awaiting_resolution`, then provide an outcome and at least one
source:

```sh
forecast-ledger question resolve \
  --file ledger.yaml \
  --question q-launch \
  --input resolution.yaml \
  --yes
```

Binary outcomes are booleans. Multiple-choice outcomes are current option IDs;
numeric outcomes are exact decimal strings; date outcomes are full-date strings.
Source URLs must be absolute. Forecasts and integrity evidence are retained.

Use `question annul` to record that a question cannot be resolved under its
criteria, or `question dispute` to challenge a resolved or annulled record.
These commands also require input and approval. Forecast Ledger v1 stores only
the current resolution object: resolve-after-dispute, annul of a terminal state,
or dispute replaces that object. The file itself does not provide internal
resolution history and does not infer Git or another external history system.

[How-to index](index.md) · [Documentation index](../index.md)
