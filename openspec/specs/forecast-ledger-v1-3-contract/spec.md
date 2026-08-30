# Forecast Ledger v1.3.0 Contract Specification

## Purpose

Defines the exact Forecast Ledger v1.3.0 contract accepted, embedded, reported,
and packaged by the application.

## Requirements

### Requirement: Exact immutable v1.3.0 contract
The application SHALL embed Forecast Ledger schema version `1.3.0` from
upstream commit `32218f682b3a650f41153e98817473bf429973a7`, whose schema SHA-256 is
`f673e4f3fc867a83d8c42a6992c6020ea28359a293580c8c742fe9dcdcd8d2c1`.
The annotated tag object SHALL be
`d3d1f06a7f27501b1419eaf78fc4a48e51de9ee3`, the release archive SHA-256
SHALL be `3b6b9f274a67d2714edaa308f9aad51b218dbf24ed95de1a1340292ad1df1f2a`,
and the published `SHA256SUMS` asset SHA-256 SHALL be
`6042508976246ddc62974ad3054dca9885525024d4bb543572b75b23c60ac284`.
Validation SHALL use only embedded bytes and SHALL NOT fetch a schema at build,
command, or ledger-operation runtime.

#### Scenario: Binary validates without schema network access
- **WHEN** a supported ledger is validated while network access is unavailable
- **THEN** validation uses the embedded v1.3.0 schema and completes without a schema request

#### Scenario: Vendored source is reproducible
- **WHEN** the vendored release archive and schema are checked during conformance testing
- **THEN** the archive, tag, commit, schema, and checksum-asset digests match the declared v1.3.0 pins

### Requirement: Exclusive v1.3.0 admission without compatibility
The application SHALL accept only a root `schema_version` of `1.3.0`. It SHALL
reject `1.2.0`, missing versions, and all other values with stable code
`unsupported_schema_version` before mutation, artifact creation, secret
creation, or network access. Human and plain CLI output SHALL label this
diagnostic as a warning and stop with exit `3`; JSON SHALL retain its stable
error envelope. The application SHALL NOT migrate, rewrite, dual-read, import,
or convert an unsupported ledger or its timestamp artifacts.

#### Scenario: Version 1.3.0 is accepted
- **WHEN** an otherwise valid ledger declares `schema_version: 1.3.0`
- **THEN** schema-version admission succeeds and normal validation continues

#### Scenario: Version 1.2.0 is rejected without side effects
- **WHEN** any operation receives an otherwise valid v1.2.0 ledger
- **THEN** it warns and fails with `unsupported_schema_version`, identifies `1.3.0` as supported, and performs no write or network request

#### Scenario: No migration interface is exposed
- **WHEN** a user inspects CLI help, MCP tools, and generated operation schemas
- **THEN** no migration, conversion, compatibility, or schema-upgrade operation is present

### Requirement: Published v1.3.0 conformance corpus
The application SHALL validate every published v1.3.0 valid fixture, reject
every published invalid fixture, and reproduce the published forecast-seal
vector byte-for-byte. The v1.3.0 question contract SHALL treat
`forecast_window` as optional and, when present, permit only `opens_at`; the
removed `closes_at` field SHALL be rejected.

#### Scenario: Published valid fixtures pass
- **WHEN** conformance tests load the pinned v1.3.0 valid ledgers
- **THEN** every fixture passes the same local schema, format, and semantic validation path used for user ledgers

#### Scenario: Published invalid fixtures fail
- **WHEN** conformance tests run the complete pinned invalid corpus
- **THEN** every case, including an obsolete `closes_at`, is rejected without compatibility parsing or network access

#### Scenario: Cryptographic vector remains enforced
- **WHEN** the pinned v1.3.0 forecast-seal vector is executed
- **THEN** the cryptographic output matches the published vector exactly

### Requirement: One v1.3.0 identity across public surfaces
Every application surface that reports or records the supported schema SHALL
use the same embedded v1.3.0 version, commit, and digest. This includes CLI
version output, MCP initialization metadata and resources, validation results,
generated schemas and examples, and publication manifests.

#### Scenario: Metadata agrees across adapters
- **WHEN** a caller reads CLI version JSON and MCP initialization metadata from the same binary
- **THEN** both report schema version `1.3.0`, commit `32218f682b3a650f41153e98817473bf429973a7`, and schema SHA-256 `f673e4f3fc867a83d8c42a6992c6020ea28359a293580c8c742fe9dcdcd8d2c1`

#### Scenario: Publication records the exact contract
- **WHEN** a publication package is built from a v1.3.0 ledger
- **THEN** its manifest records the same version, commit, and schema digest as the running binary
