//go:build windows

package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/chaoscondensate/cli/internal/app"
	"golang.org/x/sys/windows"
)

func createProtectedFile(path string, data []byte) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return app.NewError(app.CodeIO, "current Windows owner cannot be identified", err)
	}
	sid := user.User.Sid
	descriptor, err := windows.SecurityDescriptorFromString("O:" + sid.String() + "D:P(A;;FA;;;" + sid.String() + ")")
	if err != nil {
		return app.NewError(app.CodeIO, "owner-only Windows ACL cannot be built", err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return app.NewError(app.CodeUsage, "protected key path is invalid", err)
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE|windows.READ_CONTROL, 0, &attributes, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, fs.ErrExist) {
			return app.NewError(app.CodeConflict, "protected key file already exists", err)
		}
		return app.NewError(app.CodeIO, "protected key file cannot be created", err)
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		windows.CloseHandle(handle)
		_ = os.Remove(path)
		return app.NewError(app.CodeIO, "protected key file handle cannot be opened", nil)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := checkWindowsOwnerOnly(path, sid); err != nil {
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
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return app.NewError(app.CodeIO, "current Windows owner cannot be identified", err)
	}
	return checkWindowsOwnerOnly(path, user.User.Sid)
}

func checkWindowsOwnerOnly(path string, expected *windows.SID) error {
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return app.NewError(app.CodeIO, "protected key ACL cannot be read", err)
	}
	owner, _, err := descriptor.Owner()
	if err != nil || owner == nil || !owner.Equals(expected) {
		return app.NewError(app.CodeConflict, "protected key file is not owned by the current user", err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		return app.NewError(app.CodeConflict, "protected key file does not have an owner-only ACL", err)
	}
	control, _, err := descriptor.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return app.NewError(app.CodeConflict, "protected key ACL inheritance is not disabled", err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil || ace == nil {
		return app.NewError(app.CodeConflict, "protected key ACL entry cannot be read", err)
	}
	const fileAllAccess windows.ACCESS_MASK = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x1ff
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE ||
		ace.Header.AceFlags != 0 ||
		ace.Mask != fileAllAccess ||
		aceSID == nil || !aceSID.IsValid() || !aceSID.Equals(expected) {
		return app.NewError(app.CodeConflict, "protected key ACL grants access beyond the current owner", nil)
	}
	return nil
}
