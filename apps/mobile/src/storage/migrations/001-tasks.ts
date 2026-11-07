// Source documents and the desktop database never enter this device database.
export const taskSchema = `
CREATE TABLE imported_packages (
  id TEXT PRIMARY KEY NOT NULL CHECK(length(id) = 64),
  format TEXT NOT NULL CHECK(format = 'pipe-bending.tasks'),
  schema_version INTEGER NOT NULL CHECK(schema_version = 1),
  file_name TEXT NOT NULL,
  task_count INTEGER NOT NULL CHECK(task_count BETWEEN 1 AND 1000),
  imported_at TEXT NOT NULL
);
CREATE TABLE mobile_tasks (
  id TEXT PRIMARY KEY NOT NULL,
  revision INTEGER NOT NULL CHECK(revision > 0),
  number TEXT NOT NULL,
  queue_position INTEGER NOT NULL UNIQUE CHECK(queue_position > 0),
  content_json TEXT NOT NULL CHECK(json_valid(content_json)),
  content_sha256 TEXT NOT NULL CHECK(length(content_sha256) = 64),
  first_package_id TEXT NOT NULL REFERENCES imported_packages(id),
  imported_at TEXT NOT NULL
);
CREATE TABLE imported_package_tasks (
  package_id TEXT NOT NULL REFERENCES imported_packages(id),
  task_id TEXT NOT NULL REFERENCES mobile_tasks(id),
  task_revision INTEGER NOT NULL CHECK(task_revision > 0),
  PRIMARY KEY(package_id, task_id)
);
CREATE TRIGGER mobile_task_content_immutable BEFORE UPDATE ON mobile_tasks
BEGIN SELECT RAISE(ABORT, 'imported task is immutable'); END;
PRAGMA user_version = 1;
`;
