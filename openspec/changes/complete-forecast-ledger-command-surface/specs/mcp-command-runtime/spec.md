## Purpose

Defines a production MCP stdio runtime that exposes the same Forecast Ledger business operations as the CLI through closed schemas, confined roots, simple server-wide safety modes, cancellation, and secret-safe results.

## ADDED Requirements

### Requirement: Start a protocol-clean stdio server
`forecast-ledger mcp serve` SHALL start the pinned supported MCP protocol over stdin/stdout. Protocol stdout MUST contain only framed MCP messages; startup information, warnings, progress, and diagnostics SHALL use stderr. Initialization SHALL advertise binary version, source revision, embedded schema version/commit/digest, built-in network profile ID, online/offline and read/write mode, supported MCP protocol revision, tools/resources capabilities, and experimental timestamp status without writing a human banner to stdout.

#### Scenario: Client initialization
- **WHEN** a compatible client launches the binary and negotiates the supported protocol
- **THEN** initialization and capability discovery succeed with no non-protocol stdout bytes

#### Scenario: Unsupported protocol revision
- **WHEN** a client offers no supported protocol revision
- **THEN** initialization returns a protocol-level negotiation error and terminates without opening ledger files

### Requirement: Confine all files to explicit canonical roots
Serve SHALL require one or more `--ledger-root` values. Repeatable `--output-root` and `--secret-root` values SHALL be optional but required before tools needing those classes can be enabled. Every root SHALL exist, resolve canonically at startup, and be distinct according to the documented overlap policy; secret roots MUST NOT be inside ledger, package, or resource roots.

Every tool path SHALL be resolved against the appropriate root and reject absolute paths outside it, `..`, symlink/junction/reparse escape, Windows drive/UNC escape, case-folding collision, device paths, NUL, and path changes between validation and open. The server MUST NOT expand shell syntax or infer a default ledger.

#### Scenario: Ledger tool traversal
- **WHEN** a tool supplies a file path that escapes through a symlink or junction
- **THEN** it returns a recoverable path error before opening the escaped file

#### Scenario: Secret root overlap
- **WHEN** startup config places a secret root beneath an output root
- **THEN** server startup fails before protocol readiness

### Requirement: Derive access from roots and simple safety modes
The server SHALL expose its full applicable read/write tool surface by default without per-capability grants. Ledger paths SHALL remain confined to explicit ledger roots; package creation requires an output root; private input/key operations require a secret root. A missing required root SHALL fail before opening an unconfined path, reading a secret, acquiring a lock, or creating a file.

`--read-only` SHALL be an optional server-wide mode that rejects every ledger/artifact/package mutation before side effects while preserving read, status, check, and verification tools. `--offline` SHALL be an optional server-wide mode that opens no network socket: network-required stamp/upgrade calls fail as network-disabled, timestamp/layered verification performs local checks with timing not checked, and package verification remains local. Without `--offline`, timestamp and layered verification tools SHALL use the embedded `opentimestamps-public-v1` profile automatically and MUST NOT require network configuration.

MCP calls MUST never prompt. High-risk inputs SHALL include an explicit `confirm: true` field where the CLI requires `--yes`; confirmation records caller intent but is not presented as an authorization boundary. No MCP input schema SHALL accept calendar, explorer, proxy, or arbitrary Bitcoin endpoint URLs. Optional Bitcoin Core override is CLI-only in v1; MCP uses the built-in public profile or offline mode.

#### Scenario: Reveal with confined secret root
- **WHEN** a client calls `forecast_reveal` with `confirm: true` and a key reference inside a configured secret root
- **THEN** the tool executes the shared reveal operation without requiring a separate reveal grant

#### Scenario: Mutation in read-only mode
- **WHEN** a client calls a mutating tool on a server started with `--read-only`
- **THEN** the tool returns a recoverable read-only error before secret, lock, or file side effects

### Requirement: Expose a complete closed tool surface
The server SHALL expose the following CLI-parity tools backed by the same application actions:

| CLI action | MCP tool |
| --- | --- |
| `init` | `ledger_init` |
| `ledger update`, `validate`, `status` | `ledger_update`, `ledger_validate`, `ledger_status` |
| `platform add|update|list|show|remove` | `platform_add`, `platform_update`, `platform_list`, `platform_show`, `platform_remove` |
| `question add|update|list|show|resolve|annul|dispute` | `question_add`, `question_update`, `question_list`, `question_show`, `question_resolve`, `question_annul`, `question_dispute` |
| `forecast add|list|show|seal|reveal` | `forecast_add`, `forecast_list`, `forecast_show`, `forecast_seal`, `forecast_reveal` |
| `target build|check` | `target_build`, `target_check` |
| `timestamp stamp|upgrade|status|verify` | `timestamp_stamp`, `timestamp_upgrade`, `timestamp_status`, `timestamp_verify` |
| `verify` | `verification_run` |
| `publish build|verify` | `publication_build`, `publication_verify` |

Every input and output schema SHALL be closed, typed, versioned where persisted data is involved, and reject unknown fields. Every ledger tool SHALL require `file`; selector and structured input fields SHALL match the CLI contract. Side-effecting tools SHALL describe required root classes, built-in network profile use, server modes, confirmation, dry-run behavior, and rollback semantics in discovery metadata.

