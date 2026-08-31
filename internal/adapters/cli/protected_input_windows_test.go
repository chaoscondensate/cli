//go:build windows

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/service"
)

func TestPrivateInputACLFailureNamesInputArgumentWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private.yaml")
	if err := os.WriteFile(path, []byte("forecasted_at: '2026-09-01T09:00:00+01:00'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var destination service.SealedForecastInput
	err := decodePrivateOperationInputForArgument(context.Background(), path, strings.NewReader(""), service.InputSchemaForecastSealPrivate, &destination, "--secret-input")
	if app.ErrorCodeOf(err) != app.CodeConflict || !strings.Contains(err.Error(), "--secret-input") || strings.Contains(err.Error(), "key file") || strings.Contains(err.Error(), path) {
		t.Fatalf("ACL error = %v", err)
	}
}
