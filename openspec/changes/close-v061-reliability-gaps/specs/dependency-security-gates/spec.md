## Purpose

Keeps dependency vulnerability analysis executable with the selected Go toolchain and enforced as a real release gate.

## ADDED Requirements

### Requirement: Pin a project-toolchain-compatible vulnerability scanner
The repository SHALL pin the reviewed `govulncheck` module version as development tooling and SHALL provide one cross-platform command that builds and runs it with the Go toolchain selected by the project rather than a lower toolchain selected from the scanner module in isolation. The scanner MUST remain outside runtime dependencies and release binaries.

#### Scenario: Older bootstrap Go uses automatic toolchain selection
- **WHEN** a contributor's bootstrap `go` command is older than the Go version pinned by `go.mod` but automatic toolchain selection can obtain the project toolchain
- **THEN** the documented scan analyzes the project with the pinned project toolchain instead of failing because the scanner was built with an older Go release

#### Scenario: Scanner cannot analyze the project
- **WHEN** the scanner cannot load packages, parse the selected Go version, obtain required advisory data, or complete analysis
- **THEN** the command fails and MUST NOT be recorded as a clean vulnerability result

### Requirement: CI enforces vulnerability analysis
Pull-request and main-branch CI SHALL run the pinned scanner after module verification with the project toolchain. CI MUST fail for a reachable vulnerability, scanner execution error, package-loading error, or incomplete analysis; a skipped or missing scan MUST NOT satisfy the release gate.

#### Scenario: Toolchain compatibility regresses
- **WHEN** a Go or scanner update makes the pinned scan unable to analyze all repository packages
- **THEN** CI fails before snapshot or release jobs can succeed

### Requirement: Security documentation matches executable evidence
Contributor instructions, dependency review, documentation baseline, and release instructions SHALL use the same pinned command and SHALL date or qualify clean-scan claims. Documentation MUST distinguish no reachable vulnerability found from absence of all module advisories and from an independent security audit.

#### Scenario: Scan reports an unreachable module advisory
- **WHEN** `govulncheck` reports a vulnerable required module but no reachable symbol in the application
- **THEN** maintained evidence records that distinction and does not claim that every dependency is vulnerability-free
