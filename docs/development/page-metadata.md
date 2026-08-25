# Documentation metadata and ownership

<!-- doc-metadata
coverage: development
reviewed: 2026-08-25
owner: documentation
generated: false
security-critical: false
prerequisites: documentation-style.md
next: index.md
-->

Every maintained Markdown page under `docs/` carries one `doc-metadata` comment
immediately after its level-one heading. The comment keeps repository Markdown
machine-checkable without requiring a documentation site generator.

## Required fields

```text
<!-- doc-metadata
coverage: v0.1.1
reviewed: 2026-08-25
owner: documentation
generated: false
security-critical: false
prerequisites: ../getting-started/index.md
next: ../how-to/index.md
-->
```

| Field | Rule |
| --- | --- |
| `coverage` | Product version/range such as `v0.1.1`, or `development` for contributor-only material. |
| `reviewed` | ISO `YYYY-MM-DD` date of the last substantive review, not a cosmetic edit. |
| `owner` | One role from `project-maintainer`, `documentation`, `security`, `interface`, or `release`. |
| `generated` | `true` only when the page is reproduced mechanically and carries a visible generated warning; otherwise `false`. |
| `security-critical` | `true` when incorrect guidance could expose secrets, lose evidence, weaken verification, or misroute a private report. |
| `prerequisites` | Comma-separated relative page paths, or `none`. These are reading or operation prerequisites, not every inbound link. |
| `next` | Comma-separated relative page paths for useful next actions, or `none` for a terminal policy page. |

A generated page also records a `source` relative path or command in the
metadata block and states “Generated; do not edit by hand” in visible prose.

## Ownership

- `project-maintainer` owns license, conduct, governance, citation, and final
  policy approval.
- `documentation` owns information architecture, prose quality, navigation,
  accessibility, and checked examples.
- `security` owns vulnerability guidance, threat and key-custody material,
  evidence limitations, and confidential reporting routes.
- `interface` owns generated CLI/MCP reference and compatibility with the
  candidate binary and pinned contract.
- `release` owns install artifacts, checksums, signing status, changelog
  derivation, and release-note accuracy.

Andrey Korchak is the current project maintainer and default holder of roles not
yet delegated. `CODEOWNERS` and governance policy may name additional reviewers;
metadata records the accountable role rather than a username that would need to
be edited across every page.

## Review triggers

Update `reviewed` after the responsible owner checks substantive effects of a
schema, cryptography, timestamp, CLI, MCP, platform-support, policy, or release
change. Security-critical pages require their named owner in addition to normal
documentation review. A generated page's review date changes only after its
source, output, and surrounding explanation have been reviewed together.

[Development documentation](index.md) · [Documentation index](../index.md)
