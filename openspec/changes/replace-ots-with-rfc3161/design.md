## Context

See `proposal.md` for the motivation. The current binary embeds Forecast Ledger v1.1.0, models timestamps as OTS receipts, implements calendar and Bitcoin observation in `internal/timestamp/ots`, and exposes those assumptions through services, CLI, MCP, generated contracts, package manifests, build information, CI, and documentation. The released upstream v1.2.0 contract instead records RFC 3161 request, response, TSA, SHA-256, state, verified metadata, and CA-bundle paths; it deliberately rejects the old OTS object.

The CLI remains one Go binary on macOS, Linux, and Windows. Local validation cannot use the network, timestamp and secret bytes require bounded parsing and safe output, CLI and MCP must share service behavior, and artifact-plus-ledger mutations must retain the existing lock, revalidation, recoverable-write, and source-preserving guarantees. There are no deployed clients or ledgers requiring compatibility.

## Goals / Non-Goals

**Goals:**

- Make v1.2.0 the only accepted and reported contract and reproduce its exact upstream corpus.
- Replace the complete OTS/Bitcoin subsystem with a bounded pure-Go RFC 3161 client and verifier.
- Make normal timestamp verification local and portable from target, request, response, and retained trust material.
- Keep stamp, status, verify, layered verification, publication, CLI, and MCP on the same transport-neutral application services.
- Remove every live OTS/Bitcoin implementation, interface, option, generated field, test dependency, workflow, and current-documentation claim.

**Non-Goals:**

- Reading, converting, migrating, or repackaging v1.1.0 ledgers or `.ots` receipts.
- Rewriting archived/superseded OpenSpec records or immutable historical releases; current specs, active changes, code, tests, CI, and maintained documentation are in scope.
- Operating a TSA, selecting or endorsing a default TSA, supplying a system-wide trust store, or supporting authenticated/custom-header TSA services in this change.
- RFC 4998 evidence renewal, OCSP/CRL archival, qualified-TSA discovery, or a legal-validity claim. Those require a later contract that can name the additional retained evidence.
- Runtime use of OpenSSL. OpenSSL is an independent development-time fixture oracle only.

## Decisions

### 1. Cut over atomically to the exact released contract

Vendor the v1.2.0 release archive from annotated tag `v1.2.0`, commit `6c2fe3df99223945b8d1613a03f95796b3c7d1e2`, archive SHA-256 `5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e`, and schema SHA-256 `d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8`. Replace the vendored v1.1.0 directory rather than retaining two embedded schemas. Admission rejects every other root version before parsing into the typed model or performing effects.

Alternative considered: dual-read v1.1.0 and v1.2.0 or an OTS-to-RFC migration command. This would preserve code and trust behavior the user explicitly wants removed and cannot manufacture an RFC 3161 timestamp for an old forecast, so it is rejected.

### 2. Use a constrained pure-Go RFC 3161 dependency behind an internal boundary

Add `github.com/notaryproject/tspclient-go` at release `v1.0.0` (commit `543cd58803b6c7260bad9bc78973a47ab02f249a`, Apache-2.0) behind `internal/timestamp/rfc3161`. The internal package owns byte limits, supported algorithms, request construction, response parsing, CMS verification, X.509 trust policy, error normalization, and the HTTP client. Application services do not expose dependency types.

The module is narrowly scoped, pure Go, includes RFC/CMS negative conformance tests, and keeps the shipped binary independent of an OpenSSL installation. Before accepting it, implementation must verify its request, response, signed-token, certificate, and BER behavior against checked-in OpenSSL fixtures and add the license to third-party notices. Unsupported or insufficiently bounded library paths are wrapped or rejected at the internal boundary rather than exposed as a broader protocol surface.

Alternatives considered: shelling out to `openssl ts`, which breaks the single-binary and cross-platform contract; copying a private ASN.1/CMS implementation, which creates unnecessary cryptographic maintenance; or using the much larger Sigstore stack, which introduces unrelated transparency-log and signing dependencies.

