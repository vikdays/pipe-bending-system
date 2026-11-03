package registry

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/documents"
	"changeme/internal/storage"
	"github.com/xuri/excelize/v2"
)

func workbookFixture(t *testing.T, dir string) string {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	book.SetSheetName("Sheet1", "Трубный журнал")
	rows := [][]any{{"Демонстрационная книга"}, {"Номер трубы", "№ Сертификата, дата сертификата, количество листов", "Завод изготовитель", "Диаметр трубы, мм", "Толщина стенки, мм", "Длина трубы, м", "Номер плавки", "Эквивалент углерода, %", "Неизвестная графа", "Номер плавки", "Ceq"},
		{"0001.А-01", "№ TEST-1 от 17.01.2026 на 4 л.", "Вымышленный завод", 530, 8, 12, "TEST-HEAT", 0, "Сохранить исходник", "LOOKUP", 99},
		{"0002.А-01", "№ TEST-1 от 17.01.2026 на 4 л.", "Вымышленный завод", 530, 8, 12, "TEST-HEAT", 0.33, "Другой текст"},
		{"0003.А-01", "", "Вымышленный завод", "не число", 8, 12, "TEST-HEAT", ""}}
	for i, row := range rows {
		axis, _ := excelize.CoordinatesToCellName(1, i+1)
		if e := book.SetSheetRow("Трубный журнал", axis, &row); e != nil {
			t.Fatal(e)
		}
	}
	path := filepath.Join(dir, "fictional.xlsx")
	if e := book.SaveAs(path); e != nil {
		t.Fatal(e)
	}
	return path
}
func managerFixture(t *testing.T) (*ImportManager, *Repository, string) {
	t.Helper()
	r, db, dir := fixture(t)
	store, e := documents.New(db, dir)
	if e != nil {
		t.Fatal(e)
	}
	return NewImportManager(db, store), r, dir
}
func countTable(t *testing.T, r *Repository, table string, want int) {
	t.Helper()
	var n int
	if e := r.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); e != nil || n != want {
		t.Fatalf("%s count=%d want=%d err=%v", table, n, want, e)
	}
}
func TestImportPreviewCommitDedupAndPersistence(t *testing.T) {
	m, r, dir := managerFixture(t)
	path := workbookFixture(t, dir)
	p, e := m.Prepare(path, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if p.Configuration.HeaderRow != 2 || len(p.Rows) != 3 || p.Configuration.Columns["heatNumber"] != 7 || p.Configuration.Units["lengthMm"] != "m" {
		t.Fatalf("wrong mapping %+v", p.Configuration)
	}
	if p.Rows[0].Input.FactoryNumber != "0001.А-01" || *p.Rows[0].Input.LengthMm != 12000 || *p.Rows[0].Input.CarbonEquivalentPct != 0 || p.Rows[2].Input.CarbonEquivalentPct != nil || p.Rows[2].Status != "invalid" {
		t.Fatalf("values or errors lost %+v", p.Rows)
	}
	source := sourceOf(p.Rows[0].Input)
	if source.Cells["J3"].Raw != "LOOKUP" || source.Cells["I3"].Raw != "Сохранить исходник" {
		t.Fatal("unknown/auxiliary source lost")
	}
	countTable(t, r, "pipes", 0)
	countTable(t, r, "certificates", 0)
	selection := []string{p.Rows[0].ID, p.Rows[1].ID}
	result, e := m.Commit(p.ID, "actor", p.Revision, selection)
	if e != nil || result.Added != 2 {
		t.Fatal(result, e)
	}
	countTable(t, r, "pipes", 2)
	countTable(t, r, "certificates", 1)
	countTable(t, r, "pipe_certificates", 2)
	repeated, e := m.Commit(p.ID, "actor", p.Revision, selection)
	if e != nil || *repeated != *result {
		t.Fatal("commit retry is not idempotent", e)
	}
	_, e = m.Commit(p.ID, "actor", p.Revision, []string{selection[0]})
	assertCode(t, e, apperror.Conflict)
	again, e := m.Prepare(path, "actor")
	if e != nil || again.ID == p.ID || again.Status != "prepared" || again.Rows[0].Status != "existing" || again.Rows[1].Status != "existing" {
		t.Fatal("repeat file must expose skipped rows and preserve existing ones", e)
	}
	if e = r.db.Close(); e != nil {
		t.Fatal(e)
	}
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	store, e := documents.New(db, dir)
	if e != nil {
		t.Fatal(e)
	}
	reopened := NewImportManager(db, store)
	saved, e := reopened.Get(p.ID, "actor")
	if e != nil || saved.Result.Added != 2 || len(saved.Rows) != 3 {
		t.Fatal("restart lost import", e)
	}
	_, e = reopened.Get(p.ID, "different-user")
	assertCode(t, e, apperror.Forbidden)
}
func TestImportRollbackRevalidationAndEditedSource(t *testing.T) {
	m, r, dir := managerFixture(t)
	p, e := m.Prepare(workbookFixture(t, dir), "actor")
	if e != nil {
		t.Fatal(e)
	}
	input := p.Rows[1].Input
	input.CertificateIDs = []string{"not-a-certificate"}
	source := sourceOf(input)
	input.AttributesJSON = `{"source":{"sha256":"forged"},"customFields":[]}`
	edited, e := m.EditRow(p.ID, "actor", p.Revision, p.Rows[1].ID, input)
	if e != nil {
		t.Fatal(e)
	}
	if sourceOf(edited.Rows[1].Input).SHA256 != source.SHA256 {
		t.Fatal("WebView overwrote native source")
	}
	_, e = m.Commit(p.ID, "actor", p.Revision, []string{p.Rows[0].ID})
	assertCode(t, e, apperror.Conflict)
	_, e = m.Commit(p.ID, "actor", edited.Revision, []string{p.Rows[0].ID, p.Rows[1].ID})
	assertCode(t, e, apperror.NotFound)
	countTable(t, r, "pipes", 0)
	countTable(t, r, "certificates", 0)
	input = p.Rows[0].Input
	input.AttributesJSON = "{}"
	input.DiameterMm = number(720)
	_, e = r.CreatePipe(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	refreshed, e := m.Get(p.ID, "actor")
	if e != nil || refreshed.Rows[0].Status != "conflict" {
		t.Fatal("new conflicting data must be reclassified", e)
	}
	_, e = m.Commit(p.ID, "actor", edited.Revision, []string{p.Rows[0].ID})
	assertCode(t, e, apperror.Conflict)
	countTable(t, r, "pipes", 1)
}
func TestDuplicateAndAmbiguousRows(t *testing.T) {
	a := completeInput()
	b := a
	b.InventoryNumber = "ANOTHER"
	rows := []ImportRow{{Input: a, Issues: []ImportIssue{}}, {Input: b, Issues: []ImportIssue{}}}
	classifyRows(rows, nil)
	if rows[0].Status != "new" || rows[1].Status != "duplicate" {
		t.Fatal(rows)
	}
	b.DiameterMm = number(720)
	rows = []ImportRow{{Input: a}, {Input: b}}
	classifyRows(rows, nil)
	if rows[0].Status != "conflict" || rows[1].Status != "conflict" {
		t.Fatal("must not choose a first conflicting row", rows)
	}
	existing := Pipe{PipeData: a.PipeData, ID: "existing"}
	rows = []ImportRow{{Input: a}}
	classifyRows(rows, []Pipe{existing, existing})
	if rows[0].Status != "conflict" {
		t.Fatal("ambiguous matches silently merged")
	}
}
func TestUnmappedWorkbookCanBeMappedAndDraftSurvives(t *testing.T) {
	m, r, dir := managerFixture(t)
	book := excelize.NewFile()
	book.SetCellValue("Sheet1", "A1", "Мой номер")
	book.SetCellValue("Sheet1", "A2", "00042")
	path := filepath.Join(dir, "unknown.xlsx")
	book.SaveAs(path)
	book.Close()
	p, e := m.Prepare(path, "actor")
	if e != nil || len(p.Rows) != 0 {
		t.Fatal("unknown headers should still open mapping", e)
	}
	c := p.Configuration
	c.Columns = map[string]int{"factoryNumber": 1}
	next, e := m.Configure(p.ID, "actor", p.Revision, c)
	if e != nil || len(next.Rows) != 1 || next.Rows[0].Input.FactoryNumber != "00042" {
		t.Fatal(e, next)
	}
	history, e := m.List("actor")
	if e != nil || len(history) != 1 {
		t.Fatal(e)
	}
	countTable(t, r, "pipes", 0)
}
func TestWorkbookLimitsAndNumbers(t *testing.T) {
	for _, value := range []string{"1,000.3", "1 000", "NaN", "Inf", "=2+2"} {
		if _, e := parseNumber(value, "mm"); e == nil {
			t.Fatal("ambiguous number accepted", value)
		}
	}
	if n, e := parseNumber("0,33%", "percent"); e != nil || *n != 0.33 {
		t.Fatal(n, e)
	}
	if n, e := parseNumber("0", "C"); e != nil || n == nil || *n != 0 {
		t.Fatal(n, e)
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	w, _ := z.Create("xl/worksheets/sheet1.xml")
	w.Write([]byte(`<worksheet><sheetData><row r="999999999"><c r="A999999999"/></row></sheetData></worksheet>`))
	z.Close()
	path := filepath.Join(t.TempDir(), "bad.xlsx")
	os.WriteFile(path, out.Bytes(), 0600)
	f, _ := os.Open(path)
	defer f.Close()
	if validateWorkbook(f) == nil {
		t.Fatal("unbounded row index accepted")
	}
}

// Opt-in integration test reads local production files; no originals or extracted data are tracked.
func TestLocalWorkbookIntegration(t *testing.T) {
	path := os.Getenv("PIPE_TEST_XLSX")
	if path == "" {
		t.Skip("set PIPE_TEST_XLSX to check a local original")
	}
	m, r, _ := managerFixture(t)
	p, e := m.Prepare(path, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Rows) != 285 {
		t.Fatalf("expected 285 source records, got %d", len(p.Rows))
	}
	for _, row := range p.Rows {
		if row.Status != "new" {
			t.Fatalf("row %d: %s %+v", row.SourceRow, row.Status, row.Issues)
		}
	}
	certificateID := ""
	for i := 0; i < 10; i++ {
		input := p.Rows[i].Input
		if certificateID != "" {
			input.NewCertificate = nil
			input.CertificateIDs = []string{certificateID}
		}
		input.InventoryNumber = "ТЕСТ-" + strings.Repeat("0", 3) + string(rune('A'+i))
		pipe, err := r.CreatePipe(input, "actor")
		if err != nil {
			t.Fatal(err)
		}
		certificateID = pipe.Certificates[0].ID
	}
	p, e = m.Get(p.ID, "actor")
	if e != nil {
		t.Fatal(e)
	}
	selection := []string{}
	existing := 0
	for _, row := range p.Rows {
		if row.Status == "new" {
			selection = append(selection, row.ID)
		} else if row.Status == "existing" {
			existing++
		} else {
			t.Fatalf("row %d: %s", row.SourceRow, row.Status)
		}
	}
	if existing != 10 || len(selection) != 275 {
		t.Fatal(existing, len(selection))
	}
	result, e := m.Commit(p.ID, "actor", p.Revision, selection)
	if e != nil || result.Added != 275 || result.Existing != 10 {
		t.Fatal(result, e)
	}
	countTable(t, r, "pipes", 285)
	countTable(t, r, "certificates", 11)
	countTable(t, r, "pipe_certificates", 285)
	completed, e := m.Get(p.ID, "actor")
	if e != nil {
		t.Fatal(e)
	}
	addedRows, existingRows := 0, 0
	for _, row := range completed.Rows {
		if row.Status == "imported" {
			addedRows++
		}
		if row.Status == "existing" {
			existingRows++
		}
	}
	if addedRows != 275 || existingRows != 10 {
		t.Fatalf("committed row statuses: added=%d existing=%d", addedRows, existingRows)
	}
	pipes, e := r.ListPipes()
	if e != nil {
		t.Fatal(e)
	}
	for _, pipe := range pipes {
		var a map[string]any
		if json.Unmarshal([]byte(pipe.AttributesJSON), &a) != nil || a["source"] == nil {
			t.Fatal("source lost")
		}
	}
	t.Log("285 rows parsed; 10 existing preserved; 275 added atomically")
}
