## Purpose

Defines deterministic, transport-neutral evidence-package creation and local verification so a ledger and its exact public evidence can be copied or published without source control, hosted services, secrets, or machine-specific paths.

## ADDED Requirements

### Requirement: Build only from an explicit complete ledger
`forecast-ledger publish build` SHALL require a real `--file` ledger path and a new explicit `--output` directory. It SHALL fully validate the ledger and enumerate every target and OpenTimestamps receipt referenced by the ledger. It MUST NOT inspect source-control metadata, infer an upload destination, contact a remote service, or silently omit a referenced artifact. The package SHALL contain the complete selected ledger rather than a rewritten subset.

#### Scenario: Standalone ledger outside source control
- **WHEN** a valid ledger and its referenced artifacts live in an ordinary directory
- **THEN** build creates the same complete local evidence package without invoking a source-control tool or requiring a hosted account

#### Scenario: Missing referenced receipt
- **WHEN** ledger metadata references a receipt that cannot be read safely
- **THEN** build fails before creating the output directory contents and identifies the missing safe relative path

### Requirement: Include a deterministic allowlisted file set
The package root SHALL contain `manifest.json`, one ledger at `ledger/<original-base-name>`, targets at their stable `proofs/targets/...` relative paths, and receipts at their stable `proofs/receipts/...` relative paths. Every included file other than the manifest SHALL be represented exactly once in the manifest. Paths SHALL use forward slashes, be relative, normalized, case-collision checked, and sorted by UTF-8 byte order. The closed v1 manifest role vocabulary SHALL be exactly:

| Role | Required path and meaning |
| --- | --- |
| `ledger` | exactly one `ledger/<original-base-name>` entry containing the byte-exact selected ledger |
| `forecast_target` | one entry for each referenced `proofs/targets/<forecast-id>.json` target |
| `opentimestamps_receipt` | one entry for each referenced `proofs/receipts/<forecast-id>.json.ots` detached receipt |

No other entry role is valid in v1. Supporting another public evidence file SHALL require a new reviewed manifest profile rather than accepting an unknown role.

The manifest SHALL use a versioned closed schema and contain at least: manifest profile, embedded ledger schema version/commit/digest, packaged ledger path, and sorted file entries with role, path, byte length, SHA-256 algorithm, and lowercase digest. It MUST NOT contain creation time, host/user name, absolute path, source-control data, random ID, or platform separator so identical evidence produces byte-identical canonical manifest bytes across platforms.

#### Scenario: Cross-platform repeat build
- **WHEN** identical evidence is packaged on supported macOS, Linux, and Windows systems
- **THEN** every copied content file, manifest byte, and manifest SHA-256 is identical

### Requirement: Exclude secrets and undisclosed material
Build MUST exclude key files, actual `--key-file` locations, secret roots and paths, credentials, lock/journal/temp files, raw private seal input, decrypted sealed plaintext, and any file not required by the public ledger evidence. A sealed forecast MAY contribute only public note, commitment, ciphertext, safe logical key hint, target, receipt, and other data already allowed by the ledger contract. A revealed forecast MAY contribute only disclosed material already represented in the validated ledger; the manifest and command output SHALL still redact raw key values and machine-local key locations.

Because the complete ledger is copied byte-for-byte, its schema-required `key_hint` remains present. Package build SHALL accept it only when it matches the package-safe v1 `scheme:opaque` grammar defined by the cryptography capability and SHALL otherwise fail before output creation with guidance to run `forecast key-hint update`. It MUST NOT guess whether a free-form string is a path or silently rewrite imported bytes. CLI-created hints use `forecast-key:<forecast-id>` and never record the actual key-file path.

Before writing, build SHALL scan the prospective manifest and generated/copy set for protected-root membership, known secret file roles/names, absolute paths, and secret canaries used by acceptance tests. Detection SHALL fail the operation rather than merely warn.

#### Scenario: Key located under output source tree
- **WHEN** a key file is adjacent to a target or otherwise discoverable during package enumeration
- **THEN** it is not included, and any attempt to include it explicitly is rejected

#### Scenario: Imported machine-local key hint
- **WHEN** the exact ledger contains a key hint that exposes an absolute or relative key-file path
- **THEN** package build fails the closed logical-hint validation, identifies the selected forecast, and directs the user to the explicit repair command without rewriting or copying anything

#### Scenario: Unrevealed plaintext found
- **WHEN** prospective package content contains a sealed forecast's private bundle outside ciphertext
- **THEN** build fails secret-disclosure checks and removes or recovers only output it created

### Requirement: Create packages without overwrite or partial success
The output path MUST be absent. Build SHALL validate and hash every source, resolve every destination, detect case-insensitive collisions, and prepare the complete manifest before committing package files. It SHALL create files with safe permissions under a new confined directory, flush them, write the manifest last, and use a journal to recover interruption. It MUST NOT merge into an existing directory or overwrite unrelated content.

