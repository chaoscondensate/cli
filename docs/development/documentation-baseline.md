# Documentation baseline

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: project-maintainer
generated: false
security-critical: true
prerequisites: ../../AGENTS.md
next: documentation-traceability.md
-->

Reviewed: 2026-08-26

This page records the implementation and policy inputs used by the maintained
product documentation. It describes the repository at the reviewed commit; it
does not make unfinished interfaces available or stable.

## Documentation ownership

The `establish-open-source-product-documentation` OpenSpec change owns public
documentation and community-health deliverables. The older documentation tasks
in `build-forecast-ledger-cli-mcp` are handoff records, not a second checklist.

| Earlier task | Owning tasks in this change |
| --- | --- |
| 11.1 README and workflow quick starts | 4.1–4.8, 5.3–5.4, and 6.1–6.3 |
| 11.2 CLI reference, JSON, and exit codes | 7.1–7.2 and 7.5–7.6 |
| 11.3 Security and evidence limits | 3.5, 4.4, 8.2–8.3, and 8.6 |
| 11.4 MCP setup and safety | 6.4 and 7.3 |
| 11.5 Schema update and conformance | 6.7, 7.5, 8.1, and 8.6 |
| 11.6 Checked examples and tutorials | 5.3–5.4, 6.1–6.6, and 9.1–9.6 |

Completion is recorded only in the owning checklist. A handoff does not mean
that the corresponding document or test has been implemented.

## Implementation inventory

This inventory was checked against the current unreleased `main` source after
release `v0.1.1`.

### Working CLI surface

The following commands have connected application services:

| Command | Behavior | Network |
| --- | --- | --- |
| `forecast-ledger init --file <new-path> ... --input <path|->` | Exclusively creates a schema-valid JSON/YAML ledger with one typed question and one public or sealed forecast. A sealed forecast requires a separate new `--key-file`. | None |
| `forecast-ledger ledger update --file <path> --input <path|->` | Minimally patches allowed root and current forecaster metadata. | None |
| `forecast-ledger platform add|update|list|show|remove ...` | Creates, patches, lists, inspects, or removes unreferenced platform records. List/show accept `--file -`; removal requires approval. | None |
| `forecast-ledger question add|update|list|show|resolve|annul|dispute ...` | Creates a typed question with its first public or sealed forecast, applies safe patches, reads redacted summaries, and records approved lifecycle claims. List/show accept `--file -`. | None |
| `forecast-ledger forecast add|list|show|seal|reveal ...`, `forecast key-hint update ...` | Appends public or sealed forecasts, authenticates approved reveal, repairs safe logical hints, or reads redacted history. List/show accept `--file -`. | None |
| `forecast-ledger target build|check ...` | Builds or checks exact `forecast-envelope/v1` RFC 8785 target bytes at deterministic ledger-relative paths. | None |
| `forecast-ledger timestamp stamp|upgrade|status|verify ...` | Creates, upgrades, inspects, and verifies experimental pure-Go OpenTimestamps receipts. Built-in stamp is 2-of-4; Bitcoin verification requires two public observers to agree. CLI-only custom calendars and Bitcoin Core are advanced overrides. | Built-in profile, explicit custom mode, Core, or offline as applicable |
| `forecast-ledger verify --file <path> ...` | Reports document, content-binding, existence-timing, reveal, and outcome-evidence layers with stable states, reasons, evidence, and limitations. | Built-in Bitcoin observers unless `--offline`; outcome URLs only with `--check-sources` |
| `forecast-ledger publish build|verify ...` | Builds a deterministic allowlisted package with canonical manifest and verifies it offline by default. It does not use Git or a hosted publisher. | None by default; verify uses built-in Bitcoin observers only with `--online` |
| `forecast-ledger mcp serve --ledger-root <name=path> ...` | Starts a protocol-clean stdio server with closed CLI-parity tools and redacted addressed resources. Default is read-write/online within roots; read-only/offline are whole-server modes and reveal is separately default-off. | Built-in profile unless server is offline |
| `forecast-ledger validate --file <path>` | Parses and validates an embedded-schema JSON or YAML ledger. Eligible for `--file -`. | None |
| `forecast-ledger status --file <path>` | Validates a ledger and summarizes questions, forecasts, and receipt states. Eligible for `--file -`. | None |
| `forecast-ledger version [--json]` | Reports binary, source, Go, schema, and MCP protocol metadata. | None |
| `forecast-ledger completion <shell>` | Generates Bash, Zsh, Fish, or PowerShell completion text through urfave. | None |

