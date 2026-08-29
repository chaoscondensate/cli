## 1. Classify Bitcoin observation outcomes

- [x] 1.1 Add closed typed observer-issue kinds and a sanitized public projection with deterministic stable source IDs and no endpoint, body, credential, or raw-error fields.
- [x] 1.2 Refactor the built-in dual-source observer to retain and classify all per-source outcomes, including unavailable HTTP responses, malformed observations, source disagreement, header rejection, and request/height budget exhaustion.
- [x] 1.3 Apply the same acquisition classification to Bitcoin Core and injected observers, preserving context interruption behavior and treating unknown acquisition errors conservatively rather than as proof mismatch.
- [x] 1.4 Add deterministic local observer and HTTP fixture tests for each issue kind, concurrent response order, source-ID ordering, budget bounds, caching, and secret/URL redaction.

## 2. Return honest timestamp verification reports

- [x] 2.1 Define the shared timestamp verification report DTO, public result schema, timing layer, safe observer issue fields, and internal application-category field without changing persisted ledger or receipt shapes.
- [x] 2.2 Implement the multi-attestation reducer so any verified branch wins, acquisition uncertainty prevents a proof-failure claim, and `timing.bitcoin_mismatch` occurs only when every candidate received a complete observation and mismatched.
- [x] 2.3 Refactor timestamp verification to use report outcomes after safe receipt evaluation, keep ordinary preflight errors on the error path, and mutate the ledger only after successful Bitcoin verification.
- [x] 2.4 Present pending, offline, source-unavailable, inconclusive, budget-limited, mismatch, late-valid, and verified reports consistently in CLI human, plain, and JSON modes with the specified stdout/stderr and exit behavior.
- [x] 2.5 Return equivalent recoverable `timestamp_verify` MCP outcomes and prove that expected domain failures keep the stdio session alive.
- [x] 2.6 Add permutation and parity tests for mixed Bitcoin branches, exact network/incomplete/verification codes, request summaries, safe source IDs, unchanged ledgers on non-success, and absence of the v0.3.0 false-failure message.

## 3. Reserve aggregate pass for applicable evidence

- [x] 3.1 Add `no_evidence` to the stable verification-overall contract and implement the shared fail, incomplete, pending, no-evidence, pass reducer with an explicit applicable-layer count.
- [x] 3.2 Update layered ledger verification tests for empty ledgers, question-only selections, all-`not_applicable` forecasts, mixed applicable/not-applicable layers, pending plus not-checked precedence, and zero network requests on empty selections.
- [x] 3.3 Update package verification tests to separate manifest/file integrity from the evidence aggregate and cover empty packages, all-`not_applicable` evidence, applicable passes, pending evidence, source outage, and tampering.
- [x] 3.4 Update CLI and MCP result codes, human/plain/JSON presentation, goldens, and adapter parity tests so `no_evidence` uses `incomplete`/exit 9 and never appears as `pass`/exit 0.

## 4. Align contracts and documentation

- [x] 4.1 Regenerate and verify result JSON Schemas, MCP tool schemas/descriptions, operation contracts, help fixtures, and any maintained examples affected by the timestamp report or `no_evidence` state; do not edit generated files by hand.
- [x] 4.2 Update the canonical timestamp, ledger-verification, package-verification, verification-claims, MCP, security/evidence-boundary, error/exit, and documentation-baseline pages to distinguish observation failure, proof mismatch, structural integrity, and applicable evidence.
- [x] 4.3 Review README, getting-started, command reference, release/maturity notes, platform guidance, and navigation for impact; update affected text and record a no-change conclusion for reviewed areas that do not need edits.
- [x] 4.4 Reconcile or explicitly supersede conflicting empty-pass and timestamp-failure language in active OpenSpec command artifacts before either overlapping change is archived.
- [x] 4.5 Run `go test ./internal/doccheck` plus link, generated-file, documented-command, and example checks required by the documentation baseline.

## 5. Verify the complete change

- [x] 5.1 Run focused OTS, timestamp service, aggregation, publication, presentation, CLI, MCP, redaction, and race tests using deterministic local fixtures and no live public-service dependency.
- [x] 5.2 Run `gofmt -w cmd internal`, `go mod verify`, `go test ./...`, and `go vet ./...` with the pinned toolchain.
- [x] 5.3 Run `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` and resolve or document any finding according to project policy.
- [x] 5.4 Run `openspec validate make-verification-outcomes-honest --strict`, confirm generated/public contracts are clean, and complete the documentation-impact handoff before marking the change done.
