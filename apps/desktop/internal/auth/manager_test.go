package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"changeme/internal/storage"
)

const testPassword = "Fictional password phrase 2026!"

func fixture(t *testing.T) (*Manager, *sql.DB, TokenPair) {
	t.Helper()
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	manager, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) }
	if _, err = manager.Bootstrap("supervisor", testPassword); err != nil {
		t.Fatal(err)
	}
	pair, err := manager.Login("supervisor", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	return manager, db, pair
}

func TestBootstrapIsOneTimeAndPasswordsAreSalted(t *testing.T) {
	m, db, pair := fixture(t)
	if needed, err := m.NeedsSetup(); err != nil || needed {
		t.Fatalf("setup=%v %v", needed, err)
	}
	if _, err := m.Bootstrap("intruder", testPassword); !errors.Is(err, ErrConfigured) {
		t.Fatal(err)
	}
	user, err := m.CreateUser(pair.Session.AccessToken, "operator.one", testPassword, "operator")
	if err != nil {
		t.Fatal(err)
	}
	var first, second string
	db.QueryRow("SELECT password_hash FROM users WHERE username='supervisor'").Scan(&first)
	db.QueryRow("SELECT password_hash FROM users WHERE id=?", user.ID).Scan(&second)
	if first == second || !strings.HasPrefix(first, passwordPrefix) || strings.Contains(first, testPassword) || !verifyPassword(testPassword, first) {
		t.Fatal("passwords must be independently salted Argon2id hashes")
	}
	for _, password := range []string{"demo2026", strings.Repeat("x", 129), "               "} {
		if _, err := m.CreateUser(pair.Session.AccessToken, "bad.password", password, "operator"); err == nil {
			t.Fatal("accepted weak/oversized password")
		}
	}
}