The root exposes stable presentation flags `--json`, `--plain`, `--quiet`,
`--verbose`, `--no-color`, `--no-input`, `--yes`, and `--timeout`. JSON, plain, and quiet modes are
mutually exclusive. Stable application error codes are `usage`,
`invalid_data`, `unsupported_schema_version`, `not_found`, `conflict`,
`verification`, `io`, `network`, `network_disabled`, `pending`, `incomplete`,
`unavailable`, `internal`, and `interrupted`; their CLI exits are implemented in
`internal/app/errors.go`.

### Availability and maturity

Every visible leaf in the current source has a real action and an availability
test; the old preview unavailable handler has been removed. This does not make
every subsystem stable. The OpenTimestamps backend is explicitly experimental
until official-client differential coverage, real-calendar liveness, complete
native-platform recovery, budget and malformed-input matrices, and an
independent review are recorded. Installed releases may expose less than
unreleased `main`; candidate-binary help and `version --json` are authoritative
for an installed artifact.

### Contract and protocol pins

| Item | Reviewed value |
| --- | --- |
| Forecast Ledger schema | `1.0.0` |
| Schema commit | `e409463d702888fefd253b32f21b9b2f864aabed` |
| Embedded schema SHA-256 | `e63bdd01f0241aa4d94d5ccc45e84bcea70a6a7fd46ab77cff4802b3f8b8fc65` |
| MCP protocol target | `2026-07-28` |
| Timestamp profile | `opentimestamps-public-v1` (experimental; four calendars, threshold two, two Bitcoin observers; per invocation: 32 heights, 128 requests, four concurrent) |
| Go toolchain | `1.27.0` |

Validation uses embedded contract bytes and does not resolve remote schema
references. The moved `v1.0.0` upstream tag is not a safe pin; documentation
and builds use the exact commit and digest above.

### Platforms and release artifacts

GoReleaser builds six `CGO_ENABLED=0` archives:

- macOS arm64 and x86-64 as `.tar.gz`;
- Linux arm64 and x86-64 as `.tar.gz`; and
- Windows arm64 and x86-64 as `.zip`.

Release `v0.1.1` contains those six archives and eight native Linux packages:
`deb`, `rpm`, `apk`, and Arch Linux packages for arm64 and x86-64. Each package
installs the binary in `/usr/bin` and the Apache-2.0 license in
`/usr/share/licenses/forecast-ledger`. The main checksum manifest covers the
archives, Linux packages, and their fourteen SBOMs. GitHub artifact attestations
bind those published digests. CI runs unit tests and vet on current
GitHub-hosted macOS, Ubuntu, and Windows runners, checks the full artifact matrix
on Ubuntu and Windows, and installs and removes the Debian package on Ubuntu. A
cross-build does not establish native filesystem correctness on every
architecture.

Windows x86-64 also receives a Chocolatey `nupkg`, built, published,
smoke-tested, and separately attested on the Windows release runner; Windows
ARM64 uses the native ZIP. GoReleaser creates the `nupkg` after the main checksum
manifest, so that package is not a `checksums.txt` entry. Release `v0.1.0`
predates the native Linux and Chocolatey packages and contains archives only.

These are GitHub Release assets, not APT, RPM, APK, Arch, public Chocolatey,
Winget, or Scoop repositories. Package-manager update discovery is therefore
not available yet.

Homebrew installs the stable release as
`chaoscondensate/tap/forecast-ledger`. The macOS archives are not Developer ID
signed or notarized; the formula removes quarantine metadata and applies a
local ad-hoc signature in the Cellar. Windows archives and the Chocolatey
package are not code signed. Linux packages are not package-signed.

## Licensing and contribution policy

The maintainer approved the following policy on 2026-08-25:

- Original software, documentation, examples, and generated assets use the
  Apache License 2.0, SPDX identifier `Apache-2.0`.
- The copyright notice is `Copyright 2026 Chaos Condensate contributors`.
- Vendored or otherwise incorporated third-party material keeps its upstream
  license and attribution. Its license is stored with the material where
  practical and represented in third-party notices and release SBOMs.
- Dependencies are not relicensed. Their resolved versions and licenses are
  reviewed and recorded separately from the project's own license.
- The project does not require a Contributor License Agreement or Developer
  Certificate of Origin sign-off. Contributions intentionally submitted for
  inclusion are handled under the contribution terms of Apache-2.0. A future
  policy change applies prospectively and must be documented before use.

