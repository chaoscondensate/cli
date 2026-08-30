#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
canary_tmp=$(mktemp -d "${TMPDIR:-/tmp}/forecast-ledger-freetsa-canary.XXXXXX")
trap 'rm -rf "$canary_tmp"' EXIT HUP INT TERM

target="$repository_root/internal/timestamp/rfc3161/testdata/freetsa/target.txt"
bundle="$repository_root/internal/timestamp/rfc3161/providers/freetsa/ca.pem"
request="$canary_tmp/request.tsq"
response="$canary_tmp/response.tsr"

openssl ts -query -sha256 -cert -data "$target" -out "$request"
curl --fail --silent --show-error \
  --connect-timeout 10 \
  --max-time 30 \
  --max-redirs 0 \
  --header 'Content-Type: application/timestamp-query' \
  --header 'Accept: application/timestamp-reply' \
  --data-binary "@$request" \
  --output "$response" \
  https://freetsa.org/tsr
openssl ts -verify -queryfile "$request" -in "$response" -CAfile "$bundle"

printf 'provider=freetsa\n'
printf 'endpoint=https://freetsa.org/tsr\n'
printf 'request_sha256='
openssl dgst -sha256 -r "$request" | awk '{print $1}'
printf 'response_sha256='
openssl dgst -sha256 -r "$response" | awk '{print $1}'
openssl ts -reply -in "$response" -text | awk '
  /Status:/ || /Policy OID:/ || /Hash Algorithm:/ || /Serial number:/ || /Time stamp:/ {print}
'
