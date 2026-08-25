//go:build !windows

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
)

func TestUpdateLedgerReportsReadOnlyDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ledger.json")
	if err := os.WriteFile(path, []byte("{\"value\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	err := UpdateLedger(context.Background(), path, TransactionOptions{Mutate: func(parsed *document.Document) ([]byte, error) {
		return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
	}})
	if app.ErrorCodeOf(err) != app.CodeIO {
		t.Fatalf("read-only directory err=%v", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || string(data) != "{\"value\":1}\n" {
		t.Fatalf("ledger changed data=%q err=%v", data, readErr)
	}
}
