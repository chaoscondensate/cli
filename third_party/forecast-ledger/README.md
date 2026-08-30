# Forecast Ledger v1.3.0 attribution

This directory and `internal/schema` contain exact files copied from the
[Chaos Condensate Forecast Ledger schema](https://github.com/chaoscondensate/schema)
at commit `32218f682b3a650f41153e98817473bf429973a7`.

Upstream version: `1.3.0`

Annotated tag object:
`d3d1f06a7f27501b1419eaf78fc4a48e51de9ee3`

Release archive SHA-256:
`3b6b9f274a67d2714edaa308f9aad51b218dbf24ed95de1a1340292ad1df1f2a`

Published `SHA256SUMS` asset SHA-256:
`6042508976246ddc62974ad3054dca9885525024d4bb543572b75b23c60ac284`

| Upstream file | Local use | SHA-256 |
| --- | --- | --- |
| `schema/forecast-ledger.schema.json` | Embedded runtime contract | `f673e4f3fc867a83d8c42a6992c6020ea28359a293580c8c742fe9dcdcd8d2c1` |
| `tests/vectors/forecast-seal-v1.json` | Seal/JCS conformance | `59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5` |
| `examples/valid/empty-ledger.json` | Empty-ledger fixture | `d7718493a5dcdb4d6af8ce398e103788adc9c43739916eb82475af0ab0617426` |
| `examples/valid/question-without-forecasts.yaml` | Backlog-question fixture | `7b53e9db6cdd669562b94136ed5a7a882d27d8b43bfc752de6e37894a9aeee5a` |
| `examples/valid/individual-ledger.json` | JSON validation fixture | `77cb761c714e36a31347e1d8630c99a0c73cf2ae3425438faef5a060e436828c` |
| `examples/valid/team-ledger.yaml` | YAML/reveal fixture | `ed00a6bfc47711727bc9d40c1c40fb0e5a0efa08a94cf6d1b8912ecfeca65386` |
| `tests/invalid-cases.json` | Semantic negative cases | `d0e4b8036bf9119bb402687b753740ea8dcbd2d8dbef49b7673b7745add2cfee` |
| `LICENSE` | Upstream license notice | `7084b3fb14e3a306691af23e58ab0ccfa336b202853740f5e1ea0ebab39cacf2` |

The copied files are distributed under the upstream MIT license in `LICENSE`.
Local Go code is an independent compatible implementation. Reference tools and
normative documentation are not copied; reviews pin their exact-commit URLs:

- [`tools/validate.py`](https://github.com/chaoscondensate/schema/blob/32218f682b3a650f41153e98817473bf429973a7/tools/validate.py)
- [`tools/forecast_crypto.py`](https://github.com/chaoscondensate/schema/blob/32218f682b3a650f41153e98817473bf429973a7/tools/forecast_crypto.py)
- [`tools/build_targets.py`](https://github.com/chaoscondensate/schema/blob/32218f682b3a650f41153e98817473bf429973a7/tools/build_targets.py)
- [`docs/cryptographic-verification.md`](https://github.com/chaoscondensate/schema/blob/32218f682b3a650f41153e98817473bf429973a7/docs/cryptographic-verification.md)
- [`docs/forecast-verification-workflows.md`](https://github.com/chaoscondensate/schema/blob/32218f682b3a650f41153e98817473bf429973a7/docs/forecast-verification-workflows.md)

Do not update these files from a floating tag. Review and record a new exact
commit and every digest before changing the embedded contract.
