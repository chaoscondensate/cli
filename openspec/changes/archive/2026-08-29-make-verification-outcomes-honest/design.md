## Context

See `proposal.md` for motivation and `specs/verification-outcome-semantics/spec.md` for observable behavior.

The built-in Bitcoin observer currently returns one undifferentiated `error` for transport failure, an invalid source response, source disagreement, budget exhaustion, and local block-header rejection. `CommitTimestampVerify` reuses the same variable for that observation error and for the later OpenTimestamps-to-Bitcoin comparison, then maps either path to `CodeVerification`. Its partial `TimestampArtifactResult` is discarded by the CLI when the error is returned.

Layered ledger and package verification already represent source outages as `not_checked`, but aggregation treats a report with zero forecast rows or only `not_applicable` layers as `pass`. Package verification reuses that aggregator, so manifest and file integrity can be sound while the top-level evidence claim is still too strong.

The change crosses the OTS observer, shared services, result presentation, CLI/MCP adapters, generated contracts, tests, and security-sensitive documentation. It must preserve bounded network behavior, safe source labels, redaction, explicit files, local/offline operation, and the existing ledger/target/receipt byte contracts.

## Goals / Non-Goals

**Goals:**

- Make the acquisition-versus-comparison boundary explicit and machine-testable.
- Preserve useful, safe reports for expected non-zero verification outcomes.
- Give all verification aggregates one conservative applicability rule.
- Keep CLI and MCP classifications and public fields derived from shared service results.
- Cover the observed outage without relying on live public services.

**Non-Goals:**

- Relaxing the built-in requirement for two agreeing public Bitcoin sources.
- Changing OpenTimestamps parsing, target canonicalization, stored integrity fields, or proof bytes.
- Adding fallback explorers, retry policy, endpoint configuration, source-control evidence, or hosted verification.
- Implementing v1.0.0 migration, changing question forecast windows, redesigning schema-validation diagnostics, or addressing other independent v0.3.0 findings.
- Changing the global rule that ordinary preflight/usage/data errors use the application error channel.

## Decisions

### 1. Give observer failures a typed, sanitized classification

Add an error type at the OTS observation boundary with a closed kind and sorted stable source IDs. The initial kinds are:

- `source_unavailable`: a required source could not be contacted or returned an unavailable HTTP/RPC result;
- `observation_inconclusive`: sources responded but could not form one accepted observation because their data disagreed or failed structural/header checks; and
- `observation_budget_exhausted`: the bounded local request or height budget prevented completion.

The type wraps an internal cause for tests and logs that are already safely bounded, but its public projection contains only the closed kind and source IDs. Public projection never includes endpoints, response bodies, credential-bearing values, or raw error strings. The public observer will retain all per-source outcomes long enough to report every affected ID deterministically instead of returning whichever goroutine error happens to be read first. Bitcoin Core uses the same projection with source ID `bitcoin-core`.

`source_unavailable` maps to application code `network`; inconclusive and budget-limited observation map to `incomplete`. Unknown errors returned by an injected observer remain acquisition failures and map conservatively to `incomplete`, never to proof mismatch.

Alternative considered: classify failures by matching text from existing errors. Rejected because wording is not a stable contract, wrapped network errors vary by platform, and raw strings risk leaking endpoints or credentials.

### 2. Reduce all candidate attestations before choosing an outcome

Timestamp verification will record one of three internal results for each supported Bitcoin attestation: observation unavailable/inconclusive, complete observation plus mismatch, or verified. It will then reduce them in this order:

1. any verified candidate wins;
2. otherwise, any acquisition failure keeps the timing check `not_checked`, using `network` if any candidate was source-unavailable and `incomplete` for other acquisition limits; and
3. only a non-empty set made entirely of complete-observation mismatches becomes `fail`/`verification`.

This prevents an early failed branch from hiding a later valid branch and prevents a missing observation from becoming evidence against the receipt. The observer and comparison remain bounded by the existing profile; no retries or extra sources are introduced.

Alternative considered: stop on the first observer error and change only its application code. Rejected because receipt evaluation can yield multiple Bitcoin branches, so first-error behavior can still make a stronger claim than the complete candidate set supports.

### 3. Return a dedicated timestamp verification report after safe preflight

Introduce a transport-neutral timestamp verification report rather than overloading the artifact mutation result used by stamp, upgrade, and status. The report contains the artifact identity and local receipt state, an `existence_timing` layer, optional height, network profile, request summary, warnings, safe observation issue fields, side effects, and an internal non-serialized application code.

