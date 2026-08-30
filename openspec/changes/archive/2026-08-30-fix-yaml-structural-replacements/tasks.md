## 1. Capture the Regression

- [x] 1.1 Add document tests that reproduce replacement of JSON-normalized ordered scalars inside block-style YAML mappings and sequence records, including status, timestamp-like, boolean-like, numeric-like, and null-like strings.
- [x] 1.2 Add document tests for populated mapping and sequence replacements beneath block mapping values, block sequence items, and existing flow collections, asserting parseable output and decoded value order.
- [x] 1.3 Add LF and CRLF fixtures with comments, quoting, blank lines, empty collections, and untouched siblings; assert exact preservation outside the addressed subtree and expanded block style inside it.
- [x] 1.4 Extend bounded negative and fuzz/property coverage so replacement either yields a reparsed document with preserved out-of-scope bytes or returns a non-mutating error without panic.

## 2. Repair YAML Replacement Rendering

- [x] 2.1 Add one document-owned semantic value classifier for built-in values and `OrderedValue`, and route normalized null, boolean, integer, and string replacements through the existing scalar-splice path.
- [x] 2.2 Derive a bounded replacement context from the existing YAML node and source bytes, including splice range, mapping-value or sequence-item role, first-line placement, continuation indentation, and newline convention.
- [x] 2.3 Render populated mapping and sequence replacements with the captured source context, stable field order, expanded block style, two-space nesting, and explicit `{}` or `[]` only for empty collections.
- [x] 2.4 Remove reliance on line or column metadata from freshly encoded YAML nodes and keep flow-to-block conversion bounded to the addressed subtree and minimum parent syntax.
- [x] 2.5 Keep per-patch reparse and prospective ledger validation mandatory; do not add a whole-document serialization fallback or change JSON patch rendering.

## 3. Restore Shared-Service and Adapter Parity

- [x] 3.1 Add YAML/JSON service parity tests for question title/date/status and collection updates plus resolve, annul, and dispute replacements, comparing changed pointers, lifecycle state, and decoded ledgers.
- [x] 3.2 Add YAML/JSON parity coverage for platform update and any root metadata replacement shapes that use normalized scalar or structured patch values.
- [x] 3.3 Add forecast reveal parity tests using protected temporary secret/key files, asserting equivalent revealed records, stable results, and no private value, key, salt, or absolute secret path in output or errors.
- [x] 3.4 Add deterministic RFC 3161 stamp and promotion parity tests for YAML/JSON using retained request, response, and trust fixtures; assert equivalent integrity metadata and unchanged canonical targets without live network access.
- [x] 3.5 Add a CLI regression matrix for the dogfooding operations and representative in-process MCP parity cases, proving valid YAML requests no longer return `internal` and both adapters still use the shared service behavior.

## 4. Preserve Transaction and Evidence Safety

- [x] 4.1 Add fault-injection coverage for prospective parse and post-mutation validation failures, asserting the original ledger is byte-identical, valid, and free of partial sibling files or unresolved recovery artifacts.
- [x] 4.2 Re-run source-preservation and populated-flow-style audits over every covered replacement output, including repeated patches that reparse between edits.
- [x] 4.3 Re-run the published v1.3.0 seal vector and deterministic target/timestamp fixtures byte-for-byte to prove YAML presentation changes do not affect canonical evidence.
- [x] 4.4 Exercise native replacement, lock, CRLF/LF, flush, and recovery behavior on macOS, Linux, and Windows through the existing platform test strategy; do not treat cross-builds as filesystem evidence.

## 5. Documentation and Release Readiness

- [x] 5.1 Review `README.md`, getting-started, lifecycle, platform, seal/reveal, timestamp, CLI/MCP reference, security, documentation baseline, and release guidance for impact; update affected text and record why unchanged pages need no edit.
- [x] 5.2 Add the YAML structural-replacement fix and 0.6.0 release-blocker context to the maintained changelog or next-release notes without claiming release readiness before the full matrix passes.
- [x] 5.3 Confirm command examples continue to use real `ledger.yaml` workflows and that no CLI flags, MCP properties, operation inventory, schema pins, compatibility claims, or security boundaries changed.
- [x] 5.4 Run `go test ./internal/doccheck` and proportional link/example checks after documentation updates.

## 6. Final Verification

- [x] 6.1 Run the dogfooding reproducer against a built candidate with deterministic timestamp configuration where required; confirm every valid YAML result matches the corresponding JSON result and every output ledger validates.
- [x] 6.2 Run `gofmt -w cmd internal`, `go mod verify`, and `go test ./...` with no unreviewed generated or vendored-byte changes.
- [x] 6.3 Run `go vet ./...` and `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`, recording any external tool or vulnerability-database limitation separately from product test results.
- [x] 6.4 Run `openspec validate fix-yaml-structural-replacements --strict` and perform a final documentation-impact and diff review before marking the change complete.
