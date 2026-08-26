// Package buildinfo exposes reproducible build and compatibility metadata.
package buildinfo

import (
	"runtime"

	ledgerschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

const (
	BinaryName         = "forecast-ledger"
	MCPProtocolVersion = "2026-07-28"
)

var (
	version        = "dev"
	sourceRevision = "unknown"
)

// Schema identifies the exact embedded Forecast Ledger contract.
type Schema struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	SHA256  string `json:"sha256"`
}

// Info is the stable machine-readable version result.
type Info struct {
	Binary           string            `json:"binary"`
	Version          string            `json:"version"`
	SourceRevision   string            `json:"source_revision"`
	GoVersion        string            `json:"go_version"`
	Schema           Schema            `json:"schema"`
	MCPProtocol      string            `json:"mcp_protocol"`
	TimestampProfile ots.PublicProfile `json:"timestamp_profile"`
}

// Current returns metadata for the running binary.
func Current() Info {
	return Info{
		Binary:         BinaryName,
		Version:        version,
		SourceRevision: sourceRevision,
		GoVersion:      runtime.Version(),
		Schema: Schema{
			Version: ledgerschema.Version,
			Commit:  ledgerschema.Commit,
			SHA256:  ledgerschema.SchemaSHA256,
		},
		MCPProtocol:      MCPProtocolVersion,
		TimestampProfile: ots.Profile(),
	}
}
