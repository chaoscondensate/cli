## Purpose

Defines complete business behavior for creating a Forecast Ledger and safely managing mutable root metadata, its platform registry, questions, resolutions, and append-only public forecasts.

## ADDED Requirements

### Requirement: Initialize a schema-valid ledger with an initial question
`forecast-ledger init` SHALL create a new `.json`, `.yaml`, or `.yml` file at the explicit `--file` path and MUST refuse an existing destination. It SHALL require `--ledger-id`, `--timezone`, `--forecaster-id`, `--forecaster-name`, and a bounded `--input` document containing exactly one initial question with its required `type` and exactly one initial public or sealed forecast because the pinned Forecast Ledger v1 contract requires non-empty `questions` and `forecasts` arrays. Additional questions and forecasts are added after initialization through their normal commands. Unlike `question add`, init has no separate question-type flag because its input owns the complete initial question. Both paths SHALL normalize the type into the same question builder and validation rules while retaining intentionally distinct closed input schemas. `--forecaster-kind` SHALL default to `individual`; team forecasters SHALL supply at least two uniquely identified members in the input. Optional title, description, contact, profiles, platforms, and explicit `created_at` MAY be supplied in the same input.

An initial sealed forecast SHALL use the exact normal seal profile, SHALL require an explicit new `--key-file`, and SHALL cause the complete init input to be treated as private. A public initial forecast MUST reject `--key-file`. Key creation, ledger creation, rollback/recovery, redaction, and dry-run behavior SHALL match `forecast seal`; no incomplete ledger may be committed if either resource fails.

The created document SHALL use schema version `1.0.0`, an empty platform map unless platforms were supplied, no inferred publication/source-control metadata, and the requested initial question. IDs SHALL match the v1 slug grammar, the timezone SHALL be a known IANA name, and the complete prospective document SHALL pass structural and semantic validation before exclusive creation.

#### Scenario: Minimal valid initialization
- **WHEN** an individual forecaster supplies all root flags and one valid binary question in `--input`
- **THEN** the command exclusively creates a schema-valid ledger with exact reported timestamps, no publication metadata, and no implicit network or source-control operation

#### Scenario: Initialize with a sealed first forecast
- **WHEN** the single initial forecast is sealed and a safe new `--key-file` is supplied
- **THEN** the key is protected before the valid ledger is created, and output exposes neither the private init fields nor key material

#### Scenario: Empty initial question set
- **WHEN** the init input contains no question
- **THEN** the command rejects it before file creation and explains that v1 requires at least one initial question

#### Scenario: Existing destination
- **WHEN** the ledger path already exists as any file, symlink, junction, or directory entry
- **THEN** init returns `conflict` and does not replace or follow it

### Requirement: Update mutable root metadata
`forecast-ledger ledger update` SHALL require `--file` and a closed `--input` patch. It SHALL allow only root `title`, `description`, and `default_timezone`, plus `forecaster.kind`, `forecaster.name`, `forecaster.contact`, `forecaster.profiles`, and `forecaster.members`. Omission SHALL preserve a value; explicit null SHALL remove only optional title, description, contact, profiles, or members when the resulting forecaster is individual. The resulting timezone SHALL be a known IANA name; profiles SHALL be valid; member IDs SHALL be unique. An atomic individual-to-team transition SHALL include at least two valid members in the same patch; an atomic team-to-individual transition SHALL remove members in the same patch. Output and documentation SHALL state that v1 stores current forecaster metadata, retains no internal identity history, and does not prove authorship.

The operation MUST NOT change `schema_version`, `ledger_id`, `created_at`, forecaster ID, platforms, questions, forecast history, integrity evidence, or the schema's legacy publication object. v1 authoring SHALL not create or edit publication/source-control metadata; an existing schema-valid publication object is preserved byte-for-byte as imported data.

#### Scenario: Update forecaster contact
- **WHEN** a patch changes forecaster email and omits profiles
- **THEN** the email changes, profiles remain unchanged, and all questions and forecasts remain byte-for-byte unchanged

#### Scenario: Reduce a team below schema minimum
- **WHEN** a team member patch would leave fewer than two members
- **THEN** update returns invalid data and leaves the ledger unchanged