Apache-2.0 permits commercial and non-commercial use. Choosing an OSI-approved
license does not state or require that the maintainers plan to commercialize
the project.

## Project contacts and authority

The maintained project routes are:

| Concern | Route |
| --- | --- |
| Canonical repository | <https://github.com/chaoscondensate/cli> |
| Public usage, bug, feature, and documentation support | <https://github.com/chaoscondensate/cli/issues> |
| Private vulnerability report | <https://github.com/chaoscondensate/cli/security/advisories/new> |
| Private conduct report | `andrey@chaoscondensate.com` |
| Forecast Ledger interoperability | <https://github.com/chaoscondensate/schema> |
| Broader project context | <https://chaoscondensate.com/> |

Andrey Korchak (`@57uff3r`) is the current project maintainer, governance owner,
security triage owner, conduct enforcement contact, and release authority.
These roles may be delegated publicly as the maintainer group grows. No other
person may publish an official release or change a protected policy solely by
virtue of having contributed code.

The project supports the latest stable release on a best-effort basis. The
unreleased `main` branch and preview or experimental components receive no
separate compatibility or response-time promise. A security fix may be issued
for an older release when the maintainer judges that users cannot reasonably
upgrade, but that exception does not create a standing long-term-support line.
Support policy does not promise an acknowledgment or resolution service level.

## Product identity and language

**Product name:** Forecast Ledger CLI  
**Executable name:** `forecast-ledger`  
**One-sentence value:** Create and independently check portable forecast
evidence without requiring Git or a hosted service.

The intended audiences are:

- individuals and teams that keep explicit forecast histories;
- reviewers and researchers who independently inspect forecast evidence; and
- local-tool and MCP integrators who need a structured, permission-bounded
  interface.

The maintained maturity labels have these meanings:

| Label | Meaning |
| --- | --- |
| Development | Unreleased work may be incomplete and may change without migration support. |
| Preview | A release is available for evaluation, but feature, compatibility, native-platform, or review gates are still open. |
| Stable | The documented interface and required production gates are complete for the stated support window. |
| Experimental | A named component has additional unresolved conformance, security, or interoperability risk. |
| Deprecated | The behavior still works for a stated period but has a documented replacement and removal version. |
| Unsupported | The behavior is outside the maintained contract and must not be presented as working. |

Release `v0.1.1` is **Preview**. A stable SemVer tag identifies a reproducible
release; it does not promote unfinished commands or unaudited components to the
Stable product maturity label.

Evidence language is bounded as follows:

- **valid** means that named structural, format, and semantic checks passed;
- **sealed** means that forecast plaintext was processed by the named sealing
  profile and is not present in the sealed ledger field;
- **timestamped** names a receipt state and never by itself means independently
  verified timing;
- **anchored** means the named receipt was verified against its stated
  cryptographic trust source;
- **verified** must name the layer that passed, failed, is pending, was not
  checked, or does not apply;
- **revealed** means a disclosed value was checked against the named sealed
  commitment; it does not erase the original commitment evidence;
- **published** means material was made available through some transport and
  adds no cryptographic claim by itself;
- **evidence** is inspectable material relevant to a bounded claim; and
- **proof** is used only when a specific protocol and conclusion are named.

The product does not use validity, timestamps, publication, or layer results to
claim authorship, ledger completeness, forecast truth, exact self-reported
time, or substantive outcome-source correctness.

### Maturity, audit, and limitations

- The approved public status currently visible in README is **active
  pre-release development**. `v0.1.1` is published as a stable SemVer release,
  so maintained documentation must explain that version stability does not
  imply feature completeness.
- No independent security or cryptographic audit is recorded. A pinned
  `govulncheck` run reported no reachable dependency vulnerability on
  2026-08-25; that result is not an audit or a security guarantee.
- Local validation and status are offline. No telemetry or runtime update check
  is implemented. Planned OpenTimestamps calendar operations would use the
  network only at an explicit command and grant boundary, but are unavailable.
- Sealing, reveal, target generation, OpenTimestamps, layered verification,
  evidence packages, and MCP tools/resources are not usable from this build.
- Pending receipts are not verified timing. Validation and release provenance
  do not prove authorship, completeness, forecast truth, exact self-reported
  time, or outcome-source correctness.
- The embedded schema and conformance fixtures retain upstream attribution in
  `third_party/forecast-ledger`; they are not maintained as original project
  material.

[Development documentation](index.md) · [Documentation index](../index.md)
