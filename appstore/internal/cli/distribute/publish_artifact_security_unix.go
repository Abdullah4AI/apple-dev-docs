//go:build !windows

package distribute

import (
	"fmt"
	"os"
	"reflect"
)

func validateProtectedPublishArtifactPlatform(_ *os.File, info os.FileInfo) error {
	stat := reflect.ValueOf(info.Sys())
	if stat.Kind() == reflect.Pointer {
		if stat.IsNil() {
			return fmt.Errorf("cannot inspect artifact ownership")
		}
		stat = stat.Elem()
	}
	uid, ok := unsignedStatField(stat, "Uid")
	if !ok || uid != uint64(os.Geteuid()) {
		return fmt.Errorf("must be owned by the current user")
	}
	nlink, ok := unsignedStatField(stat, "Nlink")
	if !ok {
		return fmt.Errorf("cannot inspect artifact link count")
	}
	if nlink != 1 {
		return fmt.Errorf("must not have multiple hard links")
	}
	return nil
}

func syncPublishArtifactDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open publish artifact directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync publish artifact directory: %w", err)
	}
	return nil
}

func unsignedStatField(stat reflect.Value, name string) (uint64, bool) {
	if stat.Kind() != reflect.Struct {
		return 0, false
	}
	field := stat.FieldByName(name)
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := field.Int()
		if value < 0 {
			return 0, false
		}
		return uint64(value), true
	default:
		return 0, false
	}
}
