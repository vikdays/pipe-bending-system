package registry

import (
	"changeme/internal/storage"
	"testing"
)

func TestImportMigrationPreservesRegistryAndTaskSnapshot(t *testing.T) {
	r, db, dir := fixture(t)
	pipe, e := r.CreatePipe(completeInput(), "actor")
	if e != nil {
		t.Fatal(e)
	}
	task, e := r.CreateTask(TaskInput{Number: "TEST", PipeID: pipe.ID, PipeRevision: 1, BendNumber: "TEST", Machine: "Fictional", TargetAngle: "9"}, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec("DROP TABLE registry_import_rows; DROP TABLE registry_imports; DROP TABLE document_link_events; DROP TABLE certificate_documents; DROP TABLE document_files; PRAGMA user_version=3")
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
	if e != nil || len(tasks) != 1 || tasks[0].ID != task.ID || tasks[0].PipeSnapshot.ID != pipe.ID {
		t.Fatal("migration changed task", e)
	}
}