#### Scenario: Convert an individual identity to a team
- **WHEN** one patch changes forecaster kind to team and supplies at least two uniquely identified members
- **THEN** the root metadata changes atomically, existing questions and evidence remain unchanged, and output reports the lack of internal identity history or authorship proof

#### Scenario: Attempt immutable root change
- **WHEN** a patch includes ledger ID, forecaster ID, publication, or questions
- **THEN** the closed input is rejected before mutation

### Requirement: Add, update, inspect, and remove platforms
`platform add` SHALL require `--platform` and `--input` with `name`, `kind`, and optional `url`/account fields. The platform ID is the map key and MUST be unique. Kind SHALL be exactly one of `scoring_platform`, `prediction_market`, `self_hosted`, `internal`, or `informal`; URLs SHALL be absolute valid URIs; an account object SHALL contain at least one non-empty field.

`platform update` SHALL apply a closed partial input to `name`, `kind`, `url`, or account fields without changing the platform ID. Explicit JSON `null` SHALL remove an optional field; omission SHALL preserve it. The resulting platform and every referencing question SHALL remain valid.

`platform list` SHALL return platforms sorted by ID with reference counts. `platform show` SHALL return the exact selected public record and sorted referencing question IDs. `platform remove` SHALL require approval and SHALL succeed only when no question references the platform.

#### Scenario: Add platform
- **WHEN** a unique platform ID and valid prediction-market record are supplied
- **THEN** the platform is added at `/platforms/<id>` without reordering or rewriting unrelated ledger content

#### Scenario: Update optional account field
- **WHEN** update sets `account.profile_url` and omits the existing username
- **THEN** the profile URL changes and the username remains unchanged

#### Scenario: Remove referenced platform
- **WHEN** one or more `platform_refs` use the selected platform
- **THEN** removal returns `conflict` with sorted referencing question IDs and leaves the ledger unchanged

#### Scenario: Platform list from stdin
- **WHEN** a valid ledger is piped to `platform list --file - --json`
- **THEN** one stable sorted list is returned without attempting to locate sibling artifacts

### Requirement: Add a typed question
`question add` SHALL require a globally unique `--question` ID, the existing required scalar `--type` with one of `binary`, `multiple_choice`, `numeric`, or `date`, and a closed `--input` object containing required `title`, `resolution_criteria`, `forecast_window`, `expected_resolution_at`, exactly one initial public or sealed forecast bundle, optional `created_at`, and applicable type-specific fields. The structured input MUST NOT contain a second `type` field. The default status SHALL be `open`, and no resolution SHALL be accepted on add. Question and initial forecast SHALL be validated and committed atomically so the ledger never contains an invalid empty forecast array. An initial sealed forecast SHALL require an explicit new `--key-file`, make the full input private, and reuse the normal seal/key transaction; a public initial forecast MUST reject `--key-file`.

Binary and date questions MUST NOT contain options or unit. Multiple-choice questions SHALL contain at least two uniquely identified options and no unit. Numeric questions SHALL contain a non-empty exact unit and no options. `opens_at` SHALL default to `created_at`; it MUST NOT be after `closes_at`; expected resolution MUST NOT precede close. Platform references MUST exist and tags/options MUST be unique.

#### Scenario: Add multiple-choice question
- **WHEN** input contains unique options, valid chronology, and existing platform references
- **THEN** the exact typed question and any contract-required initial forecast are appended atomically

#### Scenario: Wrong type-specific field
- **WHEN** a binary question input includes a unit or options
- **THEN** the command returns invalid data with precise field locations and makes no change

### Requirement: Constrained question updates
`question update` SHALL accept a closed patch for mutable descriptive fields (`title`, `resolution_criteria`, `forecast_window.closes_at`, `expected_resolution_at`, `platform_refs`, `tags`, and `notes`) and unresolved lifecycle status (`open`, `closed`, or `awaiting_resolution`). It MUST NOT change question ID, type, `created_at`, type-defining options/unit, any forecast, or terminal resolution through this command. A changed window MUST still contain every existing `forecasted_at`; an unresolved status MUST NOT carry a resolution.

Before changing any field included by `forecast-envelope/v1`, update SHALL reconstruct every affected old and prospective forecast target. If any bytes would change and matching integrity target metadata or a deterministic target artifact already exists, update MUST return conflict rather than silently invalidating retained content/timestamp evidence. Changes to fields excluded from the target MAY proceed when all other rules pass.

