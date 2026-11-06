package exchange

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteJSON stages in the destination directory so publishing never exposes
// half a JSON file. The OS save dialog has already confirmed replacement.
func WriteJSON(path string, data []byte) error {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".json") {
		return fmt.Errorf("choose an absolute JSON path")
	}
	if info, e := os.Lstat(path); e == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination is not a regular file")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".pipe-task-*.tmp")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	// Recheck the chosen target just before replacement. Do not follow symlinks.
	if info, e := os.Lstat(path); e == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination changed")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return replace(temp, path)
}
