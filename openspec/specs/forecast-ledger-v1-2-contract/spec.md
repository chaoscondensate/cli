# Forecast Ledger v1.2.0 Contract Specification

## Purpose

Defines the exact Forecast Ledger v1.2.0 contract accepted, embedded, reported,
and packaged by the application after the breaking RFC 3161 cutover.

## Requirements

### Requirement: Exact immutable v1.2.0 contract
The application SHALL embed Forecast Ledger schema version `1.2.0` from
upstream commit `6c2fe3df99223945b8d1613a03f95796b3c7d1e2`, whose schema SHA-256 is
`d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8`.
Validation SHALL use only the embedded bytes and SHALL NOT fetch a schema at
build-command runtime or ledger-operation runtime.

#### Scenario: Binary validates without schema network access
- **WHEN** a supported ledger is validated while network access is unavailable
- **THEN** validation uses the embedded v1.2.0 schema and completes without a schema request

#### Scenario: Vendored source is reproducible
- **WHEN** the vendored release archive and schema are checked during conformance testing
- **THEN** the archive SHA-256 is `5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e` and the tag, commit, and schema digests match the declared pins

### Requirement: Exclusive v1.2.0 admission without compatibility
The application SHALL accept only a root `schema_version` of `1.2.0`. It SHALL
reject `1.1.0`, missing versions, and all other values with stable code
`unsupported_schema_version` before mutation, artifact creation, secret
creation, or network access. Human and plain CLI output SHALL label this
diagnostic as a warning and then stop with exit `3`; JSON SHALL retain its
stable error envelope. The application SHALL NOT migrate, rewrite, dual-read,
import, or convert an unsupported ledger or its timestamp artifacts.

#### Scenario: Version 1.2.0 is accepted
- **WHEN** an otherwise valid ledger declares `schema_version: 1.2.0`
- **THEN** schema-version admission succeeds and normal validation continues

#### Scenario: Version 1.1.0 is rejected without side effects
- **WHEN** any read or mutating operation receives an otherwise valid v1.1.0 ledger
- **THEN** it warns and fails with `unsupported_schema_version`, identifies `1.2.0` as the supported version, and performs no write or network request

#### Scenario: No migration interface is exposed
- **WHEN** a user inspects CLI help, MCP tools, and generated operation schemas
- **THEN** no migration, legacy timestamp import, conversion, compatibility, or schema-upgrade operation is present

### Requirement: Published v1.2.0 conformance corpus
The application SHALL validate all published v1.2.0 valid fixtures, SHALL
reject every published invalid case, and SHALL reproduce the published
forecast-seal vector byte-for-byte. Conformance SHALL include the upstream case
that rejects the superseded timestamp object and the chronology rule requiring
at least one verified RFC 3161 `gen_time` before a known outcome.

#### Scenario: Published valid fixtures pass
- **WHEN** conformance tests load the pinned v1.2.0 empty, backlog, individual, and team ledgers
- **THEN** every fixture passes the same local schema, format, and semantic validation path used for user ledgers

#### Scenario: Superseded timestamp fixture remains invalid
- **WHEN** conformance tests run the pinned superseded-protocol case
- **THEN** the ledger is rejected and no compatibility parser or network request is invoked

#### Scenario: Negative and cryptographic fixtures remain enforced
- **WHEN** the complete pinned invalid-case corpus and forecast-seal vector are executed
- **THEN** all invalid cases are rejected and the cryptographic output matches the published vector exactly

### Requirement: One v1.2.0 identity across public surfaces
Every application surface that reports or records the supported schema SHALL
use the same embedded v1.2.0 version, commit, and digest. This includes CLI
version output, MCP initialization metadata and resources, validation results,
generated schemas and examples, and publication manifests. No public surface
SHALL report a removed timestamp profile or blockchain source count.

#### Scenario: Metadata agrees across adapters
- **WHEN** a caller reads CLI version JSON and MCP initialization metadata from the same binary
- **THEN** both report schema version `1.2.0`, commit `6c2fe3df99223945b8d1613a03f95796b3c7d1e2`, and schema SHA-256 `d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8`

#### Scenario: Publication records the exact contract
- **WHEN** a publication package is built from a v1.2.0 ledger
- **THEN** its manifest records the same version, commit, and schema digest as the running binary and contains no removed timestamp-profile metadata
