# Dependency baseline

<!-- doc-metadata
coverage: development
reviewed: 2026-08-25
owner: security
generated: false
security-critical: true
prerequisites: ../../AGENTS.md
next: build.md
-->

Reviewed: 2026-08-25

Runtime dependencies are pinned to stable tagged releases in `go.mod`. The
project uses Go 1.27.0 as its module language floor and reviewed toolchain.
Release and CI updates must keep the current and previous supported Go release
in the test matrix.

## Direct dependencies

| Purpose | Module | Version | License | Review note |
| --- | --- | --- | --- | --- |
| CLI | `github.com/urfave/cli/v3` | `v3.11.0` | MIT | Stable v3; nested commands, typed flags, exit handling, and completion support. |
| MCP | `github.com/modelcontextprotocol/go-sdk` | `v1.7.0` | Apache-2.0 and MIT; documentation CC-BY-4.0 | Official stable SDK; target MCP `2026-07-28` over stdio and test negotiation with the previous supported revision. |
| JSON Schema | `github.com/santhosh-tekuri/jsonschema/v6` | `v6.0.3` | Apache-2.0 | Draft 2020-12 support; schemas are loaded only from embedded bytes and formats are enabled explicitly. |
| JSON Schema ECMA-262 patterns | `github.com/dlclark/regexp2` | `v1.11.0` | MIT | Required for the pinned schema's lookaheads; matches use a fixed timeout. |
| YAML | `go.yaml.in/yaml/v3` | `v3.0.5` | Apache-2.0 and MIT | Maintained v3 import path. Parsing requires explicit size/depth/alias limits and source-node golden tests. |
| File locks | `github.com/gofrs/flock` | `v0.13.0` | BSD-3-Clause | Cross-platform advisory locking behind a project interface; never treated as a security boundary or atomic-write replacement. |
| ChaCha20-Poly1305 | `golang.org/x/crypto` | `v0.55.0` | BSD-3-Clause | Go team module; only the published standard-nonce profile is exposed. Randomness and SHA-256 use the standard library. |

The MCP SDK brings additional pinned modules for JSON schemas, URI templates,
encoding, authentication helpers, and protocol support. Their resolved versions
are recorded in `go.sum`; releases must also generate an SBOM and complete
third-party notices, including the MCP SDK's mixed-license history.

## Security review

- `go mod verify` reported that every downloaded module matched its checksum.
- `govulncheck` from `golang.org/x/vuln/cmd/govulncheck@v1.7.0` reported no
  reachable vulnerabilities on 2026-08-25.
- MCP SDK releases before the selected baseline had high-severity advisories;
  `v1.7.0` includes the fixes and a patched `segmentio/encoding v0.5.4`.
- Stdio is the only MCP v1 transport. HTTP-specific origin and DNS-rebinding
  risks remain out of scope until an HTTP transport is separately designed.
- YAML v3 is conservative and effectively feature-frozen. Security fixes remain
  acceptable for v1; migration to a later major version requires compatibility
  and presentation-preservation review.
- `gofrs/flock` remains pre-v1, so project code must isolate it behind a narrow
  interface and test native Windows and Unix behavior.

A clean scan is not a security guarantee. Run these commands after every module
update and before release:

```sh
go mod verify
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
```

Do not add a runtime logging, configuration, assertion, secret-manager, or
general-purpose cryptography dependency without a documented need, license
review, vulnerability scan, and conformance impact review.

[Development documentation](index.md) · [Documentation index](../index.md)
