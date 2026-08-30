# Manage questions and resolutions

<!-- doc-metadata
coverage: v0.5.2
reviewed: 2026-08-30
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/create-ledger.md
next: manage-public-forecasts.md
-->

Forecast Ledger v1.3 allows a question to exist before its first forecast.
`question add` always creates the typed question and optionally creates its
first public or sealed forecast in the same atomic operation.

The question type and all ordinary question data are direct flags:

```sh
forecast-ledger question add \
  --file ledger.yaml \
  --question q-launch \
  --type binary \
  --title "Will the launch happen by the deadline?" \
  --resolution-criteria "Resolve from the operator's public launch record." \
  --expected-resolution-at "15 Jan 2027"
```

The supported types are `binary`, `multiple_choice`, `numeric`, and `date`.
Multiple-choice questions require at least two repeated `--option id,label`
values. Numeric questions require `--unit-name` and may use `--unit-symbol` or
`--unit-ucum-code`. Binary and date questions accept neither options nor a unit.
Every referenced platform must already exist, and question and forecast IDs
must be unique in their respective ledger-wide namespaces.

Omit `--initial-forecast` to leave `forecasts: []`, then use `forecast add` or
`forecast seal` when the first forecast is ready. A first forecast does not
implicitly supersede anything; an explicit supersedes ID must already exist in
that question.

For a sealed initial forecast, public ID, visibility, times and note remain
flags; value, rationale, key factors and comment use protected
`--initial-secret-input`. The command also requires a new protected
`--key-file`. A public first forecast must not supply those protected flags.
The protected key is created before the ledger update; if the update fails, the
key is retained and recovery output tells you what to do.
Dry-run validates the destination and prospective shape without generating a
real key, salt, or nonce. `--key-file` is invalid when `initial_forecast` is
absent or public.

Update only allowed fields; omitted flags stay unchanged:

```sh
forecast-ledger question update \
  --file ledger.yaml \
  --question q-launch \
  --status closed \
  --opens-at "1 Oct 2026 09:00" \
  --notes "Ready for resolution review."
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
  --outcome-boolean=true \
  --outcome-known-at 2027-01-02T00:00:00Z \
  --source "Official result,https://example.com/result,2027-01-02T00:10:00Z" \
  --yes
```

Binary outcomes are booleans. Multiple-choice outcomes are current option IDs;
numeric outcomes are exact decimal strings; date outcomes are full-date strings.
Source URLs must be absolute. Forecasts and integrity evidence are retained.

Use `question annul` to record that a question cannot be resolved under its
criteria, or `question dispute` to challenge a resolved or annulled record.
These commands require `--reason`, optional repeated `--source`, and approval.
All public lifecycle data is supplied by flags. Forecast Ledger v1 stores only
the current resolution object: resolve-after-dispute, annul of a terminal state,
or dispute replaces that object. The file itself does not provide internal
resolution history and does not infer Git or another external history system.

[How-to index](index.md) · [Documentation index](../index.md)
