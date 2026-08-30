## MODIFIED Requirements

### Requirement: Every non-secret authoring field has a CLI representation
Every `forecast-ledger` command that creates or changes ledger data SHALL provide leaf-local flags or a dedicated subcommand for every non-secret field accepted by that operation's typed application request. The CLI SHALL NOT expose `--input`, accept a public authoring document from a path or stdin, require shell-generated JSON/YAML, or require manual ledger editing. This requirement SHALL cover `init`, `ledger update`, `platform add|update`, `question add|update|resolve|annul|dispute`, `forecast add`, non-secret metadata for `forecast seal|reveal`, and `forecast key-hint update`, plus every future authoring command.

#### Scenario: Add a minimal platform from flags
- **WHEN** a user runs `forecast-ledger platform add --file l.yaml --platform metaculus --name Metaculus --kind scoring_platform` against a valid ledger
- **THEN** the command appends a valid platform with ID `metaculus` and exposes no `--input` flag or document-source error

#### Scenario: Complete a rich mutation from flags
- **WHEN** a user supplies every applicable non-secret value for a mutation through its documented flags
- **THEN** the command constructs one typed application request and performs the normal validation and source-preserving mutation

#### Scenario: Audit catches incomplete flag coverage
- **WHEN** a command request gains a non-secret authorable field without a corresponding flag or dedicated subcommand representation
- **THEN** the maintained command-surface test fails before release

### Requirement: Contributor policy prevents input-file-only regressions
Maintained contributor guidance SHALL state that new or changed CLI authoring commands MUST provide complete direct flags or dedicated subcommands for every non-secret field, MUST NOT add a generic authoring-document flag or stdin document mode, and MUST include flag-only acceptance tests and documentation. Review and release checks SHALL inventory every authoring leaf command and fail when ordinary use requires JSON/YAML preparation or when active code, generated contracts, help, scripts, tests, or current documentation expose `--input`.

#### Scenario: A future authoring command is reviewed
- **WHEN** a contributor adds or changes an authoring leaf command
- **THEN** the change includes direct-flag coverage, help, and a copyable flag-only test or is rejected by the command-surface audit

#### Scenario: Generic input mode is reintroduced
- **WHEN** an active runtime or maintained current artifact adds `--input` or routes public authoring through a generic document source
- **THEN** the release audit fails and identifies the active occurrence

## ADDED Requirements

### Requirement: CLI authoring has one public input route
For each operation, selectors, leaf-local public authoring flags, output controls, approval controls, and purpose-named protected secret references SHALL be the complete CLI request surface. All public values SHALL converge on the existing transport-neutral service request before domain validation, locking, cryptography, or mutation. CLI parsing MUST NOT implement a second domain mutation path.

#### Scenario: Invalid direct input has no side effect
- **WHEN** direct flags produce an invalid typed request
- **THEN** the command fails through the normal validation path before any ledger, key, target, receipt, or journal effect

#### Scenario: Help exposes no compatibility document mode
- **WHEN** a user inspects authoring help or completion
- **THEN** every public authoring field is represented directly and neither `--input` nor a public input-document alternative is present

### Requirement: Purpose-named secret channels remain separate
Private forecast values, encryption keys, salts, credentials, and other secret material SHALL NOT be accepted through command arguments or environment variables. Commands handling sealed data SHALL accept all non-secret metadata through flags while reading private bundles only from an explicit `--secret-input` or `--initial-secret-input` protected file/stdin channel and writing keys only to an explicit protected key file. These channels MUST NOT be generalized into a public authoring document mode.

#### Scenario: Seal separates public flags from private material
- **WHEN** a user creates a sealed forecast
- **THEN** IDs and public metadata are accepted as flags, private material is read from the purpose-named protected channel, and no secret appears in argv, logs, diagnostics, JSON, or normal stdout

## REMOVED Requirements

### Requirement: Input documents are optional and exclusive
**Reason**: The optional compatibility path keeps a second public authoring interface alive, makes help and tests ambiguous, and contradicts the direct-only product contract.

**Migration**: Replace each JSON/YAML member with its documented leaf-local flag or dedicated subcommand. Use purpose-named protected secret channels only for private material that is forbidden in argv.

#### Scenario: Removed flag is rejected
- **WHEN** a caller supplies `--input` to any command
- **THEN** CLI parsing rejects it as an unknown flag before reading stdin, opening an input path, or mutating state

### Requirement: Direct and document modes share application behavior
**Reason**: Public document mode no longer exists, so parity is between direct CLI fields and direct MCP properties rather than between two CLI input modes.

**Migration**: Route the single CLI flag request and the corresponding flattened MCP request through the same typed application service.

#### Scenario: Only one CLI request path remains
- **WHEN** an authoring command is invoked
- **THEN** it constructs its request from direct flags and does not branch on a generic document-input mode
