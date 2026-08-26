//go:build windows

package storage

import (
	"io/fs"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

type reparseFileInfo struct{ attributes uint32 }

func (reparseFileInfo) Name() string       { return "entry" }
func (reparseFileInfo) Size() int64        { return 0 }
func (reparseFileInfo) Mode() fs.FileMode  { return 0 }
func (reparseFileInfo) ModTime() time.Time { return time.Time{} }
func (reparseFileInfo) IsDir() bool        { return false }
func (info reparseFileInfo) Sys() any {
	return &syscall.Win32FileAttributeData{FileAttributes: info.attributes}
}

func TestWindowsReparseAttributesAreRejected(t *testing.T) {
	if !isLinkOrReparse(reparseFileInfo{attributes: windows.FILE_ATTRIBUTE_REPARSE_POINT}) {
		t.Fatal("reparse-point attribute was accepted")
	}
	if isLinkOrReparse(reparseFileInfo{}) {
		t.Fatal("ordinary file was classified as a reparse point")
	}
}
