## Purpose

Defines deterministic, non-LLM date and timestamp authoring that is convenient for people while preserving exact RFC 3339 ledger values and evidence chronology.

## ADDED Requirements

### Requirement: CLI accepts a bounded human date grammar
Applicable direct CLI timestamp flags SHALL accept exact Forecast Ledger RFC 3339 plus a documented English, locale-independent allowlist: ISO calendar dates `YYYY-MM-DD`; calendar dates `D Mon YYYY`, `D Month YYYY`, `Mon D YYYY`, and `Month D YYYY`; and those date forms followed by a 24-hour `HH:MM` or `HH:MM:SS` time with an optional `Z` or numeric `±HH:MM` offset. Month matching SHALL be ASCII case-insensitive. The parser MUST use deterministic grammar and timezone rules without an LLM, network request, locale database, or fuzzy natural-language parser.

#### Scenario: Parse the requested human date
- **WHEN** `--closes-at "10 Aug 2030"` is supplied for a ledger whose default timezone is `Europe/London`
- **THEN** it is accepted as the end of 10 August 2030 in that timezone and normalized to an RFC 3339 timestamp with seconds and the applicable explicit offset

#### Scenario: Preserve exact input
- **WHEN** a caller supplies a valid exact RFC 3339 timestamp
- **THEN** the parser preserves the represented instant and explicit offset without applying a calendar-date default

### Requirement: Missing time and offset use field-specific deterministic defaults
Date-only input SHALL be accepted only for planned question scheduling fields. `opens_at` date-only input SHALL mean `00:00:00` at the start of that local date; `closes_at` and `expected_resolution_at` date-only input SHALL mean `23:59:59` at the end of that local date. A human timestamp with a time but no offset SHALL use the ledger `default_timezone`; during init it SHALL use the required init timezone. No input SHALL use the host process timezone. Explicit creation, forecast, recording, outcome, retrieval, publication, reveal, or cryptographic evidence timestamps MUST include a time, whether expressed in exact RFC 3339 or another allowed human timestamp form.

#### Scenario: Opening date starts the day
- **WHEN** `--opens-at 2030-08-10` is supplied for a valid question
- **THEN** it normalizes to local `00:00:00` in the ledger default timezone

#### Scenario: Evidence timestamp rejects a date alone
- **WHEN** a caller supplies `--recorded-at "10 Aug 2030"`
- **THEN** the command rejects the value without inventing an observation time

#### Scenario: Init uses its explicit timezone
- **WHEN** init supplies a human question scheduling value without an offset
- **THEN** normalization uses the same required IANA timezone that will be stored as `default_timezone`

### Requirement: Ambiguous and relative input is rejected
The parser MUST reject numeric slash dates, two-digit years, reordered numeric dates other than `YYYY-MM-DD`, relative phrases, locale-dependent names, timezone abbreviations, trailing words, fuzzy spelling, invalid calendar dates, and unsupported precision. A local wall time that is skipped or repeated by an IANA offset transition MUST be rejected unless the caller supplies an explicit numeric offset. Errors SHALL identify the field, explain the ambiguity or invalidity, and list accepted forms without suggesting a guessed value.

#### Scenario: Ambiguous numeric date is rejected
- **WHEN** a caller supplies `10/08/2030`
- **THEN** parsing fails as ambiguous and no request reaches mutation

#### Scenario: Relative phrase is rejected
- **WHEN** a caller supplies `next Friday` or `in two weeks`
- **THEN** parsing fails without consulting the operation clock to guess the intended date

#### Scenario: DST fold requires an offset
- **WHEN** a wall-clock time occurs twice in the ledger timezone and the input omits its offset
- **THEN** parsing fails and asks for an explicit `Z` or numeric offset

### Requirement: Stored and machine-facing values remain exact
Every accepted human CLI value SHALL be normalized before domain chronology checks and persistence to the exact v1.3.0 RFC 3339 form with seconds and an explicit offset. The embedded schema, stored JSON/YAML ledger model, ledger validation, and direct MCP timestamp properties SHALL remain strict RFC 3339 machine interfaces. Dry-run and structured success data SHALL expose each normalized value when the caller supplied a non-canonical human form.

#### Scenario: Human date produces a conforming ledger
- **WHEN** a human scheduling value passes parsing and chronology validation
- **THEN** the resulting ledger validates against the embedded v1.3.0 timestamp contract

#### Scenario: MCP human phrase is rejected
- **WHEN** an MCP timestamp property contains `10 Aug 2030` instead of RFC 3339
- **THEN** the closed tool schema rejects it as a non-conforming machine timestamp

### Requirement: Forecast time defaults to the operation clock
`forecasted_at` SHALL be optional for public, sealed, and initial forecast creation through CLI and MCP. When omitted, the application SHALL capture the operation clock exactly once and use that instant for `forecasted_at`, formatted with the ledger `default_timezone` offset; init SHALL use its explicit timezone. Existing `recorded_at` defaulting SHALL use the same captured operation instant, so when both are omitted they SHALL compare equal. The derived value MUST pass the same inclusive forecast-window and ordering checks as an explicit value and MUST NOT be clamped or moved into the window.

#### Scenario: Public forecast defaults both times
- **WHEN** forecast add omits both `forecasted_at` and `recorded_at` while the operation instant lies inside the question window
- **THEN** the appended forecast stores the same captured instant in both fields with the ledger-timezone offset

#### Scenario: Sealed forecast uses the same default
- **WHEN** forecast seal omits `forecasted_at`
- **THEN** sealing targets the single derived operation instant and retains the existing atomic key-before-ledger behavior

#### Scenario: Current time is after close
- **WHEN** forecast creation omits `forecasted_at` and the operation instant is after the question close
- **THEN** the command returns the normal after-close validation error and creates no forecast, key, or journal effect

#### Scenario: Explicit forecast time still wins
- **WHEN** the caller supplies a valid explicit `forecasted_at`
- **THEN** the application uses that value and defaults only fields that remain omitted
