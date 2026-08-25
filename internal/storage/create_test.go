package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestCreateExclusiveNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.json")
	if err := CreateExclusive(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CreateExclusive(path, []byte("second"), 0o600); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("existing output err=%v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first" {
		t.Fatalf("existing output was overwritten: %q", data)
	}
}

func TestCreateExclusiveFollowsNativeCaseFoldingWithoutSilentOverwrite(t *testing.T) {
	root := t.TempDir()
	upper := filepath.Join(root, "Target.json")
	lower := filepath.Join(root, "target.json")
	if err := CreateExclusive(upper, []byte("upper"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := CreateExclusive(lower, []byte("lower"), 0o600)
	if err != nil && app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("case-fold collision returned wrong error: %v", err)
	}
	upperData, readErr := os.ReadFile(upper)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if err != nil && string(upperData) != "upper" {
		t.Fatalf("case-fold collision overwrote existing file: %q", upperData)
	}
}
