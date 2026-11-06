package registry

import (
	"changeme/internal/apperror"
	"changeme/internal/storage"
	"math"
	"reflect"
	"testing"
)

func taskFixture(t *testing.T) (*Repository, TaskInput, string) {
	t.Helper()
	r, _, dir := fixture(t)
	pipe, e := r.CreatePipe(completeInput(), "actor")
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := r.SaveConfiguration("", 0, ConfigurationInput{Name: "Вымышленная конфигурация", Version: "TEST-v1", MachineID: "m_gt532", TechnicalCardReference: "Условная ТК — только тест", DiameterMm: 530, Parameters: []ConfigurationParameter{{Name: "Условный параметр", Value: "непроизводственное значение", Unit: ""}}}, "actor")
	if e != nil {
		t.Fatal(e)
	}
	return r, TaskInput{Number: "TASK-1", PipeID: pipe.ID, PipeRevision: pipe.Revision, BendNumber: "TEST-BEND", BendType: "1ГО", TargetAngle: "9,5", ConfigurationID: cfg.ID, ConfigurationRevision: cfg.Revision}, dir
}
func TestTaskDraftEditPrepareSnapshotsAndRestart(t *testing.T) {
	r, input, dir := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if draft.Machine != "ГТ532" || draft.Revision != 1 || draft.QueuePosition != 1 || draft.SnapshotVersion != 0 {
		t.Fatal(draft)
	}
	input.Notes = "Проверить в заданном порядке"
	updated, e := r.UpdateTask(draft.ID, 1, input, "actor")
	if e != nil || updated.Revision != 2 {
		t.Fatal(e)
	}
	_, e = r.UpdateTask(draft.ID, 1, input, "actor")
	assertCode(t, e, apperror.Conflict)
	prepared, e := r.PrepareTask(draft.ID, updated.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if prepared.Status != "ready" || prepared.Revision != 3 || prepared.SnapshotVersion != 1 || prepared.PreparedAt == "" || prepared.ConfigurationSnapshot == nil || prepared.TargetAngle != "9.5" {
		t.Fatal(prepared)
	}
	repeat, e := r.PrepareTask(draft.ID, updated.Revision, "actor")
	if e != nil || !reflect.DeepEqual(prepared, repeat) {
		t.Fatal("preparation retry changed frozen content", e)
	}
	pipe := prepared.PipeSnapshot
	changed := PipeInput{PipeData: pipe.PipeData}
	changed.DiameterMm = number(720)
	if _, e = r.UpdatePipe(pipe.ID, pipe.Revision, changed, "actor"); e != nil {
		t.Fatal(e)
	}
	cfg := *prepared.ConfigurationSnapshot
	cfg.Name = "Позднее изменение каталога"
	cfg.Parameters[0].Value = "Новое значение"
	if _, e = r.SaveConfiguration(cfg.ID, cfg.Revision, cfg.ConfigurationInput, "actor"); e != nil {
		t.Fatal(e)
	}
	_, e = r.UpdateTask(prepared.ID, prepared.Revision, input, "actor")
	assertCode(t, e, apperror.Conflict)
	for _, query := range []string{"UPDATE manufacturing_tasks SET target_angle='1' WHERE id=?", "DELETE FROM manufacturing_tasks WHERE id=?"} {
		if _, e = r.db.Exec(query, prepared.ID); e == nil {
			t.Fatal("SQL bypassed prepared task lock")
		}
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	saved, e := New(db).GetTask(prepared.ID)
	if e != nil {
		t.Fatal(e)
	}
	// Compare to the original frozen value, unaffected by mutations to local test slices.
	if *saved.PipeSnapshot.DiameterMm != 530 || saved.ConfigurationSnapshot.Name != "Вымышленная конфигурация" || saved.ConfigurationSnapshot.Parameters[0].Value != "непроизводственное значение" || saved.Revision != 3 || saved.Status != "ready" || saved.QueuePosition != 1 {
		t.Fatal("frozen sources changed after restart", saved)
	}
}
func TestPreparationRequiresCompleteCurrentSources(t *testing.T) {
	for _, change := range []func(*TaskInput){func(i *TaskInput) { i.BendNumber = "" }, func(i *TaskInput) { i.TargetAngle = "" }, func(i *TaskInput) { i.BendType = "" }, func(i *TaskInput) { i.ConfigurationID = ""; i.ConfigurationRevision = 0 }} {
		r, input, _ := taskFixture(t)
		change(&input)
		task, e := r.CreateTask(input, "actor")
		if e != nil {
			t.Fatal(e)
		}
		_, e = r.PrepareTask(task.ID, 1, "actor")
		assertCode(t, e, apperror.ValidationError)
		saved, _ := r.GetTask(task.ID)
		if saved.Status != "draft" || saved.Revision != 1 || saved.PreparedAt != "" {
			t.Fatal("failed validation partially prepared a task")
		}
	}
	r, input, _ := taskFixture(t)
	draft, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	cfg := draft.ConfigurationSnapshot
	config := cfg.ConfigurationInput
	config.DiameterMm = 720
	next, e := r.SaveConfiguration(cfg.ID, cfg.Revision, config, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.PrepareTask(draft.ID, 1, "actor")
	assertCode(t, e, apperror.Conflict)
	input.ConfigurationRevision = next.Revision
	edited, e := r.UpdateTask(draft.ID, 1, input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.PrepareTask(draft.ID, edited.Revision, "actor")
	assertCode(t, e, apperror.ValidationError)
	config.DiameterMm = 530
	next, e = r.SaveConfiguration(next.ID, next.Revision, config, "actor")
	if e != nil {
		t.Fatal(e)
	}
	input.ConfigurationRevision = next.Revision
	edited, e = r.UpdateTask(draft.ID, edited.Revision, input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	pipe := edited.PipeSnapshot
	raw := PipeInput{PipeData: pipe.PipeData}
	raw.LengthMm = nil
	newPipe, e := r.UpdatePipe(pipe.ID, pipe.Revision, raw, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.PrepareTask(draft.ID, edited.Revision, "actor")
	assertCode(t, e, apperror.Conflict)
	input.PipeRevision = newPipe.Revision
	edited, e = r.UpdateTask(draft.ID, edited.Revision, input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.PrepareTask(draft.ID, edited.Revision, "actor")
	assertCode(t, e, apperror.ValidationError)
}
func TestBatchAtomicOrderingRetryAndRestart(t *testing.T) {
	r, input, dir := taskFixture(t)
	a, b := input, input
	a.Number = ""
	b.Number = ""
	b.BendNumber = "SECOND"
	batch := TaskBatchInput{RequestID: "fictional-request", Tasks: []TaskInput{a, b}}
	tasks, e := r.CreateTasks(batch, "actor")
	if e != nil || len(tasks) != 2 {
		t.Fatal(e)
	}
	if tasks[0].QueuePosition != 1 || tasks[1].QueuePosition != 2 || tasks[0].Number == tasks[1].Number {
		t.Fatal("batch order or generated numbers lost")
	}
	r.db.Close()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	r = New(db)
	again, e := r.CreateTasks(batch, "actor")
	if e != nil || !reflect.DeepEqual(again, tasks) {
		t.Fatal("retry duplicated a batch", e)
	}
	changed := batch
	changed.Tasks = []TaskInput{b}
	_, e = r.CreateTasks(changed, "actor")
	assertCode(t, e, apperror.Conflict)
	first, second := input, input
	first.Number = "NEW"
	second.Number = tasks[0].Number
	_, e = r.CreateTasks(TaskBatchInput{RequestID: "rollback", Tasks: []TaskInput{first, second}}, "actor")
	assertCode(t, e, apperror.Conflict)
	countTable(t, r, "manufacturing_tasks", 2)
	countTable(t, r, "task_creation_requests", 1)
	first.Number = "THIRD"
	third, e := r.CreateTask(first, "actor")
	if e != nil || third.QueuePosition != 3 {
		t.Fatal("rollback consumed order positions", e)
	}
	all, e := r.ListTasks()
	if e != nil || all[0].ID != tasks[0].ID || all[1].ID != tasks[1].ID || all[2].ID != third.ID {
		t.Fatal("list not in persisted execution order", e)
	}
	_, e = r.GetTask("missing")
	assertCode(t, e, apperror.NotFound)
}
func TestTaskAndConfigurationValidationInGo(t *testing.T) {
	r, input, _ := taskFixture(t)
	for _, value := range []string{"-1", "NaN", "Inf", "1e3", "1.2.3", "0"} {
		invalidInput := input
		invalidInput.TargetAngle = value
		_, e := r.CreateTask(invalidInput, "actor")
		assertCode(t, e, apperror.ValidationError)
	}
	bad := input
	bad.BendType = "unsupported"
	_, e := r.CreateTask(bad, "actor")
	assertCode(t, e, apperror.ValidationError)
	bad = input
	bad.ConfigurationID = "missing"
	_, e = r.CreateTask(bad, "actor")
	assertCode(t, e, apperror.NotFound)
	configs, e := r.ListConfigurations()
	if e != nil {
		t.Fatal(e)
	}
	c := configs[0]
	_, e = r.SaveConfiguration(c.ID, 99, c.ConfigurationInput, "actor")
	assertCode(t, e, apperror.Conflict)
	for _, change := range []func(*ConfigurationInput){func(c *ConfigurationInput) { c.DiameterMm = math.NaN() }, func(c *ConfigurationInput) { c.DiameterMm = 0 }, func(c *ConfigurationInput) { c.TechnicalCardReference = "" }, func(c *ConfigurationInput) {
		c.Parameters = []ConfigurationParameter{{Name: "one", Value: "1"}, {Name: "ONE", Value: "2"}}
	}} {
		cfg := c.ConfigurationInput
		change(&cfg)
		_, e = r.SaveConfiguration("", 0, cfg, "actor")
		assertCode(t, e, apperror.ValidationError)
	}
}
