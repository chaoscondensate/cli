## 1. Capture v0.6.1 Regressions

- [ ] 1.1 Add presentation tests that reproduce the embedded `TimestampVerifyResult` nesting, empty-but-omitted collections, `,string`, custom marshalers, large integers, ignored fields, nested secret keys, and raw-byte canaries.
- [ ] 1.2 Add deterministic CLI/MCP parity tests that reproduce verified timestamp code drift, publication dry-run claiming files were built, each supported unchanged mutation claiming an update, and MCP target-check failure dropping its safe report.
- [ ] 1.3 Add a registry audit that enumerates every public operation and the success, dry-run, unchanged, pending, and safe partial-failure states applicable to it.

## 2. Preserve JSON Shape and Centralize Outcomes

- [ ] 2.1 Replace reflection-based presentation serialization with a fallible marshal/decode-with-`UseNumber`/recursive-redact pipeline and propagate encoding failures through CLI and MCP envelopes.
- [ ] 2.2 Audit registered public result types so raw artifact bytes remain unexported or `json:"-"`, and add secret/byte canary coverage through successful results, safe partial results, application-error details, and MCP resources.
- [ ] 2.3 Add the closed service `Outcome` contract and require every registered operation to provide a classifier for its applicable result states.
- [ ] 2.4 Move success, planned, unchanged, pending, verification, and safe partial-failure codes/messages from CLI and MCP into the shared classifiers, retaining adapter-only human/plain formatting and approval behavior.
- [ ] 2.5 Make MCP preserve typed data and set `isError` for report-bearing non-zero outcomes; keep fatal pre-report failures on the ordinary application-error path and keep the session alive.
- [ ] 2.6 Correct the CLI timestamp-verification JSON fixture to the flat public shape and update both adapters to use `timestamp.verified`, `publication.build.planned`, stable `*.unchanged`, and `target.failed` consistently.
- [ ] 2.7 Regenerate result schemas, operation contracts, MCP tool references, and every affected CLI/MCP golden; make generation and parity tests fail on adapter-local outcome drift.

## 3. Integrate Ledger Journal Recovery

- [ ] 3.1 Refactor ledger recovery into one lock-assuming helper and make both `UpdateLedger` and the existing explicit test seam delegate to it.
- [ ] 3.2 Move unfinished-journal inspection under the writer lock and continue mutation under that same lock after valid pre-replace or post-replace recovery.
- [ ] 3.3 Validate journal structure, exact ledger and temporary sibling identities, digests, format, schema, semantics, and retained-artifact context before recovery; preserve every relevant file on ambiguous or invalid state.
- [ ] 3.4 Extend transaction fault tests across every journal/write/replace/sync/cleanup stage to prove automatic retry recovery, stale-journal cleanup, tampered-current conflict, missing or changed temp refusal, invalid prospective refusal, and cancellation behavior.
- [ ] 3.5 Add representative service and CLI/MCP retries for metadata, question/forecast, and artifact-referencing mutations, plus concurrent-writer tests proving only the lock holder can recover.

## 4. Bind Outcome-Source Connections to Public Addresses

- [ ] 4.1 Extract a narrow shared public-address policy with IPv4, IPv6, IPv4-mapped IPv6, CGNAT, benchmarking, documentation, multicast, unspecified, loopback, link-local, and private-range coverage, preserving every current RFC 3161 rejection.
- [ ] 4.2 Replace the default outcome HTTP client with a proxy-disabled constrained transport whose dialer resolves once per connection, rejects a mixed answer set, and dials only approved numeric addresses while retaining the original hostname for Host and TLS verification.
- [ ] 4.3 Reapply the closed HTTPS, credential, redirect-count, origin, and destination rules to every redirect and keep cancellation, deadlines, response bounds, and safe error text intact.
- [ ] 4.4 Add deterministic resolver/dialer tests for public success, public-then-private rebinding, mixed answers, CGNAT, redirected private destinations, TLS hostname identity, disabled environment proxies, cancellation, and size limits without public network access.
- [ ] 4.5 Run the complete RFC 3161 transport/provider suite after sharing address policy and prove request profiles, built-in provider behavior, custom HTTPS behavior, fixtures, and no-live-network normal tests remain unchanged.

## 5. Repair the Dependency Security Gate

- [ ] 5.1 Pin `golang.org/x/vuln/cmd/govulncheck` as a Go project tool at the reviewed compatible version, update module sums, and standardize the cross-platform command as `go tool govulncheck ./...`.
- [ ] 5.2 Add a dedicated CI vulnerability-analysis job after module verification and require it before snapshot/release jobs; fail for findings, package-loading failures, advisory-data failures, or incomplete execution.
- [ ] 5.3 Update `AGENTS.md`, `CONTRIBUTING.md`, dependency and release guidance, and documentation baseline to use the executable command and distinguish reachable findings, module-only advisories, failed scans, and independent audits.
- [ ] 5.4 Run the pinned scan with the selected Go 1.27 toolchain and record its date and exact bounded result only after it completes successfully.

## 6. Documentation and Compatibility Handoff

- [ ] 6.1 Update README stable-JSON/MCP guidance, generated references, changelog, and release notes with the corrected flat timestamp shape and exact old-to-new outcome-code migration notes.
- [ ] 6.2 Add or update a maintained recovery how-to with `doc-metadata`, navigation, automatic retry behavior, conflict cases, preservation rules, and a warning not to edit or delete journals manually.
- [ ] 6.3 Update verification and security documentation for connection-bound outcome-source checks, CGNAT/reserved-range rejection, redirect policy, proxy exclusion, optional-network limits, and unchanged evidence-claim boundaries.
- [ ] 6.4 Remove only redaction reflection code, adapter-local outcome literals, placeholder parameters, helpers, imports, or unreachable branches made unused by this implementation; record the remaining review cleanup and performance proposals as out of scope rather than changing them silently.

## 7. Verification and Release Gates

- [ ] 7.1 Run `gofmt -w cmd internal tools`, `go mod tidy`, `go mod verify`, `go test ./...`, `go vet ./...`, `go test ./internal/doccheck`, and `go tool govulncheck ./...`; resolve every product, documentation, module, scanner, or vulnerability regression.
- [ ] 7.2 Run focused race tests for storage, service, RFC 3161, CLI, and MCP plus deterministic fault, redaction-canary, result-parity, DNS-rebinding, offline, and generated-tree checks.
- [ ] 7.3 Validate this OpenSpec change strictly and confirm the macOS, Linux, and Windows native CI matrix covers lock/recovery, path/network policy decisions, stable JSON, MCP session survival, and the vulnerability gate before implementation is marked complete.
