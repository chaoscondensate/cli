package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestEnsureDeterministicFileCreatesConfirmsAndConflicts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.json")
	want := []byte(`{"target":true}`)
	result, err := EnsureDeterministicFile(path, want, 0o644, 1024)
	if err != nil || result.State != DeterministicCreated {
		t.Fatalf("create result=%#v err=%v", result, err)
	}
	result, err = EnsureDeterministicFile(path, want, 0o600, 1024)
	if err != nil || result.State != DeterministicUnchanged {
		t.Fatalf("identical result=%#v err=%v", result, err)
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
		t.Fatalf("idempotent check changed mode to %o", info.Mode().Perm())
	}
	_, err = EnsureDeterministicFile(path, []byte(`{"target":false}`), 0o600, 1024)
	if app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("different bytes error = %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(want) {
		t.Fatalf("conflict replaced existing bytes: %s", after)
	}
}

func TestEnsureDeterministicFileNeverFollowsExistingEntry(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureDeterministicFile(destination, []byte("value"), 0o600, 1024); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("directory destination error = %v", err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	target := filepath.Join(directory, "outside")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureDeterministicFile(link, []byte("value"), 0o600, 1024); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("link destination error = %v", err)
	}
	after, err := os.ReadFile(target)
	if err != nil || string(after) != "outside" {
		t.Fatalf("link target changed: data=%q err=%v", after, err)
	}
	if _, err := os.Lstat(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
