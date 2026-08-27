package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationInputDiagnosticsStayStructuredAndRedacted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	const canary = "PRIVATE-DIAGNOSTIC-CANARY"
	input := "name: Example\nkind: informal\nextra: " + canary + "\n"
	base := []string{"forecast-ledger", "platform", "add", "--file", path, "--platform", "example", "--input", "-"}

	code, stdout, stderr := runCLIWithStdin(input, base...)
	if code != 3 || stdout != "" || strings.Contains(stderr, canary) || !strings.Contains(stderr, "/extra (line 3, column 8)") || !strings.Contains(stderr, "schema.additionalProperties") || !strings.Contains(stderr, "unknown field extra") {
		t.Fatalf("human diagnostic code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	jsonArgs := append([]string{"forecast-ledger", "--json"}, base[1:]...)
	code, stdout, stderr = runCLIWithStdin(input, jsonArgs...)
	if code != 3 || stdout != "" || strings.Contains(stderr, canary) {
		t.Fatalf("JSON diagnostic code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope struct {
		Details struct {
			Issues []struct {
				Code     string `json:"code"`
				Location struct {
					Pointer string `json:"pointer"`
					Start   struct {
						Line   int `json:"line"`
						Column int `json:"column"`
					} `json:"start"`
				} `json:"location"`
			} `json:"issues"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Details.Issues) != 1 || envelope.Details.Issues[0].Code != "schema.additionalProperties" || envelope.Details.Issues[0].Location.Pointer != "/extra" || envelope.Details.Issues[0].Location.Start.Line != 3 {
		t.Fatalf("JSON issues = %#v", envelope.Details.Issues)
	}
}

func TestSemanticInputDiagnosticUsesSafePointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := "name: '   '\nkind: informal\n"
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "platform", "add", "--file", path, "--platform", "example", "--input", "-")
	if code != 3 || stdout != "" || !strings.Contains(stderr, "/platform/name") || !strings.Contains(stderr, "semantic.invalid_field") {
		t.Fatalf("semantic diagnostic code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.Contains(stderr, "line 0") || strings.Contains(stderr, "column 0") {
		t.Fatalf("semantic diagnostic fabricated a position: %s", stderr)
	}
	code, stdout, stderr = runCLIWithStdin(input, "forecast-ledger", "--json", "platform", "add", "--file", path, "--platform", "example", "--input", "-")
	if code != 3 || stdout != "" || strings.Contains(stderr, `"start"`) || strings.Contains(stderr, `"line":0`) {
		t.Fatalf("unknown semantic span was not omitted: %s", stderr)
	}
}
