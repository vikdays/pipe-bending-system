package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/auth"
	"changeme/internal/exchange"
	"changeme/internal/registry"
	"changeme/internal/storage"
)

func exportFixture(t *testing.T) (*DesktopRegistryService, *sql.DB, auth.TokenPair, registry.Task, string) {
	t.Helper()
	dir := t.TempDir()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	manager, e := auth.New(db)
	if e != nil {
		t.Fatal(e)
	}
	const password = "Fictional export test phrase 2026!"
	if _, e = manager.Bootstrap("export.admin", password); e != nil {
		t.Fatal(e)
	}
	login, e := manager.Login("export.admin", password)
	if e != nil {
		t.Fatal(e)
	}
	service := &DesktopRegistryService{auth: manager, repository: registry.New(db)}
	actor := login.Session.User.ID
	n := func(v float64) *float64 { return &v }
	pipe, e := service.repository.CreatePipe(registry.PipeInput{PipeData: registry.PipeData{FactoryNumber: "FICTIONAL-0001", InventoryNumber: "TEST-1", DiameterMm: n(530), WallThicknessMm: n(8), LengthMm: n(12000)}}, actor)
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := service.repository.SaveConfiguration("", 0, registry.ConfigurationInput{Name: "Тестовая конфигурация", Version: "TEST", MachineID: "m_gt532", DiameterMm: 530, TechnicalCardReference: "Условная ТК"}, actor)
	if e != nil {
		t.Fatal(e)
	}
	draft, e := service.repository.CreateTask(registry.TaskInput{Number: "TEST-EXPORT", PipeID: pipe.ID, PipeRevision: pipe.Revision, BendNumber: "DEMO-BEND", BendType: "1ГО", TargetAngle: "9", ConfigurationID: cfg.ID, ConfigurationRevision: cfg.Revision}, actor)
	if e != nil {
		t.Fatal(e)
	}
	task, e := service.repository.PrepareTask(draft.ID, draft.Revision, actor)
	if e != nil {
		t.Fatal(e)
	}
	return service, db, login, task, dir
}
func TestExportDialogCancellationFailureAndSuccessfulJournal(t *testing.T) {
	s, db, login, task, dir := exportFixture(t)
	access := login.Session.AccessToken
	calls := 0
	s.saveFile = func(string) (string, error) { calls++; return "", nil }
	result, e := s.ExportTasks(access, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	if e != nil || result.Outcome != "cancelled" {
		t.Fatal(result, e)
	}
	events, e := s.ListExchangeEvents(access)
	if e != nil || len(events) != 0 {
		t.Fatal("cancel marked success", e)
	}
	cancelled, _ := s.repository.GetTask(task.ID)
	if cancelled.Status != "ready" {
		t.Fatal("cancel marked task exported")
	}
	s.saveFile = func(string) (string, error) { return "", errors.New("internal system details") }
	_, e = s.ExportTasks(access, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	requireError(t, e, apperror.InternalError)
	events, _ = s.ListExchangeEvents(access)
	if len(events) != 0 {
		t.Fatal("dialog failure logged success")
	}
	s.saveFile = func(string) (string, error) { return filepath.Join(dir, "missing", "task.json"), nil }
	_, e = s.ExportTasks(access, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	requireError(t, e, apperror.InternalError)
	events, _ = s.ListExchangeEvents(access)
	if len(events) != 1 || events[0].Status != "failed" {
		t.Fatal(events)
	}
	failed, _ := s.repository.GetTask(task.ID)
	if failed.Status != "ready" {
		t.Fatal("failed write marked task exported")
	}
	s.saveFile = func(name string) (string, error) { return filepath.Join(dir, name), nil }
	first, e := s.ExportTasks(access, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	if e != nil || first.Outcome != "saved" {
		t.Fatal(first, e)
	}
	data, e := os.ReadFile(filepath.Join(dir, first.FileName))
	if e != nil {
		t.Fatal(e)
	}
	for _, secret := range []string{access, login.RefreshToken, "Fictional export test phrase 2026!", "password_hash", "export.admin"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("credentials leaked")
		}
	}
	if len(first.ExportedTasks) != 1 || first.ExportedTasks[0].ID != task.ID || first.ExportedTasks[0].Revision != task.Revision {
		t.Fatal("missing status confirmation", first)
	}
	_, e = s.ExportTasks(access, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	requireError(t, e, apperror.Conflict)
	events, _ = s.ListExchangeEvents(access)
	if len(events) != 2 || events[0].Status != "saved" || events[0].SizeBytes != len(data) {
		t.Fatal(events)
	}
	loaded, _ := s.repository.GetTask(task.ID)
	if loaded.Status != "exported" || loaded.ExportedAt == "" || loaded.Revision != task.Revision {
		t.Fatal("saved export state missing", loaded)
	}
	// A new ready task is still exportable after the first task leaves the queue.
	input := task.TaskInput
	input.Number = "TEST-NEXT"
	nextDraft, e := s.repository.CreateTask(input, login.Session.User.ID)
	if e != nil {
		t.Fatal(e)
	}
	next, e := s.repository.PrepareTask(nextDraft.ID, nextDraft.Revision, login.Session.User.ID)
	if e != nil {
		t.Fatal(e)
	}
	// Logout while a dialog is open must prevent writing and adding an event.
	s.saveFile = func(string) (string, error) {
		s.auth.Logout(login.RefreshToken)
		return filepath.Join(dir, "expired.json"), nil
	}
	_, e = s.ExportTasks(access, []registry.TaskExportSelection{{ID: next.ID, Revision: next.Revision}})
	requireError(t, e, apperror.Unauthenticated)
	if _, e = os.Stat(filepath.Join(dir, "expired.json")); !os.IsNotExist(e) {
		t.Fatal("wrote after logout")
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM exchange_events").Scan(&count)
	if count != 2 || calls != 1 {
		t.Fatal(count, calls)
	}
}
func TestExportWrittenButJournalFailureIsNotSuccess(t *testing.T) {
	s, db, login, task, dir := exportFixture(t)
	path := filepath.Join(dir, "task.json")
	s.saveFile = func(string) (string, error) { return path, nil }
	s.writeExport = func(path string, data []byte) error {
		if e := exchange.WriteJSON(path, data); e != nil {
			return e
		}
		_, e := db.Exec("CREATE TRIGGER fail_journal BEFORE UPDATE ON exchange_events BEGIN SELECT RAISE(ABORT,'test journal failure'); END")
		return e
	}
	result, e := s.ExportTasks(login.Session.AccessToken, []registry.TaskExportSelection{{ID: task.ID, Revision: task.Revision}})
	if e != nil || result.Outcome != "written_unlogged" || result.Message == "" {
		t.Fatal(result, e)
	}
	if _, e = os.Stat(path); e != nil {
		t.Fatal(e)
	}
	events, _ := s.ListExchangeEvents(login.Session.AccessToken)
	if len(events) != 1 || events[0].Status != "pending" {
		t.Fatal(events)
	}
	pending, _ := s.repository.GetTask(task.ID)
	if pending.Status != "ready" || len(result.ExportedTasks) != 0 {
		t.Fatal("unlogged write marked task exported")
	}
}

func TestPackageExportHasOneJournalEventWithTenOrderedMembers(t *testing.T) {
	s, db, login, first, dir := exportFixture(t)
	actor := login.Session.User.ID
	tasks := []registry.TaskExportSelection{{ID: first.ID, Revision: first.Revision}}
	for i := 1; i < 10; i++ {
		input := first.TaskInput
		input.Number = fmt.Sprintf("TEST-%02d", i)
		draft, e := s.repository.CreateTask(input, actor)
		if e != nil {
			t.Fatal(e)
		}
		ready, e := s.repository.PrepareTask(draft.ID, draft.Revision, actor)
		if e != nil {
			t.Fatal(e)
		}
		tasks = append(tasks, registry.TaskExportSelection{ID: ready.ID, Revision: ready.Revision})
	}
	calls := 0
	s.saveFile = func(name string) (string, error) { calls++; return filepath.Join(dir, name), nil }
	result, e := s.ExportTasks(login.Session.AccessToken, tasks)
	if e != nil || result.TaskCount != 10 || result.Outcome != "saved" || calls != 1 {
		t.Fatal(result, calls, e)
	}
	data, e := os.ReadFile(filepath.Join(dir, result.FileName))
	if e != nil {
		t.Fatal(e)
	}
	var p registry.TaskPackage
	if e = json.Unmarshal(data, &p); e != nil || len(p.Tasks) != 10 {
		t.Fatal("not one package", e)
	}
	events, e := s.ListExchangeEvents(login.Session.AccessToken)
	if e != nil || len(events) != 1 || len(events[0].Tasks) != 10 || events[0].Status != "saved" {
		t.Fatal(events, e)
	}
	for i, v := range events[0].Tasks {
		if v.QueuePosition != i+1 || v.TaskUUID != p.Tasks[i].ID || v.TaskRevision != p.Tasks[i].Revision {
			t.Fatal("journal lost package composition", v)
		}
	}
	if _, e = db.Exec("SELECT 1 FROM exchange_event_tasks"); e != nil {
		t.Fatal(e)
	}
}
