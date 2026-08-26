package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/presentation"
)

func TestForecastShowHelpGolden(t *testing.T) {
	expected, err := os.ReadFile("testdata/help/forecast-show.txt")
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "forecast", "show", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if stdout != string(expected) {
		t.Fatalf("help output changed (-want +got):\nWANT:\n%s\nGOT:\n%s", expected, stdout)
	}
}

func TestInitHelpGolden(t *testing.T) {
	expected, err := os.ReadFile("testdata/help/init.txt")
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "init", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if stdout != string(expected) {
		t.Fatalf("help output changed (-want +got):\nWANT:\n%s\nGOT:\n%s", expected, stdout)
	}
}

func TestInitPublicJSONGolden(t *testing.T) {
	input, err := os.ReadFile("testdata/input/init-public.json")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/result/init-public.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	code, stdout, stderr := runCLIWithStdin(string(input), "forecast-ledger", "--json", "init", "--file", path, "--ledger-id", "research", "--timezone", "UTC", "--forecaster-id", "andrey", "--forecaster-name", "Andrey", "--input", "-")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if stdout != string(expected) {
		t.Fatalf("JSON output changed (-want +got):\nWANT:\n%s\nGOT:\n%s", expected, stdout)
	}
}

func TestLedgerUpdateHelpAndJSONGoldens(t *testing.T) {
	help, err := os.ReadFile("testdata/help/ledger-update.txt")
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "ledger", "update", "--help")
	if code != 0 || stderr != "" || stdout != string(help) {
		t.Fatalf("help code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, help, stdout)
	}
	ledgerBytes := fixtureBytes(t, "individual-ledger.json")
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, ledgerBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	patch, err := os.ReadFile("testdata/input/root-metadata-patch.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/result/ledger-update.json")
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLIWithStdin(string(patch), "forecast-ledger", "--json", "ledger", "update", "--file", path, "--input", "-")
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestPlatformHelpAndListJSONGolden(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "add", want: []string{"--file", "--platform", "--input", "--dry-run"}},
		{name: "update", want: []string{"--file", "--platform", "--input", "--dry-run"}},
		{name: "list", want: []string{"--file"}},
		{name: "show", want: []string{"--file", "--platform"}},
		{name: "remove", want: []string{"--file", "--platform", "--dry-run", "--yes"}},
	}
	for _, test := range tests {
		code, stdout, stderr := runCLI("forecast-ledger", "platform", test.name, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("platform %s help code=%d stderr=%q", test.name, code, stderr)
		}
		for _, expected := range test.want {
			if !strings.Contains(stdout, expected) {
				t.Errorf("platform %s help missing %q:\n%s", test.name, expected, stdout)
			}
		}
	}
	want, err := os.ReadFile("testdata/result/platform-list.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "platform", "list", "--file", path)
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("platform list JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestForecastHelpAndListJSONGolden(t *testing.T) {
	for _, test := range []struct {
		name string
		want []string
	}{
		{name: "add", want: []string{"--file", "--question", "--forecast", "--input", "--dry-run"}},
		{name: "list", want: []string{"--file", "--question"}},
		{name: "show", want: []string{"--file", "--question", "--forecast"}},
	} {
		code, stdout, stderr := runCLI("forecast-ledger", "forecast", test.name, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("forecast %s help code=%d stderr=%q", test.name, code, stderr)
		}
		for _, expected := range test.want {
			if !strings.Contains(stdout, expected) {
				t.Errorf("forecast %s help missing %q:\n%s", test.name, expected, stdout)
			}
		}
	}
	want, err := os.ReadFile("testdata/result/forecast-list.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "forecast", "list", "--file", path, "--question", "q-central-bank-cut")
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("forecast list JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestForecastSealRevealHintHelpAndSealPlanGolden(t *testing.T) {
	for _, test := range []struct {
		args []string
		want []string
	}{
		{args: []string{"seal"}, want: []string{"--file", "--question", "--forecast", "--input", "--key-file", "--dry-run"}},
		{args: []string{"reveal"}, want: []string{"--file", "--question", "--forecast", "--key-file", "--revealed-at", "--dry-run", "--yes"}},
		{args: []string{"key-hint", "update"}, want: []string{"--file", "--question", "--forecast", "--key-hint", "--dry-run"}},
	} {
		arguments := append([]string{"forecast-ledger", "forecast"}, test.args...)
		arguments = append(arguments, "--help")
		code, stdout, stderr := runCLI(arguments...)
		if code != 0 || stderr != "" {
			t.Fatalf("forecast %v help code=%d stderr=%q", test.args, code, stderr)
		}
		for _, expected := range test.want {
			if !strings.Contains(stdout, expected) {
				t.Errorf("forecast %v help missing %q:\n%s", test.args, expected, stdout)
			}
		}
	}
	want, err := os.ReadFile("testdata/result/forecast-seal-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "f.key")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"forecasted_at":"2026-08-25T09:00:00+01:00","recorded_at":"2026-08-25T09:01:00+01:00","value":{"kind":"multiple_choice","probabilities":[{"option_id":"centre-left","probability_bp":5000},{"option_id":"centre-right","probability_bp":3500},{"option_id":"other","probability_bp":1500}]},"rationale":"private","key_factors":[],"comment":"private","supersedes_forecast_id":"f-election-coalition-001"}`
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "--json", "forecast", "seal", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--input", "-", "--key-file", keyPath, "--dry-run")
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("forecast seal plan JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestQuestionHelpAndListJSONGolden(t *testing.T) {
	for _, test := range []struct {
		name string
		want []string
	}{
		{name: "add", want: []string{"--file", "--question", "--type", "--input", "--key-file", "--dry-run"}},
		{name: "update", want: []string{"--file", "--question", "--input", "--dry-run"}},
		{name: "list", want: []string{"--file"}},
		{name: "show", want: []string{"--file", "--question"}},
		{name: "resolve", want: []string{"--file", "--question", "--input", "--dry-run", "--yes"}},
		{name: "annul", want: []string{"--file", "--question", "--input", "--dry-run", "--yes"}},
		{name: "dispute", want: []string{"--file", "--question", "--input", "--dry-run", "--yes"}},
	} {
		code, stdout, stderr := runCLI("forecast-ledger", "question", test.name, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("question %s help code=%d stderr=%q", test.name, code, stderr)
		}
		for _, expected := range test.want {
			if !strings.Contains(stdout, expected) {
				t.Errorf("question %s help missing %q:\n%s", test.name, expected, stdout)
			}
		}
	}
	want, err := os.ReadFile("testdata/result/question-list.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "question", "list", "--file", path)
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("question list JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestTargetHelpAndBuildJSONGolden(t *testing.T) {
	for _, test := range []struct {
		name string
		want []string
	}{
		{name: "build", want: []string{"--file", "--question", "--forecast", "--all", "--dry-run"}},
		{name: "check", want: []string{"--file", "--question", "--forecast", "--all"}},
	} {
		code, stdout, stderr := runCLI("forecast-ledger", "target", test.name, "--help")
		if code != 0 || stderr != "" {
			t.Fatalf("target %s help code=%d stderr=%q", test.name, code, stderr)
		}
		for _, expected := range test.want {
			if !strings.Contains(stdout, expected) {
				t.Errorf("target %s help missing %q:\n%s", test.name, expected, stdout)
			}
		}
	}
	want, err := os.ReadFile("testdata/result/target-build.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "target", "build", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-001")
	if code != 0 || stderr != "" || stdout != string(want) {
		t.Fatalf("target build JSON code=%d stderr=%q\nWANT:\n%s\nGOT:\n%s", code, stderr, want, stdout)
	}
}

func TestJSONErrorSchemaExitCodesAndCancellation(t *testing.T) {
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "validate")
	if code != 2 || stdout != "" {
		t.Fatalf("usage code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope presentation.ErrorEnvelope
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatalf("JSON error is invalid: %v (%s)", err, stderr)
	}
	if envelope.OK || envelope.Code != app.CodeUsage || envelope.Message == "" {
		t.Fatalf("unexpected JSON error: %#v", envelope)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var canceledOut, canceledErr bytes.Buffer
	code = Run(ctx, []string{"forecast-ledger", "validate", "--file", "ledger.yaml"}, strings.NewReader(""), &canceledOut, &canceledErr)
	if code != 130 || canceledOut.Len() != 0 || !strings.Contains(canceledErr.String(), "interrupted") {
		t.Fatalf("cancellation code=%d stdout=%q stderr=%q", code, canceledOut.String(), canceledErr.String())
	}
}

func TestOutputWriterFailureReturnsInternalExit(t *testing.T) {
	writer := failingWriter{}
	code := Run(context.Background(), []string{"forecast-ledger", "version"}, strings.NewReader(""), writer, writer)
	if code != 1 {
		t.Fatalf("writer failure exit=%d, want 1", code)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, os.ErrPermission }
