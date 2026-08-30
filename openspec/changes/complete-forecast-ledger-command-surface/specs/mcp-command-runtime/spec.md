> Supersession note (2026-08-30): explicit `tsa_url`/`ca_bundle` requirements
> below record completed v0.4.0 behavior and are superseded for the next release
> by `add-default-tsa-failover`.

> Supersession note (2026-08-30): generic authoring documents, public
> `--input`, MCP `input`/public `input_file`, v1.2.0 identity, and the old target
> shape below are historical implementation facts. `make-authoring-direct-readable`
> owns their v1.3.0 replacements and takes precedence.

## Purpose

Defines a production MCP stdio runtime that exposes the same Forecast Ledger business operations as the CLI through closed schemas, confined roots, simple server-wide safety modes, cancellation, and secret-safe results.

## ADDED Requirements

### Requirement: Start a protocol-clean stdio server
`forecast-ledger mcp serve` SHALL start the pinned supported MCP protocol over stdin/stdout. Protocol stdout MUST contain only framed MCP messages; startup information, warnings, progress, and diagnostics SHALL use stderr. Initialization SHALL advertise binary version, source revision, embedded schema version/commit/digest, online/offline and read/write mode, supported MCP protocol revision, tools/resources capabilities, and experimental timestamp status without writing a human banner to stdout.

#### Scenario: Client initialization
- **WHEN** a compatible client launches the binary and negotiates the supported protocol
- **THEN** initialization and capability discovery succeed with no non-protocol stdout bytes

#### Scenario: Unsupported protocol revision
- **WHEN** a client offers no supported protocol revision
- **THEN** initialization returns a protocol-level negotiation error and terminates without opening ledger files

### Requirement: Confine all files to explicit canonical roots
Serve SHALL require one or more `--ledger-root` values. Repeatable `--output-root` and `--secret-root` values SHALL be optional but required before tools needing those classes can be enabled. Every root SHALL exist, resolve canonically at startup, and be distinct according to the documented overlap policy; secret roots MUST NOT be inside ledger, package, or resource roots. A root confines locations but SHALL NOT by itself authorize irreversible reveal.

Every tool path SHALL be resolved against the appropriate root and reject absolute paths outside it, `..`, symlink/junction/reparse escape, Windows drive/UNC escape, case-folding collision, device paths, NUL, and path changes between validation and open. The server MUST NOT expand shell syntax or infer a default ledger.

A startup, missing-root, overlap, or confinement error SHALL identify the applicable stable root class (`ledger`, `output`, or `secret`), the originating flag, and, when configured roots are numbered or named, its safe root identifier. A missing or invalid configured route SHALL name that one safe class/identifier. An overlap SHALL name both conflicting class/identifier descriptors so the operator can correct the right pair. These failures MUST NOT be reduced to indistinguishable generic messages or expose configured absolute paths.

#### Scenario: Ledger tool traversal
- **WHEN** a tool supplies a file path that escapes through a symlink or junction
- **THEN** it returns a recoverable path error before opening the escaped file

#### Scenario: Secret root overlap
- **WHEN** startup config places a secret root beneath an output root
- **THEN** server startup fails before protocol readiness and identifies both the `secret` and `output` route IDs without displaying either absolute path

#### Scenario: Configured ledger route does not exist
- **WHEN** `--ledger-root main=<path>` names a path that cannot be opened as a root
- **THEN** startup identifies `--ledger-root`, root class `ledger`, and safe route ID `main` rather than only saying that an artifact root does not exist

#### Scenario: Tool requires an output root
- **WHEN** a package-building call has no configured output root
- **THEN** the recoverable error identifies the `output` root class without disclosing any absolute ledger or secret path

### Requirement: Derive access from roots, safety modes, and an explicit reveal boundary
The server SHALL expose its applicable read/write tool surface by default without general write or network grants. Ledger paths SHALL remain confined to explicit ledger roots; package creation requires an output root; private input and seal key creation require a secret root. A missing required root SHALL fail before opening an unconfined path, reading a secret, acquiring a lock, or creating a file.