#### Scenario: Existing output directory
- **WHEN** any directory entry already exists at `--output`
- **THEN** build returns `conflict` without adding, deleting, or inspecting unrelated contents beyond safe type/collision checks

#### Scenario: Interrupted package build
- **WHEN** the process stops after copying some CLI-created files
- **THEN** recovery removes or identifies only those recorded partial files and never reports a complete package without a durable matching manifest

### Requirement: Verify the source evidence while building
Before package commit, build SHALL reconstruct and compare every included target, validate every receipt's binding and syntax offline, verify every ledger/reference digest, and run applicable revealed-forecast authentication. Pending receipts MAY be packaged but SHALL remain labeled pending. A failed content/reveal/package-integrity check MUST prevent package creation; unavailable optional network verification MUST NOT trigger implicit network access.

#### Scenario: Package pending proof
- **WHEN** target and receipt are internally valid but the OTS proof is pending
- **THEN** build succeeds with pending state preserved and does not describe existence timing as verified

### Requirement: Return complete package identity
Successful build SHALL return output root, packaged ledger path, manifest path, manifest SHA-256, sorted file count/roles, total bytes, and evidence-state summary without absolute secret paths. `--dry-run` SHALL return the same prospective file roles/paths and deferred hashes where bytes are not safely available, but create no directory or file.

#### Scenario: JSON build result
- **WHEN** package build succeeds in JSON mode
- **THEN** stdout contains one stable result that is sufficient to identify and later verify the exact local package

### Requirement: Verify a package from explicit retained files
`forecast-ledger publish verify` SHALL require a real `--file` path naming the packaged ledger and `--manifest` naming its manifest. Both SHALL resolve inside the same confined package root. The command SHALL parse a bounded closed manifest, reject duplicate/non-normalized/absolute/traversing/case-colliding paths, verify the declared schema pin, and hash every listed file before parsing the ledger or trusting a role.

It SHALL fail if a listed file is missing, a digest/size/role differs, the ledger path does not equal `--file`, a symlink/junction escapes, or an unexpected regular file exists outside the manifest allowlist (excluding explicitly documented platform metadata). Only after package integrity passes SHALL it run document, content, reveal, and offline timestamp layers.

#### Scenario: Manifest path escape
- **WHEN** a manifest entry uses `..`, an absolute path, drive/UNC syntax, or a link escaping the package root
- **THEN** verification fails before opening the escaped target

#### Scenario: Extra unlisted file
- **WHEN** the package contains an unlisted key or other regular file
- **THEN** package integrity fails and the unexpected safe relative path is reported without reading secret contents

### Requirement: Keep package verification offline by default
Package verification SHALL require no original authoring location, source-control repository, hosting service, calendar, or network. `--online` MAY enable existence-timing revalidation through the built-in dual-public-source profile without requiring endpoint configuration; optional Bitcoin Core options MAY replace that profile for independently operated verification. Package integrity and content/reveal results MUST remain independently visible if an online source fails. `--offline` and `--online` MUST be mutually exclusive, and omission of both SHALL remain offline for portable-package verification.

This intentional default differs from layered `forecast-ledger verify`, which is online unless `--offline` is supplied. Help, README, and command references SHALL state the contrast next to both commands. `publish verify` is read-only and SHALL NOT expose `--dry-run`; `--online` opts into network observation while omission or explicit `--offline` keeps portable verification local.

#### Scenario: Package copied to removable media
- **WHEN** a verifier receives only the retained package and runs offline verification
- **THEN** manifest, ledger, target, receipt syntax/binding, and reveal checks complete without contacting the original author or publication location

### Requirement: Use verification exit semantics without hiding the report
Manifest/path/digest failure SHALL return verification exit `6`; invalid ledger data exit `3`; a missing CLI-selected `--file` or `--manifest` SHALL return not-found exit `4`; after a manifest parses successfully, any listed entry that is absent, replaced, or unreadable SHALL be a package-integrity verification failure with exit `6`; pending or budget-incomplete evidence without failure SHALL return exit `9`; requested network failure SHALL return exit `8`; a complete applicable pass SHALL return exit `0`. JSON mode SHALL emit the safely available package/evidence matrix on non-zero verification outcomes.

#### Scenario: Selected manifest does not exist
- **WHEN** the path passed to `--manifest` is absent
- **THEN** package verification returns not-found/exit `4` before claiming to have loaded a package identity

#### Scenario: Listed receipt is missing
- **WHEN** a valid parsed manifest lists a receipt whose package entry is absent
- **THEN** package integrity fails with exit `6` because the claimed package is incomplete

#### Scenario: One tampered target among valid files
- **WHEN** one listed target digest differs while other files match
- **THEN** package verification fails overall, identifies that entry, and still reports independently established manifest observations without claiming package validity
