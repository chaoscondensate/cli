package presentation

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestStableJSONUsesCorrectStreamsAndRedactsSecrets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	presenter := New(&stdout, &stderr, Options{JSON: true})
	data := map[string]any{"question_id": "q-one", "revealed_key": "CANARY-KEY", "nested": map[string]any{"token": "CANARY-TOKEN", "key_hint": "safe-hint"}}
	if err := presenter.Success("forecast.shown", "Forecast found", data); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("success wrote stderr: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "CANARY") || !strings.Contains(stdout.String(), `"revealed_key":"[redacted]"`) || !strings.Contains(stdout.String(), `"key_hint":"safe-hint"`) {
		t.Fatalf("unsafe JSON success: %s", stdout.String())
	}

	stdout.Reset()
	err := app.WithDetails(app.NewError(app.CodeNotFound, "forecast not found", errors.New("private cause")), map[string]any{"key": "CANARY", "forecast_id": "f-one"})
	if writeErr := presenter.Failure(err); writeErr != nil {
		t.Fatal(writeErr)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failure wrote stdout: %q", stdout.String())
	}
	var envelope ErrorEnvelope
	if decodeErr := json.Unmarshal(stderr.Bytes(), &envelope); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if envelope.Code != app.CodeNotFound || envelope.Details["key"] != "[redacted]" || strings.Contains(stderr.String(), "private cause") {
		t.Fatalf("unexpected JSON failure: %#v raw=%s", envelope, stderr.String())
	}
}

func TestNonTTYPlainQuietVerboseAndNoColor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	notTTY, isTTY := false, true
	presenter := New(&stdout, &stderr, Options{StdoutTTY: &notTTY, StderrTTY: &notTTY})
	if presenter.Mode() != ModePlain || presenter.ColorEnabled() {
		t.Fatalf("non-TTY mode=%s color=%v", presenter.Mode(), presenter.ColorEnabled())
	}
	_ = presenter.Success("ok", "done", nil)
	_ = presenter.Verbose("details")
	if stdout.String() != "done\n" || stderr.Len() != 0 {
		t.Fatalf("plain stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	presenter = New(&stdout, &stderr, Options{Quiet: true, Verbose: true, StdoutTTY: &isTTY, StderrTTY: &isTTY})
	_ = presenter.Success("ok", "done", nil)
	_ = presenter.Verbose("details")
	if stdout.Len() != 0 || stderr.String() != "details\n" {
		t.Fatalf("quiet/verbose stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	presenter = New(&stdout, &stderr, Options{StdoutTTY: &isTTY, LookupEnv: func(name string) (string, bool) { return "", name == "NO_COLOR" }})
	if presenter.ColorEnabled() {
		t.Fatal("NO_COLOR was ignored")
	}

	presenter = New(&stdout, &stderr, Options{StdoutTTY: &isTTY, LookupEnv: func(name string) (string, bool) {
		if name == "TERM" {
			return "dumb", true
		}
		return "", false
	}})
	if presenter.ColorEnabled() {
		t.Fatal("TERM=dumb was ignored")
	}

	presenter = New(&stdout, &stderr, Options{NoColor: true, StdoutTTY: &isTTY})
	if presenter.ColorEnabled() {
		t.Fatal("NoColor option was ignored")
	}
}