`forecast_reveal` SHALL additionally require the server to start with default-off `--allow-reveal`. A secret root alone MUST NOT enable disclosure. Without the flag the tool SHALL be absent from discovery and a direct call SHALL receive the protocol's unknown-tool response without reading a key or ledger. With the flag, reveal still requires a secret-root reference, write-capable mode, and request-level `confirm: true`. Combining `--allow-reveal` with `--read-only`, or enabling reveal without any secret root, SHALL fail startup as contradictory configuration.

`--read-only` SHALL be an optional server-wide mode that startup-disables every ledger/artifact/package mutation while preserving read, status, check, and verification tools. Mutating tools SHALL be absent from discovery in this mode, and a direct call SHALL receive the protocol's unknown-tool response before secret, lock, file, or network side effects. `--offline` SHALL be an optional server-wide mode that opens no network socket: network-required stamp calls fail as network-disabled, while timestamp status, timestamp verify, layered timestamp checks, and package verification still perform their complete local checks. General layered verification also skips explicitly requested outcome-source retrieval.

MCP calls MUST never prompt. High-risk inputs SHALL include an explicit `confirm: true` field where the CLI requires `--yes`; confirmation records caller intent but is not presented as an authorization boundary and does not replace `--allow-reveal`. `timestamp_stamp` SHALL require explicit `tsa_url` and `ca_bundle` fields equivalent to the CLI inputs. No other MCP input schema SHALL accept a TSA, remote timestamp source, proxy, arbitrary request headers, credential-bearing URL, or general HTTP configuration.

#### Scenario: Reveal with confined secret root
- **WHEN** a client calls `forecast_reveal` with `confirm: true` and a key reference inside a configured secret root
- **THEN** the tool executes the shared reveal operation only when the server also started with `--allow-reveal`

#### Scenario: Seal-only MCP server
- **WHEN** a write-capable server has a secret root but omits `--allow-reveal`
- **THEN** `forecast_seal` may be discovered while `forecast_reveal` is absent and cannot read or disclose an existing key

#### Scenario: Mutation in read-only mode
- **WHEN** a client calls a mutating tool on a server started with `--read-only`
- **THEN** the tool is absent from discovery and the direct call returns the protocol's unknown-tool response before secret, lock, file, or network side effects

### Requirement: Expose a complete closed tool surface
The server SHALL expose the following CLI-parity tools backed by the same application actions:

| CLI action | MCP tool |
| --- | --- |
| `init` | `ledger_init` |
| `ledger update`, `validate`, `status` | `ledger_update`, `ledger_validate`, `ledger_status` |
| `platform add|update|list|show|remove` | `platform_add`, `platform_update`, `platform_list`, `platform_show`, `platform_remove` |
| `question add|update|list|show|resolve|annul|dispute` | `question_add`, `question_update`, `question_list`, `question_show`, `question_resolve`, `question_annul`, `question_dispute` |
| `forecast add|list|show|seal|reveal`, `forecast key-hint update` | `forecast_add`, `forecast_list`, `forecast_show`, `forecast_seal`, `forecast_reveal`, `forecast_key_hint_update` |
| `target build|check` | `target_build`, `target_check` |
| `timestamp stamp|status|verify` | `timestamp_stamp`, `timestamp_status`, `timestamp_verify` |
| `verify` | `verification_run` |
| `publish build|verify` | `publication_build`, `publication_verify` |

Every input and output schema SHALL be closed, typed, versioned where persisted data is involved, and reject unknown fields. Every ledger tool SHALL require `file`; selector and structured input fields SHALL match the CLI contract. Discovery metadata for every tool SHALL describe its static effect as either `read-only` or `mutating`, independently of the server's current mode. It SHALL separately describe applicable root classes, explicit network use where applicable, current server mode, confirmation, dry-run behavior, and rollback semantics. A read tool MUST NOT be described as mutating merely because the server is write-capable, and a mutating tool MUST NOT be described as read-only because the server started in read-only mode.

