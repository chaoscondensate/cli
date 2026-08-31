//go:build windows

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
)

func TestCreateProtectedFileIsExclusiveAndOwnerOnlyWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forecast.key")
	if err := CreateProtectedFile(path, []byte("secret\n")); err != nil {
		t.Fatal(err)
	}
	if err := CheckProtectedFile(path); err != nil {
		t.Fatal(err)
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

func TestCheckProtectedFileRejectsInheritedWindowsACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inherited.key")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckProtectedFile(path); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("inherited ACL error = %v", err)
	}
}
