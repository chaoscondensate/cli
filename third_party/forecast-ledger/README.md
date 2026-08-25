# Forecast Ledger v1.0.0 attribution

This directory and `internal/schema` contain exact files copied from the
[Chaos Condensate Forecast Ledger schema](https://github.com/chaoscondensate/schema)
at commit `e409463d702888fefd253b32f21b9b2f864aabed`.

Upstream version: `1.0.0`

Release archive SHA-256:
`a3d6afcf8a3cd9b9e9a650ebac684cbe2f155a81db309797d77694b5f4b9bbda`

| Upstream file | Local use | SHA-256 |
| --- | --- | --- |
| `schema/forecast-ledger.schema.json` | Embedded runtime contract | `e63bdd01f0241aa4d94d5ccc45e84bcea70a6a7fd46ab77cff4802b3f8b8fc65` |
| `tests/vectors/forecast-seal-v1.json` | Seal/JCS conformance | `59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5` |
| `examples/valid/individual-ledger.json` | JSON validation fixture | `b05d3ad403ba85d962e1f8d1e6219b789ff763b81928f120b90926603b67dd68` |
| `examples/valid/team-ledger.yaml` | YAML/reveal fixture | `fc42a7b70c5cef89e6524cf45f8c7be07bedaf1c1368eed761739d835200e4c1` |
| `tests/invalid-cases.json` | Semantic negative cases | `a7e0275d216f8f81285bfcb2c37e4095395fa2f7bc95b3ebdce51f55d7c29d59` |
| `LICENSE` | Upstream license notice | `7084b3fb14e3a306691af23e58ab0ccfa336b202853740f5e1ea0ebab39cacf2` |

The copied files are distributed under the upstream MIT license in `LICENSE`.
Local Go code is an independent compatible implementation. Reference tools and
normative documentation are not copied; reviews pin their exact-commit URLs:

- [`tools/validate.py`](https://github.com/chaoscondensate/schema/blob/e409463d702888fefd253b32f21b9b2f864aabed/tools/validate.py)
- [`tools/forecast_crypto.py`](https://github.com/chaoscondensate/schema/blob/e409463d702888fefd253b32f21b9b2f864aabed/tools/forecast_crypto.py)
- [`tools/build_targets.py`](https://github.com/chaoscondensate/schema/blob/e409463d702888fefd253b32f21b9b2f864aabed/tools/build_targets.py)
- [`docs/cryptographic-verification.md`](https://github.com/chaoscondensate/schema/blob/e409463d702888fefd253b32f21b9b2f864aabed/docs/cryptographic-verification.md)
- [`docs/forecast-verification-workflows.md`](https://github.com/chaoscondensate/schema/blob/e409463d702888fefd253b32f21b9b2f864aabed/docs/forecast-verification-workflows.md)

Do not update these files from the floating `v1.0.0` tag. Review and record a
new exact commit and every digest before changing the embedded contract.
