//go:build windows

package storage

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func isLinkOrReparse(info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
