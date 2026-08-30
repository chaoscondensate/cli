## Purpose

Defines readable version information for people while preserving deterministic, undecorated output for scripts and non-interactive environments.

## ADDED Requirements

### Requirement: Human version output is structured and readable
`forecast-ledger version` SHALL present the program version and available build, source, embedded schema, Go, and MCP metadata as short labeled lines in human mode. Missing optional build values SHALL use one consistent human placeholder and MUST NOT suppress known metadata or cause failure. The command SHALL make no network request.

#### Scenario: Human reads complete version metadata
- **WHEN** a terminal user runs `forecast-ledger version` from a release build
- **THEN** stdout shows a compact labeled summary containing the release version and all available build and compatibility pins

#### Scenario: Development build has missing metadata
- **WHEN** optional linker-provided build fields are absent
- **THEN** the command succeeds and displays a consistent placeholder for those fields while retaining embedded schema metadata

### Requirement: Version color follows the global output contract
Human version output MAY use restrained ANSI color only for labels or status accents when stdout is a color-capable TTY. It SHALL contain no ANSI escapes when `--no-color` is set, `NO_COLOR` is non-empty, `TERM=dumb`, stdout is not a TTY, or plain or JSON mode is selected. Color MUST NOT be the only way metadata is distinguished.

#### Scenario: Interactive human output may be colored
- **WHEN** human version output is written to a supported TTY and no color-disabling control applies
- **THEN** labels may contain the same restrained palette used by other CLI presentation without changing the text values

#### Scenario: Redirected version output is clean
- **WHEN** version output is redirected or any supported color-disabling control applies
- **THEN** stdout contains no ANSI escape sequence and remains readable line by line

### Requirement: Machine-readable version output remains stable
`forecast-ledger version --json` SHALL emit exactly one stable JSON value with the established field names, values, and types and no decoration. Plain mode SHALL emit stable parseable text without color. Human formatting improvements MUST NOT alter build metadata values, compatibility pins, diagnostics routing, or successful exit status.

#### Scenario: JSON consumers are unaffected
- **WHEN** a caller runs `forecast-ledger version --json`
- **THEN** stdout contains one undecorated JSON value compatible with the existing version output contract

#### Scenario: Diagnostics do not contaminate stdout
- **WHEN** version presentation emits a warning or diagnostic
- **THEN** the diagnostic is written to stderr and the selected stdout format remains valid

