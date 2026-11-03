CREATE TABLE document_files (
    id TEXT PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE CHECK (length(sha256)=64),
    kind TEXT NOT NULL CHECK (kind IN ('xlsx','pdf')),
    original_name TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
    page_count INTEGER CHECK (page_count IS NULL OR page_count > 0),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id)
);

CREATE TABLE certificate_documents (
    certificate_id TEXT NOT NULL REFERENCES certificates(id),
    document_id TEXT NOT NULL REFERENCES document_files(id),
    display_name TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
    PRIMARY KEY (certificate_id,document_id)
);

CREATE TABLE document_link_events (
    id TEXT PRIMARY KEY,
    certificate_id TEXT NOT NULL REFERENCES certificates(id),
    document_id TEXT NOT NULL REFERENCES document_files(id),
    action TEXT NOT NULL CHECK (action IN ('attach','detach')),
    actor_id TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL
);

CREATE TABLE registry_imports (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES document_files(id),
    original_name TEXT NOT NULL,
    owner_id TEXT NOT NULL REFERENCES users(id),
    configuration_json TEXT NOT NULL CHECK (json_valid(configuration_json)),
    revision INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'prepared' CHECK (status IN ('prepared','committed')),
    result_json TEXT CHECK (result_json IS NULL OR json_valid(result_json)),
    selection_json TEXT,
    created_at TEXT NOT NULL,
    committed_at TEXT
);

CREATE TABLE registry_import_rows (
    id TEXT PRIMARY KEY,
    import_id TEXT NOT NULL REFERENCES registry_imports(id),
    source_row INTEGER NOT NULL,
    input_json TEXT NOT NULL CHECK (json_valid(input_json)),
    issues_json TEXT NOT NULL CHECK (json_valid(issues_json)),
    pipe_id TEXT REFERENCES pipes(id),
    UNIQUE(import_id,source_row)
);

CREATE INDEX registry_imports_owner_idx ON registry_imports(owner_id,created_at);
CREATE INDEX registry_import_rows_parent_idx ON registry_import_rows(import_id);
CREATE INDEX certificate_documents_document_idx ON certificate_documents(document_id);