### 3. Require an explicit TSA and retained CA bundle

`timestamp stamp` takes `--tsa-url` and `--ca-bundle`; MCP takes equivalent fields within configured roots. There is no built-in endpoint list and no environment or config discovery. The TSA URL must be public HTTPS without user-info or fragment, is resolved against private/link-local/reserved destinations, and may not redirect outside the validated origin. The CA bundle must already be a protected-by-path, readable PEM artifact inside the ledger root and is the only trust-anchor input for that timestamp.

The RFC request uses SHA-256, a CSPRNG nonce, and `certReq=true`. The HTTP client sends `application/timestamp-query`, accepts only the RFC timestamp response media type, applies the existing operation deadline plus explicit response/body limits, and never logs the body. Repeating stamp with another TSA creates a separate timestamp entry; the application does not claim two URLs are organizationally independent.

Alternative considered: system roots or an embedded default CA/TSA profile. Both weaken future reproducibility or silently centralize provider policy and are rejected.

### 4. Use a three-operation timestamp lifecycle

Keep `timestamp stamp`, `timestamp status`, and `timestamp verify`; delete `timestamp upgrade`. Stamp normally performs request creation, submission, local verification, evidence retention, and ledger update in one user action. Status is a read-only inventory and bounded parse of local evidence. Verify always re-runs local cryptographic and metadata checks and updates a pending record only after success.

Remove `--calendar`, `--calendar-min-success`, `--bitcoin-core`, `--bitcoin-auth-file`, timestamp `--verified-at`, publication `--online`/`--offline`, Bitcoin observer settings, OTS profile metadata, and all corresponding MCP fields. General ledger `verify --offline` remains because outcome-source checks are a separate optional network boundary; it no longer controls timestamp verification, which is local.

Alternative considered: expose request/export/import commands matching the manual OpenSSL workflow. That adds more states and artifact plumbing without a current user need; the retained `.tsq` and `.tsr` remain ordinary portable files, so a later explicit import change is possible.

### 5. Preserve atomicity across network-separated preflight and commit

Stamp uses three phases:

1. Read and validate the ledger, select the immutable forecast, reconstruct the target, validate the TSA and CA inputs, and derive non-colliding paths under `proofs/timestamps/<forecast-id>/` without effects.
2. Generate the request, contact the TSA, parse the bounded response, and perform local verification outside the ledger lock.
3. Acquire the ledger/resource locks, re-read and revalidate the document, prove the selected forecast and target digest are unchanged, confirm every destination is absent or byte-identical, then commit target, `.tsq`, `.tsr`, and source-preserving ledger patch through the existing recoverable resource transaction.

The TSA path component is a stable short digest of the normalized TSA URL, so it is safe and deterministic without leaking credentials or relying on hostname punctuation. A successfully retained but not fully verified response is recorded as pending with a structured recovery result. A verified response records parsed `gen_time`, policy OID, serial number, CA path, and `verified_at`; user-controlled `verified_at` is removed and test clocks are injected below adapters.

Alternative considered: hold the ledger lock during network I/O. That would make a slow TSA block unrelated local work and increase interruption risk, so the final commit instead uses revalidation as its optimistic concurrency guard.

### 6. Verify the complete local evidence chain, not stored labels

The RFC backend verifies the response status and token; request/response nonce and imprint agreement; target/request SHA-256 agreement; CMS content type, signed attributes, message digest, signature, and signing-certificate binding; timestamping EKU; supported hash and signature algorithms; certificate chain against the retained bundle at `gen_time`; and parsed metadata against ledger fields. It rejects trailing data, unsupported critical content, excessive nesting/counts/sizes, duplicate ambiguity, and weak algorithms outside the accepted profile.

Layered verification tries every retained TSA entry. One complete valid entry passes existence timing; all completely checked entries must fail before the layer fails; any remaining pending or uncheckable entry prevents an all-branches cryptographic-failure claim. Outcome chronology uses verified `gen_time`, not `verified_at` or filesystem time. No result claims authorship, TSA clock truth, outcome truth, completeness, or exact self-reported forecast time.

