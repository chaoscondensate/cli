# Security documentation

<!-- doc-metadata
coverage: v0.3.1
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
- the [OpenTimestamps trust and privacy limits](../how-to/timestamp-forecasts.md);
- the [package allowlist and manifest checks](../how-to/publish-evidence.md);
- the [MCP root, mode, and reveal boundaries](../how-to/run-mcp.md); and
- the repository [security policy](../../SECURITY.md).

Keep private operation input, protected key files, and Bitcoin Core credentials
under a separate secret root. Do not place a secret root inside a ledger or
package root. A sealed ledger protects its forecast bundle only while the key
and private input remain separate. Reveal is irreversible publication of the
private fields and requires explicit approval.

The built-in timestamp profile sends blinded commitments to third-party
calendars and later sends block heights to two public Bitcoin APIs. Blinding
does not hide request timing; height queries reveal an approximate time period.
The two APIs must agree, but are still trusted for canonical-chain selection.
If an observer is unavailable, malformed, disagreeing, or over budget, the tool
cannot make a fresh proof comparison. It returns only stable issue kinds, safe
source IDs, and bounded request counts; raw endpoints, credentials, response
bodies, and underlying errors are not public result fields. Such acquisition
failure is not evidence that the cryptographic proof mismatched.

A valid document or package manifest is a structural conclusion. Overall
verification `pass` additionally requires at least one applicable
forecast-evidence layer; empty selections return `no_evidence`.

Publication copies only the exact ledger and referenced targets/receipts. Always
inspect the new package before sharing it, because an exact ledger may contain
fields intentionally disclosed by an earlier reveal.

Do not put keys, credentials, private ledgers, or unrevealed forecast material
in public issues. Suspected vulnerabilities belong in
[GitHub Private Vulnerability Reporting](https://github.com/chaoscondensate/cli/security/advisories/new).
Conduct reports belong at `andrey@chaoscondensate.com`.

[Documentation index](../index.md)
