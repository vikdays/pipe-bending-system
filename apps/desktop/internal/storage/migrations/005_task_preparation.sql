CREATE TABLE technological_configurations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    machine TEXT NOT NULL,
    technical_card_reference TEXT NOT NULL,
    diameter_mm REAL NOT NULL CHECK (diameter_mm > 0),
    parameters_json TEXT NOT NULL CHECK (json_valid(parameters_json) AND json_type(parameters_json)='array'),
    notes TEXT NOT NULL DEFAULT '',
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    updated_by TEXT NOT NULL REFERENCES users(id)
);

ALTER TABLE manufacturing_tasks ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0);
ALTER TABLE manufacturing_tasks ADD COLUMN queue_position INTEGER NOT NULL DEFAULT 0;
ALTER TABLE manufacturing_tasks ADD COLUMN bend_type TEXT NOT NULL DEFAULT '';
ALTER TABLE manufacturing_tasks ADD COLUMN notes TEXT NOT NULL DEFAULT '';
ALTER TABLE manufacturing_tasks ADD COLUMN configuration_id TEXT REFERENCES technological_configurations(id);
ALTER TABLE manufacturing_tasks ADD COLUMN configuration_revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE manufacturing_tasks ADD COLUMN configuration_snapshot_json TEXT CHECK (configuration_snapshot_json IS NULL OR (json_valid(configuration_snapshot_json) AND json_type(configuration_snapshot_json)='object'));
ALTER TABLE manufacturing_tasks ADD COLUMN snapshot_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE manufacturing_tasks ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE manufacturing_tasks ADD COLUMN updated_by TEXT REFERENCES users(id);
ALTER TABLE manufacturing_tasks ADD COLUMN prepared_at TEXT;
ALTER TABLE manufacturing_tasks ADD COLUMN prepared_by TEXT REFERENCES users(id);

UPDATE manufacturing_tasks AS t SET
    queue_position=(SELECT COUNT(*) FROM manufacturing_tasks AS p WHERE p.created_at<t.created_at OR (p.created_at=t.created_at AND p.id<=t.id)),
    updated_at=created_at,updated_by=created_by;
CREATE UNIQUE INDEX manufacturing_tasks_queue_idx ON manufacturing_tasks(queue_position);
CREATE INDEX manufacturing_tasks_configuration_idx ON manufacturing_tasks(configuration_id);

CREATE TABLE task_creation_requests (
    id TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL REFERENCES users(id),
    payload_hash TEXT NOT NULL,
    task_ids_json TEXT NOT NULL CHECK (json_valid(task_ids_json)),
    created_at TEXT NOT NULL
);

-- Prepared content is immutable in this first version, including direct SQL writes.
CREATE TRIGGER prepared_task_no_update BEFORE UPDATE ON manufacturing_tasks
WHEN OLD.status <> 'draft'
BEGIN SELECT RAISE(ABORT,'prepared task is immutable'); END;
CREATE TRIGGER prepared_task_no_delete BEFORE DELETE ON manufacturing_tasks
WHEN OLD.status <> 'draft'
BEGIN SELECT RAISE(ABORT,'prepared task is immutable'); END;
