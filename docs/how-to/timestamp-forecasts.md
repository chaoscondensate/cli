# Timestamp a forecast

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-29
owner: security
generated: false
security-critical: true
prerequisites: build-targets.md
next: verify-evidence.md
-->

OpenTimestamps support is experimental. It binds the exact canonical target
bytes to a Bitcoin attestation; it does not prove authorship, completeness,
truth, or the exact self-reported forecast time.

Every timestamp command selects an existing forecast. A question with no
forecasts is valid, but selecting a forecast under it returns `not_found`
before entropy, artifact creation, mutation, or network access.

## Create a pending receipt

```sh
forecast-ledger timestamp stamp \
  --file ledger.yaml \
  --question q-launch \
  --forecast f-launch-002
```

The built-in `opentimestamps-public-v1` profile generates a fresh 16-byte nonce,
appends it to the target digest, hashes again, and submits only that blinded
commitment. It contacts four fixed HTTPS calendar pools concurrently and needs
two accepted branches. The target and receipt are written durably before the
ledger enters `pending`. Retrying the same operation reuses matching durable
artifacts instead of creating another implicit submission.

Use `--dry-run` to validate without entropy, network, or writes. `--offline`
fails before effects because stamping needs calendar responses.

## Check and upgrade locally retained evidence

```sh
forecast-ledger timestamp status --file ledger.yaml --question q-launch --forecast f-launch-002
forecast-ledger timestamp upgrade --file ledger.yaml --question q-launch --forecast f-launch-002
```

Status is local-only and reports `unanchored`, `pending`,
`confirmed_unverified`, `verified`, `failed`, or `inconsistent`. Upgrade contacts
only accepted built-in calendar identities and replaces a receipt only when the
new proof is a semantic superset. A not-yet-ready receipt remains pending.

For a one-invocation liveness escape hatch, CLI stamp and upgrade accept
repeatable `--calendar https://...` with `--calendar-min-success N`. Custom
calendars replace the built-in set; they never extend it. Only public HTTPS
origins are accepted. Custom mode has a caller-selected trust boundary and is
not available through MCP.

## Verify Bitcoin timing

```sh
forecast-ledger timestamp verify --file ledger.yaml --question q-launch --forecast f-launch-002
```

The default verifier queries both mempool.space and Blockstream for each needed
height and requires their block hash and header observations to agree. It then
checks the header hash, proof of work, Merkle root, and OTS attestation locally.
One invocation is limited to 32 unique heights, 128 HTTP requests, and four
concurrent requests. Repeated heights are deduplicated.

Observation acquisition and proof comparison are separate outcomes. If a
public source or Bitcoin Core is unavailable, verification returns a structured
`not_checked` layer with `timing.source_unavailable`, safe source IDs, the
network profile, and the bounded request summary. It leaves the ledger
unchanged and uses `network`/exit 8. Malformed, disagreeing, or budget-limited
observations are also incomplete, not proof mismatches. Raw endpoints,
credentials, response bodies, and underlying errors are not returned.

`timing.bitcoin_mismatch` is reserved for a complete accepted Bitcoin
observation that does not match the receipt. It uses `verification`/exit 6.
JSON, human, and plain CLI modes write these expected verification reports to
stdout; stderr remains for diagnostics rather than duplicating the report.

A proof can be cryptographically valid but too late for the forecast claim. If
the conservative Bitcoin block time is not before the recorded
`outcome_known_at`, `timestamp verify` keeps the valid proof and reports
`timestamp.valid_but_too_late`. Layered verification then fails the existence
timing layer with `timing.not_before_outcome`; offline verification reaches the
same conclusion from the stored checked bound.

After successful verification, `forecast show` and layered `verify --offline`
display the retained receipt state, Bitcoin block height, conservative
`anchored_before`, and `verified_at`. These views open no network connection.
They label the evidence as stored and disclose that schema v1 does not retain
the identity of the Bitcoin source used by the earlier check.

An advanced CLI user may replace the two public observers with an independently
operated Bitcoin Core RPC endpoint:

```sh
forecast-ledger timestamp verify \
  --file ledger.yaml --question q-launch --forecast f-launch-002 \
  --bitcoin-core http://127.0.0.1:8332 \
  --bitcoin-auth-file bitcoin-core-auth.json
```

The protected auth file is closed JSON containing `username` and `password`.
It must not be committed or published. MCP accepts no endpoint or credential
input.

## Trust and privacy

Calendar services learn request timing and blinded commitments. Public Bitcoin
services learn requested block heights and therefore an approximate timestamp
period. Agreement reduces reliance on one public API but still trusts both
services for canonical-chain selection and makes availability depend on both.
Loss of that availability prevents a fresh conclusion; it does not establish
that retained proof bytes are wrong.

[Verify evidence](verify-evidence.md) · [Security](../security/index.md)
