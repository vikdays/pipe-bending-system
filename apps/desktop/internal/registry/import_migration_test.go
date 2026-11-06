package registry

import (
	"changeme/internal/storage"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestImportMigrationPreservesRegistryAndTaskSnapshot(t *testing.T) {
	dir := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(dir, "application.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	for i, file := range []string{"001_auth.sql", "002_shorter_lockout.sql", "003_pipe_registry.sql"} {
		data, err := os.ReadFile(filepath.Join("..", "storage", "migrations", file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatal(err)
		}
		db.Exec(fmt.Sprintf("PRAGMA user_version=%d", i+1))
	}
	if _, e = db.Exec("PRAGMA foreign_keys=ON; INSERT INTO users(id,username,password_hash,role,created_at) VALUES('actor','legacy.user','unused','admin',1)"); e != nil {
		t.Fatal(e)
	}
	r := New(db)
	pipe, e := r.CreatePipe(completeInput(), "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec("INSERT INTO manufacturing_tasks(id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,created_at,created_by) VALUES('legacy-task','TEST',?,1,?,'TEST','Fictional','9','2026-01-01','actor')", pipe.ID, jsonText(pipe))
	if e != nil {
		t.Fatal(e)
	}
	db.Close()
	reopened, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	r = New(reopened)
	pipes, e := r.ListPipes()
	if e != nil || len(pipes) != 1 || pipes[0].ID != pipe.ID || pipes[0].AttributesJSON != pipe.AttributesJSON {
		t.Fatal("migration changed registry", e)
	}
	tasks, e := r.ListTasks()
	if e != nil || len(tasks) != 1 || tasks[0].ID != "legacy-task" || tasks[0].PipeSnapshot.ID != pipe.ID || tasks[0].QueuePosition != 1 || tasks[0].Revision != 1 || tasks[0].ConfigurationSnapshot != nil {
		t.Fatal("migration changed task", e)
	}
}

func TestMachineMigrationPreservesAttachmentsAndPreparedTask(t *testing.T) {
	dir := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(dir, "application.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	for i, file := range []string{"001_auth.sql", "002_shorter_lockout.sql", "003_pipe_registry.sql", "004_registry_imports_documents.sql", "005_task_preparation.sql"} {
		data, e := os.ReadFile(filepath.Join("..", "storage", "migrations", file))
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(string(data)); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(fmt.Sprintf("PRAGMA user_version=%d", i+1)); e != nil {
			t.Fatal(e)
		}
	}
	_, e = db.Exec(`PRAGMA foreign_keys=ON;
 INSERT INTO users(id,username,password_hash,role,created_at) VALUES('actor','legacy.user','unused','admin',1);
 INSERT INTO document_files(id,sha256,kind,original_name,relative_path,size_bytes,created_at,created_by) VALUES('pdf','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','pdf','fictional.pdf','pdf/test.pdf',10,'test','actor'),('xlsx','bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','xlsx','fictional.xlsx','xlsx/test.xlsx',10,'test','actor');
 INSERT INTO certificates(id,reference,created_at,created_by) VALUES('cert','Fictional certificate','test','actor');
 INSERT INTO certificate_documents(certificate_id,document_id,display_name) VALUES('cert','pdf','Fictional original');
 INSERT INTO document_link_events(id,certificate_id,document_id,action,actor_id,created_at) VALUES('event','cert','pdf','attach','actor','test');
 INSERT INTO registry_imports(id,document_id,original_name,owner_id,configuration_json,created_at) VALUES('import','xlsx','fictional.xlsx','actor','{}','test');
 INSERT INTO pipes(id,inventory_number,created_at,updated_at,created_by,updated_by) VALUES('pipe','TEST','test','test','actor','actor');
 INSERT INTO manufacturing_tasks(id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,status,created_at,created_by,queue_position,configuration_snapshot_json) VALUES('task','FROZEN','pipe',1,'{"id":"pipe"}','TEST','ГТ532','1','ready','test','actor',1,'{"name":"Frozen old version"}');`)
	if e != nil {
		t.Fatal(e)
	}
	db.Close()
	db, e = storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var count int
	if e = db.QueryRow("PRAGMA foreign_keys").Scan(&count); e != nil || count != 1 {
		t.Fatal("foreign key enforcement not restored", count, e)
	}
	for _, query := range []string{"SELECT COUNT(*) FROM document_files", "SELECT COUNT(*) FROM machines"} {
		if e = db.QueryRow(query).Scan(&count); e != nil || count != 2 {
			t.Fatal(query, count, e)
		}
	}
	for _, table := range []string{"certificate_documents", "document_link_events", "registry_imports"} {
		if e = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); e != nil || count != 1 {
			t.Fatal(table, count, e)
		}
	}
	rows, e := db.Query("PRAGMA foreign_key_check")
	if e != nil {
		t.Fatal(e)
	}
	if rows.Next() {
		t.Fatal("broken foreign keys")
	}
	rows.Close()
	task, e := New(db).GetTask("task")
	if e != nil || task.Status != "ready" || task.MachineSnapshot != nil || task.ConfigurationSnapshot.Name != "Frozen old version" {
		t.Fatal("migration changed frozen task", e)
	}
	if _, e = db.Exec("UPDATE manufacturing_tasks SET notes='changed' WHERE id='task'"); e == nil {
		t.Fatal("prepared trigger removed")
	}
}
