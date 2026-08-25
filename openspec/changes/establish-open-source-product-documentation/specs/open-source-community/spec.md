## Purpose

Defines the public policies, contribution pathways, governance signals, and security channels required for a trustworthy, welcoming, and maintainable open-source project.

## ADDED Requirements

### Requirement: Open-source rights are explicit
Before the first public release, the repository SHALL include the full text of a maintainer-approved OSI-approved software license, expose its SPDX identifier in the README and release metadata, identify copyright ownership, and document licensing for source, documentation, examples, generated assets, vendored material, and third-party dependencies. Machine-readable licensing metadata SHALL pass the project's selected compliance check.

#### Scenario: User evaluates reuse rights
- **WHEN** a user inspects the repository or a source release without network access
- **THEN** they can determine the applicable license, copyright holder, and third-party notices without relying on a hosting platform's license detection

### Requirement: Contribution path is reproducible
`CONTRIBUTING.md` SHALL explain prerequisites, repository setup, architecture and source-of-truth rules, how to find work, branch and pull-request expectations, tests and documentation checks, cryptographic and schema conformance gates, sign-off or contributor-license policy, review criteria, release boundaries, and where to ask contribution questions. Commands SHALL work on the supported contributor platforms or link to platform-specific alternatives.

#### Scenario: First external contribution
- **WHEN** a contributor follows the guide from a clean checkout
- **THEN** they can build, test, format, validate documentation, and submit a reviewable change without private maintainer knowledge

### Requirement: Community conduct is enforceable
The project SHALL adopt a recognized code of conduct, link it from README and contribution guidance, name the scope and enforcement responsibilities, and provide a private non-placeholder reporting route controlled by designated maintainers. Enforcement contact details SHALL be distinct from public issue reporting where confidentiality is required.

#### Scenario: Conduct incident report
- **WHEN** a community member needs to report a conduct incident privately
- **THEN** the code of conduct provides a working confidential route and explains what response process to expect

### Requirement: Security reporting is private and actionable
`SECURITY.md` SHALL identify supported versions, a private vulnerability-reporting channel, requested report contents, safe-harbor expectations, acknowledgment and update targets, coordinated-disclosure behavior, and where security advisories will be published. Public issues SHALL be explicitly discouraged for suspected undisclosed vulnerabilities.

#### Scenario: Researcher finds a suspected key disclosure flaw
- **WHEN** a researcher opens the security policy
- **THEN** they can report the flaw privately with enough information for triage and know how acknowledgment and disclosure will proceed

### Requirement: Support boundaries are discoverable
`SUPPORT.md` SHALL distinguish usage questions, reproducible bugs, security reports, conduct reports, schema questions, and broader Chaos Condensate inquiries, routing each to the correct maintained channel. It SHALL state expected support scope without promising response times or service levels the maintainers have not committed to.

#### Scenario: User has a schema interoperability question
- **WHEN** the user consults support guidance
- **THEN** they are directed to the appropriate public channel or authoritative schema resource rather than a security or conduct reporting route

### Requirement: Governance and maintenance are transparent
The repository SHALL document the current governance model, maintainer responsibilities, decision and change process, release authority, access expectations, conflict handling, inactive-maintainer path, and how governance can evolve. Named roles and current contacts SHALL be verifiable and placeholders MUST NOT ship in a public release.

#### Scenario: Contributor proposes a protocol-affecting change
- **WHEN** a contribution would alter canonicalization, cryptography, timestamp evidence, or schema compatibility
- **THEN** governance documentation identifies the required design review, conformance evidence, approval authority, and public decision record

### Requirement: Issue and pull-request workflows collect useful evidence
The repository SHALL provide structured issue forms for reproducible bugs, documentation problems, and feature requests; a configuration directing security and conduct reports away from public issues; and a pull-request template covering scope, testing, compatibility, documentation, security, and release-note impact. Templates SHALL ask only for information relevant to triage and MUST warn against submitting secrets or private ledger data.

#### Scenario: User reports a validation bug
- **WHEN** the user opens the bug form
- **THEN** it requests version, platform, redacted reproduction, expected and actual behavior, and relevant diagnostics while warning the reporter not to attach keys or sensitive ledger contents

### Requirement: Changes and releases are communicated for humans
The project SHALL maintain `CHANGELOG.md` as a curated, reverse-chronological record with an Unreleased section, dated versions, security and breaking-change visibility, and migration links. Release notes SHALL be derived from the changelog, add installation and verification details, identify schema and compatibility pins, credit contributors, and disclose known limitations without using raw commit history as user documentation.

#### Scenario: Release contains a breaking flag change
- **WHEN** a release removes or changes a documented CLI flag
- **THEN** the changelog and release notes mark it as breaking, identify the first affected version, and give a concrete migration command

### Requirement: Project citation is stable
The repository SHALL provide machine-readable citation metadata with project title, description, canonical repository URL, maintainers or authors as approved by them, license, release version, and preferred citation. Releases SHALL keep version and publication date fields synchronized.

#### Scenario: Researcher cites a released version
- **WHEN** a researcher reads the citation metadata for a release
- **THEN** they can cite that exact version without inferring authorship or publication data from repository history

### Requirement: Community health is a release gate
The public repository SHALL expose discoverable README, license, contribution, code-of-conduct, security, and support files; valid issue and pull-request templates; non-placeholder reporting contacts; and links that resolve to the same policies referenced by the documentation. CI or a release checklist SHALL detect missing required files, placeholder text, and broken policy links.

#### Scenario: Security contact remains a placeholder
- **WHEN** a release candidate still contains a placeholder security address
- **THEN** the community-health release gate fails and the public release is blocked
