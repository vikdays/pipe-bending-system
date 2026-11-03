package registry

import (
	"database/sql"
	"math"
	"reflect"
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/storage"
)

func fixture(t *testing.T) (*Repository, *sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err = db.Exec("INSERT INTO users(id,username,password_hash,role,created_at) VALUES('actor','fictional.user','unused-test-hash','admin',1)"); err != nil {
		t.Fatal(err)
	}
	return New(db), db, dir
}
func number(value float64) *float64 { return &value }
func completeInput() PipeInput {
	return PipeInput{PipeData: PipeData{InventoryNumber: "TEST-001", FactoryNumber: "0001.А-02", Manufacturer: "Вымышленный завод", DiameterMm: number(530), WallThicknessMm: number(8), LengthMm: number(12000), MassKg: number(1200), StrengthClass: "TEST", SteelGrade: "Условная", HeatNumber: "ПЛ-ДЕМО", CarbonEquivalentPct: number(0), PipeSpecification: "Условный документ", CoatingType: "Условное покрытие", CoatingSpecification: "Условный документ покрытия", TestPressureMpa: number(10), MinOperatingTempC: number(0), IncomingInspectionReference: "Демонстрационный акт", AllowedCategory54: "Значение из источника", AllowedCategory74: "Не установлено", Notes: "Вымышленные данные", AttributesJSON: `{"customFields":[{"name":"Дополнительная характеристика","value":"исходный текст","unit":""}],"original":{"unknown":[0,null,"—"]}}`}}
}
func assertCode(t *testing.T, err error, code apperror.Code) {
	t.Helper()
	if err == nil || apperror.Describe(err).Code != code {
		t.Fatalf("error=%v want=%s", err, code)
	}
}

func TestPipeAllFieldsCertificatesAndRestart(t *testing.T) {
	r, db, dir := fixture(t)
	input := completeInput()
	pages := 4
	input.NewCertificate = &CertificateDraft{Reference: "Вымышленный сертификат № TEST", Number: "TEST", IssuedOn: "2026-01-17", PageCount: &pages, Issuer: "Условный издатель"}
	first, err := r.CreatePipe(input, "actor")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.PipeData, input.PipeData) || len(first.Certificates) != 1 || first.Revision != 1 {
		t.Fatalf("round trip lost pipe fields: %+v", first)
	}
	input.InventoryNumber = "TEST-002"
	input.NewCertificate = nil
	input.CertificateIDs = []string{first.Certificates[0].ID}
	second, err := r.CreatePipe(input, "actor")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.FactoryNumber != second.FactoryNumber || !reflect.DeepEqual(first.Certificates, second.Certificates) {
		t.Fatal("same factory number must not merge distinct inventory entries; a certificate can be shared")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	pipes, err := New(reopened).ListPipes()
	if err != nil || len(pipes) != 2 {
		t.Fatal(len(pipes), err)
	}
	for _, pipe := range pipes {
		if pipe.AttributesJSON != input.AttributesJSON || pipe.MinOperatingTempC == nil || *pipe.MinOperatingTempC != 0 || len(pipe.Certificates) != 1 {
			t.Fatal("restart changed optional data or certificate links")
		}
	}
}

