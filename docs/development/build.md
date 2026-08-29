# Developer build

<!-- doc-metadata
coverage: development
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: false
prerequisites: ../../AGENTS.md
next: dependencies.md
-->

The project requires the reviewed Go 1.27.0 toolchain selected through
`go.mod`.

On macOS and Linux:

```sh
sh scripts/build.sh
```

On Windows PowerShell:

```powershell
./scripts/build.ps1
```

Both commands build `./cmd/forecast-ledger` with `-mod=readonly` and
`-trimpath`. Set `OUTPUT` to choose another output path. The default is
`dist/forecast-ledger` on POSIX and `dist/forecast-ledger.exe` on Windows.
Set `VERSION` and `SOURCE_REVISION` to inject deterministic release metadata;
both default to development-safe values rather than reading mutable repository
state during the build.

Run the package tests with:

```sh
go test ./...
```

To run the optional upstream Python fixture harness, install the pinned schema
repository's Python development requirements and set
`FORECAST_LEDGER_UPSTREAM_ROOT` to a checkout at commit
`c04c72a178c15cd6cbbdd2e8a7b743d58872a94a` before running
`go test ./internal/validation`. This checks the reference harness itself; it
does not replace the still-required case-by-case Go/Python parity CI gate.

[Development documentation](index.md) · [Documentation index](../index.md)
