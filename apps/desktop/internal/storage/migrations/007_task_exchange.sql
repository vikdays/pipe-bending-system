-- Local IDs remain unchanged. A separate stable UUID identifies a task on other devices.
CREATE TABLE task_exchange_ids (
    task_id TEXT PRIMARY KEY REFERENCES manufacturing_tasks(id) ON DELETE CASCADE,
    task_uuid TEXT NOT NULL UNIQUE CHECK(length(task_uuid)=36)
);
CREATE TABLE exchange_events (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES manufacturing_tasks(id),
    task_uuid TEXT NOT NULL,
    task_revision INTEGER NOT NULL CHECK(task_revision>0),
    task_number TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind='task_export'),
    status TEXT NOT NULL CHECK(status IN ('pending','saved','failed')),
    file_name TEXT NOT NULL,
    sha256 TEXT NOT NULL CHECK(length(sha256)=64),
    size_bytes INTEGER NOT NULL CHECK(size_bytes>0),
    created_at TEXT NOT NULL,
    completed_at TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL REFERENCES users(id)
);
CREATE INDEX exchange_events_task_idx ON exchange_events(task_id,created_at);
