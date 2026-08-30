# Security policy

Forecast Ledger CLI handles files that may contain unrevealed forecasts and
cryptographic material. Please report suspected vulnerabilities privately so
users have a chance to update before technical details become public.

## Supported versions

The project is Preview software. Security fixes are made only on the current
development branch and shipped in the newest release line.

| Version | Security support |
| --- | --- |
| Latest `0.5.x` release | Best-effort fixes |
| Current `main` branch | Reports accepted; not a supported end-user release |
| Earlier releases | Unsupported; upgrade to the latest release |

The [release page](https://github.com/chaoscondensate/cli/releases/latest)
identifies the latest release. Preview support does not mean that the code,
cryptography, or evidence model has received an independent audit.

## Report a vulnerability privately

Use [GitHub Private Vulnerability Reporting](https://github.com/chaoscondensate/cli/security/advisories/new).
It creates a private draft advisory visible to the reporter and repository
security maintainers. Do not disclose an unpatched vulnerability in a public
issue, pull request, discussion, social post, or conduct email.

If GitHub's private form is unavailable, open a public issue that asks only for
a private security contact. Do not include the vulnerability, affected paths,
proof of concept, keys, credentials, private ledger content, or unrevealed
forecast material in that issue.

Include as much of the following as is safe:

- affected release, commit, operating system, architecture, and installation
  method;
- the affected command, MCP tool, file type, or evidence workflow;
- expected security boundary and observed behavior;
- impact and a realistic attack scenario;
- minimal reproduction steps or synthetic proof of concept;
- logs with secrets, local paths, and private data removed;
- whether the issue is already public or being reported elsewhere; and
- your preferred disclosure timeline and credit name, if any.

Use synthetic data when possible. If sensitive material is essential, first
describe its type and ask the maintainer how to transfer it. Never submit a
real protected key, credential, unrevealed forecast, or full private ledger.

## Response targets

The security maintainer aims to:

- acknowledge a new report within five business days;
- provide an initial triage decision within ten business days; and
- provide a status update at least every fourteen calendar days while an
  accepted report remains unresolved.

These are good-faith targets, not a service-level agreement or a promise of a
fix. Complex reports, maintainer availability, or a safety concern may require
more time. If a target will be missed, the maintainer will try to explain the
delay through the private advisory.

During triage, the maintainer checks reproducibility, affected versions,
severity, exposure of secret or private material, and whether the report is a
product vulnerability, an upstream dependency issue, or expected behavior.
The reporter will be told whether the report is accepted, needs more evidence,
is a duplicate, or is out of scope.

## Coordinated disclosure

For an accepted vulnerability, the maintainer and reporter should agree on a
disclosure plan. The normal sequence is to prepare and review a fix privately,
publish a corrected release, allow reasonable update time when impact requires
it, and then publish the advisory. Please avoid public technical disclosure
before that point. The project will not ask for indefinite secrecy; if progress
stalls, both sides should discuss a revised date explicitly.

Published repository advisories appear at
[GitHub Security Advisories](https://github.com/chaoscondensate/cli/security/advisories).
They identify affected and fixed versions, impact, mitigations, and credit when
the reporter wants it. A CVE may be requested when the issue and ecosystem
support it.

## Safe harbor for good-faith research

The project will not initiate legal action against research that follows this
policy, is conducted in good faith, avoids privacy violations and unnecessary
harm, and gives the project a reasonable opportunity for coordinated
disclosure. Accidental, limited violations made while acting in good faith
should be reported promptly so they can be addressed.

This safe-harbor statement does not authorize access to third-party systems or
data, denial of service, social engineering, physical attacks, destructive
testing, persistence after access, or retention of personal or private ledger
data. Follow applicable law and the rules of systems you do not own. Stop and
report if testing exposes a real secret, private ledger, or unrevealed forecast.

There is no bug-bounty program and no payment is promised. Good-faith reports
may receive public credit with the reporter's consent.
