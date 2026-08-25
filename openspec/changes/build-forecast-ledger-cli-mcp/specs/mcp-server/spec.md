## Purpose

Defines a local MCP interface that gives agents safe, structured access to the same Forecast Ledger operations as the CLI.

## ADDED Requirements

### Requirement: MCP stdio server
The binary SHALL provide an MCP stdio server whose stdout contains only protocol messages and whose diagnostics use stderr. The server SHALL advertise its implementation version, supported MCP protocol version, and capability set.

#### Scenario: Start from an MCP client
- **WHEN** a compatible client launches the server over stdin/stdout
- **THEN** initialization succeeds without any human-readable text corrupting protocol stdout

### Requirement: Explicit file in every ledger tool
Every MCP tool that reads or changes ledger state SHALL require a `file` property and stable record selectors where applicable. The server MUST NOT maintain or infer a default ledger between calls.

#### Scenario: Tool call without file
- **WHEN** a client invokes `ledger_validate` without `file`
- **THEN** the tool returns a structured recoverable input error and reads no ledger

### Requirement: Shared behavior with the CLI
MCP tools SHALL use the same validation, canonicalization, cryptography, timestamp, locking, error classification, and mutation services as the CLI. Tools SHALL expose typed input and output schemas with unknown input properties rejected.

#### Scenario: Same invalid ledger through two adapters
- **WHEN** the CLI and MCP validate the same invalid ledger
- **THEN** both report the same machine error codes, document locations, and rule failures

### Requirement: Safe default permissions
The MCP server SHALL be read-only by default. Write, network, and reveal capabilities SHALL require separate startup grants, and a tool SHALL fail before side effects when its required grant is absent.

#### Scenario: Seal on read-only server
- **WHEN** a client calls the seal tool on a server without write permission
- **THEN** the tool returns a permission error and creates no key, target, or ledger change

#### Scenario: Package build on a read-only server
- **WHEN** a client calls evidence-package build without the write grant
- **THEN** no package files are created and the result identifies the missing grant

### Requirement: Confined filesystem access
The server SHALL canonicalize all ledger, artifact, package-output, and secret-file paths; reject traversal, symlink escape, disallowed absolute paths, and platform-specific escape forms; and confine access to explicit startup roots. Secret roots SHALL be separate from ledger resource roots.

#### Scenario: Symlink escape
- **WHEN** a tool path resolves through a symlink outside its allowed root
- **THEN** the request fails before opening the escaped file

### Requirement: Secret-redacted tools
MCP arguments and results MUST NOT contain raw keys, salts, sealed plaintext, credentials, or secret-file contents. Seal and reveal SHALL use references to authorized protected files or a supported secret channel, and generated secrets SHALL never become MCP resources.

#### Scenario: Successful seal tool
- **WHEN** a permitted seal call succeeds
- **THEN** structured output includes public record IDs, artifact paths, and next actions but excludes the generated key, salt, and plaintext

### Requirement: CLI-parity MCP surface
The MCP server SHALL expose read tools/resources for validation, inspection, questions, forecasts, targets, timestamp state, verification, and evidence-package verification; write-gated tools for ledger initialization and record mutation, seal/reveal, target and timestamp changes, and evidence-package creation; and separately gated network tools only for timestamp operations.

#### Scenario: Tool discovery
- **WHEN** a client lists tools on a read-only server
- **THEN** every tool clearly describes side effects and permission needs, and unsupported write calls remain safely rejected

### Requirement: Recoverable tool errors
Invalid input, validation failures, conflicts, pending evidence, and expected network failures SHALL be returned as structured tool execution errors that an agent can correct. Protocol errors SHALL be reserved for malformed MCP messages or server failures.

#### Scenario: Probability sum error
- **WHEN** a forecast tool receives multiple-choice values that do not sum to 10,000 basis points
- **THEN** it returns a structured execution error with the validation code and field details rather than terminating the MCP session
