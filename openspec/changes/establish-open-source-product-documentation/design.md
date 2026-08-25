## Context

See `proposal.md` for motivation and the two delta specs for observable requirements. The repository currently contains planning artifacts only. The related `build-forecast-ledger-cli-mcp` change already asks for a README, command reference, security notes, MCP guidance, schema-update instructions, and examples, but does not yet define their information architecture, quality bar, maintenance policy, or open-source community surface.

Public product text is English and uses plain language. The CLI is cross-platform, offline-first for local operations, explicit about ledger paths, and unusually sensitive to inaccurate claims because it combines encryption, timestamps, evidence packages, and verification reports. The pinned Forecast Ledger contract and released binary are authoritative technical sources. `https://chaoscondensate.com/` provides broader project context, not CLI semantics.

The design follows the user-need separation described by Diátaxis, GitHub's community-health conventions, OpenSSF Best Practices criteria, REUSE/SPDX machine-readable licensing practices, and a human-curated changelog. These are quality inputs rather than badges to pursue at the expense of actual usability.

## Goals / Non-Goals

**Goals:**

- Give a first-time reader a short path from project recognition to a safe successful command.
- Keep tutorials, operational recipes, exact reference, and conceptual explanation distinct but connected.
- Make command examples, generated reference, policy files, and critical links testable in CI.
- Make security, support, contribution, governance, licensing, and release expectations explicit before launch.
- Keep repository documentation readable without a hosted site and ready for a future site without restructuring the content model.
- Apply a coherent, accessible visual and editorial system without obscuring evidence or status.

**Non-Goals:**

- Selecting the legal license on behalf of the maintainers; implementation requires a recorded maintainer-approved OSI license.
- Building or deploying a documentation website, analytics, search service, forum, or translation platform.
- Replacing generated CLI help, MCP schemas, the Forecast Ledger schema, or OpenSpec with prose copies.
- Promising support response times, audit status, security guarantees, or product maturity that have not been achieved.
- Rebranding the broader Chaos Condensate website or making its content normative for the CLI.

## Decisions

### 1. Use README as a concise router with one complete first success

The README uses progressive disclosure in this order:

```text
name, one-sentence value, maturity/status
why it exists and who it is for
what it proves and does not prove
install and checksum verification
small public-forecast quick start
sealed, timestamp, verify, package, and MCP entry links
documentation map
security and local-data summary
Chaos Condensate project context
support, contributing, governance, license, citation
```

Only one public-forecast flow is fully expanded in the README. Sealed workflows need more safety context and link to a tutorial rather than being compressed into an unsafe snippet. Every essential status statement is text; badges are supplementary.

Alternative considered: put the complete manual in README. Rejected because a long mixed-purpose page makes onboarding slower, duplicates reference, and causes content drift.

### 2. Organize repository-first Markdown with Diátaxis

The initial information architecture is:

```text
README.md
docs/
├── index.md
├── getting-started/
│   ├── install.md
│   ├── quick-start.md
│   └── concepts.md
├── tutorials/
│   ├── public-forecast.md
│   ├── sealed-forecast.md
│   ├── timestamp-and-verify.md
│   └── mcp-client.md
├── how-to/
│   ├── manage-ledgers.md
│   ├── update-forecasts.md
│   ├── reveal.md
│   ├── verify-evidence.md
│   ├── build-and-verify-package.md
│   ├── recover.md
│   ├── upgrade.md
│   └── troubleshoot.md
├── reference/
│   ├── cli/
│   ├── mcp/
│   ├── exit-codes.md
│   ├── json-output.md
│   ├── files.md
│   ├── compatibility.md
│   └── glossary.md
├── explanation/
│   ├── architecture.md
│   ├── forecast-ledger.md
│   ├── sealing.md
│   ├── timestamps.md
│   ├── verification-claims.md
│   ├── evidence-packages.md
│   └── project-background.md
└── security/
    ├── threat-model.md
    ├── key-custody.md
    ├── trust-sources.md
    └── privacy.md
```

Directory indexes or the main docs index expose every maintained page. Relative links keep a source checkout and release archive usable offline. Front matter is optional until a site generator needs it; content roles and navigation do not depend on one.

