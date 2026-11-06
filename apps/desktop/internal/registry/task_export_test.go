package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/storage"
	"github.com/google/uuid"
)

func TestTaskExportStableIdentityFrozenSnapshotsAndRestart(t *testing.T) {
	r, input, dir := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, _, e = r.BuildTasksExport([]TaskExportSelection{{ID: draft.ID, Revision: draft.Revision}})
	assertCode(t, e, apperror.Conflict)
	prepared, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	p, first, e := r.BuildTasksExport([]TaskExportSelection{{ID: prepared.ID, Revision: prepared.Revision}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = uuid.Parse(p.Tasks[0].ID); e != nil {
		t.Fatal("not a UUID", e)
	}
	if p.Format != "pipe-bending.tasks" || p.SchemaVersion != 1 || p.Tasks[0].Revision != prepared.Revision || p.Tasks[0].Number != input.Number || p.Tasks[0].TargetAngleDeg != 9.5 || p.Tasks[0].QueuePosition != 1 {
		t.Fatal(p)
	}
	if !reflect.DeepEqual(p.Tasks[0].PipeSnapshot.PipeData, prepared.PipeSnapshot.PipeData) || len(p.Tasks[0].ConfigurationSnapshot.Parameters) != 1 || p.Tasks[0].MachineSnapshot.Model != "ГТ532" {
		t.Fatal("missing frozen data")
	}
	for _, secret := range []string{"unused-test-hash", "password", "accessToken", "refreshToken", "createdBy", "ownerId", "relativePath", "sourceJson", "confirmSourceReview"} {
		if bytes.Contains(first, []byte(secret)) {
			t.Fatal("private field leaked", secret)
		}
	}
	_, _, e = r.BuildTasksExport([]TaskExportSelection{{ID: prepared.ID, Revision: prepared.Revision - 1}})
	assertCode(t, e, apperror.Conflict)
	_, _, e = r.BuildTasksExport([]TaskExportSelection{{ID: "missing", Revision: 1}})
	assertCode(t, e, apperror.Conflict)
	pipe := prepared.PipeSnapshot
	pipe.DiameterMm = number(720)
	if _, e = r.UpdatePipe(pipe.ID, pipe.Revision, PipeInput{PipeData: pipe.PipeData}, "actor"); e != nil {
		t.Fatal(e)
	}
	cfg := *prepared.ConfigurationSnapshot
	cfg.Name = "Изменённый каталог"
	if _, e = r.SaveConfiguration(cfg.ID, cfg.Revision, cfg.ConfigurationInput, "actor"); e != nil {
		t.Fatal(e)
	}
	if _, e = r.db.Exec("UPDATE machines SET name='Изменённый станок' WHERE id=?", prepared.MachineID); e != nil {
		t.Fatal(e)
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	p2, second, e := New(db).BuildTasksExport([]TaskExportSelection{{ID: prepared.ID, Revision: prepared.Revision}})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(first, second) || p2.Tasks[0].ID != p.Tasks[0].ID {
		t.Fatal("repeat/restart changed file or identity")
	}
	events, e := New(db).ListExchangeEvents()
	if e != nil || len(events) != 0 {
		t.Fatal("building a file was logged as a transfer", e)
	}
	if path := os.Getenv("PIPE_TASK_EXAMPLE"); path != "" {
		if e = os.WriteFile(path, first, 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func TestTaskExportRejectsIncompleteSnapshots(t *testing.T) {
	r, input, _ := taskFixture(t)
	d, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	p, e := r.PrepareTask(d.ID, d.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*Task){
		func(v *Task) { v.ConfigurationSnapshot = nil }, func(v *Task) { v.MachineSnapshot = nil }, func(v *Task) { v.PipeSnapshot.LengthMm = nil }, func(v *Task) { v.PipeSnapshot.AttributesJSON = "[]" }, func(v *Task) { v.PreparedAt = "" }, func(v *Task) { v.SnapshotVersion = 0 }, func(v *Task) { v.BendNumber = "" }, func(v *Task) { v.ConfigurationSnapshot.NeedsReview = true },
	} {
		data, _ := json.Marshal(p)
		var next Task
		json.Unmarshal(data, &next)
		mutate(&next)
		_, e = exportTask(next, uuid.NewString())
		assertCode(t, e, apperror.ValidationError)
	}
}

func TestTaskExchangeMigrationPreservesPreparedTask(t *testing.T) {
	r, input, dir := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	before, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	// Emulate the pre-export schema containing an immutable prepared task.
	if _, e = r.db.Exec("DROP TABLE exchange_event_tasks; DROP TABLE exchange_events; DROP TABLE task_exchange_ids; PRAGMA user_version=6"); e != nil {
		t.Fatal(e)
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	repo := New(db)
	after, e := repo.GetTask(before.ID)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("migration changed existing task", e)
	}
	p, _, e := repo.BuildTasksExport([]TaskExportSelection{{ID: after.ID, Revision: after.Revision}})
	if e != nil || p.Tasks[0].Revision != before.Revision {
		t.Fatal("old task not exportable", e)
	}
}

func TestTenTasksExportInQueueOrderAndAtomicValidation(t *testing.T) {
	r, input, dir := taskFixture(t)
	selected := []TaskExportSelection{}
	for i := 0; i < 10; i++ {
		input.Number = fmt.Sprintf("TEST-%02d", i+1)
		d, e := r.CreateTask(input, "actor")
		if e != nil {
			t.Fatal(e)
		}
		ready, e := r.PrepareTask(d.ID, d.Revision, "actor")
		if e != nil {
			t.Fatal(e)
		}
		// Reverse the UI input: the persisted queue, not request order, is authoritative.
		selected = append([]TaskExportSelection{{ID: ready.ID, Revision: ready.Revision}}, selected...)
	}
	input.Number = "DO-NOT-EXPORT-DRAFT"
	if _, e := r.CreateTask(input, "actor"); e != nil {
		t.Fatal(e)
	}
	p, first, e := r.BuildTasksExport(selected)
	if e != nil || len(p.Tasks) != 10 {
		t.Fatal(p, e)
	}
	seen := map[string]bool{}
	for i, v := range p.Tasks {
		if v.QueuePosition != i+1 || v.Number != fmt.Sprintf("TEST-%02d", i+1) || seen[v.ID] || v.Revision != 2 || v.PipeSnapshot.LengthMm == nil || len(v.ConfigurationSnapshot.Parameters) != 1 {
			t.Fatal("order/identity/snapshot lost", v)
		}
		seen[v.ID] = true
	}
	for _, bad := range [][]TaskExportSelection{nil, selected[:9], append(append([]TaskExportSelection{}, selected...), selected[0])} {
		if _, _, e = r.BuildTasksExport(bad); e == nil {
			t.Fatal("invalid/incomplete selection accepted")
		}
	}
	stale := append([]TaskExportSelection{}, selected...)
	stale[0].Revision++
	_, _, e = r.BuildTasksExport(stale)
	assertCode(t, e, apperror.Conflict)
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	_, second, e := New(db).BuildTasksExport(selected)
	if e != nil || !bytes.Equal(first, second) {
		t.Fatal("package changed after restart", e)
	}
	// One broken ready task must reject the entire package, never return 9 tasks.
	if _, e = db.Exec("DROP TRIGGER prepared_task_no_update; UPDATE manufacturing_tasks SET machine_snapshot_json=NULL WHERE id=?", selected[0].ID); e != nil {
		t.Fatal(e)
	}
	_, data, e := New(db).BuildTasksExport(selected)
	assertCode(t, e, apperror.ValidationError)
	if data != nil {
		t.Fatal("partial package returned")
	}
}

func TestPackageJournalMigrationKeepsSingleFileHistory(t *testing.T) {
	r, input, dir := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	ready, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.db.Exec("DROP TABLE exchange_event_tasks; DROP TABLE exchange_events; DROP TABLE task_exchange_ids"); e != nil {
		t.Fatal(e)
	}
	oldSchema, e := os.ReadFile("../storage/migrations/007_task_exchange.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.db.Exec(string(oldSchema)); e != nil {
		t.Fatal(e)
	}
	externalID := uuid.NewString()
	if _, e = r.db.Exec("INSERT INTO task_exchange_ids(task_id,task_uuid) VALUES(?,?)", ready.ID, externalID); e != nil {
		t.Fatal(e)
	}
	if _, e = r.db.Exec("INSERT INTO exchange_events(id,task_id,task_uuid,task_revision,task_number,kind,status,file_name,sha256,size_bytes,created_at,actor_id) VALUES('legacy',?,?,?,?,'task_export','saved','old.json',?,42,'2026-11-04T09:00:00Z','actor')", ready.ID, externalID, ready.Revision, ready.Number, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	if _, e = r.db.Exec("PRAGMA user_version=7"); e != nil {
		t.Fatal(e)
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	repo := New(db)
	events, e := repo.ListExchangeEvents()
	if e != nil || len(events) != 1 || len(events[0].Tasks) != 1 || events[0].Tasks[0].TaskUUID != externalID || events[0].FileName != "old.json" || events[0].Status != "saved" {
		t.Fatal("lost historical export", events, e)
	}
	current, e := repo.GetTask(ready.ID)
	if e != nil || current.Status != "exported" {
		t.Fatal("historical successful export not recognized", e)
	}
	var savedUUID string
	if e = db.QueryRow("SELECT task_uuid FROM task_exchange_ids WHERE task_id=?", ready.ID).Scan(&savedUUID); e != nil || savedUUID != externalID {
		t.Fatal("migration changed task identity", e)
	}
	_, _, e = repo.BuildTasksExport([]TaskExportSelection{{ID: ready.ID, Revision: ready.Revision}})
	assertCode(t, e, apperror.Conflict)
}

func TestExportedStatePersistsAndLeavesNewTasksInQueue(t *testing.T) {
	r, input, dir := taskFixture(t)
	selections := []TaskExportSelection{}
	before := []Task{}
	for i := 0; i < 2; i++ {
		input.Number = fmt.Sprintf("EXPORTED-%d", i)
		draft, e := r.CreateTask(input, "actor")
		if e != nil {
			t.Fatal(e)
		}
		task, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
		if e != nil {
			t.Fatal(e)
		}
		before = append(before, task)
		selections = append(selections, TaskExportSelection{ID: task.ID, Revision: task.Revision})
	}
	p, data, e := r.BuildTasksExport(selections)
	if e != nil {
		t.Fatal(e)
	}
	event, e := r.BeginTasksExport("actor", p.Filename(), p, data)
	if e != nil {
		t.Fatal(e)
	}
	for _, t0 := range before {
		current, e := r.GetTask(t0.ID)
		if e != nil || current.Status != "ready" {
			t.Fatal("pending changed state", e)
		}
	}
	if e = r.FinishTaskExport(event, true); e != nil {
		t.Fatal(e)
	}
	for _, t0 := range before {
		current, e := r.GetTask(t0.ID)
		if e != nil || current.Status != "exported" || current.ExportedAt == "" {
			t.Fatal("saved status missing", e)
		}
		// Apart from the read-model export metadata every field remains identical.
		current.Status = t0.Status
		current.ExportedAt = ""
		if !reflect.DeepEqual(current, t0) {
			t.Fatal("export changed frozen task data")
		}
		if _, e = r.db.Exec("UPDATE manufacturing_tasks SET target_angle='1' WHERE id=?", t0.ID); e == nil {
			t.Fatal("export weakened immutable-row trigger")
		}
	}
	_, _, e = r.BuildTasksExport(selections)
	assertCode(t, e, apperror.Conflict)
	input.Number = "NEW-READY"
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	next, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	repo := New(db)
	all, e := repo.ListTasks()
	if e != nil {
		t.Fatal(e)
	}
	ready, exported := 0, 0
	for _, v := range all {
		if v.Status == "ready" {
			ready++
		}
		if v.Status == "exported" {
			exported++
		}
	}
	if ready != 1 || exported != 2 {
		t.Fatal("queue counts wrong after restart", ready, exported)
	}
	repeatPrepare, e := repo.PrepareTask(before[0].ID, before[0].Revision, "actor")
	if e != nil || repeatPrepare.Status != "exported" {
		t.Fatal("prepare retry regressed export state", e)
	}
	nextPackage, _, e := repo.BuildTasksExport([]TaskExportSelection{{ID: next.ID, Revision: next.Revision}})
	if e != nil || len(nextPackage.Tasks) != 1 || nextPackage.Tasks[0].Number != "NEW-READY" {
		t.Fatal("already-exported tasks entered the next package", e)
	}
}

func TestOldRevisionAndUnconfirmedExportsDoNotMarkCurrentTaskExported(t *testing.T) {
	r, input, _ := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	task, e := r.PrepareTask(draft.ID, draft.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	selected := []TaskExportSelection{{ID: task.ID, Revision: task.Revision}}
	p, data, e := r.BuildTasksExport(selected)
	if e != nil {
		t.Fatal(e)
	}
	old := p
	old.Tasks = append([]ExportedTask{}, p.Tasks...)
	old.Tasks[0].Revision--
	event, e := r.BeginTasksExport("actor", "old.json", old, data)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.FinishTaskExport(event, true); e != nil {
		t.Fatal(e)
	}
	current, e := r.GetTask(task.ID)
	if e != nil || current.Status != "ready" || current.ExportedAt != "" {
		t.Fatal("old revision marked the current one exported", e)
	}
	failed, e := r.BeginTasksExport("actor", "failed.json", p, data)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.FinishTaskExport(failed, false); e != nil {
		t.Fatal(e)
	}
	if _, _, e = r.BuildTasksExport(selected); e != nil {
		t.Fatal("failed export removed task from queue", e)
	}
}
