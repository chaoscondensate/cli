# Manage platform records

<!-- doc-metadata
coverage: unreleased-main
reviewed: 2026-08-26
owner: interface
generated: false
security-critical: false
prerequisites: ../getting-started/create-ledger.md
next: ../reference/index.md
-->

Platform records describe external or local places associated with questions.
They do not publish data or contact a service. Every command names the ledger
with `--file`.

Create `platform.yaml`:

```yaml
name: Metaculus
kind: scoring_platform
url: https://www.metaculus.com/
account:
  username: example-user
```

Add it under a stable ID:

```sh
forecast-ledger platform add \
  --file ledger.yaml \
  --platform metaculus \
  --input platform.yaml
```

The supported kinds are `scoring_platform`, `prediction_market`,
`self_hosted`, `internal`, and `informal`. URLs must be absolute. Account data
must contain at least one of username, user ID, or profile URL.

Update only named fields with a closed JSON or YAML patch:

```yaml
name: Metaculus forecasting
url: null
account:
  profile_url: https://www.metaculus.com/accounts/profile/example-user/
```

```sh
forecast-ledger platform update \
  --file ledger.yaml \
  --platform metaculus \
  --input platform-patch.yaml
```

Omitted fields stay unchanged. `null` removes an optional URL, account, or
account field. It cannot remove the required name or kind.

List and inspect records without mutation:

```sh
forecast-ledger platform list --file ledger.yaml
forecast-ledger platform show --file ledger.yaml --platform metaculus
```

Both commands accept a ledger on stdin with `--file -`. List order and
referencing question IDs are stable and sorted. JSON output includes the ledger
ID, exact platform record, and reference counts.

Removal is allowed only when no question references the platform and requires
confirmation:

```sh
forecast-ledger platform remove \
  --file ledger.yaml \
  --platform old-platform \
  --yes
```

Without `--yes`, an interactive terminal asks first. Use `--dry-run` on add,
update, or remove to validate and show the planned change without writing.

[How-to index](index.md) · [Documentation index](../index.md)
