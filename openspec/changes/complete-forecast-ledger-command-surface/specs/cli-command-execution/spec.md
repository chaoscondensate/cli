## Purpose

Defines the common observable execution contract for every Forecast Ledger CLI action so humans, scripts, and MCP clients receive consistent selection, safety, output, and failure behavior.

## ADDED Requirements

### Requirement: Only implemented commands are advertised
The root help and shell completion SHALL advertise a command only after its application action and command-specific acceptance tests are complete. A registered but incomplete action MUST remain hidden and MUST return `unavailable` with exit `10` without reading secrets, opening a network connection, or changing a file.

#### Scenario: Command becomes available
- **WHEN** the implementation and acceptance gates for `platform add` pass
- **THEN** the command becomes visible in root help and completion in the same change that removes its unavailable action

#### Scenario: Incomplete command invoked explicitly
- **WHEN** a user invokes a registered action whose implementation gate is still open
- **THEN** it returns exit `10`, emits machine code `unavailable` in JSON mode, and has no side effect

### Requirement: Leaf-local file and record selection
Every ledger action SHALL require leaf-local `--file <path>` or `-f <path>`. Record actions SHALL use `--platform`, `--question`, and `--forecast` stable IDs exactly as applicable; array positions, titles, timestamps, current directory scans, environment variables, config defaults, and prior invocations MUST NOT select a ledger or record.

The command selector contract SHALL be:

| Commands | Required selection |
| --- | --- |
| `init` | `--file` names a new ledger |
| `platform add|update|show|remove` | `--file --platform` |
| `platform list` | `--file` |
| `question add|update|show|resolve|annul|dispute` | `--file --question` |
| `question list` | `--file` |
| `forecast add|show|seal|reveal` | `--file --question --forecast` |
| `forecast list` | `--file --question` |
| `target build|check` | `--file` plus `--all` or both `--question --forecast` |
| `timestamp stamp|upgrade|status|verify` | `--file --question --forecast` |
| `verify` | `--file`; optional `--question`, and optional `--forecast` only with `--question` |
| `publish build` | `--file --output` |
| `publish verify` | `--file --manifest` |
| `mcp serve` | one or more `--ledger-root`; it is not a single-ledger action |

#### Scenario: Selector ambiguity
- **WHEN** a user combines `--all` with a question or forecast selector
- **THEN** the command fails as usage before reading the ledger

#### Scenario: Forecast without question
- **WHEN** a user supplies `--forecast` without `--question`
- **THEN** the command fails as usage and explains that forecast lookup is scoped by question even though forecast IDs are globally unique

### Requirement: Bounded typed input documents
Commands that accept a structured record or secret bundle SHALL use `--input <path>` or `--input -` and SHALL parse a closed JSON or YAML object documented for that command. Unknown fields, duplicate keys, aliases, excessive size/depth, floats where the contract uses exact strings or integers, and a second JSON/YAML document MUST be rejected. Scalar convenience flags MAY exist only when they map unambiguously to the same typed input and MUST NOT accept secrets.

#### Scenario: Unknown update field
- **WHEN** a question update document includes a field outside the documented patch schema
- **THEN** the command fails as usage or invalid input and does not ignore the field

#### Scenario: Secret bundle on stdin
- **WHEN** `forecast seal --input -` receives one bounded private bundle on stdin
- **THEN** the bundle is parsed without copying its plaintext into argv, environment variables, diagnostics, or output

### Requirement: Explicit stdin eligibility
`--file -` SHALL be accepted only by local read-only actions that do not require sibling artifacts: `validate`, `status`, `platform list|show`, `question list|show`, and `forecast list|show`. Every mutation, target, timestamp, layered verification, and publication action MUST require a real ledger path. A command using stdin for its ledger MUST NOT also use `--input -`.

#### Scenario: Two stdin consumers
- **WHEN** a user supplies both `--file -` and `--input -`
- **THEN** the command fails as usage before reading stdin

#### Scenario: Sidecar action with stdin ledger
- **WHEN** a user invokes `verify --file -`
- **THEN** the command rejects stdin because target and receipt paths require a real ledger directory

### Requirement: Stable time input
Persisted timestamps SHALL be RFC 3339 values with seconds and an explicit offset. Commands MAY default a write-time field such as `created_at`, `recorded_at`, or `revealed_at` to the current clock, but MUST report the exact stored value and MUST accept an explicit value for reproducible use and testing. The CLI MUST NOT present a self-reported timestamp or local clock as cryptographic evidence.

#### Scenario: Default recorded time
- **WHEN** a forecast input omits `recorded_at`
- **THEN** the command captures one clock value, validates it against `forecasted_at`, stores it unchanged, and returns it in the result

