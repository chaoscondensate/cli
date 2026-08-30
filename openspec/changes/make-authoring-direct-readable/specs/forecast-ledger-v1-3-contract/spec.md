## MODIFIED Requirements

### Requirement: Exact immutable v1.3.0 contract
The application SHALL embed Forecast Ledger schema version `1.3.0` from one exact upstream commit and published release asset. The checked-in source record, build metadata, contributor instructions, and conformance tests SHALL declare the same full commit ID, release-archive SHA-256, and schema SHA-256. Validation SHALL use only the embedded bytes and SHALL NOT fetch a tag, branch, schema, or fixture at build-command runtime or ledger-operation runtime.

#### Scenario: Binary validates without schema network access
- **WHEN** a supported ledger is validated while network access is unavailable
- **THEN** validation uses the exact embedded v1.3.0 schema and completes without a schema request

#### Scenario: Vendored source is reproducible
- **WHEN** the vendored v1.3.0 release archive, schema, attribution, and fixtures are checked during conformance testing
- **THEN** every byte and digest matches the exact declared commit and release pins rather than a floating tag

### Requirement: Exclusive v1.3.0 admission without compatibility
The application SHALL accept only a root `schema_version` of `1.3.0`. It SHALL reject `1.2.0`, older versions, missing versions, and all other values with stable code `unsupported_schema_version` before mutation, artifact creation, secret creation, or network access. Human and plain CLI output SHALL label this diagnostic as a warning and then stop with exit `3`; JSON and MCP SHALL retain their stable error envelopes. The application SHALL NOT migrate, rewrite, dual-read, import, or convert an unsupported ledger or its timestamp artifacts without a separate accepted change.

#### Scenario: Version 1.3.0 is accepted
- **WHEN** an otherwise valid ledger declares `schema_version: 1.3.0`
- **THEN** schema-version admission succeeds and normal validation continues

#### Scenario: Version 1.2.0 is rejected without side effects
- **WHEN** any read or mutating operation receives an otherwise valid v1.2.0 ledger
- **THEN** it warns and fails with `unsupported_schema_version`, identifies `1.3.0` as the supported version, and performs no write or network request

#### Scenario: No migration interface is exposed
- **WHEN** a user inspects CLI help, MCP tools, and generated operation schemas
- **THEN** no migration, conversion, compatibility, or schema-upgrade operation is present

### Requirement: Published v1.3.0 conformance corpus
The application SHALL validate all published v1.3.0 valid fixtures, SHALL reject every published invalid case, and SHALL reproduce every v1.3.0 cryptographic target and seal vector byte-for-byte. Vendored contract and fixture bytes MUST remain unchanged from the exact pinned release; any CLI-specific regression fixture SHALL be clearly separate from the upstream conformance corpus.

#### Scenario: Published valid fixtures pass
- **WHEN** conformance tests load the complete pinned v1.3.0 valid corpus
- **THEN** every fixture passes the same local schema, format, and semantic validation path used for user ledgers

#### Scenario: Published invalid fixtures fail
- **WHEN** conformance tests run the complete pinned v1.3.0 invalid corpus
- **THEN** every invalid case is rejected without a compatibility parser or network request

#### Scenario: Cryptographic vectors remain exact
- **WHEN** the complete v1.3.0 target and seal vectors are executed
- **THEN** application output matches every published vector byte-for-byte

#### Scenario: Cryptographic vector remains enforced
- **WHEN** the pinned v1.3.0 forecast-seal vector is executed
- **THEN** the cryptographic output matches the published vector exactly

### Requirement: One v1.3.0 identity across public surfaces
Every application surface that reports or records the supported schema SHALL use the same embedded v1.3.0 version, exact commit, schema digest, and applicable release digest. This includes CLI version output, MCP initialization metadata and resources, validation results, generated schemas and examples, publication manifests, documentation, package metadata, and release checks.

#### Scenario: Metadata agrees across adapters
- **WHEN** a caller reads CLI version JSON and MCP initialization metadata from the same binary
- **THEN** both report schema version `1.3.0` and the exact commit and schema digest declared by the vendored source record

#### Scenario: Publication records the exact contract
- **WHEN** a publication package is built from a v1.3.0 ledger
- **THEN** its manifest records the same version, commit, and schema digest as the running binary
