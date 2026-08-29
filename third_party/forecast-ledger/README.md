# Forecast Ledger v1.2.0 attribution

This directory and `internal/schema` contain exact files copied from the
[Chaos Condensate Forecast Ledger schema](https://github.com/chaoscondensate/schema)
at commit `6c2fe3df99223945b8d1613a03f95796b3c7d1e2`.

Upstream version: `1.2.0`

Release archive SHA-256:
`5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e`

| Upstream file | Local use | SHA-256 |
| --- | --- | --- |
| `schema/forecast-ledger.schema.json` | Embedded runtime contract | `d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8` |
| `tests/vectors/forecast-seal-v1.json` | Seal/JCS conformance | `59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5` |
| `examples/valid/empty-ledger.json` | Empty-ledger fixture | `80d0542b2429d50531c7cd43969799311688630840d88195156a8a52505ab710` |
| `examples/valid/question-without-forecasts.yaml` | Backlog-question fixture | `065d8acde34c2918bccc289a06f267ec0db79ee4c379f603458e8cbb7f79f7dd` |
| `examples/valid/individual-ledger.json` | JSON validation fixture | `54a22c6d154d864ed0dee85ee2f2ba7e985354e722690127c3606c8bfb582fd4` |
| `examples/valid/team-ledger.yaml` | YAML/reveal fixture | `b4d739d8f0730eeea4c147f44cd622f497e5be59a682b85dc44ffb6c6c28a5b9` |
| `tests/invalid-cases.json` | Semantic negative cases | `5999288b3eb0bfd4fc99bd8a3bdc2f6520d245bac41845976e8d09aa922c758f` |
| `LICENSE` | Upstream license notice | `7084b3fb14e3a306691af23e58ab0ccfa336b202853740f5e1ea0ebab39cacf2` |

The copied files are distributed under the upstream MIT license in `LICENSE`.
Local Go code is an independent compatible implementation. Reference tools and
normative documentation are not copied; reviews pin their exact-commit URLs:

- [`tools/validate.py`](https://github.com/chaoscondensate/schema/blob/6c2fe3df99223945b8d1613a03f95796b3c7d1e2/tools/validate.py)
- [`tools/forecast_crypto.py`](https://github.com/chaoscondensate/schema/blob/6c2fe3df99223945b8d1613a03f95796b3c7d1e2/tools/forecast_crypto.py)
- [`tools/build_targets.py`](https://github.com/chaoscondensate/schema/blob/6c2fe3df99223945b8d1613a03f95796b3c7d1e2/tools/build_targets.py)
- [`docs/cryptographic-verification.md`](https://github.com/chaoscondensate/schema/blob/6c2fe3df99223945b8d1613a03f95796b3c7d1e2/docs/cryptographic-verification.md)
- [`docs/forecast-verification-workflows.md`](https://github.com/chaoscondensate/schema/blob/6c2fe3df99223945b8d1613a03f95796b3c7d1e2/docs/forecast-verification-workflows.md)

Do not update these files from a floating tag. Review and record a new exact
commit and every digest before changing the embedded contract.