### 7. Make publication evidence protocol-neutral at the manifest boundary

Replace manifest role `opentimestamps_receipt` and Bitcoin observation fields with `timestamp_request`, `timestamp_response`, and `timestamp_ca_bundle`. Package build follows only ledger-recorded safe paths and includes every referenced item plus the exact target. Package verify checks manifest/file digests first, then invokes the same RFC service on packaged bytes with no network path. Shared CA bundles are deduplicated by package path and digest without collapsing distinct TSA entries.

Result schemas replace block height, anchored-before, observer mode, source counts, and OTS state with request/response paths, TSA identity, `gen_time`, policy OID, serial number, local check states, and bounded TSA request counts where stamping actually contacted a service.

### 8. Delete OTS as a subsystem, not as a compatibility wrapper

After RFC paths are wired, delete `internal/timestamp/ots`, OTS fixtures and differential tests, OTS liveness workflow, Bitcoin observer interfaces and implementations, calendar profiles, old manifest roles, and all adapter flags/fields. Move only genuinely protocol-neutral URL/path safety behavior into neutral packages and rewrite its names/tests so no OTS API remains.

Update the main OpenSpec capabilities and reconcile the active `complete-forecast-ledger-command-surface` and `establish-open-source-product-documentation` artifacts before implementation is considered complete. Remove the maintained OTS review page and every current README/docs/help/security/baseline claim; preserve only explicit historical statements in changelog and archived/superseded OpenSpec records. A scoped denylist scan excluding those history locations is a release gate.

## Risks / Trade-offs

- **[A supplied CA bundle is the user's trust decision]** → Require it explicitly, never fall back to system roots, report its path/digest in evidence results, and document that cryptographic validity does not certify TSA quality or clock honesty.
- **[RFC 3161/CMS parsing is security-sensitive]** → Keep a narrow profile and hard bounds, use a pinned maintained client library, add malformed/property/fuzz tests, and differentially verify checked-in fixtures with OpenSSL.
- **[Certificate revocation and multi-decade renewal are not represented by v1.2.0]** → State this limitation plainly; do not imply LTV, qualified status, or 20-year preservation beyond the retained CA-bundle verification the contract can express.
- **[Persisting a non-verified response creates recoverable pending state]** → Never mark timing verified, retain exact bytes and structured reasons, and allow local verify after trust material or implementation issues are corrected.
- **[Network occurs between preflight and commit]** → Re-read under lock and require the forecast/target and destinations to remain unchanged before any durable commit.
- **[A dependency could accept behavior broader than the product profile]** → Wrap all entry points, reject unsupported algorithms/structures explicitly, and do not re-export dependency types.
- **[Removing all compatibility prevents old artifacts from being inspected]** → This is intentional before adoption; immutable upstream tags and old binaries remain historical tools, while the new binary stays small and unambiguous.

## Migration Plan

1. Reconcile project instructions and active OpenSpec artifacts with this accepted cutover so no concurrent work reintroduces OTS.
2. Replace the vendored contract, pins, fixtures, typed model, semantic validation, generated examples, and public schema identity with v1.2.0; make v1.1.0 fail before effects.
3. Introduce the bounded RFC backend and independent fixtures, then switch timestamp services, CLI/MCP adapters, layered verification, publication, and generated contracts.
4. Delete the OTS/Bitcoin packages, options, workflows, fixtures, dependencies, documentation, and current-spec remnants; run the scoped denylist scan.
5. Run formatting, module verification, full tests, race tests for affected stateful packages, vet, vulnerability scanning, documentation checks, cross-builds, and native filesystem/network-path tests in proportion to risk.
6. Release the cutover as one coherent pre-adoption version. Rollback is a source/release revert to the prior binary; there is no data rollback or dual-format bridge, and ledgers authored as v1.2.0 will remain unsupported by the old binary.
