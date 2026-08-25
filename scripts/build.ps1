$ErrorActionPreference = "Stop"

$output = if ($env:OUTPUT) { $env:OUTPUT } else { "dist/forecast-ledger.exe" }
$outputDirectory = Split-Path -Parent $output
$version = if ($env:VERSION) { $env:VERSION } else { "dev" }
$sourceRevision = if ($env:SOURCE_REVISION) { $env:SOURCE_REVISION } else { "unknown" }
$ldflags = "-X github.com/chaoscondensate/cli/internal/buildinfo.version=$version -X github.com/chaoscondensate/cli/internal/buildinfo.sourceRevision=$sourceRevision"

if ($outputDirectory) {
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
}

go build -mod=readonly -trimpath -ldflags $ldflags -o $output ./cmd/forecast-ledger
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