func TestInvalidCredentialsDoNotRevealAccountExistence(t *testing.T) {
	m, _, _ := fixture(t)
	for _, name := range []string{"supervisor", "unknown.user", "x' OR 1=1 --"} {
		if _, err := m.Login(name, "wrong password"); !errors.Is(err, ErrCredentials) {
			t.Fatalf("%q: %v", name, err)
		}
	}
	if _, err := m.Login(" SUPERVISOR ", testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestAccessExpiryRefreshRotationAndReplayRevokesFamily(t *testing.T) {
	m, db, first := fixture(t)
	independent, err := m.Login("supervisor", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	now := m.now().Add(accessLifetime)
	m.now = func() time.Time { return now }
	if _, err = m.CurrentUser(first.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("expired access accepted", err)
	}
	second, err := m.Refresh(first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken || second.Session.AccessToken == first.Session.AccessToken {
		t.Fatal("tokens not rotated")
	}
	if _, err = m.CurrentUser(second.Session.AccessToken); err != nil {
		t.Fatal(err)
	}
	var count int
	digest, _ := digestToken(second.RefreshToken, "r_")
	if err = db.QueryRow("SELECT COUNT(*) FROM refresh_tokens WHERE digest=?", digest).Scan(&count); err != nil || count != 1 {
		t.Fatal("missing refresh digest", err)
	}
	if _, err = m.Refresh(first.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("replay accepted", err)
	}
	if _, err = m.CurrentUser(second.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("family access still valid", err)
	}
	if _, err = m.Refresh(second.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("family refresh still valid", err)
	}
	if _, err = m.Refresh(independent.RefreshToken); err != nil {
		t.Fatal("unrelated session was revoked", err)
	}
}

func TestRefreshDoesNotExtendAbsoluteSessionExpiry(t *testing.T) {
	m, _, first := fixture(t)
	now := m.now().Add(sessionLifetime - time.Minute)
	m.now = func() time.Time { return now }
	second, err := m.Refresh(first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.Session.AccessExpiresAt != now.Add(time.Minute).Unix() {
		t.Fatal("access exceeds session expiry")
	}
	now = now.Add(time.Minute)
	if _, err = m.Refresh(second.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
}

func TestLogoutAndTokenTypes(t *testing.T) {
	m, _, pair := fixture(t)
	if _, err := m.CurrentUser(pair.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("refresh used as access")
	}
	if _, err := m.Refresh(pair.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("access used as refresh")
	}
	forged, _ := randomToken("a_")
	if _, err := m.CurrentUser(forged); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("forged token accepted")
	}
	if err := m.Logout(pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CurrentUser(pair.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("logout did not revoke access")
	}
	if _, err := m.Refresh(pair.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("logout did not revoke refresh")
	}
}

func TestAuthorizationAndDisabledAccountRevokeSessions(t *testing.T) {
	m, _, admin := fixture(t)
	user, err := m.CreateUser(admin.Session.AccessToken, "operator.one", testPassword, "operator")
	if err != nil {
		t.Fatal(err)
	}
	operator, err := m.Login("operator.one", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.ListUsers(""); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	if _, err = m.ListUsers(operator.Session.AccessToken); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if _, err = m.CreateUser(operator.Session.AccessToken, "intruder", testPassword, "admin"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err = m.SetUserEnabled(operator.Session.AccessToken, admin.Session.User.ID, false); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err = m.SetUserEnabled(admin.Session.AccessToken, admin.Session.User.ID, false); err == nil {
		t.Fatal("self disable accepted")
	}
	if err = m.SetUserEnabled(admin.Session.AccessToken, user.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err = m.CurrentUser(operator.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("disabled access accepted")
	}
	if _, err = m.Refresh(operator.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("disabled refresh accepted")
	}
	if _, err = m.Login("operator.one", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal("disabled login accepted")
	}
	if err = m.SetUserEnabled(admin.Session.AccessToken, user.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Refresh(operator.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("re-enabling resurrected old session")
	}
}

func TestPasswordChangeRevokesAllSessions(t *testing.T) {
	m, _, first := fixture(t)
	second, err := m.Login("supervisor", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.ChangePassword(first.Session.AccessToken, "incorrect", testPassword+"new"); !errors.Is(err, ErrCredentials) {
		t.Fatal(err)
	}
	if err = m.ChangePassword(first.Session.AccessToken, testPassword, testPassword+"new"); err != nil {
		t.Fatal(err)
	}
	for _, pair := range []TokenPair{first, second} {
		if _, err = m.CurrentUser(pair.Session.AccessToken); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("old access survived password change")
		}
		if _, err = m.Refresh(pair.RefreshToken); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("old refresh survived password change")
		}
	}
	if _, err = m.Login("supervisor", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal("old password still valid")
	}
	if _, err = m.Login("supervisor", testPassword+"new"); err != nil {
		t.Fatal(err)
	}
}

func TestLoginThrottlePersistsAcrossManagerRestart(t *testing.T) {
	m, db, _ := fixture(t)
	for i := 0; i < 8; i++ {
		if _, err := m.Login("supervisor", "wrong"); !errors.Is(err, ErrCredentials) {
			t.Fatal(err)
		}
	}
	restarted, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	restarted.now = m.now
	if _, err = restarted.Login("supervisor", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal("restart bypassed limiter", err)
	}
	if _, err = restarted.Login("unknown.user", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal("lockout response reveals account existence", err)
	}
	blockedAt := m.now()
	now := blockedAt.Add(5*time.Minute - time.Second)
	restarted.now = func() time.Time { return now }
	if _, err = restarted.Login("supervisor", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal("lockout ended before five minutes", err)
	}
	now = blockedAt.Add(5 * time.Minute)
	restarted.now = func() time.Time { return now }
	if _, err = restarted.Login("supervisor", testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestGlobalThrottleAlsoLimitsUnknownUsernames(t *testing.T) {
	m, db, _ := fixture(t)
	if _, err := db.Exec("INSERT INTO auth_limits(scope,failures,window_start,blocked_until) VALUES('global',29,?,0)", m.now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Login("unknown.user", testPassword); !errors.Is(err, ErrCredentials) {
		t.Fatal(err)
	}
	if _, err := m.Login("supervisor", testPassword); !errors.Is(err, ErrRateLimited) {
		t.Fatal("global throttle bypassed", err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM auth_limits").Scan(&count); err != nil || count != 1 {
		t.Fatal("unknown usernames must not grow the limiter table", count, err)
	}
	blockedAt := m.now()
	now := blockedAt.Add(5*time.Minute - time.Second)
	m.now = func() time.Time { return now }
	if _, err := m.Login("supervisor", testPassword); !errors.Is(err, ErrRateLimited) {
		t.Fatal("global lockout ended before five minutes", err)
	}
	now = blockedAt.Add(5 * time.Minute)
	if _, err := m.Login("supervisor", testPassword); err != nil {
		t.Fatal("global lockout did not end after five minutes", err)
	}
}

func TestConcurrentRefreshHasOnlyOneWinnerAndDetectsReplay(t *testing.T) {
	m, _, pair := fixture(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := m.Refresh(pair.RefreshToken); results <- err }()
	}
	wg.Wait()
	close(results)
	success, denied := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrUnauthorized) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || denied != 1 {
		t.Fatalf("success=%d denied=%d", success, denied)
	}
}

func TestSessionSerializationContainsNoRefreshOrPassword(t *testing.T) {
	_, _, pair := fixture(t)
	value, err := json.Marshal(pair.Session)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{pair.RefreshToken, testPassword, "password_hash", "RefreshToken"} {
		if strings.Contains(string(value), secret) {
			t.Fatal("secret in frontend response")
		}
	}
}

func TestDatabaseContainsDigestsNotRawCredentials(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Bootstrap("supervisor", testPassword); err != nil {
		t.Fatal(err)
	}
	pair, err := m.Login("supervisor", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "application.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{testPassword, pair.Session.AccessToken, pair.RefreshToken} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("raw credential found in SQLite file")
		}
	}
}

func TestConcurrentBootstrapAcrossManagersCreatesOnlyOneAdministrator(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i, manager := range []*Manager{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.Bootstrap([]string{"first.admin", "second.admin"}[i], testPassword)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM users WHERE role='admin'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || success != 1 {
		t.Fatalf("admins=%d successful setups=%d", count, success)
	}
}