### Requirement: Transactional mutation and format preservation
Every ledger mutation SHALL acquire the cross-platform ledger lock, parse and fully validate the current document, apply a minimal source-tree patch, fully validate the prospective document, and perform a recoverable same-directory replacement. It MUST preserve JSON versus YAML, newline convention, and untouched YAML comments, order, scalar style, and unknown presentation details allowed by the schema. An error or interruption before commit MUST leave the original ledger byte-for-byte unchanged.

#### Scenario: Invalid prospective state
- **WHEN** a platform removal would leave a dangling platform reference
- **THEN** post-change validation fails, no replacement occurs, and the original bytes remain unchanged

#### Scenario: Concurrent mutation
- **WHEN** a CLI or MCP writer already holds the ledger lock
- **THEN** a second writer returns `conflict` without performing a partial read-modify-write

### Requirement: Dry-run and confirmation
Every command that mutates a ledger, creates or replaces an artifact, writes a key, creates a package, or uses the network SHALL support `--dry-run`. Dry-run SHALL perform all local parsing, selection, permission, collision, and prospective validation possible without generating a real secret, writing a file, acquiring a remote result, or changing state; it SHALL return a structured plan that identifies deferred checks. Dry-run success MUST NOT claim that a later network or concurrent write will succeed.

Commands that disclose previously sealed material, remove a platform, replace terminal question state, or would overwrite through a separately approved replacement flow SHALL require interactive confirmation or `--yes`. With `--no-input`, non-TTY stdin, or MCP execution, missing explicit approval SHALL fail before side effects.

#### Scenario: Seal dry-run
- **WHEN** a user runs `forecast seal --dry-run`
- **THEN** input, selectors, destination policy, and prospective ledger shape are checked, but no key/nonce/salt is generated, no key or ledger file is written, and output marks cryptographic bytes as deferred

#### Scenario: Non-interactive reveal without approval
- **WHEN** reveal would disclose a sealed forecast and stdin is not interactive without `--yes`
- **THEN** it fails before reading the key file or changing the ledger

### Requirement: Stable result and error envelopes
Human output SHALL be concise English. JSON success SHALL be one object with `ok: true`, stable operation `code`, plain `message`, and typed `data`; JSON failure SHALL be one object on stderr with `ok: false`, stable error `code`, plain `message`, and optional redacted `details`. Primary success output belongs only on stdout; errors, warnings, progress, and verbose diagnostics belong only on stderr.

The CLI exit mapping SHALL remain: `0` success, `1` unexpected internal failure, `2` usage, `3` invalid data, `4` not found, `5` conflict/precondition, `6` cryptographic or evidence verification failure, `7` local I/O, `8` network/remote failure, `9` pending/not ready, `10` unavailable, and `130` interrupted.

#### Scenario: JSON mutation success
- **WHEN** `platform add --json` succeeds
- **THEN** stdout contains exactly one success envelope with ledger ID, platform ID, changed path, and stored platform while stderr is empty unless verbose diagnostics were explicitly requested

#### Scenario: Expected business conflict
- **WHEN** an add command receives an existing ID
- **THEN** it returns `conflict`/exit `5`, not `internal`, with the conflicting stable ID in redacted details

### Requirement: Secret-safe and private-safe output
Raw keys, salts, nonces before publication, sealed plaintext, credentials, protected file contents, and absolute secret paths MUST NOT appear in help examples, argv values, environment inputs, normal or verbose logs, result/error envelopes, panic text, telemetry, MCP resources, or evidence packages. Ledger validation errors MUST identify locations and rules without echoing private source values. Revealed keys stored by the published schema SHALL remain redacted from CLI/MCP inspection output.

#### Scenario: Output writer failure after secret operation
- **WHEN** reporting a seal result fails
- **THEN** fallback diagnostics contain no generated key, private bundle field, or absolute key path

### Requirement: Cancellation and bounded external work
Every action SHALL honor inherited cancellation and `--timeout`. File and cryptographic loops SHALL check cancellation at bounded intervals; network requests SHALL use the operation context, response-size limits, redirect policy, and explicit endpoints. Interruption SHALL return exit `130` and preserve or recover the last committed coherent state.

#### Scenario: Interrupt during calendar request
- **WHEN** the operation context is canceled while stamping
- **THEN** outstanding requests stop promptly, no pending ledger state is recorded without a retained valid receipt, and recoverable artifacts are reported

### Requirement: Cross-platform observable parity
The same ledger and explicit inputs SHALL produce equivalent domain results, JSON field names, deterministic target/manifest bytes, error codes, and path-safety decisions on supported macOS, Linux, and Windows systems. Platform-specific locking, ACL, replacement, separators, drive letters, UNC paths, symlinks, junctions, and case folding MUST NOT weaken confinement or change logical identifiers.

#### Scenario: Windows path escape
- **WHEN** a relative artifact path resolves through a junction or drive/UNC escape outside its allowed root
- **THEN** the operation rejects the path before opening or creating the target