#### Scenario: Unknown tool input property
- **WHEN** a client sends a misspelled or unsupported property
- **THEN** schema validation rejects the call before executing the application action

#### Scenario: Static tool effect is independent of server mode
- **WHEN** `ledger_status` is discovered on a write-capable server and `forecast_add` is considered for a read-only server
- **THEN** `ledger_status` remains labeled `read-only`, while `forecast_add` remains statically `mutating` and is omitted because the current mode disables it

### Requirement: Discover only completed MCP tools
An MCP tool SHALL be registered and returned by `tools/list` only after its shared service operation, closed schemas, root/mode behavior, rollback/cancellation tests, CLI parity tests, documentation, and applicable native/conformance gates pass. A planned, partially implemented, or startup-disabled tool, including reveal without `--allow-reveal` and every mutating tool in read-only mode, MUST be absent from discovery and direct calls SHALL receive the protocol's unknown-tool response without opening files, reading secrets, or contacting the network. Completed and startup-enabled tools remain usable while later or disabled tools are absent.

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

#### Scenario: Pending timestamp verify
- **WHEN** a retained timestamp entry cannot complete local verification because required request, response, or CA-bundle material is missing
- **THEN** the tool returns a recoverable `pending` or incomplete result and the MCP session remains usable

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
Every tool call SHALL use its MCP request context and operation timeout. Cancellation SHALL stop bounded file/network work promptly and return interruption without partial mutation. Concurrent read calls MAY proceed; if one writer holds a ledger lock, every second CLI or MCP writer for that ledger SHALL return immediate deterministic `conflict` without waiting or partially reading/modifying state. Request timeout does not create a ledger-writer queue or define a lock-wait duration; callers that intentionally contend SHALL serialize calls or use bounded retry/backoff. Different ledgers MAY mutate concurrently within global resource limits.

#### Scenario: Cancel package build
- **WHEN** the client cancels after some package files are staged
- **THEN** recovery handles only server-created partial files and the tool returns interruption without a complete manifest claim

### Requirement: Bound server resource consumption
Serve SHALL enforce reviewed limits for message size, JSON/schema depth, concurrent requests, open files, ledger/input bytes, target/request/response/package bytes, ASN.1/CMS nesting and collection counts, network response bytes, redirects, operation duration, total HTTP requests, and network concurrency. Limit failures SHALL be recoverable and MUST NOT allocate or log attacker-controlled unbounded data. Timestamp stamp SHALL validate its explicit public HTTPS TSA destination and reject private, link-local, reserved, credential-bearing, and cross-origin-redirected destinations. Explicit outcome-source checking SHALL apply the same destination classes. Timestamp/layered/package verification SHALL open only confined retained local evidence.

#### Scenario: Oversized tool input
- **WHEN** a request exceeds the configured maximum before full decoding
- **THEN** it is rejected safely and the server remains available for subsequent valid requests

### Requirement: Prove real-process protocol and mode behavior
Release gates SHALL include in-memory and real child-process stdio tests for initialization/version negotiation, stdout purity, every tool schema, default full mode, reveal-disabled/reveal-enabled discovery, contradictory reveal/read-only startup, read-only/offline modes, root-class combinations, explicit TSA confinement and budgets, local RFC evidence verification, CLI parity, traversal and root races, secret canaries, concurrent calls, cancellation, oversized/malformed messages, graceful EOF/shutdown, and previous supported protocol compatibility. `mcp serve` MUST remain hidden/unavailable until the server can initialize and serve its declared surface.

#### Scenario: Human log on protocol stdout
- **WHEN** any startup or tool path writes non-protocol bytes to stdout
- **THEN** the real-process framing test and release gate fail
