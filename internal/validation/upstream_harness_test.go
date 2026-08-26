package validation

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestPinnedUpstreamPythonFixtureHarness runs the exact pinned upstream fixture
// harness when its checkout and Python test dependencies are available. It
// checks the reference implementation itself; case-by-case Go/Python parity is
// a separate required conformance gate.
func TestPinnedUpstreamPythonFixtureHarness(t *testing.T) {
	upstreamRoot := os.Getenv("FORECAST_LEDGER_UPSTREAM_ROOT")
	if upstreamRoot == "" {
		t.Skip("set FORECAST_LEDGER_UPSTREAM_ROOT to run the pinned Python reference fixture harness")
	}
	runner := filepath.Join(upstreamRoot, "tools", "run_fixture_tests.py")
	command := exec.Command("python3", runner)
	command.Dir = upstreamRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("upstream validator differs or cannot run: %v\n%s", err, output)
	}
}