#### Scenario: Unknown tool input property
- **WHEN** a client sends a misspelled or unsupported property
- **THEN** schema validation rejects the call before executing the application action

### Requirement: Discover only completed MCP tools
An MCP tool SHALL be registered and returned by `tools/list` only after its shared service operation, closed schemas, root/mode behavior, rollback/cancellation tests, CLI parity tests, documentation, and applicable native/conformance gates pass. A planned or partially implemented tool MUST be absent from discovery and direct calls SHALL receive the protocol's unknown-tool response without opening files, reading secrets, or contacting the network. Completed tools remain usable while later tools are still absent.

#### Scenario: Incremental platform rollout
- **WHEN** `platform_list` is complete but `platform_add` is not
- **THEN** discovery includes `platform_list`, omits `platform_add`, and the server remains usable for completed tools

### Requirement: Preserve exact CLI/MCP business parity
CLI and MCP adapters SHALL call the same transport-neutral operation for parsing, selection, validation, mutation, locking, canonicalization, cryptography, timestamping, verification, publication, and error classification. Given equivalent inputs and observation time, they SHALL return equivalent operation codes and typed data, differing only in transport envelope and interactive approval representation.

#### Scenario: Duplicate forecast ID through both adapters
- **WHEN** CLI and MCP attempt to add the same already-used forecast ID
- **THEN** both return the same conflict code, conflicting IDs, and unchanged ledger bytes

### Requirement: Return recoverable structured tool failures
Expected usage, invalid data, not-found, conflict, verification, I/O, network, pending, unavailable, missing-root, read-only, and offline outcomes SHALL be MCP tool execution errors with stable application code, safe message, redacted details, and retry guidance where applicable. Malformed JSON-RPC/MCP messages and unrecoverable server faults alone SHALL use protocol errors or session termination. One failed tool call MUST NOT corrupt or end an otherwise valid session.

#### Scenario: Pending timestamp upgrade
- **WHEN** no matching built-in-profile calendar can upgrade a valid pending receipt
- **THEN** the tool returns a recoverable `pending` result and the MCP session remains usable

### Requirement: Never expose secrets as tools, arguments, results, or resources
Seal/reveal inputs SHALL use relative protected file references confined to secret roots; raw key, salt, nonce source, credential, and private bundle values MUST NOT appear in schemas, results, logs, tracing, errors, completion metadata, or resources. Private seal input MAY use a protected file reference supplied to the tool but MUST NOT be returned. Revealed keys present in ledger bytes SHALL be redacted from inspection and resource views.

#### Scenario: Successful seal tool
- **WHEN** `forecast_seal` succeeds
- **THEN** result contains ledger/question/forecast IDs, safe artifact/key reference, and recovery guidance but no generated key or private bundle field

### Requirement: Publish redacted read-only resources
The server SHALL expose redacted resources for explicitly addressed ledgers, questions, forecasts, target/receipt summaries, verification reports, and evidence-package manifests under a versioned `forecast-ledger://` URI scheme. Resource URIs SHALL encode a root identifier and safe relative path rather than an absolute machine path. Resource reads SHALL revalidate confinement and current file identity, perform no network or mutation, and omit secret roots entirely.

#### Scenario: Read sealed forecast resource
- **WHEN** a client reads a sealed forecast resource
- **THEN** it receives public note, commitment and evidence state without plaintext, raw key, credential, or protected path

### Requirement: Support cancellation and concurrent clients safely
Every tool call SHALL use its MCP request context and operation timeout. Cancellation SHALL stop bounded file/network work promptly and return interruption without partial mutation. Concurrent read calls MAY proceed; if one writer holds a ledger lock, every second CLI or MCP writer for that ledger SHALL return immediate deterministic `conflict` without waiting or partially reading/modifying state. Different ledgers MAY mutate concurrently within global resource limits.

#### Scenario: Cancel package build
- **WHEN** the client cancels after some package files are staged
- **THEN** recovery handles only server-created partial files and the tool returns interruption without a complete manifest claim

### Requirement: Bound server resource consumption
Serve SHALL enforce reviewed limits for message size, JSON/schema depth, concurrent requests, open files, ledger/input bytes, target/receipt/package bytes, network response bytes, redirects, and operation duration. Limit failures SHALL be recoverable and MUST NOT allocate or log attacker-controlled unbounded data. Protocol network clients SHALL use only built-in profile endpoints with pinned HTTPS hosts and redirect/SSRF protections; explicit outcome-source checking SHALL reject private, link-local, reserved, and redirect-escaped destinations.

#### Scenario: Oversized tool input
- **WHEN** a request exceeds the configured maximum before full decoding
- **THEN** it is rejected safely and the server remains available for subsequent valid requests

### Requirement: Prove real-process protocol and mode behavior
Release gates SHALL include in-memory and real child-process stdio tests for initialization/version negotiation, stdout purity, every tool schema, default full mode, read-only/offline modes, root-class combinations, built-in network-profile confinement, CLI parity, traversal and root races, secret canaries, concurrent calls, cancellation, oversized/malformed messages, graceful EOF/shutdown, and previous supported protocol compatibility. `mcp serve` MUST remain hidden/unavailable until the server can initialize and serve its declared surface.

#### Scenario: Human log on protocol stdout
- **WHEN** any startup or tool path writes non-protocol bytes to stdout
- **THEN** the real-process framing test and release gate fail
