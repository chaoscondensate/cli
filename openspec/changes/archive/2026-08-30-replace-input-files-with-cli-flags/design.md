## Context

See `proposal.md` for motivation. The CLI currently converts bounded JSON/YAML command documents into transport-neutral service requests. That preserves a clean adapter boundary but makes the document parser the only complete authoring surface. The new flag path must keep the same service, validation, locking, cryptography, source-preserving patch, and result-presentation behavior. It must also respect the project invariant that secrets never enter argv or environment variables.

The active command surface spans creation, patch, lifecycle, public forecast, sealed forecast, and evidence operations. Optional fields and collections require presence-aware parsing; ordinary Go zero values cannot distinguish “not supplied” from “set to empty/false/zero.” Concurrent command-surface work also means coverage must be derived from the actual registered leaf commands and typed request schemas rather than a hand-maintained shortlist alone.

## Goals / Non-Goals

**Goals:**

- Provide one complete direct flag path for every non-secret authoring operation.
- Preserve optional document input as a compatibility/batch mode without making it part of the happy path.
- Centralize flag-to-request assembly so leaf commands stay consistent and testable.
- Give patch flags precise set/clear/omit semantics and give collections a shell-friendly representation.
- Keep version presentation aligned with the existing global color and output-mode policy.
- Make missing flag coverage mechanically visible in tests and contributor guidance.

**Non-Goals:**

- Passing private forecast plaintext, raw keys, salts, credentials, or other secrets in argv.
- Changing the v1.2.0 ledger schema, stable IDs, lifecycle rules, application service contracts, or MCP transport model.
- Removing bounded JSON/YAML input support in this change.
- Adding interactive prompts, a TUI, a config-derived ledger path, or inferred defaults from the current directory.

## Decisions

### 1. Two exclusive input modes converge before the service boundary

Each authoring leaf selects either direct flag mode or optional document mode. Selectors (`--file`, record IDs), output controls, dry-run controls, and protected destination paths remain outside this choice. Any document-mapped flag combined with `--input` is a usage error. Both modes construct the same existing typed request and then call the same application service.

This is preferable to field-by-field merging because merging makes precedence hard to explain, makes explicit empty values ambiguous, and risks accidentally overriding data read from a protected input. Removing document mode immediately would create avoidable compatibility breakage.

### 2. Use presence-aware reusable flag binders

The CLI adapter will define small reusable binders for shared public structures: ledger metadata, platform data, question common fields and type variants, forecast common fields and type variants, outcome sources, lifecycle evidence, and key hints. Each binder records whether a value was set, validates CLI-only shape, and builds the existing application input type. Domain validation stays below the adapter.

Leaf-local urfave flags remain the public interface. Reuse occurs in flag definitions and assembly helpers, not by moving flags to command groups or by invoking another adapter.

### 3. Collections use repeatable atomic flags; related nested records use a dedicated grammar or child operation

Simple arrays use repeated flags, such as repeated tags, options, or source values. A repeated value represents one atomic item and preserves command-line order. Public nested records with multiple coupled properties use either a small documented tuple grammar with individually escaped components or an existing dedicated child operation; no flag accepts embedded JSON/YAML. The implementation audit chooses the least ambiguous representation per existing typed field and documents it in leaf help.

Parallel arrays are rejected because mismatched counts create positional coupling. Generic `key=value` maps are used only for fields whose contract is genuinely a string map; they are not a substitute for typed records.

### 4. Patch commands expose explicit clear operations

For updates, omitted flags mean “leave unchanged.” A setter means “replace with this value,” including false or zero where valid. `--clear-<field>` means remove an optional value or reset a clearable collection. Setter and clearer together are rejected before reading the ledger. This preserves source-patching intent without magic sentinel strings.

### 5. Minimal creation uses contract-safe defaults only

Creation commands require flags for every schema-required value that cannot be derived safely. Existing documented operation-clock defaults remain available for write-time timestamps. Optional metadata is omitted. The embedded contract requires a platform name and kind in addition to its selector ID, so direct platform creation requires `--name` and `--kind`; an incomplete `platform add --file l.yaml --platform metaculus` must name those missing flags rather than demand `--input` or invent semantic defaults.

No value is inferred from cwd, environment, prior calls, array positions, remote services, or an uncontrolled platform catalog.

### 6. Secret operations retain protected channels

`forecast seal` and reveal-related operations get flags for public selectors and metadata only. Private bundles continue through stdin or protected input files, and generated/disclosed key material continues through protected files. Shared binders must classify fields explicitly as public or secret; the command-surface audit fails if a secret-classified field appears as an argv flag.

### 7. Coverage is schema- and command-driven

A maintained inventory test enumerates every registered authoring leaf and its typed request fields. Each non-secret authorable field must declare a flag/group/child-command mapping; secret fields must declare their protected channel; service-only/computed fields must declare why the user does not author them. The test also invokes each authoring leaf in flag mode far enough to prove urfave does not require `input`.

Focused end-to-end fixtures then cover every question/forecast type, create versus patch semantics, repeatable collections, set/clear conflicts, document compatibility, dry-run, JSON output, and source-preserving JSON/YAML mutation.

### 8. Version styling uses the shared presentation policy

Human version output becomes a compact label/value block produced through the existing presentation/color policy. Only labels or small accents may be colored. JSON bypasses human rendering; plain and all non-color environments use identical text without ANSI. No new styling dependency is needed.

## Risks / Trade-offs

- [A large number of flags can make help noisy] → Keep flags leaf-local, group help by record section, show type-specific examples, and prefer dedicated child operations for complex repeatable records.
- [CLI flag names can drift from typed inputs] → Make the inventory test compare registered mappings to request/schema fields and fail closed for an unclassified addition.
- [Presence semantics can accidentally clear data] → Use explicit presence types and separate clear flags; test omitted, empty, false, zero, set, and clear cases.
- [Tuple syntax can become another serialization language] → Permit it only for small fixed public records, document escaping, and prefer a dedicated subcommand once the shape is not trivially atomic.
- [Retained input documents double the parser matrix] → Converge both modes on one request type and use parity tests; document mode remains optional and secondary.
- [Shell history exposes values] → Keep the existing strict secret classification; document that public ledger data is intentionally visible in argv and process listings.
- [Concurrent OpenSpec work can introduce commands after the first audit] → Derive inventory from the final registered command tree during implementation and rerun it after rebasing or integrating adjacent changes.

## Migration Plan

1. Inventory every current authoring leaf, typed request field, existing input schema, and secret classification; record the chosen flag representation.
2. Add presence-aware binders and direct mode to creation commands, then patch/lifecycle commands, preserving document mode in each step.
3. Add the version renderer through the shared presentation policy.
4. Add parity, negative, secret-boundary, help, completion, and end-to-end tests; regenerate maintained command schemas/fixtures if their contract includes CLI flags.
5. Update contributor guidance and all affected public documentation with flag-only examples and the explicit secret exception.
6. Release only after the inventory reports no uncovered non-secret authoring fields and the full conformance suite passes. Rollback can restore the previous binary because input-document compatibility is retained; ledgers written through flags use the unchanged v1.2.0 format.
