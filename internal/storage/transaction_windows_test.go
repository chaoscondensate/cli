//go:build windows

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/document"
)

func TestWindowsMoveFileExReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte("{\"value\":1}\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := UpdateLedger(context.Background(), path, TransactionOptions{Mutate: func(parsed *document.Document) ([]byte, error) {
		return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{\"value\":2}\r\n" {
		t.Fatalf("replacement data=%q err=%v", data, err)
	}
}
