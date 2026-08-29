# Build and verify an evidence package

<!-- doc-metadata
coverage: v0.3.1
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

The output must be a new path. Build validates the complete source ledger,
rebuilds every referenced target, checks every retained OTS receipt binding and
revealed forecast authentication, and copies only this allowlist:

- the exact complete ledger at `ledger/<original-name>`;
- referenced `proofs/targets/...` files; and
- referenced `proofs/receipts/...` OpenTimestamps files.

Keys, private input files, locks, journals, temporary files, neighboring files,
and recursively discovered directories are not copied. Key hints must use the
path-free `scheme:opaque` form. Repair an imported location-like hint with
`forecast-ledger forecast key-hint update` before building.

The ledger is the publication graph's source of truth. `target build` can create
a useful standalone target beside a ledger without changing ledger integrity;
that adjacent file is intentionally not discovered or copied unless a retained
integrity record references it. This keeps package content explicit and
reviewable without requiring Git-managed storage.

The canonical `manifest.json` is written last. Its closed
`forecast-ledger-publication/v1` profile contains one exact ledger entry and
only `forecast_target` or `opentimestamps_receipt` evidence entries, sorted by
portable path with SHA-256 and size. `--dry-run` performs preflight without
creating the package.

An empty ledger produces a valid minimal package containing the copied ledger
and `manifest.json`, with no evidence entries. Package verification reports the
passing manifest and file checks, but its evidence aggregate is `no_evidence`
with `incomplete`/exit 9. Structural integrity is not evidence that a forecast,
target, timestamp, reveal, or outcome record exists.

Verify the copy independently; package verification is offline by default:

```sh
forecast-ledger publish verify \
  --file evidence-package/ledger/ledger.yaml \
  --manifest evidence-package/manifest.json
```

The verifier reads and validates the manifest first, confines every listed
path, rejects missing, changed, extra, linked, or unsupported-role entries, and
then runs document, content, reveal, outcome, and local timestamp checks.

Add `--online` only when fresh Bitcoin timing checks are wanted. The same
dual-source agreement and request budgets used by ordinary verification apply.
Network source unavailability makes timing incomplete; it does not change the
already established package byte-integrity verdict and is not reported as a
Bitcoin-proof mismatch.

The copied directory remains valid after the source ledger and source proof
files are removed. It may be shared by any file transport.

[Verify ledger evidence](verify-evidence.md) · [Publication manifest reference](../reference/generated/index.md)
