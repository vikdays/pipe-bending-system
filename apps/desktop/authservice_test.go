package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"changeme/internal/auth"
	"changeme/internal/storage"
)

func TestNativeRefreshIsNotExportedAndNewFacadeRequiresLogin(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manager, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	service := newDesktopAuthService(manager)
	if _, err = service.Bootstrap("supervisor", "Fictional password phrase 2026!"); err != nil {
		t.Fatal(err)
	}
	session, err := service.Login("supervisor", "Fictional password phrase 2026!")
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), service.refresh) {
		t.Fatal("native refresh leaked")
	}
	if _, err = service.Refresh(); err != nil {
		t.Fatal(err)
	}
	if _, err = newDesktopAuthService(manager).Refresh(); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatal("new application restored without credentials")
	}
	if err = service.Logout(); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.CurrentUser(session.AccessToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatal("logout failed to revoke access")
	}
}
