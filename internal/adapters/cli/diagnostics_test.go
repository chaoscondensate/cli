package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovedGenericInputFlagIsRejectedWithoutReadingOrMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	before := fixtureBytes(t, "individual-ledger.json")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	const canary = "PRIVATE-DIAGNOSTIC-CANARY"
	code, stdout, stderr := runCLIWithStdin(canary, "forecast-ledger", "platform", "add", "--file", path, "--platform", "example", "--input", "-")
	if code != 2 || stdout != "" || strings.Contains(stderr, canary) || !strings.Contains(stderr, "flag provided but not defined") {
		t.Fatalf("removed flag result code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("removed generic flag changed the ledger")
	}
}

func TestSemanticDirectFlagDiagnosticUsesSafePointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "platform", "add", "--file", path, "--platform", "example", "--name", "Example", "--kind", "informal", "--url", "not a url")
	if code != 3 || stdout != "" || !strings.Contains(stderr, "/platform/url") || !strings.Contains(stderr, "semantic.invalid_field") {
		t.Fatalf("semantic diagnostic code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.Contains(stderr, "line 0") || strings.Contains(stderr, "column 0") {
		t.Fatalf("semantic diagnostic fabricated a position: %s", stderr)
	}
}
