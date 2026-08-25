package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/chaoscondensate/cli/internal/app"
)

// CreateExclusive writes and flushes a new file without ever overwriting an
// existing path. The parent directory must already exist.
func CreateExclusive(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return app.NewError(app.CodeConflict, "output file already exists", err)
		}
		return app.NewError(app.CodeIO, "output file cannot be created", err)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return app.NewError(app.CodeIO, "output file cannot be written", err)
	}
	if err := file.Sync(); err != nil {
		return app.NewError(app.CodeIO, "output file cannot be flushed", err)
	}
	if err := file.Close(); err != nil {
		return app.NewError(app.CodeIO, "output file cannot be closed", err)
	}
	ok = true
	if err := syncParentDirectory(filepath.Dir(path)); err != nil {
		return app.NewError(app.CodeIO, "output directory cannot be flushed", err)
	}
	return nil
}
