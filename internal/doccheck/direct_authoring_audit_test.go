package doccheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	genericCLIInputPattern = regexp.MustCompile("(^|[^[:alnum:]-])--input([[:space:]=\\\"'`]|$)")
	genericMCPInputPattern = regexp.MustCompile(`["'](input|input_file)["'][[:space:]]*:`)
)

// These are the only active-surface exceptions. Each is a negative assertion
// proving that a removed interface is rejected. Archived and explicitly
// superseded OpenSpec directories are outside this audit because they preserve
// historical contracts and are never current runtime or user guidance.
var directAuthoringNegativeTests = map[string]bool{
	"internal/adapters/cli/authoring_flags_test.go":    true,
	"internal/adapters/cli/diagnostics_test.go":        true,
	"internal/adapters/mcp/server_test.go":             true,
	"internal/doccheck/direct_authoring_audit_test.go": true,
}

func TestActiveSurfaceHasNoGenericPublicAuthoringTransport(t *testing.T) {
	root := findRepositoryRoot(t)
	entries := []string{"README.md", "docs", "scripts", "tools", "internal"}
	for _, entry := range entries {
		start := filepath.Join(root, entry)
		err := filepath.WalkDir(start, func(path string, item os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if item.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if directAuthoringNegativeTests[relative] || strings.Contains(relative, "/schema/vendor/forecast-ledger/v1.2.0/") || strings.Contains(relative, "/schema/testdata/forecast-ledger/v1.2.0/") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if genericCLIInputPattern.Find(content) != nil {
				t.Errorf("%s contains removed generic CLI authoring input", relative)
			}
			if genericMCPInputPattern.Find(content) != nil {
				t.Errorf("%s contains removed generic MCP authoring wrapper", relative)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("audit %s: %v", entry, err)
		}
	}
}
