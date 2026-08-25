## Problem and scope

<!-- What user or maintainer problem does this solve? Link the issue and accepted OpenSpec change when applicable. State what is intentionally out of scope. -->

## Changes

<!-- Describe observable behavior and the smallest important implementation details. -->

## Evidence

<!-- List exact commands run and their results. Add synthetic before/after examples when useful. -->

- [ ] Relevant unit, integration, negative, boundary, property, fuzz, or golden tests pass.
- [ ] `go mod verify`, `go test ./...`, and `go vet ./...` pass, or an exception is explained below.
- [ ] Documentation, link, example, generated-reference, and REUSE checks pass when affected.
- [ ] I used only synthetic or redacted test and review material.

## Platform impact

<!-- Describe native macOS, Linux, and Windows behavior. A cross-build alone does not verify filesystem semantics. -->

- [ ] No platform-specific effect.
- [ ] Native platform evidence or a justified follow-up is included.
- [ ] Packaging, Homebrew, archive naming, signing, or installation effects are documented.

## Contract and compatibility

<!-- Explain effects on existing ledgers and clients. Never edit vendored contract or fixtures without the complete pin, attribution, compatibility, and conformance update. -->

- [ ] No Forecast Ledger schema or fixture change.
- [ ] No canonicalization, cryptography, seal/reveal, or timestamp-evidence change.
- [ ] No stable JSON, exit-code, CLI, MCP, package-format, or verification-meaning change.
- [ ] Any affected contract has an accepted design, migration or rejection behavior, and conformance evidence.

## Security and privacy

<!-- Describe file, path, permission, network, secret, write, reveal, logging, dependency, and trust-boundary effects. -->

- [ ] No new or changed security/privacy boundary.
- [ ] Threats, abuse cases, secret handling, and least-privilege behavior are tested and documented.
- [ ] The diff contains no keys, credentials, private ledgers, unrevealed forecasts, personal data, or sensitive local paths.

## Documentation and release communication

- [ ] Public behavior, maturity, evidence claims, examples, and navigation are documented.
- [ ] `CHANGELOG.md` has an entry, or this change has no user-visible release impact.
- [ ] Release notes should mention this change, or the reason they should not is given below.
- [ ] OpenSpec tasks are checked only where their complete behavior and verification are present.

## Exceptions and reviewer notes

<!-- Explain every unchecked item that would normally apply. Call out security-critical, generated, licensing, community-policy, and release files that need a designated owner review. -->
