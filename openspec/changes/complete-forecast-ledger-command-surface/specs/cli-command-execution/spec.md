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
| `ledger update` | `--file` |
| `platform add|update|show|remove` | `--file --platform` |
| `platform list` | `--file` |
| `question add|update|show|resolve|annul|dispute` | `--file --question` |
| `question list` | `--file` |
| `forecast add|show|seal|reveal` | `--file --question --forecast` |
| `forecast key-hint update` | `--file --question --forecast` |
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

### Requirement: Refuse unsupported ledger contracts
Every action that reads an existing ledger SHALL inspect the bounded root and require `schema_version` exactly `1.0.0` before domain mutation, artifact creation, crypto, network, or MCP resource publication. An absent, unknown, or future version SHALL return stable code `unsupported_schema_version` in the invalid-data category with exit `3`, identify the supported version, and MUST NOT coerce, migrate, fetch, or validate against a floating remote contract.

#### Scenario: Future schema version
- **WHEN** a ledger declares `schema_version: 2.0.0`
- **THEN** the action returns `unsupported_schema_version`/exit `3` before any side effect and reports that only `1.0.0` is supported

### Requirement: Stable time input
Persisted timestamps SHALL be RFC 3339 values with seconds and an explicit offset. Commands MAY default a write-time field such as `created_at`, `recorded_at`, or `revealed_at` to the current clock, but MUST report the exact stored value and MUST accept an explicit value for reproducible use and testing. The CLI MUST NOT present a self-reported timestamp or local clock as cryptographic evidence.

#### Scenario: Default recorded time
- **WHEN** a forecast input omits `recorded_at`
- **THEN** the command captures one clock value, validates it against `forecasted_at`, stores it unchanged, and returns it in the result

### Requirement: Consistent zero-config and offline network behavior
Every action that can contact the network SHALL identify the built-in profile or explicit advanced source mode it will use and SHALL support `--offline`. Offline mode SHALL override automatic protocol-source use and explicit outcome-source checking, open no socket, and either continue with documented local-only results or fail before side effects when the action inherently requires a network response. Ordinary OpenTimestamps and Bitcoin public verification MUST NOT require calendar/explorer configuration. Only timestamp stamp/upgrade MAY accept the documented explicit CLI custom-calendar mode, and only timestamp/layered/package verification MAY accept the documented CLI Bitcoin Core override; MCP accepts neither arbitrary calendar nor Bitcoin endpoint URLs in v1.

#### Scenario: Offline layered verification
- **WHEN** a user runs `verify --offline`
- **THEN** all local layers run, network-dependent layers remain pending or not checked, and no source URL is contacted

#### Scenario: Visible automatic profile
- **WHEN** a network-capable command uses the embedded public profile
- **THEN** human and JSON results identify its stable profile ID and safe contacted source IDs

### Requirement: Transactional mutation and format preservation
Every ledger mutation SHALL acquire the cross-platform ledger lock, parse and fully validate the current document, apply a minimal source-tree patch, fully validate the prospective document, and perform a recoverable same-directory replacement. It MUST preserve JSON versus YAML, newline convention, and untouched YAML comments, order, scalar style, and unknown presentation details allowed by the schema. An error or interruption before commit MUST leave the original ledger byte-for-byte unchanged.

#### Scenario: Invalid prospective state
- **WHEN** a platform removal would leave a dangling platform reference
- **THEN** post-change validation fails, no replacement occurs, and the original bytes remain unchanged

#### Scenario: Concurrent mutation
- **WHEN** a CLI or MCP writer already holds the ledger lock
- **THEN** a second writer returns `conflict` without performing a partial read-modify-write

### Requirement: Dry-run and confirmation
Every command that mutates a ledger, creates or replaces an artifact, writes a key, or creates a package SHALL support `--dry-run`. A read-only command does not gain dry-run merely because it can observe the network; `--offline` controls network use for layered `verify` and `publish verify`. Dry-run SHALL perform all local parsing, selection, permission, collision, and prospective validation possible without generating a real secret, writing a file, acquiring a remote result, or changing state; it SHALL return a structured plan that identifies deferred checks. Dry-run success MUST NOT claim that a later network or concurrent write will succeed.

Commands that disclose previously sealed material, remove a platform, replace terminal question state, or would overwrite through a separately approved replacement flow SHALL require interactive confirmation or `--yes`. With `--no-input`, non-TTY stdin, or MCP execution, missing explicit approval SHALL fail before side effects.

#### Scenario: Seal dry-run
- **WHEN** a user runs `forecast seal --dry-run`
- **THEN** input, selectors, destination policy, and prospective ledger shape are checked, but no key/nonce/salt is generated, no key or ledger file is written, and output marks cryptographic bytes as deferred

#### Scenario: Non-interactive reveal without approval
- **WHEN** reveal would disclose a sealed forecast and stdin is not interactive without `--yes`
- **THEN** it fails before reading the key file or changing the ledger

