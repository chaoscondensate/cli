package releasecheck

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestOlderCommandSurfaceChangeCannotBeSyncedOrArchivedSeparately(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	name := "build-forecast-ledger-cli-mcp"
	if _, err := os.Stat(filepath.Join(root, "openspec", "changes", name)); !os.IsNotExist(err) {
		t.Fatalf("superseded change is still active: %v", err)
	}
	retained := filepath.Join(root, "openspec", "superseded", name)
	config, err := os.ReadFile(filepath.Join(retained, ".openspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{
		"state: superseded", "superseded_by: complete-forecast-ledger-command-surface",
		"sync: forbidden", "archive: forbidden",
	} {
		if !strings.Contains(string(config), contract) {
			t.Fatalf("superseded config is missing %q", contract)
		}
	}
	mapping, err := os.ReadFile(filepath.Join(retained, "SUPERSEDED.md"))
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`(?m)^\| \x60[0-9]+\.[0-9]+\x60 \|`).FindAll(mapping, -1)
	if len(matches) != 31 {
		t.Fatalf("completed foundation mappings = %d, want 31", len(matches))
	}
}
