CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator')),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at INTEGER NOT NULL
);
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked_at INTEGER
);
CREATE TABLE access_tokens (
    digest BLOB PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);
CREATE TABLE refresh_tokens (
    digest BLOB PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    used_at INTEGER
);
CREATE INDEX access_session ON access_tokens(session_id);
CREATE INDEX refresh_session ON refresh_tokens(session_id);
CREATE TABLE auth_limits (
    scope TEXT PRIMARY KEY,
    failures INTEGER NOT NULL,
    window_start INTEGER NOT NULL,
    blocked_until INTEGER NOT NULL DEFAULT 0
);
