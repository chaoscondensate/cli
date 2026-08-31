//go:build !windows

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
)

func TestCreateProtectedFileIsExclusiveAndOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forecast.key")
	if err := CreateProtectedFile(path, []byte("secret\n")); err != nil {
		t.Fatal(err)
	}
	if err := CheckProtectedFile(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o", got)
	}
	if err := CreateProtectedFile(path, []byte("replacement\n")); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("second create error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "secret\n" {
		t.Fatalf("protected file was overwritten: %q", data)
	}
}

func TestCheckProtectedFileRejectsLooseModeWithoutChangingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loose.key")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckProtectedFile(path); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("loose mode error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("check changed mode to %o", info.Mode().Perm())
	}
}
