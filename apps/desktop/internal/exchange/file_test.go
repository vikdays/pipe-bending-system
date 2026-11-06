package exchange

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONReplacesCompleteFileAndRejectsUnsafeTargets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task.json")
	if e := os.WriteFile(path, []byte("old"), 0600); e != nil {
		t.Fatal(e)
	}
	data := []byte("{\"schemaVersion\":1}\n")
	if e := WriteJSON(path, data); e != nil {
		t.Fatal(e)
	}
	saved, e := os.ReadFile(path)
	if e != nil || string(saved) != string(data) {
		t.Fatal("partial/replacement failure", e)
	}
	for _, bad := range []string{filepath.Join(dir, "application.sqlite3"), filepath.Join(dir, "missing", "task.json"), dir, "relative.json"} {
		if e = WriteJSON(bad, data); e == nil {
			t.Fatal("unsafe destination", bad)
		}
	}
	directory := filepath.Join(dir, "directory.json")
	os.Mkdir(directory, 0700)
	if e = WriteJSON(directory, data); e == nil {
		t.Fatal("directory accepted")
	}
	link := filepath.Join(dir, "link.json")
	if os.Symlink(path, link) == nil {
		if e = WriteJSON(link, []byte("changed")); e == nil {
			t.Fatal("symlink accepted")
		}
	}
	saved, e = os.ReadFile(path)
	if e != nil || string(saved) != string(data) {
		t.Fatal("failed export damaged existing file")
	}
	temps, _ := filepath.Glob(filepath.Join(dir, ".pipe-task-*.tmp"))
	if len(temps) != 0 {
		t.Fatal("staged files left behind")
	}
}
