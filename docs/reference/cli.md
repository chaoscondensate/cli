# CLI reference

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/index.md
next: generated/index.md
-->

Every ledger leaf defines its own required `--file/-f`; groups do not own that
flag. Read-only `validate`, `status`, `platform list|show`, `question list|show`,
and `forecast list|show` accept `--file -`. Artifact, timestamp, verification,
publication, mutation, and MCP operations require real paths. Stdin provides
only ledger bytes and cannot resolve sibling target or receipt paths.

## Command surface

| Command | Required selection or destination | Main effect |
| --- | --- | --- |
| `init` | new `--file`, identity flags, `--input`; conditional new `--key-file` | Create one valid ledger with its first question and forecast. |
| `ledger update` | `--file --input` | Patch allowed root/current-forecaster fields. |
| `validate`, `status` | `--file` | Validate or summarize. |
| `platform add|update` | `--file --platform --input` | Add or patch one platform. |
| `platform list|show` | `--file`; show also `--platform` | Read sorted/redacted platform data. |
| `platform remove` | `--file --platform --yes` | Remove only an unreferenced platform. |
| `question add` | `--file --question --type --input`; conditional new `--key-file` | Add one typed question and first forecast. |
| `question update` | `--file --question --input` | Patch allowed unfrozen fields. |
| `question list|show` | `--file`; show also `--question` | Read sorted/redacted question data. |
| `question resolve|annul|dispute` | `--file --question --input --yes` | Replace the v1 current resolution state while retaining forecasts. |
| `forecast add` | `--file --question --forecast --input` | Append a public forecast revision. |
| `forecast list|show` | `--file --question`; show also `--forecast` | Read append-only/redacted history. |
| `forecast seal` | `--file --question --forecast --input` and new `--key-file` | Append ciphertext after protected key creation. |
| `forecast reveal` | `--file --question --forecast --key-file --yes` | Authenticate and disclose a sealed forecast. |
| `forecast key-hint update` | `--file --question --forecast --key-hint` | Replace only the safe logical hint. |
| `target build|check` | `--file` plus `--all` or question+forecast | Create or compare canonical target bytes. |
| `timestamp stamp|upgrade|status|verify` | `--file --question --forecast` | Manage experimental OTS evidence. |
| `verify` | `--file`; optional question+forecast | Run layered evidence checks. |
| `publish build` | `--file --output` | Create a new standalone package. |
| `publish verify` | package ledger `--file` and `--manifest` | Verify offline by default; `--online` rechecks Bitcoin timing. |
| `mcp serve` | one or more `--ledger-root name=path` | Serve the shared operations over protocol-clean stdio. |
| `version` | none | Print binary, schema, MCP, and timestamp-profile pins. |

Mutation and resource-creation leaves provide `--dry-run`. It performs complete
preflight but does not persist files; timestamp dry-runs also skip entropy and
network. Read-only network verification has online/offline modes instead of
dry-run. Approval uses an interactive prompt or `--yes`; `--no-input` never
prompts.

Global result modes are normal human output, `--json`, `--plain`, or `--quiet`.
The last three are mutually exclusive. `--no-color` and `TERM=dumb` disable
decoration. `--timeout` bounds network/wait operations.

Question and forecast show output includes business fields and type-aware public
values in normal human and plain modes. Layered verification prints its complete
ordered evidence matrix without requiring `--verbose`. Quote RFC 3339 values in
maintained YAML input, such as `recorded_at: "2026-09-01T09:01:00Z"`.

`mcp serve` also accepts repeatable named `--output-root` and `--secret-root`,
whole-server `--read-only` and `--offline`, default-off `--allow-reveal`, and
bounded `--max-concurrent` and `--max-tool-bytes` limits. It has no general
write or network grant flags.
Read-only startup omits every mutating tool from discovery; calling an omitted
name returns the MCP unknown-tool protocol response.

## Stable exits

| Exit | Codes |
| --- | --- |
| 0 | complete success |
| 1 | `internal` |
| 2 | `usage` |
| 3 | `invalid_data`, `unsupported_schema_version` |
| 4 | `not_found` |
| 5 | `conflict` |
| 6 | `verification` |
| 7 | `io` |
| 8 | `network`, `network_disabled` |
| 9 | `pending`, `incomplete` |
| 10 | `unavailable` |
| 130 | `interrupted` |

Exact closed input schemas and operation policies are generated under
[generated interface reference](generated/index.md). Candidate-binary
`--help` is authoritative for flags in an installed release.

[Reference index](index.md) · [MCP reference](mcp.md)
