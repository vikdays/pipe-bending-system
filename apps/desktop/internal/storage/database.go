package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open owns an application-specific directory, never the source checkout or USB drive.
// Unix modes and Windows DACLs are restricted to the current OS user.
func Open(directory string) (*sql.DB, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("invalid database directory")
	}
	if err := restrictPermissions(directory, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, "application.sqlite3")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid database file")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = file.Chmod(0600); err != nil {
		file.Close()
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	if err = restrictPermissions(path, 0600); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := databaseURI(filepath.ToSlash(absolute))
	db, err := sql.Open("sqlite", uri+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(DELETE)&_pragma=secure_delete(ON)")
	if err != nil {
		return nil, err
	}
	// A single connection also keeps connection-local SQLite pragmas predictable.
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err == nil {
		err = migrate(db)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func databaseURI(path string) string {
	// A drive letter must be a URI path (/C:/...), never a file://C: authority.
	if len(path) >= 2 && path[1] == ':' && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func migrate(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	files := []string{"migrations/001_auth.sql", "migrations/002_shorter_lockout.sql"}
	if version > len(files) {
		return fmt.Errorf("database version is newer than this application")
	}
	for index := version; index < len(files); index++ {
		source, err := migrations.ReadFile(files[index])
		if err != nil {
			return err
		}
		if _, err = tx.Exec(string(source)); err != nil {
			return err
		}
		if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", index+1)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
