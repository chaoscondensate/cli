## 1. Align Planning and Project Rules

- [ ] 1.1 Update `AGENTS.md` package boundaries, timestamp invariants, documentation duties, and conformance gates to name only Forecast Ledger v1.2.0 and RFC 3161.
- [ ] 1.2 Reconcile the active `complete-forecast-ledger-command-surface` proposal, design, timestamp/verification/publication/CLI/MCP specs, and remaining tasks so none plans or requires OTS, calendars, Bitcoin observers, or removed flags.
- [ ] 1.3 Reconcile the active `establish-open-source-product-documentation` proposal, product-documentation spec, and remaining tasks so its required pages and examples cover RFC 3161 artifacts and no maintained OTS content.
- [ ] 1.4 Validate this change and both reconciled active changes with strict OpenSpec validation before editing implementation code.

## 2. Adopt the Exact v1.2.0 Contract

- [ ] 2.1 Fetch the immutable v1.2.0 release archive and verify tag `v1.2.0`, commit `6c2fe3df99223945b8d1613a03f95796b3c7d1e2`, archive SHA-256 `5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e`, and schema SHA-256 `d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8` before vendoring.
- [ ] 2.2 Replace the vendored v1.1.0 schema and conformance corpus with exact v1.2.0 schema, valid examples, invalid cases, seal vector, license, attribution, and reproducibility metadata; retain no embedded v1.1.0 fallback.
- [ ] 2.3 Update schema embedding, admission, format checking, build information, MCP schema resources, version output, generated examples, and publication schema pins to one v1.2.0 identity.
- [ ] 2.4 Replace OTS ledger model fields with the exact `rfc3161Timestamp` request, response, TSA URL, SHA-256, state, `gen_time`, policy OID, serial number, and CA-bundle fields required by v1.2.0.
- [ ] 2.5 Update semantic validation for RFC 3161 state completeness, multiple timestamps, verified-before-outcome chronology, safe relative artifacts, and the upstream negative `opentimestamps` case.
- [ ] 2.6 Add admission tests proving every v1.1.0 operation fails with `unsupported_schema_version` before writes, secrets, entropy, or network and that no migration or compatibility surface exists.
- [ ] 2.7 Run the complete pinned upstream v1.2.0 valid, invalid, target, and seal conformance corpus through the same application paths used for user ledgers.

## 3. Build the Bounded RFC 3161 Backend

- [ ] 3.1 Add `github.com/notaryproject/tspclient-go` v1.0.0, verify its module checksum and commit `543cd58803b6c7260bad9bc78973a47ab02f249a`, review its Apache-2.0 license, and record it in third-party notices.
- [ ] 3.2 Create `internal/timestamp/rfc3161` with application-owned request, response, verification, metadata, limit, and safe error types that expose no dependency-specific types.
- [ ] 3.3 Implement bounded SHA-256 request creation with CSPRNG nonce and certificate inclusion requested, plus deterministic request parsing and target-imprint checks.
- [ ] 3.4 Implement bounded response/status and CMS token parsing with explicit rejection of malformed, trailing, ambiguous, excessive, unsupported, or weak-algorithm input.
- [ ] 3.5 Implement nonce, request imprint, target digest, CMS signed-attribute, signature, signing-certificate, timestamping EKU, and metadata verification.
- [ ] 3.6 Implement PEM CA-bundle loading and certificate-chain verification at parsed `gen_time` without system-root fallback or network revocation lookup.
- [ ] 3.7 Implement the constrained TSA HTTP client with public-HTTPS destination validation, DNS/IP restrictions, same-origin redirect policy, RFC media types, operation deadline, response-size limit, cancellation, and body-safe errors.
- [ ] 3.8 Create checked-in OpenSSL-generated SHA-256 `.tsq`, `.tsr`, certificate, and target fixtures with reproducible generation notes and byte/semantic interoperability tests.
- [ ] 3.9 Add negative, property, and fuzz coverage for request/response ASN.1 and CMS, bounds, unsupported algorithms, invalid chains/EKU, metadata mismatch, nonce mismatch, target mismatch, and panic resistance.

