package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestLoadRejectsMissingUnknownAndFutureSchemaVersionsFirst(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "missing", content: `{"ledger_id":"example"}`},
		{name: "wrong type", content: `{"schema_version":1}`},
		{name: "future", content: `{"schema_version":"2.0.0"}`},
		{name: "unknown", content: `{"schema_version":"preview"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ledger.json")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadAndValidateLedger(context.Background(), path, nil)
			if app.ErrorCodeOf(err) != app.CodeUnsupportedSchemaVersion || app.ExitCodeOf(err) != 3 {
				t.Fatalf("error = %#v, code=%q exit=%d", err, app.ErrorCodeOf(err), app.ExitCodeOf(err))
			}
			applicationErr, ok := err.(*app.Error)
			if !ok || applicationErr.Details["supported_schema_version"] != "1.0.0" {
				t.Fatalf("details = %#v", applicationErr)
			}
		})
	}
}
