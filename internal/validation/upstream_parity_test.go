package validation

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestUpstreamPythonValidatorParity runs the exact pinned upstream fixture
// harness when its checkout and Python test dependencies are available. The Go
// side of the same corpus is mandatory in TestAllPinnedInvalidCasesAreRejected.
func TestUpstreamPythonValidatorParity(t *testing.T) {
	upstreamRoot := os.Getenv("FORECAST_LEDGER_UPSTREAM_ROOT")
	if upstreamRoot == "" {
		t.Skip("set FORECAST_LEDGER_UPSTREAM_ROOT to the pinned schema checkout for differential validation")
	}
	runner := filepath.Join(upstreamRoot, "tools", "run_fixture_tests.py")
	command := exec.Command("python3", runner)
	command.Dir = upstreamRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("upstream validator differs or cannot run: %v\n%s", err, output)
	}
}
