## Purpose

Defines the exact Forecast Ledger v1.1.0 contract accepted, embedded, reported, and packaged by the application after an intentionally breaking cutover.

## ADDED Requirements

### Requirement: Exact immutable v1.1.0 contract
The application SHALL embed Forecast Ledger schema version `1.1.0` from upstream commit `c04c72a178c15cd6cbbdd2e8a7b743d58872a94a`, whose schema SHA-256 is `c478f0f568c0c746c343a308d0fcb53815f4c8b91b4666f8f784913ad9132d15`. Validation SHALL use only the embedded bytes and SHALL NOT fetch a schema at build-command runtime or ledger-operation runtime.

#### Scenario: Binary validates without schema network access
- **WHEN** a supported ledger is validated while network access is unavailable
- **THEN** validation uses the embedded v1.1.0 schema and completes without a schema request

#### Scenario: Vendored source is reproducible
- **WHEN** the vendored release archive and schema are checked during conformance testing
- **THEN** the archive SHA-256 is `edb2e307a7ce55984d17306556f0538f49a3a2a9fa66c9bfec973c90f0cb88dd` and the commit and schema digests match the declared pin

### Requirement: Exclusive schema-version acceptance
The application SHALL accept only a root `schema_version` of `1.1.0`. It SHALL reject `1.0.0`, missing versions, and all other values with stable code `unsupported_schema_version` before mutation, artifact creation, secret creation, or network access. Human and plain CLI output SHALL label this diagnostic as a warning and then stop with exit `3`; JSON SHALL retain its stable error envelope. The application SHALL NOT migrate, rewrite, or dual-read an unsupported ledger.

#### Scenario: Version 1.1.0 is accepted
- **WHEN** an otherwise valid ledger declares `schema_version: 1.1.0`
- **THEN** schema-version admission succeeds and normal validation continues

#### Scenario: Version 1.0.0 is rejected without side effects
- **WHEN** any read or mutating operation receives an otherwise valid v1.0.0 ledger
- **THEN** it warns and fails with `unsupported_schema_version`, identifies `1.1.0` as the supported version, and performs no write or network request

#### Scenario: No migration interface is exposed
- **WHEN** a user inspects CLI help, MCP tools, and generated operation schemas
- **THEN** no migration, conversion, compatibility, or schema-upgrade operation is present

### Requirement: Published v1.1.0 conformance corpus
The application SHALL validate all four published v1.1.0 valid fixtures, including the empty-ledger and question-without-forecasts fixtures, SHALL continue to reject every published invalid case, and SHALL reproduce the published forecast-seal vector byte-for-byte.

#### Scenario: New valid empty fixtures pass
- **WHEN** conformance tests load `empty-ledger.json` and `question-without-forecasts.yaml` from the pinned upstream release
- **THEN** both pass local schema, format, and semantic validation without modification

#### Scenario: Existing valid fixtures pass under v1.1.0
- **WHEN** conformance tests load the pinned `individual-ledger.json` and `team-ledger.yaml`
- **THEN** both pass the same validation path used for user ledgers

#### Scenario: Negative and cryptographic fixtures remain enforced
- **WHEN** the pinned invalid-case corpus and forecast-seal vector are executed
- **THEN** all invalid cases are rejected and the cryptographic output matches the published vector exactly

### Requirement: One schema identity across public surfaces
Every application surface that reports or records the supported schema SHALL use the same embedded v1.1.0 version, commit, and digest. This includes CLI version output, MCP initialization metadata and resources, validation results, generated schemas and examples, and publication manifests.

#### Scenario: Metadata agrees across adapters
- **WHEN** a caller reads CLI version JSON and MCP initialization metadata from the same binary
- **THEN** both report schema version `1.1.0`, commit `c04c72a178c15cd6cbbdd2e8a7b743d58872a94a`, and schema SHA-256 `c478f0f568c0c746c343a308d0fcb53815f4c8b91b4666f8f784913ad9132d15`

#### Scenario: Publication records the exact contract
- **WHEN** a publication package is built from a v1.1.0 ledger
- **THEN** its manifest records the same version, commit, and schema digest as the running binary
