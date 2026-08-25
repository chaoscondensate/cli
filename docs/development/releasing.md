# Release process

Forecast Ledger releases use GoReleaser 2.18.0. A pushed annotated SemVer tag
creates a GitHub Release, six cross-platform archives, SHA-256 checksums, archive
SBOMs, a GitHub artifact attestation, and a Homebrew cask update.

The release targets are:

- macOS: amd64 and arm64
- Linux: amd64 and arm64
- Windows: amd64 and arm64

Unix archives use `tar.gz`; Windows archives use `zip`. All builds use
`CGO_ENABLED=0`, `-trimpath`, read-only modules, and embedded version and source
revision metadata.

## One-time repository setup

1. Confirm that the top-level Apache-2.0 `LICENSE` and public `README.md` are
   present and accurate. The release workflow refuses to publish without them.
2. Create the public `chaoscondensate/homebrew-tap` repository and make an
   initial README commit on `main`; an empty repository has no branch for
   GoReleaser to update. GoReleaser will create and maintain
   `Casks/forecast-ledger.rb`.
3. Create a fine-grained GitHub personal access token or GitHub App credential
   with `Contents: Read and write` access to only
   `chaoscondensate/homebrew-tap`. It does not need access to this repository.
4. Add that credential to this repository as the Actions secret
   `HOMEBREW_TAP_TOKEN`.
5. Configure the `release` GitHub environment. Require approval for stable
   releases and allow release tags from protected `main`.
6. In repository Actions settings, keep the default `GITHUB_TOKEN` restricted.
   The release workflow grants only `contents`, `id-token`, and `attestations`
   permissions explicitly.
7. Protect `main` with the `CI` workflow. Enable immutable releases and prevent
   release-tag deletion or movement if the organization policy supports it.

The tap requires a separate token because GitHub's workflow token cannot write
to a different repository. Never reuse a broad classic PAT when a tap-only
fine-grained credential is available.

## Pre-release checks

Run these commands from a clean checkout:

```sh
gofmt -w cmd internal
go mod verify
go test ./...
go vet ./...
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

Then inspect `dist/artifacts.json`, run each native binary available on the
current host, and confirm that `forecast-ledger version --json` contains the
expected version, commit, schema pin, Go version, and MCP protocol version.

The CI workflow repeats tests on Ubuntu, macOS, and Windows and builds a full
six-target snapshot without publishing. Treat a snapshot failure as a release
blocker.

## Publishing

Start with a release candidate:

```sh
git switch main
git pull --ff-only
git status --short
git tag -a v0.1.0-rc.1 -m "forecast-ledger v0.1.0-rc.1"
git push origin v0.1.0-rc.1
```

GoReleaser marks prerelease tags as prereleases and does not update the stable
Homebrew cask. After checking the candidate archives on macOS, Linux, and
Windows, publish the stable tag:

```sh
git tag -a v0.1.0 -m "forecast-ledger v0.1.0"
git push origin v0.1.0
```

Do not reuse or move a release tag. Fix a failed or incorrect release in a new
patch or prerelease version.

## Post-release verification

1. Confirm the GitHub Release has six archives, `checksums.txt`, and SBOM files.
2. Download the archives and verify them with `sha256sum -c checksums.txt` or
   `shasum -a 256 -c checksums.txt`.
3. Verify provenance with:

   ```sh
   gh attestation verify --owner chaoscondensate <archive>
   ```

4. Check that the tap contains `Casks/forecast-ledger.rb` for the new stable
   version.
5. On both Apple Silicon and Intel macOS, run:

   ```sh
   brew update
   brew install --cask chaoscondensate/tap/forecast-ledger
   forecast-ledger version --json
   brew uninstall --cask forecast-ledger
   ```

6. Smoke-test the native Linux and Windows archives, including help, version,
   validation, and MCP startup without protocol output corruption.
7. Update installation documentation and announce the release through
   <https://chaoscondensate.com/> when appropriate.

## Signing status

The initial pipeline creates checksums, SBOMs, and GitHub artifact attestations,
but it does not Apple-sign or notarize the macOS binaries. The Homebrew cask
uses Homebrew's documented unsigned-binary fallback to remove the quarantine
attribute after installation. This status must be stated in release notes.

Before calling the release channel stable, prefer Apple Developer ID signing
and notarization, remove the cask's `xattr` post-install hook, and add a native
verification step for the signed archives. Windows code signing is also not yet
configured and must be documented as such.
