//go:build darwin || linux

package rootfs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func copyReplacementMetadata(destination, source *os.File, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("inspect replacement ownership: unsupported stat type %T", info.Sys())
	}
	if err := unix.Fchown(int(destination.Fd()), int(stat.Uid), int(stat.Gid)); err != nil {
		return fmt.Errorf("preserve replacement ownership: %w", err)
	}
	// Preserve the special mode bits as well as ordinary permissions. The
	// replacement is written from the already-verified source descriptor, so
	// dropping these bits would violate the preserveMetadata contract.
	// Apply ACLs before the final mode update so the restored ACL mask cannot
	// discard special permission bits.
	if err := copyAccessControlList(destination, source); err != nil {
		return err
	}
	if err := copyExtendedAttributes(destination, source); err != nil {
		return err
	}
	// ACL restoration may update the mode mask, so apply the complete source
	// mode last, including setuid, setgid, and sticky bits.
	mode := uint32(stat.Mode) & 0o7777
	if err := unix.Fchmod(int(destination.Fd()), mode); err != nil {
		return fmt.Errorf("preserve replacement permissions: %w", err)
	}
	return nil
}

func restoreReplacementMode(destination *os.File, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("inspect replacement mode: unsupported stat type %T", info.Sys())
	}
	want := uint32(stat.Mode) & 0o7777
	if err := unix.Fchmod(int(destination.Fd()), want); err != nil {
		return fmt.Errorf("preserve replacement permissions: %w", err)
	}
	var actual unix.Stat_t
	if err := unix.Fstat(int(destination.Fd()), &actual); err != nil {
		return fmt.Errorf("verify replacement permissions: %w", err)
	}
	if got := uint32(actual.Mode) & 0o7777; got != want {
		return fmt.Errorf("verify replacement permissions: got mode %#o, want %#o", got, want)
	}
	return nil
}

func copyExtendedAttributes(destination, source *os.File) error {
	size, err := unix.Flistxattr(int(source.Fd()), nil)
	if err != nil {
		if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
			return nil
		}
		return fmt.Errorf("list replacement extended attributes: %w", err)
	}
	if size == 0 {
		return nil
	}
	names := make([]byte, size)
	size, err = unix.Flistxattr(int(source.Fd()), names)
	if err != nil {
		return fmt.Errorf("list replacement extended attributes: %w", err)
	}
	for _, rawName := range bytes.Split(names[:size], []byte{0}) {
		if len(rawName) == 0 {
			continue
		}
		name := string(rawName)
		valueSize, err := unix.Fgetxattr(int(source.Fd()), name, nil)
		if err != nil {
			return fmt.Errorf("read replacement extended attribute %q: %w", name, err)
		}
		value := make([]byte, valueSize)
		if valueSize > 0 {
			valueSize, err = unix.Fgetxattr(int(source.Fd()), name, value)
			if err != nil {
				return fmt.Errorf("read replacement extended attribute %q: %w", name, err)
			}
			value = value[:valueSize]
		}
		if err := unix.Fsetxattr(int(destination.Fd()), name, value, 0); err != nil {
			return fmt.Errorf("preserve replacement extended attribute %q: %w", name, err)
		}
	}
	return nil
}
