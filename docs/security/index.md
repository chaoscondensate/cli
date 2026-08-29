# Security documentation

<!-- doc-metadata
coverage: v0.4.0
reviewed: 2026-08-29
owner: security
generated: false
security-critical: true
prerequisites: ../getting-started/index.md
next: ../explanation/verification-claims.md
-->

Forecast Ledger CLI is Preview software and has not received a recorded
independent security or cryptographic audit. Before using it with private
material, review:

- the [evidence, maturity, and audit baseline](../development/documentation-baseline.md);
- the [dependency security review](../development/dependencies.md); and
- the [README evidence boundaries](../../README.md#evidence-boundaries); and
- the [RFC 3161 trust and privacy limits](../how-to/timestamp-forecasts.md);
- the [package allowlist and manifest checks](../how-to/publish-evidence.md);
- the [MCP root, mode, and reveal boundaries](../how-to/run-mcp.md); and
- the repository [security policy](../../SECURITY.md).

Keep private operation input and protected key files under a separate secret
root. Do not place a secret root inside a ledger or package root. A sealed
ledger protects its forecast bundle only while the key
and private input remain separate. Reveal is irreversible publication of the
private fields and requires explicit approval.

`timestamp stamp` sends the SHA-256 digest of the canonical target, a random
nonce, and request timing to the explicitly selected RFC 3161 authority. It
does not send forecast plaintext unless the digest itself is already known to
the authority. The authority learns when the request was made and may retain
network metadata. The CLI accepts only a public HTTPS endpoint, bounded
responses, and same-origin redirects.

Later status, timestamp verification, layered verification, and package
verification are local. They trust only the CA bundle retained with the ledger,
not the operating-system root store. Preserve that bundle, the request, the
response, and the target together. A valid signature proves only that the named
authority issued the token for the digest at its asserted generation time.

A valid document or package manifest is a structural conclusion. Overall
verification `pass` additionally requires at least one applicable
forecast-evidence layer; empty selections return `no_evidence`.

Publication copies only the exact ledger and referenced timestamp artifacts. Always
inspect the new package before sharing it, because an exact ledger may contain
fields intentionally disclosed by an earlier reveal.

Do not put keys, credentials, private ledgers, or unrevealed forecast material
in public issues. Suspected vulnerabilities belong in
[GitHub Private Vulnerability Reporting](https://github.com/chaoscondensate/cli/security/advisories/new).
Conduct reports belong at `andrey@chaoscondensate.com`.

[Documentation index](../index.md)
