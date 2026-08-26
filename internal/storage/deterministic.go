package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/chaoscondensate/cli/internal/app"
)

type DeterministicState string

const (
	DeterministicCreated   DeterministicState = "created"
	DeterministicReplaced  DeterministicState = "replaced"
	DeterministicUnchanged DeterministicState = "unchanged"
)

type DeterministicResult struct {
	State  DeterministicState
	Path   string
	Size   int64
	SHA256 string
}

// EnsureDeterministicFile exclusively creates expected bytes or confirms an
// existing byte-identical regular file. It never follows, replaces, truncates,
// chmods, or removes an existing directory entry.
func EnsureDeterministicFile(path string, expected []byte, mode fs.FileMode, maxExistingBytes int64) (DeterministicResult, error) {
	result := DeterministicResult{Path: path, Size: int64(len(expected)), SHA256: ResourceDigest(expected)}
	if maxExistingBytes <= 0 {
		maxExistingBytes = 64 << 20
	}
	if int64(len(expected)) > maxExistingBytes {
		return result, app.NewError(app.CodeInvalidData, "deterministic artifact exceeds its size limit", nil)
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := CreateExclusive(path, expected, mode); err != nil {
			// A racing creator may have installed the same deterministic bytes.
			if app.ErrorCodeOf(err) == app.CodeConflict {
				return compareDeterministicFile(path, expected, maxExistingBytes)
			}
			return result, err
		}
		result.State = DeterministicCreated
		return result, nil
	}
	if err != nil {
		return result, app.NewError(app.CodeIO, "deterministic artifact cannot be inspected", err)
	}
	if isLinkOrReparse(info) {
		return result, app.NewError(app.CodeConflict, "deterministic artifact path is a link or reparse point", nil)
	}
	return compareDeterministicFile(path, expected, maxExistingBytes)
}

func compareDeterministicFile(path string, expected []byte, maxBytes int64) (DeterministicResult, error) {
	result := DeterministicResult{Path: path, Size: int64(len(expected)), SHA256: ResourceDigest(expected)}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, app.NewError(app.CodeConflict, "deterministic artifact disappeared during collision check", err)
		}
		return result, app.NewError(app.CodeIO, "deterministic artifact cannot be inspected", err)
	}
	if isLinkOrReparse(info) || !info.Mode().IsRegular() {
		return result, app.NewError(app.CodeConflict, "deterministic artifact destination is not a regular file", nil)
	}
	if info.Size() > maxBytes {
		return result, app.WithDetails(app.NewError(app.CodeConflict, "deterministic artifact contains different bytes", nil), map[string]any{
			"expected_sha256": result.SHA256, "actual_size": info.Size(),
		})
	}
	file, err := os.Open(path)
	if err != nil {
		return result, app.NewError(app.CodeIO, "deterministic artifact cannot be opened", err)
	}
	actual, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return result, app.NewError(app.CodeIO, "deterministic artifact cannot be read", readErr)
	}
	if closeErr != nil {
		return result, app.NewError(app.CodeIO, "deterministic artifact cannot be closed", closeErr)
	}
	if bytes.Equal(actual, expected) {
		result.State = DeterministicUnchanged
		return result, nil
	}
	digest := sha256.Sum256(actual)
	return result, app.WithDetails(app.NewError(app.CodeConflict, "deterministic artifact contains different bytes", nil), map[string]any{
		"expected_sha256": result.SHA256,
		"actual_sha256":   hex.EncodeToString(digest[:]),
		"expected_size":   len(expected),
		"actual_size":     len(actual),
	})
}

func DeterministicRelativePath(directory, base string) string {
	return filepath.ToSlash(filepath.Join(directory, base))
}

// ReplaceDeterministicFile installs validated replacement bytes through a
// flushed same-directory temporary file. A crash leaves either the old or new
// complete regular file; links and type changes are rejected before replace.
func ReplaceDeterministicFile(path string, expected []byte, mode fs.FileMode, maxBytes int64) (DeterministicResult, error) {
	result := DeterministicResult{Path: path, Size: int64(len(expected)), SHA256: ResourceDigest(expected)}
	if maxBytes <= 0 {
		maxBytes = 64 << 20
	}
	if int64(len(expected)) > maxBytes {
		return result, app.NewError(app.CodeInvalidData, "replacement artifact exceeds its size limit", nil)
	}
	if same, err := compareDeterministicFile(path, expected, maxBytes); err == nil {
		return same, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return EnsureDeterministicFile(path, expected, mode, maxBytes)
		}
		return result, app.NewError(app.CodeIO, "replacement artifact cannot be inspected", err)
	}
	if isLinkOrReparse(info) || !info.Mode().IsRegular() {
		return result, app.NewError(app.CodeConflict, "replacement artifact destination is not a regular file", nil)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-replace-*.tmp")
	if err != nil {
		return result, app.NewError(app.CodeIO, "replacement temporary file cannot be created", err)
	}
	temporaryPath := temporary.Name()
	installed := false
	defer func() {
		_ = temporary.Close()
		if !installed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(mode.Perm()); err != nil {
		return result, app.NewError(app.CodeIO, "replacement temporary permissions cannot be set", err)
	}
	if _, err := temporary.Write(expected); err != nil {
		return result, app.NewError(app.CodeIO, "replacement temporary file cannot be written", err)
	}
	if err := temporary.Sync(); err != nil {
		return result, app.NewError(app.CodeIO, "replacement temporary file cannot be flushed", err)
	}
	if err := temporary.Close(); err != nil {
		return result, app.NewError(app.CodeIO, "replacement temporary file cannot be closed", err)
	}
	if err := safeReplace(temporaryPath, path); err != nil {
		return result, app.NewError(app.CodeIO, "artifact replacement failed", err)
	}
	installed = true
	if err := syncParentDirectory(filepath.Dir(path)); err != nil {
		return result, app.NewError(app.CodeIO, "artifact directory flush failed", err)
	}
	result.State = DeterministicReplaced
	return result, nil
}