Once any forecast under the question has a retained deterministic target or target-bearing integrity metadata, target-covered question wording and timing fields are frozen in v1: there is no override, target replacement, or evidence rewrite. Help and conflict output SHALL direct the user to annul the original question, create a new question with a new globally unique ID, and record the predecessor in notes when the real-world question materially changed; the original question, forecasts, targets, and receipts remain intact.

#### Scenario: Shorten forecast window past a forecast
- **WHEN** a proposed close time is earlier than an existing forecast's `forecasted_at`
- **THEN** update returns `conflict` or invalid data and preserves the original question

#### Scenario: Attempt to reopen terminal question
- **WHEN** update targets a resolved, annulled, or disputed question
- **THEN** it refuses the lifecycle rewrite and directs the user to the applicable explicit transition command

#### Scenario: Change targeted question wording
- **WHEN** title or resolution criteria changes target bytes for a question with a retained target artifact
- **THEN** update returns conflict and preserves the question and evidence bytes

#### Scenario: Replace a materially changed targeted question
- **WHEN** an operator needs to change frozen wording or timing after evidence exists
- **THEN** help identifies annul-plus-new-question as the supported workflow and does not offer an overwrite or target-rebuild escape hatch

### Requirement: List and show questions
`question list` SHALL return questions sorted by stable ID with type, status, forecast count, forecast window, expected resolution time, and integrity counts. `question show` SHALL return the selected question, its public metadata, resolution if present, and forecast summaries without exposing sealed plaintext or revealed key material. Neither action SHALL fetch outcome URLs, verify timestamps, or mutate state.

#### Scenario: Show sealed question
- **WHEN** the selected question contains sealed forecasts
- **THEN** output includes public notes and commitment/target state but contains no decrypted bundle, raw key, or absolute secret path

### Requirement: Resolve a question with typed evidence
`question resolve` SHALL require approval, a selected question in `closed`, `awaiting_resolution`, or `disputed`, and a closed `--input` object containing `outcome`, `outcome_known_at`, optional `recorded_at`, one or more evidence sources, and optional notes. Outcome type SHALL match the question: boolean for binary, existing option ID for multiple-choice, exact decimal string for numeric, and full date for date. Each source SHALL include non-empty title, absolute URL, and `retrieved_at`, with optional publisher, published time, and SHA-256 content digest. `recorded_at` MUST NOT precede `outcome_known_at`.

The command SHALL set both question and resolution status to `resolved`, retain every forecast byte-for-byte, and warn without rejecting when existing verified timestamps do not predate the outcome; layered verification decides evidentiary sufficiency. When replacing a disputed resolution, it SHALL report the prior status and explicitly state that v1 retains only the current resolution object and does not provide internal resolution history.

#### Scenario: Resolve binary question
- **WHEN** a closed binary question receives a boolean outcome, valid chronology, and one source
- **THEN** its terminal resolution is stored atomically and all forecasts remain unchanged

#### Scenario: Resolution without evidence
- **WHEN** the source list is empty
- **THEN** the command returns invalid data and leaves status and resolution unchanged

#### Scenario: Resolve after dispute review
- **WHEN** a disputed question receives a new valid typed outcome and evidence with confirmation
- **THEN** the dispute object is replaced by the current resolved object, prior status is reported, and forecasts remain unchanged

### Requirement: Annul a question
`question annul` SHALL require approval and `--input` containing a non-empty reason, optional `recorded_at`, and optional evidence sources. It SHALL accept unresolved, resolved, or disputed questions, replace any current resolution only after explicit confirmation, set question/resolution status to `annulled`, and retain all forecast records and integrity evidence unchanged. When replacing a disputed or resolved object it SHALL report prior status and the absence of internal v1 resolution history. The result SHALL make clear that annulment is a recorded claim, not deletion.

#### Scenario: Annul before resolution
- **WHEN** an open or unresolved question becomes invalid under its criteria and a reason is supplied
- **THEN** it transitions to annulled without deleting the question or forecasts