func TestValidationAndMissingCertificateDoNotPartiallyWrite(t *testing.T) {
	r, db, _ := fixture(t)
	for _, change := range []func(*PipeInput){
		func(p *PipeInput) { p.InventoryNumber = "" },
		func(p *PipeInput) { p.DiameterMm = number(-1) },
		func(p *PipeInput) { p.LengthMm = number(math.Inf(1)) },
		func(p *PipeInput) { p.CarbonEquivalentPct = number(math.NaN()) },
		func(p *PipeInput) { p.WallThicknessMm = number(300) },
		func(p *PipeInput) { p.AttributesJSON = "null" },
		func(p *PipeInput) { p.AttributesJSON = "[]" },
		func(p *PipeInput) { p.AttributesJSON = "{" },
		func(p *PipeInput) { p.NewCertificate = &CertificateDraft{Reference: "TEST", IssuedOn: "2026-02-30"} },
	} {
		input := completeInput()
		change(&input)
		_, err := r.CreatePipe(input, "actor")
		assertCode(t, err, apperror.ValidationError)
	}
	input := completeInput()
	input.CertificateIDs = []string{"missing"}
	input.NewCertificate = &CertificateDraft{Reference: "Must roll back"}
	_, err := r.CreatePipe(input, "actor")
	assertCode(t, err, apperror.NotFound)
	for _, table := range []string{"pipes", "certificates", "pipe_certificates", "manufacturing_tasks"} {
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal(table, count, err)
		}
	}
	input = PipeInput{PipeData: PipeData{InventoryNumber: "MIN-1", FactoryNumber: "0001"}}
	pipe, err := r.CreatePipe(input, "actor")
	if err != nil || pipe.DiameterMm != nil || pipe.AttributesJSON != "{}" {
		t.Fatal("unknown values must remain null", err)
	}
	_, err = r.CreateTask(TaskInput{Number: "TASK", PipeID: pipe.ID, PipeRevision: 1, BendNumber: "BEND", Machine: "Указано вручную", TargetAngle: "9"}, "actor")
	assertCode(t, err, apperror.ValidationError)
}

func TestTaskReferencesPipeSnapshotAndOptimisticRevision(t *testing.T) {
	r, db, dir := fixture(t)
	input := completeInput()
	pipe, err := r.CreatePipe(input, "actor")
	if err != nil {
		t.Fatal(err)
	}
	taskInput := TaskInput{Number: "TASK-1", PipeID: pipe.ID, PipeRevision: pipe.Revision, BendNumber: "BEND-1", Machine: "Условный станок", TargetAngle: "9,5"}
	task, err := r.CreateTask(taskInput, "actor")
	if err != nil {
		t.Fatal(err)
	}
	input.DiameterMm = number(720)
	updated, err := r.UpdatePipe(pipe.ID, pipe.Revision, input, "actor")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatal("revision not incremented")
	}
	_, err = r.UpdatePipe(pipe.ID, pipe.Revision, input, "actor")
	assertCode(t, err, apperror.Conflict)
	taskInput.Number = "TASK-STALE"
	_, err = r.CreateTask(taskInput, "actor")
	assertCode(t, err, apperror.Conflict)
	if _, err = db.Exec("DELETE FROM pipes WHERE id=?", pipe.ID); err == nil {
		t.Fatal("foreign key allowed deleting the task's source pipe")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	tasks, err := New(reopened).ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatal(len(tasks), err)
	}
	if !reflect.DeepEqual(tasks[0], task) || tasks[0].PipeID != pipe.ID || *tasks[0].PipeSnapshot.DiameterMm != 530 || tasks[0].TargetAngle != "9.5" {
		t.Fatal("task lost identity or original pipe snapshot after edit/restart")
	}
}

func TestConflictsAndParameterizedValues(t *testing.T) {
	r, _, _ := fixture(t)
	input := completeInput()
	input.FactoryNumber = "'); DROP TABLE pipes; --"
	pipe, err := r.CreatePipe(input, "actor")
	if err != nil || pipe.FactoryNumber != input.FactoryNumber {
		t.Fatal(err)
	}
	_, err = r.CreatePipe(input, "actor")
	assertCode(t, err, apperror.Conflict)
	_, err = r.UpdatePipe("missing", 1, input, "actor")
	assertCode(t, err, apperror.NotFound)
	task := TaskInput{Number: "TASK", PipeID: "missing", PipeRevision: 1, BendNumber: "BEND", Machine: "Условный", TargetAngle: "9"}
	_, err = r.CreateTask(task, "actor")
	assertCode(t, err, apperror.NotFound)
	task.PipeID = pipe.ID
	if _, err = r.CreateTask(task, "actor"); err != nil {
		t.Fatal(err)
	}
	_, err = r.CreateTask(task, "actor")
	assertCode(t, err, apperror.Conflict)
}
