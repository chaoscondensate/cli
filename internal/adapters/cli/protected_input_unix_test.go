//go:build !windows

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/service"
)

func TestPrivateInputPermissionFailureNamesInputArgument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private.yaml")
	if err := os.WriteFile(path, []byte("forecasted_at: '2026-09-01T09:00:00+01:00'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var destination service.SealedForecastInput
	err := decodePrivateOperationInput(context.Background(), path, strings.NewReader(""), service.InputSchemaForecastSeal, &destination)
	if app.ErrorCodeOf(err) != app.CodeConflict || !strings.Contains(err.Error(), "--input") || strings.Contains(err.Error(), "key file") {
		t.Fatalf("permission error = %v", err)
	}
}
