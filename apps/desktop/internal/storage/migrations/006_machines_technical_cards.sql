CREATE TABLE machines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    model TEXT NOT NULL,
    serial_number TEXT NOT NULL DEFAULT '',
    inventory_number TEXT NOT NULL DEFAULT ''
);
INSERT INTO machines(id,name,model) VALUES ('m_gt532','ГТ532','ГТ532'),('m_gt1022','ГТ1022','ГТ1022');

-- Extend the document type while preserving every original and dependent link.
CREATE TABLE document_files_next (
    id TEXT PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE CHECK (length(sha256)=64),
    kind TEXT NOT NULL CHECK (kind IN ('xlsx','pdf','docx')),
    original_name TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    size_bytes INTEGER NOT NULL CHECK (size_bytes>0),
    page_count INTEGER CHECK (page_count IS NULL OR page_count>0),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id)
);
INSERT INTO document_files_next SELECT * FROM document_files;
DROP TABLE document_files;
ALTER TABLE document_files_next RENAME TO document_files;

CREATE TABLE technical_cards (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES document_files(id),
    machine_id TEXT NOT NULL REFERENCES machines(id),
    owner_id TEXT NOT NULL REFERENCES users(id),
    parsed_json TEXT NOT NULL CHECK (json_valid(parsed_json)),
    created_at TEXT NOT NULL,
    UNIQUE(document_id,machine_id)
);

ALTER TABLE technological_configurations ADD COLUMN machine_id TEXT REFERENCES machines(id);
ALTER TABLE technological_configurations ADD COLUMN technical_card_id TEXT REFERENCES technical_cards(id);
ALTER TABLE technological_configurations ADD COLUMN source_key TEXT;
ALTER TABLE technological_configurations ADD COLUMN source_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(source_json));
ALTER TABLE technological_configurations ADD COLUMN needs_review INTEGER NOT NULL DEFAULT 0 CHECK(needs_review IN(0,1));
ALTER TABLE technological_configurations ADD COLUMN bend_type TEXT NOT NULL DEFAULT '';
ALTER TABLE technological_configurations ADD COLUMN target_angle TEXT NOT NULL DEFAULT '';
ALTER TABLE technological_configurations ADD COLUMN min_pipe_length_mm REAL;
CREATE UNIQUE INDEX configuration_source_idx ON technological_configurations(technical_card_id,source_key);
UPDATE technological_configurations SET machine_id=CASE replace(replace(upper(machine),' ',''),'-','') WHEN 'ГТ532' THEN 'm_gt532' WHEN 'ГТ1022' THEN 'm_gt1022' END;

ALTER TABLE manufacturing_tasks ADD COLUMN machine_id TEXT REFERENCES machines(id);
ALTER TABLE manufacturing_tasks ADD COLUMN machine_snapshot_json TEXT CHECK(machine_snapshot_json IS NULL OR json_valid(machine_snapshot_json));
UPDATE manufacturing_tasks SET machine_id=(SELECT machine_id FROM technological_configurations WHERE id=configuration_id) WHERE status='draft';
