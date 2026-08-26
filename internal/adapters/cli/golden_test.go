package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
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
