#!/usr/bin/env bash

set -euo pipefail

binary=${1:-}
if [[ -z "$binary" || ! -x "$binary" ]]; then
  echo "usage: scripts/dogfood.sh /path/to/forecast-ledger" >&2
  exit 2
fi

work=$(mktemp -d)
cleanup() {
	if [[ "${DOGFOOD_KEEP:-}" == "1" ]]; then
		echo "dogfood files retained at $work" >&2
		return
	fi
  if [[ -n "${work:-}" && "$work" != "/" && -d "$work" ]]; then
    rm -rf -- "$work"
  fi
}
trap cleanup EXIT

ledger="$work/ledger.json"
key="$work/f-two.key"
package="$work/package"

"$binary" --json init \
  --file "$ledger" \
  --ledger-id dogfood-ledger \
  --timezone UTC \
  --forecaster-id dogfood-forecaster \
  --forecaster-name "Dogfood Forecaster" \
  --input internal/adapters/cli/testdata/input/init-public.json >/dev/null

printf '%s\n' '{"title":"Dogfood ledger","description":"Cross-platform command lifecycle"}' |
  "$binary" --json ledger update --file "$ledger" --input - >/dev/null

printf '%s\n' '{"name":"Dogfood platform","kind":"internal"}' |
  "$binary" --json platform add --file "$ledger" --platform dogfood --input - >/dev/null
printf '%s\n' '{"name":"Updated dogfood platform"}' |
  "$binary" --json platform update --file "$ledger" --platform dogfood --input - >/dev/null
"$binary" --plain platform list --file "$ledger" | grep -F 'dogfood' >/dev/null
"$binary" --json platform show --file "$ledger" --platform dogfood | grep -F 'Updated dogfood platform' >/dev/null
"$binary" --json platform remove --file "$ledger" --platform dogfood --yes >/dev/null

printf '%s\n' '{"title":"Will the second event happen?","resolution_criteria":"Resolve from the named source.","created_at":"2026-02-01T00:00:00Z","forecast_window":{"closes_at":"2026-12-31T00:00:00Z"},"expected_resolution_at":"2027-01-02T00:00:00Z","initial_forecast":{"id":"f-second-001","visibility":"public","forecasted_at":"2026-02-01T00:00:00Z","recorded_at":"2026-02-01T00:01:00Z","value":{"kind":"binary","probability_bp":4000}}}' |
  "$binary" --json question add --file "$ledger" --question q-second --type binary --input - >/dev/null

printf '%s\n' '{"forecasted_at":"2026-03-01T00:00:00Z","recorded_at":"2026-03-01T00:01:00Z","value":{"kind":"binary","probability_bp":6000},"rationale":"Public dogfood revision.","supersedes_forecast_id":"f-one"}' |
  "$binary" --json forecast add --file "$ledger" --question q-one --forecast f-public-002 --input - >/dev/null

private_canary='PRIVATE-DOGFOOD-CANARY'
printf '%s\n' "{\"forecasted_at\":\"2026-04-01T00:00:00Z\",\"recorded_at\":\"2026-04-01T00:01:00Z\",\"value\":{\"kind\":\"binary\",\"probability_bp\":6500},\"rationale\":\"$private_canary\",\"key_factors\":[\"private factor\"],\"comment\":\"private comment\",\"supersedes_forecast_id\":\"f-public-002\"}" |
  "$binary" --json forecast seal --file "$ledger" --question q-one --forecast f-sealed-003 --input - --key-file "$key" >/dev/null

sealed_output=$("$binary" --plain forecast show --file "$ledger" --question q-one --forecast f-sealed-003)
if grep -F "$private_canary" <<<"$sealed_output" >/dev/null; then
  echo "sealed forecast output exposed private input" >&2
  exit 1
fi

"$binary" --json forecast reveal --file "$ledger" --question q-one --forecast f-sealed-003 --key-file "$key" --revealed-at 2026-04-02T00:00:00Z --yes >/dev/null
"$binary" --plain forecast list --file "$ledger" --question q-one | grep -F 'f-sealed-003' >/dev/null
"$binary" --plain question show --file "$ledger" --question q-one | grep -F 'Will it happen?' >/dev/null

"$binary" --json target build --file "$ledger" --question q-one --forecast f-one >/dev/null
"$binary" --json target check --file "$ledger" --question q-one --forecast f-one >/dev/null
"$binary" --json timestamp status --file "$ledger" --question q-one --forecast f-one | grep -F 'unanchored' >/dev/null

set +e
"$binary" --json timestamp stamp --file "$ledger" --question q-one --forecast f-one --offline >"$work/offline.out" 2>"$work/offline.err"
offline_exit=$?
set -e
if [[ $offline_exit -ne 8 ]]; then
  echo "offline timestamp stamp returned $offline_exit, want 8" >&2
  exit 1
fi

printf '%s\n' '{"status":"closed","notes":"Closed by dogfood lifecycle."}' |
  "$binary" --json question update --file "$ledger" --question q-one --input - >/dev/null
printf '%s\n' '{"outcome":true,"outcome_known_at":"2027-01-01T00:00:00Z","recorded_at":"2027-01-01T00:01:00Z","sources":[{"title":"Official result","url":"https://example.org/result","retrieved_at":"2027-01-01T00:00:30Z"}]}' |
  "$binary" --json question resolve --file "$ledger" --question q-one --input - --yes >/dev/null

printf '%s\n' '{"reason":"The second event was cancelled.","recorded_at":"2027-01-02T00:01:00Z"}' |
  "$binary" --json question annul --file "$ledger" --question q-second --input - --yes >/dev/null
printf '%s\n' '{"reason":"The cancellation is under review.","recorded_at":"2027-01-02T00:02:00Z"}' |
  "$binary" --json question dispute --file "$ledger" --question q-second --input - --yes >/dev/null

set +e
verify_output=$("$binary" --plain verify --file "$ledger" --offline)
verify_exit=$?
set -e
if [[ $verify_exit -ne 9 ]] || ! grep -F $'overall\tincomplete' <<<"$verify_output" >/dev/null; then
  echo "unexpected layered verification result (exit $verify_exit):" >&2
  echo "$verify_output" >&2
  exit 1
fi
"$binary" --json publish build --file "$ledger" --output "$package" >/dev/null
set +e
package_output=$("$binary" --plain publish verify --file "$package/ledger/ledger.json" --manifest "$package/manifest.json")
package_exit=$?
set -e
if [[ $package_exit -ne 9 ]] || ! grep -F $'overall\tincomplete' <<<"$package_output" >/dev/null; then
  echo "unexpected package verification result (exit $package_exit):" >&2
  echo "$package_output" >&2
  exit 1
fi

if find "$package" -type f -name '*.key' -print -quit | grep -q .; then
  echo "publication package contains a key file" >&2
  exit 1
fi
if find "$package" -type f -name 'f-one.json' -path '*/proofs/targets/*' -print -quit | grep -q .; then
  echo "publication package discovered an adjacent unreferenced target" >&2
  exit 1
fi
if awk 'length($0) > 300 && $0 !~ /"ciphertext"[[:space:]]*:/ { exit 1 }' "$ledger"; then
  :
else
  echo "ledger contains an unreadably long line" >&2
  exit 1
fi

"$binary" mcp serve --ledger-root "main=$work" --read-only </dev/null
"$binary" mcp serve --ledger-root "main=$work" --offline </dev/null

echo "dogfood lifecycle passed"
