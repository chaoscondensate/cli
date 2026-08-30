## Context

See `proposal.md` for motivation. Application services normalize patch values through JSON into `document.OrderedValue` so object field order survives later JSON or YAML rendering. That wrapper also contains normalized scalars, but `isScalarReplacement` recognizes only built-in scalar Go types. Consequently normalized status, title, date, and similar values are incorrectly routed through YAML structural rendering.

Structural rendering encodes a fresh `yaml.Node`, clears flow style on populated collections, and splices its bytes into the source. Fresh nodes have no source `Line` or `Column`; the current continuation indentation therefore falls back to column zero. Inline flow-to-block tests pass because a preceding `key:` supplies a special indentation branch, while block-to-block replacements nested inside mappings or sequence items can emit continuation lines at the root and fail the mandatory reparse. The existing transaction then correctly refuses the whole mutation.

The fix must preserve ordered fields, expanded block style, unrelated source bytes, JSON behavior, canonical cryptographic bytes, and the lock/validate/recoverable-write transaction. It must cover every shared service consumer rather than special-case the commands found during dogfooding.

## Goals / Non-Goals

**Goals:**

- Classify normalized patch values by their semantic JSON kind so scalar replacements retain the established scalar-splice path.
- Render mapping and sequence replacements from the existing YAML node's source context, including mapping values, sequence items, block collections, flow collections, LF, and CRLF.
- Prove document-level source preservation and end-to-end YAML/JSON semantic parity for every structural replacement class used by services.
- Keep prospective reparse, schema/semantic validation, locking, journaling, and safe replacement as mandatory commit gates.

**Non-Goals:**

- Whole-document YAML reformatting or canonical YAML serialization.
- Changes to the Forecast Ledger v1.3.0 model, accepted YAML feature set, CLI flags, MCP properties, error codes, cryptographic profiles, or network policy.
- Recovery or mutation of unsupported schema versions, malformed input, or invalid prospective domain states.
- Preserving comments or presentation bytes inside the subtree an operation explicitly replaces; preservation remains required outside that bounded subtree.

## Decisions

### 1. Classify the semantic patch value before choosing a renderer

Add one document-owned classification path that understands both ordinary Go values and `OrderedValue`'s retained `ValueKind`. A normalized null, boolean, integer, or string will use `ReplaceScalars`; arrays and objects will use structural rendering. Additions continue to use the collection renderer because their parent syntax must be created or expanded.

This keeps ordered object rendering without treating its wrapper type as proof that every value is structural. The alternative of stopping service-side normalization for scalars would duplicate representation decisions across builders and make future call sites easy to miss.

### 2. Derive replacement layout from the existing source node, never the encoded node

Capture a small replacement context before encoding: the exact splice range, existing node kind, source line and column, line prefix, parent role implied by the source syntax, newline convention, and continuation indentation. Render the new semantic value independently, then place its first and continuation lines using that context. The encoded node contributes value content and collection style only; its zero-valued source coordinates are never consulted.

The context must distinguish a value following `key:`, a block collection beginning on the next line, and a mapping or sequence used as a sequence item. Existing flow values may be locally expanded, but the edit remains bounded to the addressed node and minimum parent syntax.

The alternative of copying `Line` and `Column` onto only the new root node is rejected because descendant layout, sequence markers, and key-on-previous-line cases still require source syntax context. Whole-document decode/encode is rejected because it would erase comments and unrelated presentation.

### 3. Keep reparse and prospective validation as the renderer postcondition

`ApplyPatch` will continue applying one operation at a time and reparsing after each splice. Shared storage transactions will continue decoding, validating, patching a copy, validating again, writing a sibling temporary file, flushing, and safely replacing under the existing journal rules. No fallback serializer will commit a file merely because the local splice failed.

This retains the safe behavior observed in dogfooding: the regression blocks operations but does not corrupt the original ledger. Genuine unsupported or invalid patches keep their current stable application classification; valid YAML operations are prevented from reaching that path by correctness and regression coverage.

### 4. Use a layered replacement matrix instead of one reproducer

Document tests will cover normalized scalar wrappers plus mapping and sequence replacements under block mappings, block sequences, and flow parents. Fixtures will include LF/CRLF, comments, quoted scalars, empty collections, stable field order, and untouched sibling byte assertions. Negative and fuzz/property cases will require either a valid bounded result or a non-mutating error without panic.

Shared-service and adapter acceptance tests will compare semantic results for YAML and JSON across question field/status update, lifecycle replacement, platform replacement, reveal, and timestamp integrity replacement. Timestamp tests will use the retained deterministic RFC 3161 fixtures; reveal tests will use protected temporary files and redaction assertions. One CLI matrix will reproduce the user-visible 0.6.0 failure, while MCP parity is established through the same service path and representative in-process tools rather than duplicating mutation logic in adapters.

## Risks / Trade-offs

- **[A source-context classifier misses an uncommon valid YAML layout]** → Cover mapping values, sequence items, flow parents, comments, quoted scalars, LF/CRLF, and feed bounded generated layouts through property/fuzz tests.
- **[Local replacement consumes a sibling comment or delimiter]** → Assert exact untouched prefixes/suffixes and calculate splice ranges only from the parsed existing node plus bounded source scanning.
- **[Scalar fast-path changes quoting or YAML tag meaning]** → Reuse the existing safe scalar replacement and mandatory reparse, with tests for strings that resemble booleans, dates, null, or numbers.
- **[Formatting fixes alter evidence bytes]** → Re-run published seal vectors and deterministic target/timestamp fixtures; canonical targets are derived from typed ledger content, not YAML presentation.
- **[A command passes while another replacement shape remains broken]** → Maintain the operation-by-shape matrix and fail release coverage when a shared service introduces an unclassified replacement kind.

## Migration Plan

1. Land the document renderer and regression tests without changing public request or result contracts.
2. Run the deterministic service/CLI/MCP YAML and JSON matrix, document checks, cryptographic vectors, and the normal Go verification suite.
3. Add the fix to the next release notes and keep 0.6.0 marked unsuitable for release; existing YAML ledgers require no data migration because failed operations left them unchanged.
4. Roll back the code change normally if verification regresses. No ledger downgrade or repair step is needed because the on-disk schema and transaction format do not change.