Alternative considered: adopt a hosted documentation framework immediately. Rejected because the repository has no implemented product yet and hosting would add deployment, dependency, accessibility, and versioning decisions before the content contract is proven.

### 3. Generate exact reference and author judgment-heavy guidance

Command/flag help, exit-code tables, JSON schema fragments, and MCP tool/resource signatures originate from the compiled binary or the same declarations used to build it. Generation is deterministic and checked for a clean working tree. Handwritten introductions, examples, security explanations, tutorials, and troubleshooting surround generated blocks but do not restate every field.

Generated files include a visible source version and a “do not edit by hand” marker. CI regenerates them and fails on a diff. Documentation for an unreleased interface is labelled unreleased rather than silently appearing as stable.

Alternative considered: hand-maintain all reference pages for better prose. Rejected because exact interfaces would inevitably diverge; prose belongs around, not instead of, generated contracts.

### 4. Execute documentation examples through a scenario harness

Checked examples live with deterministic fixtures and invoke the built candidate binary in isolated temporary directories. Each scenario asserts exit status, normalized stdout/stderr, created-file set, validation result, and absence of known secret patterns. Platform-neutral workflows run everywhere; OS-specific install and filesystem behavior run on matching native CI jobs. Network examples use recorded or local test doubles in normal CI, with separate explicitly labelled integration checks where needed.

Output snapshots normalize only inherently variable values such as temporary paths, generated IDs, timestamps, and cryptographic randomness. The surrounding assertions still verify type, format, and safety properties. Terminal captures and diagrams are regenerated from these checked scenarios or carry a source file and reproduction command.

Alternative considered: copy-edit command transcripts manually. Rejected because plausible but stale transcripts create false confidence in a safety-sensitive CLI.

### 5. Keep product claims in a shared evidence vocabulary

A short terminology and claims guide defines the allowed meanings of “valid,” “sealed,” “timestamped,” “anchored,” “verified,” “revealed,” “published,” “evidence,” and “proof.” README, docs, help examples, release notes, and issue templates reuse that vocabulary. Security-critical pages link to the exact verification-layer explanation and pinned schema information.

The docs state current audit and experimental status from release metadata or a single maintained status source. They never infer trust from a badge, hosted URL, source-control timestamp, or association with the broader website.

Alternative considered: optimize top-level copy for stronger marketing claims and explain caveats later. Rejected because readers often stop at the README and overclaiming would undermine the product's central evidence model.

### 6. Use conventional root policies with one owner per concern

The root contains `LICENSE` or the license layout required by the selected compliance policy, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `SUPPORT.md`, `GOVERNANCE.md`, `CHANGELOG.md`, and `CITATION.cff`. `.github/` contains issue forms, configuration, a pull-request template, and ownership/review rules where appropriate.

Security and conduct reporting use distinct private contacts. The public support matrix routes product bugs, usage questions, schema issues, broader Chaos Condensate inquiries, security findings, and conduct incidents without conflating them. Placeholder detection blocks release because a formally complete policy with fake contacts is worse than an explicit pre-release status.

The licensing task starts with a maintainer decision record. After selection, SPDX identifiers and third-party notices are applied consistently and checked with a pinned REUSE/SPDX-compatible tool. No license is inferred from “open source” or copied from another Chaos Condensate repository without approval.

Alternative considered: rely on organization-wide defaults. Rejected because source archives and mirrors must remain self-contained and project-specific security or contribution rules may differ.

### 7. Treat CHANGELOG as the release communication source

`CHANGELOG.md` is curated for humans with an Unreleased section and explicit Added, Changed, Deprecated, Removed, Fixed, and Security entries. Release notes are derived from it and add install/checksum instructions, compatibility pins, known limitations, and contributor credit. Breaking changes include migration steps at the point they are listed.

The docs identify the latest stable behavior and retain a compact compatibility and migration history for supported releases. A future hosted docs system may publish versioned snapshots, but the release archive remains sufficient to read the matching documentation.

Alternative considered: use generated commit lists as release notes. Rejected because commit history is an implementation record, not a complete user-impact narrative.

### 8. Apply a minimal accessible documentation design system

