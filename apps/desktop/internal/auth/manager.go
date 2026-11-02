package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"changeme/internal/apperror"
)

const (
	accessLifetime  = 5 * time.Minute
	sessionLifetime = 8 * time.Hour
	limitWindow     = 15 * time.Minute
	lockoutDuration = 5 * time.Minute
)

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,63}$`)

type Manager struct {
	mu        sync.Mutex
	db        *sql.DB
	now       func() time.Time
	dummyHash string
}

func New(db *sql.DB) (*Manager, error) {
	dummy, err := randomToken("d_")
	if err != nil {
		return nil, err
	}
	hash, err := hashPassword(dummy)
	if err != nil {
		return nil, err
	}
	return &Manager{db: db, now: time.Now, dummyHash: hash}, nil
}

func normalizeUsername(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func randomToken(prefix string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func digestToken(token, prefix string) ([]byte, bool) {
	if len(token) != 45 || !strings.HasPrefix(token, prefix) {
		return nil, false
	}
	bytes, err := base64.RawURLEncoding.Strict().DecodeString(token[2:])
	if err != nil || len(bytes) != 32 {
		return nil, false
	}
	digest := sha256.Sum256([]byte(token))
	return digest[:], true
}

func (m *Manager) NeedsSetup() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int
	if err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, ErrStorage
	}
	return count == 0, nil
}

func (m *Manager) Bootstrap(username, password string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int
	if err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return User{}, ErrStorage
	}
	if count != 0 {
		return User{}, ErrConfigured
	}
	username = normalizeUsername(username)
	if !usernamePattern.MatchString(username) {
		return User{}, usernameError()
	}
	if !validPassword(password) {
		return User{}, passwordPolicyError()
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, ErrStorage
	}
	id, err := randomToken("u_")
	if err != nil {
		return User{}, ErrStorage
	}
	tx, err := m.db.Begin()
	if err != nil {
		return User{}, ErrStorage
	}
	defer tx.Rollback()
	// Repeat inside the transaction: multiple application processes must not create two initial admins.
	if err = tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return User{}, ErrStorage
	}
	if count != 0 {
		return User{}, ErrConfigured
	}
	if _, err = tx.Exec("INSERT INTO users(id,username,password_hash,role,created_at) VALUES(?,?,?,'admin',?)", id, username, hash, m.now().Unix()); err != nil {
		return User{}, ErrStorage
	}
	if err = tx.Commit(); err != nil {
		return User{}, ErrStorage
	}
	return User{ID: id, Username: username, Role: "admin", Enabled: true}, nil
}

func usernameError() error {
	return apperror.New(apperror.ValidationError, "Логин: 3–64 символа, латинские буквы, цифры, точка, дефис или подчёркивание; первая — буква.")
}

func (m *Manager) Login(username, password string) (TokenPair, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	username = normalizeUsername(username)
	if !usernamePattern.MatchString(username) || len(password) > 512 {
		return TokenPair{}, ErrCredentials
	}
	now := m.now().Unix()
	if blocked, err := m.isBlocked("global", now); err != nil {
		return TokenPair{}, ErrStorage
	} else if blocked {
		return TokenPair{}, ErrRateLimited
	}
	if blocked, err := m.isBlocked("user:"+username, now); err != nil {
		return TokenPair{}, ErrStorage
	} else if blocked {
		// Do not reveal that this username exists via a distinct lockout response.
		verifyPassword(password, m.dummyHash)
		return TokenPair{}, m.recordLoginFailure(username, false, now)
	}
	var user User
	var hash string
	err := m.db.QueryRow("SELECT id,username,role,enabled,password_hash FROM users WHERE username=?", username).Scan(&user.ID, &user.Username, &user.Role, &user.Enabled, &hash)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return TokenPair{}, ErrStorage
	}
	if !exists {
		hash = m.dummyHash
	}
	matches := verifyPassword(password, hash)
	if !exists || !matches || !user.Enabled {
		return TokenPair{}, m.recordLoginFailure(username, exists, now)
	}
	tx, err := m.db.Begin()
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	defer tx.Rollback()
	var currentHash string
	var enabled bool
	if err = tx.QueryRow("SELECT password_hash,enabled FROM users WHERE id=?", user.ID).Scan(&currentHash, &enabled); err != nil {
		return TokenPair{}, ErrStorage
	}
	if currentHash != hash || !enabled {
		return TokenPair{}, ErrCredentials
	}
	if _, err = tx.Exec("DELETE FROM auth_limits WHERE scope=?", "user:"+username); err != nil {
		return TokenPair{}, ErrStorage
	}
	if _, err = tx.Exec("DELETE FROM auth_sessions WHERE expires_at < ?", now-int64(24*time.Hour/time.Second)); err != nil {
		return TokenPair{}, ErrStorage
	}
	sid, err := randomToken("s_")
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	if _, err = tx.Exec("INSERT INTO auth_sessions(id,user_id,created_at,expires_at) VALUES(?,?,?,?)", sid, user.ID, now, now+int64(sessionLifetime/time.Second)); err != nil {
		return TokenPair{}, ErrStorage
	}
	pair, err := issueTokens(tx, sid, user, now, now+int64(sessionLifetime/time.Second))
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	if err = tx.Commit(); err != nil {
		return TokenPair{}, ErrStorage
	}
	return pair, nil
}

func (m *Manager) recordLoginFailure(username string, accountLimit bool, now int64) error {
	tx, err := m.db.Begin()
	if err != nil {
		return ErrStorage
	}
	defer tx.Rollback()
	if err = recordFailure(tx, "global", now, 30); err != nil {
		return ErrStorage
	}
	// Unknown names and already locked accounts don't create/extend per-account locks.
	if accountLimit {
		if err = recordFailure(tx, "user:"+username, now, 8); err != nil {
			return ErrStorage
		}
	}
	if err = tx.Commit(); err != nil {
		return ErrStorage
	}
	return ErrCredentials
}

func (m *Manager) isBlocked(scope string, now int64) (bool, error) {
	var until int64
	err := m.db.QueryRow("SELECT blocked_until FROM auth_limits WHERE scope=?", scope).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return until > now, err
}

func recordFailure(tx *sql.Tx, scope string, now int64, threshold int) error {
	var failures int
	var start int64
	err := tx.QueryRow("SELECT failures,window_start FROM auth_limits WHERE scope=?", scope).Scan(&failures, &start)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if start == 0 || now-start >= int64(limitWindow/time.Second) {
		failures = 0
		start = now
	}
	failures++
	var until int64
	if failures >= threshold {
		until = now + int64(lockoutDuration/time.Second)
	}
	_, err = tx.Exec(`INSERT INTO auth_limits(scope,failures,window_start,blocked_until) VALUES(?,?,?,?)
        ON CONFLICT(scope) DO UPDATE SET failures=excluded.failures,window_start=excluded.window_start,blocked_until=excluded.blocked_until`, scope, failures, start, until)
	return err
}

func issueTokens(tx *sql.Tx, sid string, user User, now, sessionExpiry int64) (TokenPair, error) {
	access, err := randomToken("a_")
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := randomToken("r_")
	if err != nil {
		return TokenPair{}, err
	}
	accessDigest, _ := digestToken(access, "a_")
	refreshDigest, _ := digestToken(refresh, "r_")
	expiry := now + int64(accessLifetime/time.Second)
	if expiry > sessionExpiry {
		expiry = sessionExpiry
	}
	if _, err = tx.Exec("INSERT INTO access_tokens(digest,session_id,expires_at) VALUES(?,?,?)", accessDigest, sid, expiry); err != nil {
		return TokenPair{}, err
	}
	if _, err = tx.Exec("INSERT INTO refresh_tokens(digest,session_id) VALUES(?,?)", refreshDigest, sid); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{Session: Session{AccessToken: access, AccessExpiresAt: expiry, User: user}, RefreshToken: refresh}, nil
}

func (m *Manager) Refresh(refresh string) (TokenPair, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	digest, ok := digestToken(refresh, "r_")
	if !ok {
		return TokenPair{}, ErrUnauthorized
	}
	tx, err := m.db.Begin()
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	defer tx.Rollback()
	var sid string
	var expiry int64
	var used, revoked sql.NullInt64
	var user User
	err = tx.QueryRow(`SELECT s.id,s.expires_at,s.revoked_at,r.used_at,u.id,u.username,u.role,u.enabled
        FROM refresh_tokens r JOIN auth_sessions s ON s.id=r.session_id JOIN users u ON u.id=s.user_id WHERE r.digest=?`, digest).
		Scan(&sid, &expiry, &revoked, &used, &user.ID, &user.Username, &user.Role, &user.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenPair{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	now := m.now().Unix()
	if used.Valid {
		// A rotated token has been replayed: revoke the entire family, including current access tokens.
		if _, err = tx.Exec("UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,?) WHERE id=?", now, sid); err != nil {
			return TokenPair{}, ErrStorage
		}
		if err = tx.Commit(); err != nil {
			return TokenPair{}, ErrStorage
		}
		return TokenPair{}, ErrUnauthorized
	}
	if revoked.Valid || expiry <= now || !user.Enabled {
		return TokenPair{}, ErrUnauthorized
	}
	result, err := tx.Exec("UPDATE refresh_tokens SET used_at=? WHERE digest=? AND used_at IS NULL", now, digest)
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return TokenPair{}, ErrUnauthorized
	}
	pair, err := issueTokens(tx, sid, user, now, expiry)
	if err != nil {
		return TokenPair{}, ErrStorage
	}
	if err = tx.Commit(); err != nil {
		return TokenPair{}, ErrStorage
	}
	return pair, nil
}

// Logout revokes by refresh token even if its access token has already expired.
func (m *Manager) Logout(refresh string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	digest, ok := digestToken(refresh, "r_")
	if !ok {
		return nil
	}
	_, err := m.db.Exec(`UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,?)
        WHERE id IN (SELECT session_id FROM refresh_tokens WHERE digest=?)`, m.now().Unix(), digest)
	if err != nil {
		return ErrStorage
	}
	return nil
}

type querier interface{ QueryRow(string, ...any) *sql.Row }

func requireUser(q querier, token string, now int64, admin bool) (User, error) {
	digest, ok := digestToken(token, "a_")
	if !ok {
		return User{}, ErrUnauthorized
	}
	var user User
	err := q.QueryRow(`SELECT u.id,u.username,u.role,u.enabled FROM access_tokens a
        JOIN auth_sessions s ON s.id=a.session_id JOIN users u ON u.id=s.user_id
        WHERE a.digest=? AND a.expires_at>? AND s.expires_at>? AND s.revoked_at IS NULL AND u.enabled=1`, digest, now, now).
		Scan(&user.ID, &user.Username, &user.Role, &user.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthorized
	}
	if err != nil {
		return User{}, ErrStorage
	}
	if admin && user.Role != "admin" {
		return User{}, ErrForbidden
	}
	return user, nil
}

func (m *Manager) CurrentUser(access string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return requireUser(m.db, access, m.now().Unix(), false)
}

func (m *Manager) ListUsers(access string) ([]User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := requireUser(m.db, access, m.now().Unix(), true); err != nil {
		return nil, err
	}
	rows, err := m.db.Query("SELECT id,username,role,enabled FROM users ORDER BY username")
	if err != nil {
		return nil, ErrStorage
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.ID, &user.Username, &user.Role, &user.Enabled); err != nil {
			return nil, ErrStorage
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, ErrStorage
	}
	return users, nil
}

func (m *Manager) CreateUser(access, username, password, role string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := requireUser(m.db, access, m.now().Unix(), true); err != nil {
		return User{}, err
	}
	username = normalizeUsername(username)
	if !usernamePattern.MatchString(username) {
		return User{}, usernameError()
	}
	if role != "admin" && role != "operator" {
		return User{}, apperror.New(apperror.ValidationError, "Выберите роль руководителя работ или машиниста.")
	}
	if !validPassword(password) {
		return User{}, passwordPolicyError()
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, ErrStorage
	}
	id, err := randomToken("u_")
	if err != nil {
		return User{}, ErrStorage
	}
	tx, err := m.db.Begin()
	if err != nil {
		return User{}, ErrStorage
	}
	defer tx.Rollback()
	if _, err = requireUser(tx, access, m.now().Unix(), true); err != nil {
		return User{}, err
	}
	var exists int
	if err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", username).Scan(&exists); err != nil {
		return User{}, ErrStorage
	}
	if exists > 0 {
		return User{}, apperror.New(apperror.Conflict, "Этот логин уже занят.")
	}
	if _, err = tx.Exec("INSERT INTO users(id,username,password_hash,role,created_at) VALUES(?,?,?,?,?)", id, username, hash, role, m.now().Unix()); err != nil {
		return User{}, ErrStorage
	}
	if err = tx.Commit(); err != nil {
		return User{}, ErrStorage
	}
	return User{ID: id, Username: username, Role: role, Enabled: true}, nil
}

func (m *Manager) SetUserEnabled(access, userID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, err := m.db.Begin()
	if err != nil {
		return ErrStorage
	}
	defer tx.Rollback()
	actor, err := requireUser(tx, access, m.now().Unix(), true)
	if err != nil {
		return err
	}
	if actor.ID == userID {
		return apperror.New(apperror.ValidationError, "Нельзя менять блокировку собственной учётной записи.")
	}
	result, err := tx.Exec("UPDATE users SET enabled=? WHERE id=?", enabled, userID)
	if err != nil {
		return ErrStorage
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ErrStorage
	}
	if count == 0 {
		return apperror.New(apperror.NotFound, "Учётная запись не найдена.")
	}
	if count != 1 {
		return ErrStorage
	}
	if !enabled {
		if _, err = tx.Exec("UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,?) WHERE user_id=?", m.now().Unix(), userID); err != nil {
			return ErrStorage
		}
	}
	if err = tx.Commit(); err != nil {
		return ErrStorage
	}
	return nil
}

func (m *Manager) ChangePassword(access, currentPassword, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, err := requireUser(m.db, access, m.now().Unix(), false)
	if err != nil {
		return err
	}
	if !validPassword(newPassword) {
		return passwordPolicyError()
	}
	now := m.now().Unix()
	if blocked, err := m.isBlocked("password:"+user.ID, now); err != nil {
		return ErrStorage
	} else if blocked {
		return ErrRateLimited
	}
	var hash string
	if err = m.db.QueryRow("SELECT password_hash FROM users WHERE id=?", user.ID).Scan(&hash); err != nil {
		return ErrStorage
	}
	if !verifyPassword(currentPassword, hash) {
		tx, err := m.db.Begin()
		if err != nil {
			return ErrStorage
		}
		defer tx.Rollback()
		if err = recordFailure(tx, "password:"+user.ID, now, 8); err != nil {
			return ErrStorage
		}
		if err = tx.Commit(); err != nil {
			return ErrStorage
		}
		return ErrCredentials
	}
	if verifyPassword(newPassword, hash) {
		return apperror.New(apperror.ValidationError, "Новый пароль должен отличаться от текущего.")
	}
	next, err := hashPassword(newPassword)
	if err != nil {
		return ErrStorage
	}
	tx, err := m.db.Begin()
	if err != nil {
		return ErrStorage
	}
	defer tx.Rollback()
	if _, err = requireUser(tx, access, m.now().Unix(), false); err != nil {
		return err
	}
	// Optimistic check also prevents a second process overwriting a concurrent password change.
	result, err := tx.Exec("UPDATE users SET password_hash=? WHERE id=? AND password_hash=?", next, user.ID, hash)
	if err != nil {
		return ErrStorage
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return ErrUnauthorized
	}
	if _, err = tx.Exec("UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,?) WHERE user_id=?", m.now().Unix(), user.ID); err != nil {
		return ErrStorage
	}
	if _, err = tx.Exec("DELETE FROM auth_limits WHERE scope=?", "password:"+user.ID); err != nil {
		return ErrStorage
	}
	if err = tx.Commit(); err != nil {
		return ErrStorage
	}
	return nil
}