#### Scenario: Read-only online verification
- **WHEN** layered `verify` or `publish verify --online` is selected
- **THEN** the command performs no persistent mutation, exposes no `--dry-run`, and uses `--offline` or omission of `--online` as its documented network control

### Requirement: Stable result and error envelopes
Human output SHALL be concise English. JSON success SHALL be one object with `ok: true`, stable operation `code`, plain `message`, and typed `data`; JSON failure SHALL be one object on stderr with `ok: false`, stable error `code`, plain `message`, and optional redacted `details`. Primary success output belongs only on stdout; errors, warnings, progress, and verbose diagnostics belong only on stderr.

`--plain` SHALL produce undecorated UTF-8 line-oriented output with no color, animation, border, heading, or progress on stdout; repeated records SHALL use a documented stable tab-separated field order. `--quiet` SHALL suppress all successful primary stdout while preserving warnings and failures on stderr and the process exit. `--json`, `--plain`, and `--quiet` SHALL be mutually exclusive; `--verbose` MAY add redacted diagnostics only to stderr.

The CLI exit mapping SHALL remain: `0` success, `1` unexpected internal failure, `2` usage, `3` invalid or unsupported-schema data, `4` not found, `5` conflict/precondition including read-only or missing-root conditions, `6` cryptographic or evidence verification failure, `7` local I/O including operating-system access denial, `8` network/remote/network-disabled failure, `9` pending/not ready/incomplete verification, `10` unavailable, and `130` interrupted. There is no separate CLI `permission` class.

#### Scenario: JSON mutation success
- **WHEN** `platform add --json` succeeds
- **THEN** stdout contains exactly one success envelope with ledger ID, platform ID, changed path, and stored platform while stderr is empty unless verbose diagnostics were explicitly requested

#### Scenario: Expected business conflict
- **WHEN** an add command receives an existing ID
- **THEN** it returns `conflict`/exit `5`, not `internal`, with the conflicting stable ID in redacted details

#### Scenario: Quiet success
- **WHEN** a successful mutation runs with `--quiet`
- **THEN** stdout is empty, warnings remain on stderr, and the process exits `0`

#### Scenario: Conflicting output modes
- **WHEN** a user combines any two of `--json`, `--plain`, and `--quiet`
- **THEN** argument handling returns usage/exit `2` before the action

### Requirement: Secret-safe and private-safe output
Raw keys, salts, nonces before publication, sealed plaintext, credentials, protected file contents, and absolute secret paths MUST NOT appear in help examples, argv values, environment inputs, normal or verbose logs, result/error envelopes, panic text, telemetry, MCP resources, or evidence packages. Ledger validation errors MUST identify locations and rules without echoing private source values. Revealed keys stored by the published schema SHALL remain redacted from CLI/MCP inspection output.

#### Scenario: Output writer failure after secret operation
- **WHEN** reporting a seal result fails
- **THEN** fallback diagnostics contain no generated key, private bundle field, or absolute key path

### Requirement: Cancellation and bounded external work
Every action SHALL honor inherited cancellation and `--timeout`. File and cryptographic loops SHALL check cancellation at bounded intervals; network requests SHALL use the operation context, response-size limits, redirect policy, and the built-in profile or documented CLI Core endpoint. Interruption SHALL return exit `130` and preserve or recover the last committed coherent state.

#### Scenario: Interrupt during calendar request
- **WHEN** the operation context is canceled while stamping
- **THEN** outstanding requests stop promptly, no pending ledger state is recorded without a retained valid receipt, and recoverable artifacts are reported

### Requirement: Cross-platform observable parity
The same ledger and explicit inputs SHALL produce equivalent domain results, JSON field names, deterministic target/manifest bytes, error codes, and path-safety decisions on supported macOS, Linux, and Windows systems. Platform-specific locking, ACL, replacement, separators, drive letters, UNC paths, symlinks, junctions, and case folding MUST NOT weaken confinement or change logical identifiers.

#### Scenario: Windows path escape
- **WHEN** a relative artifact path resolves through a junction or drive/UNC escape outside its allowed root
- **THEN** the operation rejects the path before opening or creating the target

### Requirement: Correct the registered preview tree before availability
The existing hidden/unavailable urfave tree is a preview scaffold, not a compatibility promise for completed actions. Before the affected leaf becomes visible, implementation and help goldens SHALL change `target check`, all timestamp actions, and layered `verify` to reject `--file -`; SHALL make `timestamp verify` a mutating/network-capable action with `--dry-run`; SHALL keep the already registered required scalar `question add --type`; SHALL make MCP root flags repeatable; and SHALL retain the registered reveal gate while removing the general write/network grants. Release notes SHALL identify these preview corrections, and no command may be advertised with the old contradictory contract.

#### Scenario: Preview timestamp verify becomes available
- **WHEN** `timestamp verify` passes its implementation gate
- **THEN** its visible help requires a real file, includes dry-run/offline behavior, and no golden or completion advertises the old read-only stdin form
