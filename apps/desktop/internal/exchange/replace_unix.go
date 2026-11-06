//go:build !windows

package exchange

import "os"

func replace(source, destination string) error { return os.Rename(source, destination) }
