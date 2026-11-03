package storage

import (
	"fmt"
	"os"
)

// ProtectManagedPath is for application-owned paths, never arbitrary import sources.
func ProtectManagedPath(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || (directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular()) {
		return fmt.Errorf("invalid managed path")
	}
	if directory {
		return restrictPermissions(path, 0700)
	}
	return restrictPermissions(path, 0600)
}
