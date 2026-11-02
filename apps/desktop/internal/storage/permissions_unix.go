//go:build !windows

package storage

import "os"

func restrictPermissions(path string, mode os.FileMode) error { return os.Chmod(path, mode) }
