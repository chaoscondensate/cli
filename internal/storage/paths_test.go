package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestValidateRelativePathPortableRules(t *testing.T) {
	for _, valid := range []string{"target.json", "targets/q-one/f-one.json", "proofs/one.ots"} {
		if err := ValidateRelativePath(valid); err != nil {
			t.Errorf("valid path %q rejected: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", ".", "../escape", "a/../b", "/absolute", `C:\target`, `C:/target`, `\\server\share`, "//server/share", `a\b`, "file:stream", "a//b", "a/./b"} {
		err := ValidateRelativePath(invalid)
		if err == nil || !errors.Is(err, ErrUnsafePath) {
			t.Errorf("unsafe path %q: got %v", invalid, err)
		}
	}
}

func TestPathResolverConfinesAndRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "targets", "one.json")
	if err := os.WriteFile(file, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewPathResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve("targets/one.json", true)
	if err != nil || resolved != filepath.Join(canonicalRoot, "targets", "one.json") {
		t.Fatalf("resolved=%q err=%v", resolved, err)
	}
	if _, err := resolver.Resolve("targets/new.json", false); err != nil {
		t.Fatalf("new leaf path rejected: %v", err)
	}
	if _, err := resolver.Resolve("missing/new.json", false); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("missing parent: got %v", err)
	}

	if runtime.GOOS != "windows" {
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
			t.Fatal(err)
		}
		if _, err := resolver.Resolve("linked/escape.json", false); err == nil || !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("symlink path accepted: %v", err)
		}
	}
}

func TestResolveLedgerPathRequiresExplicitRegularFile(t *testing.T) {
	if _, err := ResolveLedgerPath("", true); app.ErrorCodeOf(err) != app.CodeUsage {
		t.Fatalf("empty file: %v", err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "ledger.json")
	resolved, err := ResolveLedgerPath(path, false)
	canonicalRoot, canonicalErr := filepath.EvalSymlinks(root)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if err != nil || resolved != filepath.Join(canonicalRoot, "ledger.json") {
		t.Fatalf("new ledger path=%q err=%v", resolved, err)
	}
	if _, err := ResolveLedgerPath(path, true); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("missing existing ledger: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveLedgerPath(path, true); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveLedgerPath(root, true); err == nil || !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("directory accepted as ledger: %v", err)
	}
}
