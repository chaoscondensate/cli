## Purpose

Defines documentation as a reliable product interface that helps users adopt, operate, integrate, and evaluate the CLI safely while keeping claims and examples synchronized with released behavior.

## ADDED Requirements

### Requirement: README is the product landing page
The repository SHALL provide an English `README.md` that identifies the product and intended users, states its maturity, explains the problem it solves, names its principal trust boundaries, lists supported platforms, and links directly to installation, quick start, full documentation, security, support, contribution, changelog, and license information. The README SHALL prioritize the first successful workflow over exhaustive reference material and MUST NOT rely on badges or images to communicate essential status.

#### Scenario: First repository visit
- **WHEN** a reader opens the repository without prior product knowledge
- **THEN** the README lets them determine what the CLI does, whether it is ready for their use, how to install and try it, where their data stays, and where to learn about limitations without opening source files

### Requirement: Quick starts are complete and safe
The documentation SHALL provide copyable quick starts for a public forecast and a sealed forecast that use an explicit `--file`, avoid literal secrets in argv and environment variables, explain each created file, show representative output, and end with a validation or verification step. Platform-specific installation prerequisites SHALL be stated before the first command.

#### Scenario: New user follows the public forecast quick start
- **WHEN** a user follows the documented commands on a supported platform with a released binary
- **THEN** the commands complete in order, create only the documented files, and finish with the documented successful validation result

#### Scenario: New user follows the sealed forecast quick start
- **WHEN** a user follows the sealed workflow
- **THEN** secret input uses a protected channel, key custody and backup warnings appear before sealing, and the resulting ledger contains no private forecast plaintext

### Requirement: Documentation has a task-oriented information architecture
Maintained documentation SHALL have a navigable index and clearly separate tutorials, how-to guides, reference, and explanations. Every maintained page SHALL have one primary user need, a descriptive title, links to prerequisites and next steps where applicable, and a route back to the documentation index. No maintained page SHALL be orphaned from navigation.

#### Scenario: Reader needs to verify an existing package
- **WHEN** a reader searches the documentation index for package verification
- **THEN** they can reach a task-focused how-to without reading a tutorial, architecture explanation, or complete command reference first

### Requirement: Core user journeys are documented
The documentation SHALL cover installation, checksum verification, upgrade and uninstall on every supported operating system; ledger initialization and validation; platform and question management; public and append-only forecast updates; sealing and reveal; target and RFC 3161 request/response workflows with explicit TSA and retained CA-bundle inputs; layered local timestamp verification; portable evidence packages; MCP stdio setup and permission grants; troubleshooting; recovery; and schema compatibility updates.

#### Scenario: User selects a supported platform
- **WHEN** a macOS, Linux, or Windows user opens installation documentation
- **THEN** they receive platform-correct install, checksum verification, shell completion, upgrade, and uninstall instructions without having to translate commands from another operating system

### Requirement: Reference matches released interfaces
CLI command, flag, exit-code, JSON output, environment, file-format, and MCP tool/resource references SHALL be generated from or mechanically checked against the released binary, embedded schemas, and MCP declarations. Reference pages SHALL identify the product version and pinned Forecast Ledger schema version, commit, and digest to which they apply.

#### Scenario: Command surface changes
- **WHEN** a command, flag, exit code, or MCP schema changes without its reference being updated
- **THEN** documentation validation fails before release and identifies the stale reference

### Requirement: Security and evidence claims are precise
Documentation SHALL distinguish structural validity, content binding, existence timing, reveal validity, outcome evidence, authorship, completeness, and truth. It SHALL disclose experimental or unaudited components, key-loss consequences, timestamp and network trust sources, offline boundaries, local data behavior, and the absence of telemetry unless that behavior changes. It MUST NOT describe a timestamp, source-control record, hosted file, or passing validation as proof of a stronger claim.

#### Scenario: Reader reviews a successful verification example
- **WHEN** all automated technical layers pass in the example
- **THEN** the text names exactly what passed and repeats that the result alone does not prove authorship, ledger completeness, forecast truth, or outcome-source correctness

### Requirement: Project identity includes the broader context
The README and project-background explanation SHALL link to `https://chaoscondensate.com/` using the name “Chaos Condensate” and describe it as broader project context. They SHALL state that this repository's documentation and pinned Forecast Ledger contract are authoritative for CLI behavior and MUST NOT claim that the separate website or related practice is non-commercial.

#### Scenario: Website and repository differ
- **WHEN** descriptive website content conflicts with the released CLI documentation or pinned schema
- **THEN** the documentation directs users to follow the repository contract for CLI behavior and treats the website as non-normative context

### Requirement: Presentation is accessible and evidence-based
Documentation SHALL use plain international English, consistent terminology, descriptive link text, meaningful headings, syntax-labelled code blocks, alt text for informative images, text equivalents for diagrams and terminal captures, and sufficient contrast in maintained visual assets. Status badges SHALL link to the evidence they summarize, reflect current project state, and remain limited to signals that help users make a decision.

#### Scenario: Essential image is unavailable
- **WHEN** images do not load or a reader uses a text-only or assistive interface
- **THEN** all installation, operation, safety, status, and navigation information remains available in text

### Requirement: Documentation is continuously verified
CI SHALL validate internal links, approved external links, Markdown structure, terminology, generated-reference drift, safe executable examples, prohibited secret patterns, and required documentation coverage. A release SHALL fail when critical documentation checks fail or when user-visible behavior changes without the required documentation and changelog updates.

#### Scenario: Stale quick-start command
- **WHEN** a quick-start command no longer succeeds against the candidate release
- **THEN** CI fails with the page and command location before release artifacts are published

### Requirement: Documentation is versioned and maintained
Documentation SHALL declare which product versions it covers, retain upgrade and compatibility information for supported releases, identify page owners for security-critical topics, and define review triggers for schema, cryptography, timestamp, CLI, MCP, platform-support, and release changes. Deprecated or removed behavior SHALL include a migration path and the release in which it changed.

#### Scenario: New schema pin is adopted
- **WHEN** the supported Forecast Ledger schema commit or digest changes
- **THEN** compatibility, reference, examples, security notes, and release documentation are reviewed together before the new binary is released
