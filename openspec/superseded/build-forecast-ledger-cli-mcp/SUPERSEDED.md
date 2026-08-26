# Superseded change record

`build-forecast-ledger-cli-mcp` was retired on 2026-08-26. Its complete
replacement is `complete-forecast-ledger-command-surface`. Keeping this record
outside `openspec/changes` prevents OpenSpec from selecting, applying, syncing,
or archiving the conflicting delta separately.

The mapping below preserves evidence from all 31 completed historical tasks.
It does not mark a replacement task complete: each replacement checkbox still
requires all behavior and tests in the new, stricter contract.

| Historical task | Replacement requirement or evidence use |
| --- | --- |
| `1.1` | Command/package naming evidence for replacement tasks `14.1`, `15.6`, and `15.8`. |
| `1.2` | Go module, entrypoint, and package foundation used by `1.1` and all implementation slices. |
| `1.3` | Pinned dependency evidence reused by `13.1`, `15.6`, and `15.8`. |
| `1.4` | Exact schema, fixtures, attribution, and seal-vector foundation reused by `1.9`, `7.2`, `8.2`, and `15.5`. |
| `1.5` | Build/version metadata foundation extended by `9.4`, `13.1`, and `14.2`. |
| `1.6` | Contributor context foundation updated by replacement tasks `14.6` and `14.7`. |
| `2.1` | Typed v1 ledger model foundation used by `1.2`, `1.4`, and authoring builders. |
| `2.2` | Bounded JSON parser evidence reused by replacement tasks `1.3` and `15.3`. |
| `2.3` | Bounded YAML/source-tree evidence reused by `1.3`, `2.1`, and `15.3`. |
| `2.4` | Embedded offline schema validator foundation reused by `1.9`, `11.2`, and `15.4`. |
| `2.5` | Semantic validation foundation extended by `1.4`, `5.1`, `6.1`, and lifecycle tasks. |
| `2.6` | Source-preserving patch foundation extended and reverified by `2.1`. |
| `2.7` | Upstream conformance fixture evidence reused by `15.4` and `15.8`. |
| `2.8` | Parser fuzz/resource-limit foundation extended by `1.3` and `15.3`. |
| `3.1` | Existing public error/exit foundation extended by replacement task `1.7`. |
| `3.2` | Existing path resolver foundation extended by replacement task `2.5`. |
| `3.3` | Existing lock foundation extended by replacement task `2.2`. |
| `3.4` | Existing ledger transaction foundation extended by replacement tasks `2.3` and `2.4`. |
| `3.5` | Existing storage fault/concurrency evidence extended by `2.7` and `15.2`. |
| `4.1` | Hidden urfave preview tree is corrected and gated by `14.1`, `14.2`, and per-command wiring tasks. |
| `4.2` | Leaf selector foundation is corrected and reverified by `14.2` and `15.1`. |
| `4.3` | Help/completion foundation is regenerated and completed by `14.2`, `14.3`, and `14.8`. |
| `4.4` | Presentation foundation is reused by `1.1`, command JSON gates, and `15.1`. |
| `4.5` | Runtime confirmation/cancellation foundation is extended by replacement task `1.6`. |
| `5.2` | Working validate/status implementation remains foundation for `11.2`, `13.4`, and CLI parity tests. |
| `11.1` | README/workflow handoff is owned by replacement tasks `14.4`, `14.5`, and the documentation change. |
| `11.2` | CLI/JSON/exit reference handoff is owned by `14.2`, `14.3`, and `14.8`. |
| `11.3` | Security/evidence-limit handoff is owned by `11.7`, `14.4`, and `14.5`. |
| `11.4` | MCP setup/safety handoff is owned by `13.1`–`13.10` and `14.4`. |
| `11.5` | Schema/conformance handoff is owned by `15.4`, `15.5`, and `15.8`. |
| `11.6` | Checked example/tutorial handoff is owned by `14.2`, `14.4`, and `14.8`. |

Only `complete-forecast-ledger-command-surface` may supply the replacement
delta specs. When it is complete, archive that change once; do not move this
record back under `openspec/changes`.