Errors before a trustworthy report exists—path confinement, ledger parsing, selection, target mismatch, missing receipt, malformed receipt, or binding failure—continue through the ordinary error contract. Once the receipt is safely parsed, bound, and evaluated, pending, offline, observation-incomplete, mismatch, late-but-valid, and verified outcomes use the report path. Ledger mutation occurs only for a verified candidate and retains the current atomic transaction.

CLI presentation follows the existing layered/package pattern: present the primary human/plain/JSON report, then return a `presentedApplicationError` for a non-zero application category. MCP dispatch returns the same public report with a recoverable tool error category. This keeps protocol stdout clean and prevents error adaptation from discarding the result.

Alternative considered: attach the result as generic `details` on an application error. Rejected because it would duplicate result schemas, keep JSON on the diagnostic channel, and diverge from the report-first behavior already used by layered and package verification.

### 4. Add `no_evidence` to the shared aggregate state machine

Extend `VerificationOverall` with `no_evidence`. The shared reducer counts only forecast-evidence layers. A layer is applicable when its state is anything other than `not_applicable`; document validation, manifest parsing, and package-file hashing remain visible but do not increment this count.

The reducer uses this order:

1. any `fail` -> `fail` / `verification`;
2. any `not_checked` -> `incomplete`, using `network` when its reason is source unavailability and `incomplete` otherwise;
3. any `pending` -> `pending` / `pending`;
4. zero applicable forecast-evidence layers -> `no_evidence` / `incomplete`; and
5. otherwise -> `pass` / success.

Both `VerifyLedgerEvidence` and `VerifyPublicationPackage` continue to call this reducer. Empty forecast selections and all-`not_applicable` rows therefore become `no_evidence`; one or more passing applicable layers can still produce `pass` when every other applicable layer passes.

Alternative considered: reuse `incomplete` as the displayed overall state. Rejected because an applicable check that could not complete and a valid object to which no evidence check applies are different facts. They can share exit 9 and application code `incomplete` without sharing the user-facing state.

Alternative considered: keep vacuous `pass` and add only a warning. Rejected because exit-0 automation commonly ignores warnings and would retain the misleading claim found by dogfooding.

### 5. Treat public contracts and documentation as one compatibility surface

Add `no_evidence` and the timestamp report to stable result DTOs and generated JSON/MCP schemas. Human and plain output put the aggregate or timing state first and include stable reason codes. JSON remains one primary value on stdout for report outcomes. CLI and MCP adapter tests will assert parity from the same deterministic fixtures.

Documentation updates must remove the current claim that an empty ledger is a complete pass, explain that structural/package integrity can pass while evidence is `no_evidence`, distinguish network observation failure from proof mismatch, and record the preview exit-code changes. Generated references are regenerated from the service registry rather than edited by hand.

Alternative considered: document the current discrepancy. Rejected because the commands would still make contradictory machine-readable claims.

## Risks / Trade-offs

- [Existing scripts treat empty verification as a successful readiness check] -> Mark the preview change explicitly, use the stable `no_evidence` value and exit 9, and update examples and release notes together.
- [Typed observer classification can drift between public HTTP and Bitcoin Core paths] -> Centralize the closed issue projection and run the same table-driven reducer tests for both observer modes.
- [Partial reports might expose network internals] -> Allowlist stable source IDs and numeric budgets/counts; test absence of endpoints, response bodies, auth data, and injected secret markers across CLI JSON, human/plain, MCP, and errors.
- [Multiple candidate branches make precedence subtle] -> Add permutation tests covering verified, mismatch, unavailable, inconclusive, and mixed branch sets so ordering cannot change the outcome.
- [The new change overlaps command capabilities in an active OpenSpec change] -> Keep this spec cross-cutting, reference the active command contracts, and update any conflicting acceptance text during implementation before either change is archived.
- [Changing aggregation affects both ordinary and package verification] -> Use one reducer with explicit applicability counts and golden tests for empty, all-not-applicable, mixed applicable, pending, unavailable, and failed matrices.

## Migration Plan

1. Add typed observer issues and deterministic local fixtures without changing public output.
2. Introduce the timestamp report and candidate reducer, then switch CLI and MCP adapters together.
3. Add `no_evidence` to the shared aggregate reducer and update layered/package output goldens and public result schemas.
4. Regenerate maintained contracts and update documentation, help, release compatibility notes, and dogfooding regressions in the same change.
5. Run focused OTS, service, adapter, MCP, presentation, doccheck, and generated-contract checks, followed by the repository's full Go verification suite.

No persisted-data migration is required. Rolling back restores the old output/exit behavior but does not require ledger, receipt, target, key, or package conversion because this change writes no new persistent format.
