# Run the MCP server

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: true
prerequisites: publish-evidence.md
next: ../reference/mcp.md
-->

Forecast Ledger provides a local stdio MCP server backed by the same Go
services as the CLI. Protocol stdout contains only MCP messages.

```sh
forecast-ledger mcp serve \
  --ledger-root main=/data/forecast-ledgers \
  --output-root packages=/data/forecast-packages \
  --secret-root keys=/data/forecast-secrets
```

Root flags are repeatable. Use a short lowercase name and an existing absolute
or relative directory. Tool file references then use
`root-name:relative/path`, for example `main:research/ledger.yaml`. The server
does not infer a default ledger, scan directories, expand shell syntax, or
accept a path outside its configured class. Configured roots may not overlap.

The default server is read-write and online within its roots. These optional
whole-server modes reduce the surface:

```sh
forecast-ledger mcp serve --ledger-root main=/data/ledgers --read-only
forecast-ledger mcp serve --ledger-root main=/data/ledgers --offline
```

Read-only rejects mutation before secret, lock, or write effects. Offline opens
no network socket; stamp and upgrade are disabled, while verification returns
explicit local-only layers.

The server accepts at most 16 concurrent tool calls and 8 MiB of decoded
arguments per call by default. Set smaller or larger bounded values at startup
when the client workload needs them:

```sh
forecast-ledger mcp serve \
  --ledger-root main=/data/ledgers \
  --max-concurrent 8 \
  --max-tool-bytes 1048576
```

`--max-concurrent` accepts 1 through 256. `--max-tool-bytes` accepts 1 KiB
through 64 MiB. Calls above either limit fail as recoverable tool errors.

Reveal is the only separate capability boundary. `forecast_reveal` is absent
from discovery unless all three conditions hold: a secret root exists, the
server is not read-only, and startup includes `--allow-reveal`. A reveal call
still needs a protected key reference and `confirm: true`.

MCP never accepts custom calendar, explorer, proxy, or Bitcoin endpoint URLs.
It uses only the embedded `opentimestamps-public-v1` profile or offline mode.
Private seal input and keys are protected-file references under a secret root;
they are never raw tool arguments or resource content.

Addressed redacted resources use the versioned form
`forecast-ledger://v1/<kind>/<root>/<path>`, with optional question and forecast
query fields. Resource reads are local and non-mutating. Supported kinds are
`ledger`, `question`, `forecast`, `target`, `receipt`, `report`, and `manifest`.

[MCP reference](../reference/mcp.md) · [Security](../security/index.md)
