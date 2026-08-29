# MCP reference

<!-- doc-metadata
coverage: v0.4.0
reviewed: 2026-08-29
owner: interface
generated: false
security-critical: true
prerequisites: ../how-to/run-mcp.md
next: generated/index.md
-->

`forecast-ledger mcp serve` uses the official pinned Go SDK and negotiates the
protocol revisions supported by that SDK. Initialization identifies the binary,
source/schema pins, RFC 3161 with SHA-256, access and network modes, and the
timestamp support status.

The server exposes CLI-parity tools named `ledger_init`, `ledger_update`,
`ledger_validate`, `ledger_status`, every `platform_*`, `question_*`, and
`forecast_*` action, `target_build`, `target_check`, `timestamp_stamp`,
`timestamp_status`, `timestamp_verify`,
`verification_run`, `publication_build`, and `publication_verify` when their
required root class and startup capability are available.

Every schema is a closed JSON object. Unknown fields fail before application
work. Every ledger tool requires `file`; selectors match the CLI. Mutations use
`dry_run`, and actions that need approval use `confirm: true`. Expected domain
errors are successful MCP protocol responses with `isError: true` and a stable
application envelope, so one failed call does not terminate the session.
Timestamp acquisition outcomes retain their structured timing report and safe
authority issue inside that envelope. Authority unavailability is `not_checked`
and `network`, while malformed, untrusted, or mismatching retained evidence is
`fail` and `verification`. Status and verification use only retained target,
request, response, and CA-bundle bytes; they open no network connection.
Layered and package verification use overall `no_evidence` with `incomplete`
when no forecast-evidence layer applies.

`ledger_init` may omit both `input` and `input_file` to create an empty ledger.
Init input may contain a question without `initial_forecast`. `question_add`
still requires exactly one of inline `input` or protected `input_file`, but its
question may also omit `initial_forecast`. A sealed initial forecast remains
file-only and requires `key_file`; inline secret forecast material is rejected.

All startup roots use explicit `name=path` syntax. There is no inferred default
root. The default resource limits are 16 concurrent tool calls and 8 MiB of
decoded arguments per call; `--max-concurrent` and `--max-tool-bytes` change
them within the documented bounded ranges.

The generated [tool catalog](generated/mcp-tool-schemas.json),
[operation contracts](generated/operation-contracts.md), and
[input schemas](generated/input-schemas/index.md) are the machine-readable
reference. Startup mode can remove `forecast_reveal` or package tools from
discovery.

`timestamp_stamp` requires `tsa_url` and ledger-relative `ca_bundle`. It accepts
only a public HTTPS authority URL, creates an RFC 3161 SHA-256 request, and
retains all bytes needed for later local verification. Repeating the call with a
different authority appends independent evidence; it does not replace earlier
entries. MCP resources expose these artifacts with resource kind `timestamp`.

[Run the MCP server](../how-to/run-mcp.md) · [Reference index](index.md)
