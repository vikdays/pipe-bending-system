package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"changeme/internal/apperror"
	"changeme/internal/auth"
	"changeme/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func requireError(t *testing.T, err error, code apperror.Code) {
	t.Helper()
	if err == nil {
		t.Fatal("operation succeeded instead of returning an error")
	}
	payload := apperror.Describe(err)
	if payload.Code != code {
		t.Fatalf("got %+v, want %s", payload, code)
	}
	// This is the SDK's error envelope: runtime.js exposes its JSON cause to the client.
	data, marshalErr := json.Marshal(&application.CallError{Message: err.Error(), Cause: json.RawMessage(apperror.Marshal(err)), Kind: application.RuntimeError})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var envelope struct {
		Message string           `json:"message"`
		Cause   apperror.Payload `json:"cause"`
	}
	if marshalErr = json.Unmarshal(data, &envelope); marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if envelope.Cause != payload || envelope.Message != payload.Message {
		t.Fatalf("invalid Wails envelope: %s", data)
	}
}

func TestAuthMethodsReturnPublicCodes(t *testing.T) {
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
	password := "Fictional password phrase 2026!"
	_, err = service.Bootstrap("!", password)
	requireError(t, err, apperror.ValidationError)
	_, err = service.Bootstrap("supervisor", "short")
	requireError(t, err, apperror.ValidationError)
	if _, err = service.Bootstrap("supervisor", password); err != nil {
		t.Fatal(err)
	}
	_, err = service.Bootstrap("another.admin", password)
	requireError(t, err, apperror.Conflict)
	_, err = service.Login("unknown.user", password)
	requireError(t, err, apperror.InvalidCredentials)
	_, err = service.Login("supervisor", "wrong")
	requireError(t, err, apperror.InvalidCredentials)
	_, err = service.CurrentUser("")
	requireError(t, err, apperror.Unauthenticated)
	session, err := service.Login("supervisor", password)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateUser(session.AccessToken, "someone", password, "invalid-role")
	requireError(t, err, apperror.ValidationError)
	_, err = service.CreateUser(session.AccessToken, "supervisor", password, "admin")
	requireError(t, err, apperror.Conflict)
	err = service.SetUserEnabled(session.AccessToken, "unknown-id", false)
	requireError(t, err, apperror.NotFound)
	err = service.SetUserEnabled(session.AccessToken, session.User.ID, false)
	requireError(t, err, apperror.ValidationError)
	err = service.ChangePassword(session.AccessToken, "wrong", password+"new")
	requireError(t, err, apperror.InvalidCredentials)
	err = service.ChangePassword(session.AccessToken, password, password)
	requireError(t, err, apperror.ValidationError)
	if _, err = service.CreateUser(session.AccessToken, "operator.one", password, "operator"); err != nil {
		t.Fatal(err)
	}
	operator, err := service.Login("operator.one", password)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ListUsers(operator.AccessToken)
	requireError(t, err, apperror.Forbidden)
	if _, err = db.Exec("INSERT INTO auth_limits(scope,failures,window_start,blocked_until) VALUES('global',30,?,?) ON CONFLICT(scope) DO UPDATE SET blocked_until=excluded.blocked_until", time.Now().Unix(), time.Now().Add(time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}
	_, err = service.Login("supervisor", password)
	requireError(t, err, apperror.RateLimited)
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = service.NeedsSetup()
	requireError(t, err, apperror.InternalError)
}

func TestPublicResultDropsUnexpectedDetailsAndResult(t *testing.T) {
	value, err := publicResult("secret token", fmt.Errorf("SQL failed: secret password"))
	if value != "" || strings.Contains(err.Error(), "secret") {
		t.Fatal("internal details leaked")
	}
	requireError(t, err, apperror.InternalError)
	value, err = publicResult("ok", nil)
	if value != "ok" || err != nil {
		t.Fatal("success changed")
	}
}
