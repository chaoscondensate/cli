package storage

import (
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
)

// CreateProtectedFile exclusively creates and durably writes a secret file.
// Platform implementations establish owner-only protection before bytes are
// written and fail closed when that protection cannot be verified.
func CreateProtectedFile(path string, data []byte) error {
	return createProtectedFile(path, data)
}

// CheckProtectedFile verifies the native owner-only protection contract for an
// existing regular file without changing its permissions or ACL.
func CheckProtectedFile(path string) error {
	return checkProtectedFile(path)
}

// ReadProtectedFile validates owner-only native protection and reads one
// bounded regular file without accepting a link swap between inspection and
// the opened file handle.
func ReadProtectedFile(path string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	resolved, err := ResolveLedgerPath(path, true)
	if err != nil {
		return nil, err
	}
	if err := CheckProtectedFile(resolved); err != nil {
		return nil, err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, app.NewError(app.CodeIO, "protected file cannot be opened", err)
	}
	defer file.Close()
	handleInfo, err := file.Stat()
	if err != nil {
		return nil, app.NewError(app.CodeIO, "protected file handle cannot be inspected", err)
	}
	pathInfo, err := os.Lstat(resolved)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, app.NewError(app.CodeConflict, "protected file changed while it was opened", err)
		}
		return nil, app.NewError(app.CodeIO, "protected file cannot be inspected", err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !handleInfo.Mode().IsRegular() || !os.SameFile(handleInfo, pathInfo) {
		return nil, app.NewError(app.CodeConflict, "protected file changed or is not a regular file", nil)
	}
	if handleInfo.Size() > maxBytes {
		return nil, app.NewError(app.CodeInvalidData, "protected file exceeds its size limit", nil)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, app.NewError(app.CodeIO, "protected file cannot be read", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, app.NewError(app.CodeInvalidData, "protected file exceeds its size limit", nil)
	}
	return data, nil
}