The repository defines a small palette and usage rules for diagrams and generated terminal assets, heading and callout conventions, plain-English style, capitalization, terminology, alt-text guidance, and status labels such as Experimental, Preview, Stable, Deprecated, and Unsupported. Safety callouts use both words/icons and color, never color alone. Text diagrams or adjacent prose preserve meaning outside rich rendering.

The README badge budget is intentionally small: release, CI, license, and security/best-practice status only when each badge links to live evidence and reflects the current repository. Popularity, vanity, donation, and redundant technology badges are excluded from the top product surface.

Alternative considered: maximize visual branding and badge density. Rejected because it slows recognition, adds external tracking/failure points, and can visually overstate maturity.

### 9. Add layered documentation gates

Fast pull-request checks validate Markdown structure, local links, terminology, placeholders, generated-reference drift, and selected executable snippets. Broader CI checks external links with retry/allowlist policy, all checked scenarios, native platform documentation, licensing compliance, and community-health coverage. Release checks require updated changelog, compatibility/status metadata, install commands, checksums, and security limitations.

External-link failures caused by transient remote outages are reported separately from broken internal links and do not silently rewrite URLs. Links essential to installation or security are mirrored or accompanied by enough local information to avoid a single external dependency.

Alternative considered: a single strict external-link job on every commit. Rejected because third-party availability is noisy and would train maintainers to ignore or disable the entire documentation gate.

### 10. Make Chaos Condensate attribution useful and bounded

README contains a short “Project” or “About” paragraph linking to `https://chaoscondensate.com/`. `docs/explanation/project-background.md` may provide fuller history and motivation. Both distinguish the open-source CLI repository, the published Forecast Ledger schema, and the broader Chaos Condensate presence. Behavioral and security claims always resolve to repository sources; the website is not fetched by builds, tests, or runtime operations.

Alternative considered: use the website as the primary documentation root. Rejected because it weakens version pinning, offline access, reviewability, and the repository's authority over released behavior.

## Risks / Trade-offs

- **[Documentation scope delays the first release]** → Prioritize launch-critical README, quick starts, security, license, contribution, exact reference, and core workflows; track lower-priority explanations without weakening release gates for safety-critical content.
- **[Generated reference becomes unreadable]** → Keep generation limited to exact interface material and add authored navigation, examples, and cross-links around it.
- **[Executable examples become brittle]** → Normalize only declared variable fields, prefer semantic assertions over whole-screen snapshots, and keep fixtures small.
- **[External links produce noisy CI]** → Separate internal and external link policies, cache results, retry boundedly, and maintain an explicit exception list with owners and expiry.
- **[Community policies promise unavailable maintainer capacity]** → State channels and process honestly, avoid unapproved service levels, and review contacts before each release.
- **[License metadata is applied before a legal choice]** → Block licensing implementation until maintainers record an OSI-approved license and copyright policy; never infer it from another repository.
- **[Website language drifts from repository behavior]** → Keep the website explicitly non-normative, link back to versioned repository docs, and do not import website copy automatically.
- **[Visual polish overstates maturity]** → Require explicit maturity and audit labels in text, restrict badges to evidence-backed signals, and review screenshots when status changes.

## Migration Plan

1. Record the documentation baseline, maintainer-approved license/copyright decision, reporting contacts, governance owners, product maturity label, and canonical repository/support URLs.
2. Establish terminology, claims, style, information architecture, navigation, and documentation validation configuration.
3. Create the root README and community-health files with real contacts and links, then verify the hosting platform's community profile.
4. Build generated CLI/MCP reference and the checked-example harness against the implemented candidate binary.
5. Write launch-critical tutorials, how-to guides, explanations, security material, and platform instructions; generate accessible diagrams and terminal captures from checked sources.
6. Add pull-request, full CI, and release documentation gates; resolve all placeholder, link, licensing, example, and reference drift failures.
7. Publish documentation with the first release archive and derive release notes from the reviewed changelog.

Rollback removes a broken generated publication artifact or restores the previous known-good documentation snapshot, but never removes required license, security, conduct, or attribution history. Incorrect safety guidance receives a visible correction and changelog entry rather than a silent rewrite when released users could have relied on it.