#### Scenario: Annul after dispute review
- **WHEN** review concludes that a disputed question must be annulled and the user confirms replacement
- **THEN** the current dispute is replaced by an annulled resolution while every forecast and integrity artifact remains unchanged

### Requirement: Record a disputed resolution
`question dispute` SHALL require approval, a selected question that has a resolved or annulled terminal state, and `--input` containing a non-empty dispute reason, optional `recorded_at`, and optional sources. It SHALL set question/resolution status to `disputed` while retaining all forecasts and integrity artifacts. Because v1 stores one current resolution object, output and documentation MUST state that this command replaces the prior resolution object in the current ledger file and that external file history is not inferred or guaranteed.

#### Scenario: Dispute resolved outcome
- **WHEN** a user records a reason and evidence challenging the current resolved outcome
- **THEN** the current resolution becomes disputed, forecasts stay unchanged, and the result reports the prior status that was replaced

#### Scenario: Dispute unresolved question
- **WHEN** the selected question has never had a resolved or annulled terminal state
- **THEN** the command returns `conflict` and does not invent a disputed resolution

### Requirement: Append a public forecast
`forecast add` SHALL require an open question, a globally unique `--forecast` ID, and a closed `--input` object containing `forecasted_at`, optional `recorded_at`, a typed `value`, and optional rationale, key factors, comment, public note, and `supersedes_forecast_id`. It SHALL set visibility to `public` and integrity to `unanchored` and MUST NOT accept commitment/encryption fields.

Binary probability SHALL be integer basis points from 0 through 10,000. Multiple-choice values SHALL cover every current option exactly once with unique option IDs and total exactly 10,000 basis points. Numeric and date values SHALL provide at least one of point, interval, or quantiles; exact decimals remain strings, interval lower MUST NOT exceed upper, quantile probabilities SHALL be unique and ordered, and quantile values SHALL be non-decreasing. `forecasted_at` SHALL lie within the question window, MUST NOT follow `recorded_at`, and records SHALL remain ordered by recorded time.

A superseded ID MUST identify an earlier forecast in the same question. Adding a revision SHALL append a new record and MUST NOT modify, delete, or mark the earlier record mutable.

#### Scenario: Append forecast revision
- **WHEN** a new valid forecast identifies an earlier forecast in the same question
- **THEN** the new record is appended with the supersession link and the prior forecast remains byte-for-byte unchanged

#### Scenario: Duplicate global forecast ID
- **WHEN** the requested ID exists under any question
- **THEN** add returns `conflict` and does not append a record

#### Scenario: Incomplete multiple-choice distribution
- **WHEN** input omits an option or totals 9,999 basis points
- **THEN** validation identifies coverage and/or sum errors and leaves the ledger unchanged

### Requirement: List and show forecasts
`forecast list` SHALL return the selected question's forecasts in recorded order with stable ID, times, visibility, supersession link, and integrity status. `forecast show` SHALL return the exact selected public/revealed forecast or a redacted sealed summary. Revealed keys SHALL remain redacted even though the v1 ledger stores the disclosed key. Neither action SHALL decrypt, contact the network, or change integrity metadata.

#### Scenario: List append-only history
- **WHEN** three forecasts form a supersession chain
- **THEN** list returns all three records in recorded order and exposes each link without collapsing them into a current value

### Requirement: Preserve imported evidence states outside the authoring surface
v1 authoring commands SHALL not create or edit schema-supported `external_anchors`; every pending-to-verified transition SHALL preserve them byte-for-byte and verification SHALL report them only as external claims, never as OpenTimestamps proof. v1 commands SHALL not create `integrity.status: failed` automatically. An imported failed integrity state is terminal for that forecast: target, timestamp, seal, and integrity mutation commands SHALL refuse to replace it, while list/show/status/verify remain available. Recovery SHALL append a new forecast revision with a new globally unique ID and optional supersession link, preserving the failed record as history.

#### Scenario: Verify imported failed integrity
- **WHEN** a schema-valid ledger contains a forecast with imported failed integrity
- **THEN** verification reports its retained failure evidence without rewriting the state

#### Scenario: Preserve external anchors on confirmation
- **WHEN** timestamp verification promotes pending integrity containing external anchors to verified
- **THEN** every external anchor is retained byte-for-byte and remains separately labeled from OTS evidence
