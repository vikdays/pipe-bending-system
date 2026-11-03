package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDatabaseMigrationAndRestart(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO users(id,username,password_hash,role,created_at) VALUES('test-id','test-user','test-hash','admin',1)"); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count, version int
	if err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatal(version, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "application.sqlite3"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("file mode %o", info.Mode().Perm())
		}
		info, err = os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0700 {
			t.Fatalf("directory mode %o", info.Mode().Perm())
		}
	}
}

func TestUpgradeShortensExistingLockoutsOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	const blockedAt = 1800000000
	if _, err = db.Exec("INSERT INTO auth_limits(scope,failures,window_start,blocked_until) VALUES('global',30,?,?)", blockedAt-60, blockedAt+15*60); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("DROP TABLE registry_import_rows; DROP TABLE registry_imports; DROP TABLE document_link_events; DROP TABLE certificate_documents; DROP TABLE document_files; DROP TABLE manufacturing_tasks; DROP TABLE pipe_certificates; DROP TABLE certificates; DROP TABLE pipes; PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		db, err = Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		var failures int
		var start, until int64
		err = db.QueryRow("SELECT failures,window_start,blocked_until FROM auth_limits WHERE scope='global'").Scan(&failures, &start, &until)
		db.Close()
		if err != nil || failures != 30 || start != blockedAt-60 || until != blockedAt+5*60 {
			t.Fatalf("failures=%d start=%d until=%d err=%v", failures, start, until, err)
		}
	}
}

func TestRefusesNewerDatabaseAndSymbolicLink(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("PRAGMA user_version=999"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db, err = Open(dir); err == nil {
		db.Close()
		t.Fatal("newer schema accepted")
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if err = os.Symlink(dir, linked); err != nil {
		t.Skip("symlinks unavailable")
	}
	if db, err = Open(linked); err == nil {
		db.Close()
		t.Fatal("symbolic link accepted")
	}
}

func TestSQLiteURIHandlesWindowsDriveAndSpaces(t *testing.T) {
	for path, expected := range map[string]string{
		"C:/Users/Example User/AppData/application.sqlite3": "file:///C:/Users/Example%20User/AppData/application.sqlite3",
		"/Users/Example User/application.sqlite3":           "file:///Users/Example%20User/application.sqlite3",
	} {
		if actual := databaseURI(path); actual != expected {
			t.Fatalf("URI: got %q, want %q", actual, expected)
		}
	}
}
