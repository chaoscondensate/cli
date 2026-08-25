# Forecast Ledger CLI documentation

<!-- doc-metadata
coverage: v0.1.0
reviewed: 2026-08-25
owner: documentation
generated: false
security-critical: false
prerequisites: none
next: getting-started/index.md
-->

This documentation covers the current Forecast Ledger CLI repository. Release
`v0.1.0` is Preview: local validation and status work, while authoring,
cryptography, OpenTimestamps, publication packages, and MCP operations remain
unavailable. See the [reviewed implementation baseline](development/documentation-baseline.md)
before relying on a workflow.

## Start here

- [Getting started](getting-started/index.md) — install the CLI and understand
  the currently supported first commands.
- [Tutorials](tutorials/index.md) — learn complete workflows after their
  implementation and executable checks are available.
- [How-to guides](how-to/index.md) — solve one operational task.
- [Reference](reference/index.md) — inspect exact interfaces, formats, versions,
  and compatibility.
- [Explanation](explanation/index.md) — understand the contract, architecture,
  evidence model, and project context.
- [Security](security/index.md) — review threats, key custody, trust sources,
  privacy, and reporting boundaries.
- [Development](development/index.md) — build, test, release, and maintain the
  project and its documentation.

Project help and reporting routes are listed in the root
[support guide](../SUPPORT.md).

## Current safe first workflow

After installing a released binary, validate an existing Forecast Ledger file
without a network request:

```sh
forecast-ledger validate --file ledger.yaml
```

Every ledger operation requires an explicit file. Do not use planned authoring,
sealing, timestamp, publication, or MCP help entries as evidence that those
operations are implemented.

[Repository README](../README.md) · [Project repository](https://github.com/chaoscondensate/cli)
