# Accessible visual system

<!-- doc-metadata
coverage: development
reviewed: 2026-08-25
owner: documentation
generated: false
security-critical: false
prerequisites: documentation-style.md
next: index.md
-->

Reviewed: 2026-08-25

Documentation must remain complete as text. Diagrams, terminal captures,
colors, icons, and badges may clarify information but never carry an essential
instruction or status alone.

## Palette

Maintained light-background assets use this restrained palette:

| Role | Color | Text label or shape requirement |
| --- | --- | --- |
| Background | `#FFFFFF` | Not applicable |
| Primary text and lines | `#111827` | Default prose and diagram edges |
| Information | `#1D4ED8` | Label `Note` or use a circle marker |
| Success | `#166534` | Label `Pass` or use a check marker |
| Pending or caution | `#92400E` | Label `Pending` or `Warning`; use a triangle marker |
| Failure or danger | `#B91C1C` | Label `Fail` or `Danger`; use a cross or octagon marker |
| Secondary text | `#4B5563` | Supporting text only, never tiny text |

These foreground colors are selected for at least WCAG AA normal-text contrast
against white. Generated assets must test their actual foreground/background
pair rather than assuming a palette entry is sufficient. Do not lower opacity
on essential text. A dark-background variant requires its own recorded contrast
check; do not invert these colors mechanically.

## Status labels

Write the status word in every visual and nearby prose:

- maturity: `Development`, `Preview`, `Stable`, `Experimental`, `Deprecated`,
  or `Unsupported`;
- evidence result: `Pass`, `Fail`, `Pending`, `Not applicable`, or
  `Not checked`; and
- workflow result: `Success`, `Warning`, or `Error` only when the named action
  has that result.

Do not use a green dot to mean verified, a lock icon to mean secure, or a check
mark to combine independent evidence layers.

## Diagrams

- Store editable source beside a generated diagram or in a documented source
  directory. Record the pinned generator and exact reproduction command.
- Prefer a small text diagram when it communicates the relationship clearly.
- Give every informative image concise alt text that states its purpose, not a
  list of decorative details.
- Add an adjacent text equivalent describing nodes, order, trust boundaries,
  and meanings that are essential to the page.
- Use labels on lines and regions. Do not encode trust, direction, or state only
  through color, dashed edges, position, or animation.
- Keep reading order meaningful and avoid crossing edges when a sequence or
  table would be clearer.
- Decorative images use empty alt text and must not be the target of an
  essential link.

Example text equivalent:

> The CLI reads the explicitly named ledger inside the ledger root. Local
> validation uses only the embedded schema. A timestamp command crosses a
> separate network boundary only after the operator grants network access.

## Terminal captures

- The primary artifact is a checked text transcript with language `text`.
- A rendered SVG or image is optional and must be generated from the same
  checked scenario, not retyped.
- Include the command, normalized stdout, normalized stderr, exit status, and
  declared created-file set in the source scenario.
- Alt text identifies the command and outcome. Adjacent prose explains any
  warning, truncation, or normalized value.
- Do not imitate a prompt in copyable blocks. Do not capture real usernames,
  home directories, credentials, keys, private ledger values, or host-specific
  paths.
- Animation is unnecessary for essential workflows. If used, provide pause and
  static alternatives and respect reduced-motion preferences.

## Safety callouts

Use the words `Note`, `Important`, or `Warning` through the Markdown callout
syntax in the [documentation style guide](documentation-style.md#callouts-and-safety).
Name the concrete risk and safe action in text. For example:

> [!WARNING]
> Losing the only protected key file makes the sealed forecast impossible to
> reveal. Back up the protected file before sealing.

Place the callout before the risky or irreversible command. Color and icons are
supplementary.

## Badges and product images

- A badge links to the current evidence it summarizes and repeats no status
  that is unavailable in text.
- Do not add popularity, vanity, technology, or redundant badges to README.
- Product images use real checked behavior and show the product version.
- Recreate captures when the command, output, maturity, or safety message
  changes.

## Asset review

Before committing a maintained asset, verify:

1. editable or generator source and reproduction instructions exist;
2. contrast passes for every text and essential graphical pair;
3. status and relationships remain clear in grayscale;
4. alt text and an adequate text equivalent exist;
5. no secret, private path, or unsupported claim is present; and
6. the page remains usable when the asset is missing.

[Development documentation](index.md) · [Documentation index](../index.md)
