#!/bin/sh
set -eu

fixture_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
RFC3161_FIXTURE_TMP=$(mktemp -d "${TMPDIR:-/tmp}/forecast-ledger-rfc3161.XXXXXX")
export RFC3161_FIXTURE_TMP
trap 'rm -rf "$RFC3161_FIXTURE_TMP"' EXIT HUP INT TERM

cp "$fixture_dir/target.txt" "$RFC3161_FIXTURE_TMP/target.txt"
printf '%s\n' '01' >"$RFC3161_FIXTURE_TMP/serial"

openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 36500 \
  -subj '/CN=Forecast Ledger RFC 3161 Test Root' \
  -keyout "$RFC3161_FIXTURE_TMP/root-key.pem" \
  -out "$RFC3161_FIXTURE_TMP/root.pem"
openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 36500 \
  -subj '/CN=Unrelated RFC 3161 Test Root' \
  -keyout "$RFC3161_FIXTURE_TMP/wrong-root-key.pem" \
  -out "$RFC3161_FIXTURE_TMP/wrong-root.pem"
openssl req -newkey rsa:2048 -nodes -sha256 \
  -subj '/CN=Forecast Ledger RFC 3161 Test TSA' \
  -keyout "$RFC3161_FIXTURE_TMP/tsa-key.pem" \
  -out "$RFC3161_FIXTURE_TMP/tsa.csr"
openssl x509 -req -sha256 -days 36500 -set_serial 2 \
  -in "$RFC3161_FIXTURE_TMP/tsa.csr" \
  -CA "$RFC3161_FIXTURE_TMP/root.pem" \
  -CAkey "$RFC3161_FIXTURE_TMP/root-key.pem" \
  -extfile "$fixture_dir/tsa-ext.cnf" \
  -out "$RFC3161_FIXTURE_TMP/tsa.pem"
openssl ts -query -sha256 -cert \
  -data "$RFC3161_FIXTURE_TMP/target.txt" \
  -out "$RFC3161_FIXTURE_TMP/request.tsq"
openssl ts -reply \
  -config "$fixture_dir/tsa.cnf" \
  -queryfile "$RFC3161_FIXTURE_TMP/request.tsq" \
  -out "$RFC3161_FIXTURE_TMP/response.tsr"
openssl ts -verify \
  -queryfile "$RFC3161_FIXTURE_TMP/request.tsq" \
  -in "$RFC3161_FIXTURE_TMP/response.tsr" \
  -CAfile "$RFC3161_FIXTURE_TMP/root.pem"

cp "$RFC3161_FIXTURE_TMP/target.txt" "$fixture_dir/target.txt"
cp "$RFC3161_FIXTURE_TMP/request.tsq" "$fixture_dir/request.tsq"
cp "$RFC3161_FIXTURE_TMP/response.tsr" "$fixture_dir/response.tsr"
cp "$RFC3161_FIXTURE_TMP/root.pem" "$fixture_dir/root.pem"
cp "$RFC3161_FIXTURE_TMP/wrong-root.pem" "$fixture_dir/wrong-root.pem"
cp "$RFC3161_FIXTURE_TMP/tsa.pem" "$fixture_dir/tsa.pem"
