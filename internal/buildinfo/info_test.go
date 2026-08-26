package buildinfo

import (
	"strings"
	"testing"

	ledgerschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestCurrentIncludesCompatibilityPins(t *testing.T) {
	t.Parallel()

	info := Current()
	if info.Binary != BinaryName || info.Version == "" || info.SourceRevision == "" {
		t.Fatalf("incomplete binary metadata: %#v", info)
	}
	if !strings.HasPrefix(info.GoVersion, "go1.") {
		t.Fatalf("unexpected Go version %q", info.GoVersion)
	}
	if info.Schema.Version != ledgerschema.Version ||
		info.Schema.Commit != ledgerschema.Commit ||
		info.Schema.SHA256 != ledgerschema.SchemaSHA256 {
		t.Fatalf("unexpected schema metadata: %#v", info.Schema)
	}
	if info.MCPProtocol != MCPProtocolVersion {
		t.Fatalf("unexpected MCP protocol %q", info.MCPProtocol)
	}
	if info.TimestampProfile.ID != "opentimestamps-public-v1" || len(info.TimestampProfile.Calendars) != 4 || len(info.TimestampProfile.BitcoinSources) != 2 {
		t.Fatalf("unexpected timestamp profile: %#v", info.TimestampProfile)
	}
}
