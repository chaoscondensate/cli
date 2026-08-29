# Forecast Ledger v1.1.0 attribution

This directory and `internal/schema` contain exact files copied from the
[Chaos Condensate Forecast Ledger schema](https://github.com/chaoscondensate/schema)
at commit `c04c72a178c15cd6cbbdd2e8a7b743d58872a94a`.

Upstream version: `1.1.0`

Release archive SHA-256:
`edb2e307a7ce55984d17306556f0538f49a3a2a9fa66c9bfec973c90f0cb88dd`

| Upstream file | Local use | SHA-256 |
| --- | --- | --- |
| `schema/forecast-ledger.schema.json` | Embedded runtime contract | `c478f0f568c0c746c343a308d0fcb53815f4c8b91b4666f8f784913ad9132d15` |
| `tests/vectors/forecast-seal-v1.json` | Seal/JCS conformance | `59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5` |
| `examples/valid/empty-ledger.json` | Empty-ledger fixture | `e31ffdf26a63742871686c4cbf6a62ed26dd7289e33d644a315dedba1ddc4a1f` |
| `examples/valid/question-without-forecasts.yaml` | Backlog-question fixture | `f4786c542a11d1bd411f03e6b3246fa3972e9ba1f90ab680262679d82b1d53a5` |
| `examples/valid/individual-ledger.json` | JSON validation fixture | `1ce53fef071e017bf629c93f4b6304316321933d1b2b47ce95e89c4dde8a35ed` |
| `examples/valid/team-ledger.yaml` | YAML/reveal fixture | `68179c3b38ea7a54f0c7d3562c56e1890a975d10d57d7e2dc3c5d880cb04b6db` |
| `tests/invalid-cases.json` | Semantic negative cases | `a7e0275d216f8f81285bfcb2c37e4095395fa2f7bc95b3ebdce51f55d7c29d59` |
| `LICENSE` | Upstream license notice | `7084b3fb14e3a306691af23e58ab0ccfa336b202853740f5e1ea0ebab39cacf2` |

The copied files are distributed under the upstream MIT license in `LICENSE`.
Local Go code is an independent compatible implementation. Reference tools and
normative documentation are not copied; reviews pin their exact-commit URLs:

- [`tools/validate.py`](https://github.com/chaoscondensate/schema/blob/c04c72a178c15cd6cbbdd2e8a7b743d58872a94a/tools/validate.py)
- [`tools/forecast_crypto.py`](https://github.com/chaoscondensate/schema/blob/c04c72a178c15cd6cbbdd2e8a7b743d58872a94a/tools/forecast_crypto.py)
- [`tools/build_targets.py`](https://github.com/chaoscondensate/schema/blob/c04c72a178c15cd6cbbdd2e8a7b743d58872a94a/tools/build_targets.py)
- [`docs/cryptographic-verification.md`](https://github.com/chaoscondensate/schema/blob/c04c72a178c15cd6cbbdd2e8a7b743d58872a94a/docs/cryptographic-verification.md)
- [`docs/forecast-verification-workflows.md`](https://github.com/chaoscondensate/schema/blob/c04c72a178c15cd6cbbdd2e8a7b743d58872a94a/docs/forecast-verification-workflows.md)

Do not update these files from a floating tag. Review and record a new exact
commit and every digest before changing the embedded contract.
