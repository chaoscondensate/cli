# Governance

Forecast Ledger CLI uses a maintainer-led open-source governance model. Design
and review happen in public where security and privacy allow; repository access
and release publication remain explicit responsibilities.

## Current roles

Andrey Korchak (GitHub [`@57uff3r`](https://github.com/57uff3r)) is the current:

- Project Maintainer, responsible for roadmap, scope, architecture, and final
  merge decisions;
- Release Maintainer, authorized to approve and publish release tags, GitHub
  Releases, attestations, and Homebrew updates;
- Security Maintainer, responsible for private vulnerability triage and
  coordinated disclosure; and
- Community Manager and Moderator, responsible for the Code of Conduct and
  community access decisions.

Contributors own the accuracy of their submissions and participate in review,
but a contribution does not grant repository, release, security-advisory, or
organization access. New maintainers are appointed in a public pull request to
this file, except that private security access may be staged before disclosure
when needed to protect users.

## Decision process

Routine fixes and documentation changes are decided through issue and pull
request review. The maintainer seeks a clear problem statement, relevant tests,
compatibility evidence, and contributor agreement when practical. The current
Project Maintainer makes the final merge or rejection decision and explains
material trade-offs in the issue, pull request, or OpenSpec record.

Large behavior changes start with a public issue and an OpenSpec proposal.
Accepted OpenSpec artifacts define implementation scope; completing a task does
not by itself approve a merge. Decisions may be revisited when new security,
compatibility, user, or maintenance evidence appears.

## Protocol-affecting changes

A change affects the protocol when it alters the embedded schema,
canonicalization, commitment or reveal bytes, timestamp evidence, layered
verification meaning, portable package format, stable machine output, MCP
contract, or interpretation shared with another implementation.

Such a proposal requires:

1. a public OpenSpec design and compatibility analysis;
2. exact authoritative sources and pinned versions or commits;
3. positive, negative, boundary, and byte-for-byte conformance evidence;
4. security and privacy review appropriate to the affected layer;
5. migration or rejection behavior for existing files and clients;
6. matching CLI, MCP, reference, and user-documentation updates; and
7. explicit approval by the Project Maintainer recorded in the pull request or
   accepted OpenSpec history.

An urgent private vulnerability fix may be developed in a GitHub security
advisory. Its public advisory and follow-up records must explain protocol and
compatibility effects once disclosure is safe.

## Release authority

Only the Release Maintainer may create or push an official release tag, approve
the protected release environment, publish or amend a GitHub Release, or update
the official Homebrew tap. Release tags are immutable and are not reused.
Contributors may create local GoReleaser snapshots, but snapshots are not
official releases.

The release decision requires passing the documented code, conformance,
documentation, licensing, community-health, packaging, and provenance gates.
The [release runbook](docs/development/releasing.md) is operational guidance;
this file identifies who has authority.

## Access expectations

Repository roles follow least privilege. People with write, release, security,
or organization access are expected to:

- use multi-factor authentication and protect recovery methods;
- keep personal and automation credentials separate and narrowly scoped;
- avoid sharing credentials or copying private advisories into public tools;
- remove unused access promptly; and
- disclose account compromise or a conflict that could affect a decision.

Homebrew tap access is separate from source-repository access. Security-advisory
access does not automatically grant release authority. Access changes should be
reviewable in platform audit logs and, when safe, reflected in the roles above.

## Conflicts and appeals

A decision-maker must disclose a financial, employment, personal, or other
material conflict and recuse when the conflict could reasonably affect trust in
the result. The Project Maintainer may ask an uninvolved subject-matter reviewer
to advise or decide the matter. Conduct reports involving the current moderator
follow the recusal process in `CODE_OF_CONDUCT.md`.

A contributor may ask for reconsideration with new evidence in the original
issue or pull request. Repetition without new evidence does not require a new
decision. Personal disputes are handled under the Code of Conduct, not by
turning technical review into public adjudication.

## Inactive-maintainer path

If the sole Project Maintainer cannot be reached, contributors should make two
public, respectful attempts through a governance issue over at least 30 days,
unless a security emergency requires the private advisory channel. The
administrators of the `chaoscondensate` GitHub organization may then appoint an
interim maintainer, preserve release and security access, and open a public
governance change.

An interim maintainer should prioritize security response, release integrity,
and continuity. They must not make an avoidable protocol-breaking change before
the community has a reasonable opportunity to review it.

## Changing governance

Governance evolves through a dedicated public issue, an OpenSpec proposal when
behavior or release policy is affected, and a focused pull request to this file.
The proposal must describe the problem, new roles or decision rights,
transition, access changes, conflicts, and effect on existing contributors.

As the maintainer group grows, the project should add independent security and
conduct contacts, require multiple reviewers for protocol and release changes,
and document how maintainers are appointed and removed. Until such a change is
merged, the current roles and single-maintainer limitations above remain the
authoritative model.
