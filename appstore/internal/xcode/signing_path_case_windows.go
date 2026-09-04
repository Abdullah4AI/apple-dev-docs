//go:build windows

package xcode

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// signingCaseInsensitiveVolumeFor reports the case comparison semantics of
// the directory containing path. Windows permits case-sensitive directories
// on an otherwise case-insensitive volume, so a volume-wide lowercased key is
// not safe for missing or case-variant xcconfig paths. The nearest existing
// directory is sufficient when path itself has not been created yet.
func signingCaseInsensitiveVolumeFor(path string) (caseInsensitive, known bool) {
	candidate := normalizeSigningLexicalPath(path)
	for {
		info, err := os.Lstat(candidate)
		if err == nil {
			if !info.IsDir() {
				candidate = filepath.Dir(candidate)
				continue
			}

			handle, err := windows.CreateFile(
				windows.StringToUTF16Ptr(candidate),
				windows.FILE_READ_ATTRIBUTES,
				windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
				nil,
				windows.OPEN_EXISTING,
				windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
				0,
			)
			if err != nil {
				return false, false
			}
			defer windows.CloseHandle(handle)

			var caseInfo struct {
				Flags uint32
			}
			buffer := (*byte)(unsafe.Pointer(&caseInfo))
			if err := windows.GetFileInformationByHandleEx(
				handle,
				windows.FileCaseSensitiveInfo,
				buffer,
				uint32(unsafe.Sizeof(caseInfo)),
			); err != nil {
				return false, false
			}
			return caseInfo.Flags&windows.FILE_CS_FLAG_CASE_SENSITIVE_DIR == 0, true
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, false
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return false, false
		}
		candidate = parent
	}
}
