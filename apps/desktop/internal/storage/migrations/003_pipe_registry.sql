CREATE TABLE pipes (
    id TEXT PRIMARY KEY,
    inventory_number TEXT NOT NULL DEFAULT '' UNIQUE,
    factory_number TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL DEFAULT '',
    diameter_mm REAL CHECK (diameter_mm IS NULL OR diameter_mm > 0),
    wall_thickness_mm REAL CHECK (wall_thickness_mm IS NULL OR wall_thickness_mm > 0),
    length_mm REAL CHECK (length_mm IS NULL OR length_mm > 0),
    mass_kg REAL CHECK (mass_kg IS NULL OR mass_kg > 0),
    strength_class TEXT NOT NULL DEFAULT '',
    steel_grade TEXT NOT NULL DEFAULT '',
    heat_number TEXT NOT NULL DEFAULT '',
    carbon_equivalent_pct REAL,
    pipe_specification TEXT NOT NULL DEFAULT '',
    coating_type TEXT NOT NULL DEFAULT '',
    coating_specification TEXT NOT NULL DEFAULT '',
    test_pressure_mpa REAL CHECK (test_pressure_mpa IS NULL OR test_pressure_mpa > 0),
    min_operating_temp_c REAL,
    incoming_inspection_reference TEXT NOT NULL DEFAULT '',
    allowed_category54 TEXT NOT NULL DEFAULT '',
    allowed_category74 TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    attributes_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(attributes_json) AND json_type(attributes_json) = 'object'),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    updated_by TEXT NOT NULL REFERENCES users(id),
    CHECK (wall_thickness_mm IS NULL OR diameter_mm IS NULL OR 2 * wall_thickness_mm < diameter_mm)
);

CREATE TABLE certificates (
    id TEXT PRIMARY KEY,
    reference TEXT NOT NULL,
    number TEXT NOT NULL DEFAULT '',
    issued_on TEXT NOT NULL DEFAULT '',
    page_count INTEGER CHECK (page_count IS NULL OR page_count > 0),
    issuer TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id)
);

CREATE TABLE pipe_certificates (
    pipe_id TEXT NOT NULL REFERENCES pipes(id) ON DELETE RESTRICT,
    certificate_id TEXT NOT NULL REFERENCES certificates(id) ON DELETE RESTRICT,
    PRIMARY KEY (pipe_id, certificate_id)
);

CREATE TABLE manufacturing_tasks (
    id TEXT PRIMARY KEY,
    number TEXT NOT NULL UNIQUE,
    pipe_id TEXT NOT NULL REFERENCES pipes(id) ON DELETE RESTRICT,
    pipe_revision INTEGER NOT NULL,
    pipe_snapshot_json TEXT NOT NULL CHECK (json_valid(pipe_snapshot_json) AND json_type(pipe_snapshot_json) = 'object'),
    bend_number TEXT NOT NULL,
    machine TEXT NOT NULL,
    target_angle TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'ready', 'sent', 'completed')),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id)
);

CREATE INDEX pipes_factory_number_idx ON pipes(factory_number);
CREATE INDEX pipes_heat_number_idx ON pipes(heat_number);
CREATE INDEX pipe_certificates_certificate_idx ON pipe_certificates(certificate_id);
CREATE INDEX manufacturing_tasks_pipe_idx ON manufacturing_tasks(pipe_id);

