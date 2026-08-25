package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
)

var ErrUnsafePath = errors.New("path is not safe")

// ResolveLedgerPath resolves an explicitly supplied ledger filename. Existing
// ledgers must be regular files and may not themselves be symlinks or reparse
// points. Ancestor symlinks are canonicalized before an artifact root is made.
func ResolveLedgerPath(input string, mustExist bool) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", app.NewError(app.CodeUsage, "--file is required", ErrUnsafePath)
	}
	if strings.IndexByte(input, 0) >= 0 {
		return "", unsafePathError("ledger path contains a NUL byte")
	}
	if runtime.GOOS != "windows" && looksLikeWindowsAbsolute(input) {
		return "", unsafePathError("Windows drive and UNC paths are not valid on this platform")
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", app.NewError(app.CodeUsage, "ledger path is not valid", err)
	}
	abs = filepath.Clean(abs)
	info, statErr := os.Lstat(abs)
	if statErr == nil {
		if isLinkOrReparse(info) {
			return "", unsafePathError("ledger path must not be a symlink or junction")
		}
		if !info.Mode().IsRegular() {
			return "", unsafePathError("ledger path must be a regular file")
		}
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return "", app.NewError(app.CodeIO, "ledger path cannot be resolved", err)
		}
		return filepath.Clean(resolved), nil
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return "", app.NewError(app.CodeIO, "ledger path cannot be inspected", statErr)
	}
	if mustExist {
		return "", app.NewError(app.CodeNotFound, "ledger file does not exist", statErr)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", app.NewError(app.CodeNotFound, "ledger parent directory does not exist", err)
		}
		return "", app.NewError(app.CodeIO, "ledger parent directory cannot be resolved", err)
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

type PathResolver struct {
	root string
}

func NewPathResolver(root string) (*PathResolver, error) {
	if strings.TrimSpace(root) == "" {
		return nil, unsafePathError("artifact root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, unsafePathError("artifact root is not valid")
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, app.NewError(app.CodeNotFound, "artifact root does not exist", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return nil, app.NewError(app.CodeIO, "artifact root cannot be inspected", err)
	}
	if !info.IsDir() {
		return nil, unsafePathError("artifact root must be a directory")
	}
	return &PathResolver{root: filepath.Clean(canonical)}, nil
}

func (r *PathResolver) Root() string { return r.root }

// Resolve confines a schema relativePath under Root and rejects existing
// symlinks, junctions, and other reparse points in every descendant component.
func (r *PathResolver) Resolve(relative string, mustExist bool) (string, error) {
	if r == nil || r.root == "" {
		return "", app.NewError(app.CodeInternal, "path resolver is not initialized", nil)
	}
	if err := ValidateRelativePath(relative); err != nil {
		return "", err
	}
	candidate := filepath.Join(r.root, filepath.FromSlash(relative))
	contained, err := pathContained(r.root, candidate)
	if err != nil || !contained {
		return "", unsafePathError("path escapes the allowed root")
	}
	if err := rejectDescendantLinks(r.root, candidate, mustExist); err != nil {
		return "", err
	}
	return candidate, nil
}

func ValidateRelativePath(path string) error {
	if path == "" || path == "." || strings.IndexByte(path, 0) >= 0 {
		return unsafePathError("relative path is empty or invalid")
	}
	if strings.Contains(path, "\\") {
		return unsafePathError("relative paths must use forward slashes")
	}
	if strings.Contains(path, ":") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || looksLikeWindowsAbsolute(path) {
		return unsafePathError("absolute, drive, UNC, and alternate-stream paths are not allowed")
	}
	if !fs.ValidPath(path) {
		return unsafePathError("relative path contains an empty, dot, or parent segment")
	}
	return nil
}

func pathContained(root, candidate string) (bool, error) {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false, err
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative), nil
}

func rejectDescendantLinks(root, candidate string, mustExist bool) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return unsafePathError("path cannot be made relative to its root")
	}
	current := root
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if mustExist || index < len(parts)-1 {
					return app.NewError(app.CodeNotFound, "path component does not exist", err)
				}
				return nil
			}
			return app.NewError(app.CodeIO, "path component cannot be inspected", err)
		}
		if isLinkOrReparse(info) {
			return unsafePathError("symlinks, junctions, and reparse points are not allowed inside the artifact root")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return unsafePathError("intermediate path component is not a directory")
		}
	}
	return nil
}

func looksLikeWindowsAbsolute(path string) bool {
	return len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' ||
		strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `//`)
}

func unsafePathError(message string) error {
	return app.NewError(app.CodeUsage, message, ErrUnsafePath)
}

func (r *PathResolver) String() string {
	if r == nil {
		return "<nil>"
	}
	return fmt.Sprintf("PathResolver(%s)", r.root)
}
