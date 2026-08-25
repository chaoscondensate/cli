#!/bin/sh

set -eu

output=${OUTPUT:-dist/forecast-ledger}
output_dir=${output%/*}
version=${VERSION:-dev}
source_revision=${SOURCE_REVISION:-unknown}
ldflags="-X github.com/chaoscondensate/cli/internal/buildinfo.version=$version -X github.com/chaoscondensate/cli/internal/buildinfo.sourceRevision=$source_revision"

if [ "$output_dir" != "$output" ]; then
	mkdir -p "$output_dir"
fi

exec go build -mod=readonly -trimpath -ldflags "$ldflags" -o "$output" ./cmd/forecast-ledger
