# Executable name review

<!-- doc-metadata
coverage: development
reviewed: 2026-08-25
owner: release
generated: false
security-critical: false
prerequisites: ../../AGENTS.md
next: releasing.md
-->

Reviewed: 2026-08-25

## Decision

The executable name is `forecast-ledger`. The Go module remains
`github.com/chaoscondensate/cli`.

The provisional name `chaos` was rejected because two active projects install
that exact command. [Chaos Toolkit](https://chaostoolkit.org/reference/usage/install/)
uses it as its primary cross-platform CLI, while Homebrew's
[`chaos-client`](https://formulae.brew.sh/formula/chaos-client) formula and
[ProjectDiscovery installation](https://docs.projectdiscovery.io/opensource/chaos/install)
also install `chaos`.

## Collision checks

- macOS/Linux local `PATH`: no `forecast-ledger` executable was present.
- Homebrew formulae and casks: an exact-name `brew search` returned no result
  after refreshing the current index.
- Scoop Main and Extras: GitHub code search of the current bucket manifests
  returned no `forecast-ledger` match.
- Winget: GitHub code search of the current `microsoft/winget-pkgs` manifests
  returned no `forecast-ledger` match.
- A public exact-name search found uses of “forecast ledger” as a generic phrase,
  but no established CLI distributed as `forecast-ledger`.

The name is valid as a Unix executable and as `forecast-ledger.exe` on Windows.
This review reduces collision risk but does not reserve the name. Repeat the
registry checks before the first stable release and before publishing a new
package-manager manifest.

Use `ChaosCondensate.ForecastLedger` as the Winget package identifier and
`forecast-ledger` as the Homebrew/Scoop package token and installed command. Do
not install `chaos` as a compatibility alias.

[Development documentation](index.md) · [Documentation index](../index.md)
