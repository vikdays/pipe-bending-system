-- One event represents one file; its members retain individual task identities.
CREATE TABLE exchange_events_next (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('task_export','tasks_export')),
    status TEXT NOT NULL CHECK(status IN ('pending','saved','failed')),
    file_name TEXT NOT NULL,
    sha256 TEXT NOT NULL CHECK(length(sha256)=64),
    size_bytes INTEGER NOT NULL CHECK(size_bytes>0),
    created_at TEXT NOT NULL,
    completed_at TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL REFERENCES users(id)
);
INSERT INTO exchange_events_next SELECT id,kind,status,file_name,sha256,size_bytes,created_at,completed_at,error_message,actor_id FROM exchange_events;
CREATE TEMP TABLE old_exchange_members AS
 SELECT e.id AS event_id,e.task_id,e.task_uuid,e.task_revision,e.task_number,t.queue_position
 FROM exchange_events e JOIN manufacturing_tasks t ON t.id=e.task_id;
DROP TABLE exchange_events;
ALTER TABLE exchange_events_next RENAME TO exchange_events;
CREATE TABLE exchange_event_tasks (
    event_id TEXT NOT NULL REFERENCES exchange_events(id),
    task_id TEXT NOT NULL REFERENCES manufacturing_tasks(id),
    task_uuid TEXT NOT NULL,
    task_revision INTEGER NOT NULL CHECK(task_revision>0),
    task_number TEXT NOT NULL,
    queue_position INTEGER NOT NULL,
    PRIMARY KEY(event_id,task_id),
    UNIQUE(event_id,queue_position)
);
INSERT INTO exchange_event_tasks SELECT * FROM old_exchange_members;
DROP TABLE old_exchange_members;
CREATE INDEX exchange_event_tasks_task_idx ON exchange_event_tasks(task_id,event_id);