## 4. Replace Timestamp Application Services

- [ ] 4.1 Replace OTS and Bitcoin observer service contracts, inputs, results, registry descriptions, and generated input schemas with transport-neutral RFC 3161 fields and stable reason codes.
- [ ] 4.2 Implement stamp preflight that validates the ledger, selector, target, TSA URL, retained CA bundle, dry-run/offline rules, and deterministic non-colliding target/request/response paths without effects.
- [ ] 4.3 Implement network request and local verification outside ledger locks, followed by locked re-read, forecast/target concurrency guard, identical-file checks, and recoverable atomic commit of target, `.tsq`, `.tsr`, and source-preserving ledger patch.
- [ ] 4.4 Implement honest stamp outcomes that record verified metadata only after complete checks and otherwise preserve recoverable pending evidence without claiming timing success.
- [ ] 4.5 Implement local read-only timestamp status over target, request, response, CA bundle, declared metadata, and safe artifact identities.
- [ ] 4.6 Implement local timestamp verify that rechecks exact bytes and metadata, promotes pending evidence only on success, and never contacts a TSA, blockchain, Git host, or system trust service.
- [ ] 4.7 Implement repeat stamp for multiple TSA URLs with stable URL-derived path components, idempotent retry behavior, independent timestamp entries, and preservation of earlier verified evidence.
- [ ] 4.8 Add service tests for dry-run entropy, offline no-socket behavior, TSA outage versus proof failure, response-retained pending recovery, interruption journals, retry, path collision, concurrent ledger change, rollback, and two-TSA precedence.

## 5. Cut Over CLI and MCP Adapters

- [ ] 5.1 Replace the CLI timestamp group with `stamp`, `status`, and `verify`; add only `--tsa-url`, `--ca-bundle`, and stamp `--offline` beyond the shared selectors, and remove upgrade, calendar, Bitcoin, and user-supplied verification-time controls.
- [ ] 5.2 Remove Bitcoin-specific flags and behavior from layered `verify` while retaining general offline/outcome-source behavior, and remove publication verify `--online`, timestamp-specific `--offline`, Bitcoin Core, and Bitcoin credential options.
- [ ] 5.3 Update human, plain, quiet, and stable JSON presentation for RFC request/response paths, TSA identity, state, `gen_time`, policy OID, serial number, local check matrix, and bounded stamp request summary.
- [ ] 5.4 Replace MCP timestamp tool schemas, registry entries, dispatch, server configuration, effects, and recoverable results with the same RFC service inputs and remove every calendar/profile/Bitcoin observer field.
- [ ] 5.5 Update MCP initialization metadata and resources to report the v1.2.0 schema and fixed RFC 3161/SHA-256 support without an endpoint profile or source count.
- [ ] 5.6 Add CLI/MCP parity tests for verified, pending, unavailable, malformed, binding-failure, trust-failure, metadata-mismatch, multi-TSA, dry-run, offline, root-confinement, and session-survival outcomes.
- [ ] 5.7 Regenerate and inspect all CLI/MCP input and result contracts, verifying that removed tools, flags, properties, OTS fields, Bitcoin fields, and credentials cannot be accepted or emitted.

## 6. Replace Layered and Publication Verification

- [ ] 6.1 Replace existence-timing verification with local RFC 3161 checks that pass on any independently verified TSA response, fail only after all applicable responses fail completely, and preserve pending/not-checked precedence.
- [ ] 6.2 Preserve the existing document, content-binding, reveal, outcome-source, empty-selection `no_evidence`, aggregate precedence, exit-code, and safe-partial-report behavior while replacing all OTS/Bitcoin reasons and evidence fields.
- [ ] 6.3 Replace publication manifest role `opentimestamps_receipt` with distinct target, timestamp-request, timestamp-response, and timestamp-CA-bundle roles and update path/digest validation.
- [ ] 6.4 Update publication build to include every referenced `.tsq`, `.tsr`, target, and CA bundle, safely deduplicate shared CA files, and reject missing, escaping, conflicting, or tampered artifacts.
- [ ] 6.5 Update publication verify to run complete RFC checks from packaged bytes without network options or fallback and preserve manifest/file observations separately from forecast evidence.
- [ ] 6.6 Add deterministic layered and package tests for complete offline verification, pending evidence, one-of-two TSA success, after-outcome timestamps, missing artifacts, manifest tampering, trust-bundle tampering, metadata mismatch, and empty evidence packages.

