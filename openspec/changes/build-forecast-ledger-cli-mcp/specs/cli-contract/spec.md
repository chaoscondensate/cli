## Purpose

Defines a safe, discoverable, scriptable command-line contract that behaves consistently across supported desktop and server platforms.

## ADDED Requirements

### Requirement: Explicit ledger selection
Every command that reads or changes a ledger SHALL require `--file <path>` or `-f <path>` and SHALL NOT infer a ledger from the current directory, configuration, or environment. Read-only commands that do not need sidecars MAY accept `--file -` for stdin; mutating, timestamp, and evidence-package build commands MUST require a real path.

#### Scenario: Missing file flag
- **WHEN** a user invokes a ledger command without `--file`
- **THEN** the command fails as invalid usage, explains how to pass the file, and does not read or change any ledger

#### Scenario: Two ledgers in one directory
- **WHEN** a directory contains multiple ledger files and the user passes one with `--file`
- **THEN** the command operates only on the named ledger and its explicitly derived artifacts

### Requirement: Discoverable English interface
The CLI SHALL use short, plain English command names, descriptions, errors, and examples. The root command and every command group SHALL provide concise no-argument help plus complete `-h`, `--help`, and `help` output.

#### Scenario: Command group without an action
- **WHEN** a user invokes a command group without a child command
- **THEN** the CLI prints a short description, one or two examples, common child commands, and a pointer to full help

### Requirement: Stable output channels and formats
Primary results SHALL be written to stdout; diagnostics, warnings, progress, and errors SHALL be written to stderr. Human output SHALL be the default, while `--json` SHALL produce one documented JSON value with stable field names and no decoration.

#### Scenario: JSON validation result in a pipeline
- **WHEN** a user runs validation with `--json` and redirects stdout
- **THEN** stdout contains only valid JSON and any progress or diagnostic text appears only on stderr

#### Scenario: Non-interactive output
- **WHEN** stdout is not a TTY, `NO_COLOR` is set, `TERM=dumb`, or `--no-color` is passed
- **THEN** the CLI emits no color escapes, animation, or interactive progress controls on stdout

### Requirement: Stable failure classification
The CLI SHALL return `0` on success and stable non-zero exit categories for usage, invalid ledger data, missing resources, conflicts, verification failure, local I/O failure, network failure, pending/not-ready state, and interruption. JSON errors SHALL include a stable machine code, a plain-English message, and optional structured details without secrets.

#### Scenario: Semantic validation failure
- **WHEN** a ledger is well-formed JSON or YAML but violates a Forecast Ledger semantic rule
- **THEN** the command exits with the invalid-data category and reports the exact document path and rule

#### Scenario: Interrupted operation
- **WHEN** the process receives an interrupt during a long network or file operation
- **THEN** it stops promptly, exits with the interruption category, and leaves no partial ledger mutation

### Requirement: Safe and recoverable mutation
Mutating commands SHALL parse and validate the current file, lock it against concurrent writers, apply the change to an in-memory copy, validate the result, and replace the original through a recoverable same-directory write. They SHALL preserve JSON versus YAML format, newline convention, and comments/order for untouched YAML nodes, refuse unintended overwrite, and support `--dry-run` where an operation changes multiple files or crosses a network boundary.

#### Scenario: Post-mutation validation fails
- **WHEN** a requested mutation would produce an invalid ledger
- **THEN** the command reports the validation errors and leaves the original ledger byte-for-byte unchanged

#### Scenario: Concurrent writer
- **WHEN** another CLI or MCP operation holds the ledger write lock
- **THEN** the command fails with a conflict and does not race or partially overwrite the file

### Requirement: Secret-safe input
Keys, salts, sealed plaintext, credentials, and other secrets MUST NOT be accepted as literal command-line flags or environment variables and MUST NOT appear in normal, JSON, verbose, or error output. Secret input SHALL use protected files, stdin, or a supported secret-manager channel.

#### Scenario: Seal command help
- **WHEN** a user inspects help for a sealing or reveal command
- **THEN** no option permits a raw key, salt, or sealed plaintext value in argv

### Requirement: Portable single-binary distribution
Official releases SHALL provide self-contained binaries for macOS, Linux, and Windows on the declared architectures, with version metadata and checksums. Core ledger, cryptographic, MCP, and timestamp behavior SHALL NOT require Python or another language runtime.

#### Scenario: Fresh supported machine
- **WHEN** a user downloads the correct release artifact to a supported machine with no project runtime installed
- **THEN** the binary can print help, validate a ledger, and start the MCP stdio server
