//go:build !windows

package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/chaoscondensate/cli/internal/app"
)

func createProtectedFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return app.NewError(app.CodeConflict, "protected key file already exists", err)
		}
		return app.NewError(app.CodeIO, "protected key file cannot be created", err)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := checkProtectedOpenFile(file); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return app.NewError(app.CodeIO, "protected key file cannot be written", err)
	}
	if err := file.Sync(); err != nil {
		return app.NewError(app.CodeIO, "protected key file cannot be flushed", err)
	}
	if err := file.Close(); err != nil {
		return app.NewError(app.CodeIO, "protected key file cannot be closed", err)
	}
	if err := syncParentDirectory(filepath.Dir(path)); err != nil {
		return app.NewError(app.CodeIO, "protected key directory cannot be flushed", err)
	}
	ok = true
	return nil
}

func checkProtectedFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return app.NewError(app.CodeIO, "protected key file cannot be inspected", err)
	}
	if isLinkOrReparse(info) || !info.Mode().IsRegular() {
		return app.NewError(app.CodeConflict, "protected key path is not a regular file", nil)
	}
	if info.Mode().Perm() != 0o600 {
		return app.NewError(app.CodeConflict, "protected key file must have mode 0600", nil)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return app.NewError(app.CodeConflict, "protected key file is not owned by the current user", nil)
	}
	return nil
}

func checkProtectedOpenFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return app.NewError(app.CodeIO, "protected key permissions cannot be inspected", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return app.NewError(app.CodeConflict, "protected key file could not be created with mode 0600", nil)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return app.NewError(app.CodeConflict, "protected key file could not be created for the current owner", nil)
	}
	return nil
}