## 7. Remove the OTS and Bitcoin Subsystem

- [ ] 7.1 Delete `internal/timestamp/ots` and every OTS codec, calendar, profile, Bitcoin observer, Bitcoin Core, explorer, fixture, differential test, and liveness test implementation.
- [ ] 7.2 Delete `.github/workflows/ots-liveness.yml` and remove OTS package targets, liveness assumptions, and Bitcoin services from CI and release checks.
- [ ] 7.3 Remove every OTS/calendar/Bitcoin import, type, constant, error, result field, manifest role, input property, flag, help string, generated schema fragment, build-info field, and test helper from live code and tests.
- [ ] 7.4 Remove obsolete direct and indirect modules with `go mod tidy`, verify no removed network or Bitcoin dependency remains, and retain only the reviewed RFC 3161 dependency footprint.
- [ ] 7.5 Move only still-required protocol-neutral URL/path safety logic into neutrally named packages with neutral tests; prove no compatibility wrapper or OTS API remains.
- [ ] 7.6 Run a scoped case-insensitive denylist scan for `opentimestamps`, standalone `ots`, calendar endpoints, `mempool.space`, Blockstream, Bitcoin Core, Bitcoin observers, and removed flags across current code, tests, CI, generated artifacts, main specs, active changes, and maintained docs; allow only explicit history in changelog and archived/superseded OpenSpec records.

## 8. Update Maintained Documentation

- [ ] 8.1 Rewrite README installation, quick start, timestamp, verification, publication, status, limitations, and maturity sections for the real v1.2.0 RFC command surface.
- [ ] 8.2 Rewrite getting-started and how-to material for explicit TSA URL/CA bundle, stamp/status/verify, multiple TSAs, offline package verification, failure recovery, and exact retained `.tsq`/`.tsr`/PEM artifacts using copy-tested commands.
- [ ] 8.3 Rewrite CLI, MCP, generated result, file/artifact, error/exit, schema compatibility, and publication reference material and remove the maintained OTS review page and navigation links.
- [ ] 8.4 Rewrite verification-claim and security explanations to distinguish TSA trust, signed `gen_time`, content binding, verified-at time, chronology, authorship, outcome truth, system-root exclusion, and the absence of revocation/LTV/RFC 4998 support.
- [ ] 8.5 Update documentation baseline, platform/release support, documentation style examples, CHANGELOG, license/third-party notices, and all current product-status statements for the breaking pre-adoption cutover.
- [ ] 8.6 Run documentation metadata, navigation, link, generated-contract, and command-example checks, including `go test ./internal/doccheck`, and manually verify cross-platform syntax shown to users.

## 9. Verify and Prepare the Cutover Release

- [ ] 9.1 Run `gofmt -w cmd internal`, `go mod verify`, and the focused schema, RFC 3161, service, storage, CLI, MCP, publication, validation, build-info, and documentation tests.
- [ ] 9.2 Run `go test ./...`, affected-package race tests, `go vet ./...`, and `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`; resolve every regression or vulnerability before completion.
- [ ] 9.3 Run bounded RFC parser fuzz targets for a documented interval and preserve any discovered regression corpus.
- [ ] 9.4 Cross-build the supported macOS, Linux, and Windows targets and run native filesystem, permission, safe-replacement, interrupt-recovery, and constrained-network tests on the required platforms; do not treat cross-build as native proof.
- [ ] 9.5 Re-run strict OpenSpec validation for this and reconciled active changes, the scoped OTS/Bitcoin denylist, schema/archive digest checks, generated-contract drift checks, and release artifact/checksum validation.
- [ ] 9.6 Update release notes and version metadata for one coherent breaking pre-adoption release and hand off a verified release-ready tree; publish only through the repository's explicit release workflow.
