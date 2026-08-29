# Build and verify an evidence package

<!-- doc-metadata
coverage: v0.4.0
reviewed: 2026-08-29
owner: security
generated: false
security-critical: true
prerequisites: verify-evidence.md
next: run-mcp.md
-->

Publication is local and transport-neutral. It does not require Git, a remote
repository, or a hosted publisher.

```sh
forecast-ledger publish build --file ledger.yaml --output evidence-package
```

The destination must not exist. Build validates the ledger, rebuilds referenced
targets, validates timestamp artifact structure, and copies only:

- the ledger at `ledger/<original-name>`;
- each referenced `forecast_target`;
- each referenced `timestamp_request` (`.tsq`);
- each referenced `timestamp_response` (`.tsr`); and
- each referenced `timestamp_ca_bundle` (PEM).

A shared CA path is copied once only when its bytes and role agree. Missing,
escaping, symlinked, conflicting, or tampered artifacts fail the build. Secret
keys, private input, locks, journals, temporary files, and unrelated neighbors
are excluded.

The canonical `forecast-ledger-publication/v2` manifest pins schema v1.2.0 and
records each allowlisted path, role, size, and SHA-256 digest.

Verify on another machine with no network option:

```sh
forecast-ledger publish verify \
  --file evidence-package/ledger/ledger.yaml \
  --manifest evidence-package/manifest.json
```

Verification first checks the manifest and every listed file, rejects extra
files, then runs the same target, RFC 3161, reveal, chronology, and outcome
metadata checks against packaged bytes. It does not contact a TSA, blockchain,
Git host, system trust store, or outcome URL. Manifest observations remain
available even when the evidence aggregate is pending, failed, or
`no_evidence`.

The package is portable evidence, not proof of authorship, completeness, truth,
TSA clock honesty, current revocation status, or long-term validation.

[Run the MCP server](run-mcp.md)
