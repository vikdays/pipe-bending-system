package registry

import (
	"archive/zip"
	"changeme/internal/apperror"
	"changeme/internal/documents"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cardFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fictional.docx")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	w, e := z.Create("word/document.xml")
	if e != nil {
		t.Fatal(e)
	}
	_, e = w.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + text + `</w:body></w:document>`))
	if e != nil {
		t.Fatal(e)
	}
	if e = z.Close(); e != nil {
		t.Fatal(e)
	}
	f.Close()
	return path
}
func fictionalCard(t *testing.T) string {
	return cardFile(t, `<w:p><w:r><w:t>ГТ532 2026 — вымышленные тестовые данные</w:t></w:r></w:p><w:p><w:r><w:t>Таблица 4 - Режимы изготовления отводов из труб 219 мм</w:t></w:r></w:p><w:tbl>`+xmlRow("Угол гибки", "Класс прочности", "Единичный угол", "Кол-во гибов", "Передвижка", "Кол-во передвижек", "L2", "Минимальная длина трубы")+xmlRow("1", "Условный класс", "0,5", "2", "100", "1", "100", "4000")+`</w:tbl>`)
}
func xmlRow(cells ...string) string {
	s := "<w:tr>"
	for _, v := range cells {
		s += `<w:tc><w:p><w:r><w:t>` + v + `</w:t></w:r></w:p></w:tc>`
	}
	return s + "</w:tr>"
}
func TestCardImportIdempotenceAndReviewGate(t *testing.T) {
	r, _, dir := fixture(t)
	store, e := documents.New(r.db, dir)
	if e != nil {
		t.Fatal(e)
	}
	manager := NewTechnicalCardManager(r.db, store)
	p, e := manager.Prepare(fictionalCard(t), "m_gt532", "actor")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Rows) != 1 {
		t.Fatalf("rows=%d", len(p.Rows))
	}
	key := p.Rows[0].Key
	if n, e := manager.Commit(p.ID, []string{key, "unknown"}, "actor"); e == nil || n != 0 {
		t.Fatal("non-atomic selection", n, e)
	}
	configs, _ := r.ListConfigurations()
	if len(configs) != 0 {
		t.Fatal("partial import")
	}
	n, e := manager.Commit(p.ID, []string{key}, "actor")
	if e != nil || n != 1 {
		t.Fatal(n, e)
	}
	n, e = manager.Commit(p.ID, []string{key}, "actor")
	if e != nil || n != 0 {
		t.Fatal("duplicate import", n, e)
	}
	configs, _ = r.ListConfigurations()
	c := configs[0]
	if !c.NeedsReview || c.MachineID != "m_gt532" || c.SourceJSON == "{}" {
		t.Fatal(c)
	}
	pi := completeInput()
	pi.DiameterMm = number(219)
	pi.LengthMm = number(5000)
	pipe, e := r.CreatePipe(pi, "actor")
	if e != nil {
		t.Fatal(e)
	}
	input := TaskInput{PipeID: pipe.ID, PipeRevision: pipe.Revision, BendNumber: "TEST", BendType: "1ГО", TargetAngle: "1", MachineID: "m_gt532", ConfigurationID: c.ID, ConfigurationRevision: c.Revision}
	task, e := r.CreateTask(input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.PrepareTask(task.ID, 1, "actor")
	assertCode(t, e, apperror.ValidationError)
	ci := c.ConfigurationInput
	ci.ConfirmSourceReview = true
	c, e = r.SaveConfiguration(c.ID, c.Revision, ci, "actor")
	if e != nil || c.NeedsReview {
		t.Fatal(e)
	}
	// Reviewing a catalog entry must not silently rewrite a saved draft.
	stale, e := r.GetTask(task.ID)
	if e != nil || stale.ConfigurationRevision == c.Revision || !stale.ConfigurationSnapshot.NeedsReview {
		t.Fatal("review unexpectedly changed the task's saved configuration", e)
	}
	_, e = r.PrepareTask(task.ID, task.Revision, "actor")
	assertCode(t, e, apperror.Conflict)
	input.ConfigurationRevision = c.Revision
	input.Number = task.Number
	task, e = r.UpdateTask(task.ID, 1, input, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if task.ConfigurationRevision != c.Revision || task.ConfigurationSnapshot.NeedsReview {
		t.Fatal("saving the draft did not pick up the reviewed configuration")
	}
	ready, e := r.PrepareTask(task.ID, task.Revision, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if ready.MachineSnapshot == nil || ready.MachineSnapshot.ID != "m_gt532" {
		t.Fatal("missing machine snapshot")
	}
	input.MachineID = "m_gt1022"
	_, e = r.CreateTask(input, "actor")
	assertCode(t, e, apperror.ValidationError)
	ci.MachineID = "m_gt1022"
	_, e = r.SaveConfiguration(c.ID, c.Revision, ci, "actor")
	assertCode(t, e, apperror.ValidationError)
	ci.MachineID = ""
	_, e = r.SaveConfiguration("", 0, ci, "actor")
	assertCode(t, e, apperror.ValidationError)
}
func TestCardRejectsWrongMachineAndUnknownLayout(t *testing.T) {
	for _, test := range []struct{ path, model string }{{fictionalCard(t), "ГТ1022"}, {cardFile(t, `<w:p><w:r><w:t>ГТ532</w:t></w:r></w:p>`), "ГТ532"}} {
		f, e := os.Open(test.path)
		if e != nil {
			t.Fatal(e)
		}
		_, _, e = ParseTechnicalCard(f, Machine{ID: "test", Name: test.model, Model: test.model})
		f.Close()
		assertCode(t, e, apperror.ValidationError)
	}
}

// Optional integration uses private originals; source files are never copied into the repository.
func TestPrivateTechnicalCardOriginals(t *testing.T) {
	r, _, dir := fixture(t)
	store, e := documents.New(r.db, dir)
	if e != nil {
		t.Fatal(e)
	}
	manager := NewTechnicalCardManager(r.db, store)
	for _, test := range []struct {
		env, id string
		want    int
	}{{"PIPE_TEST_CARD_532", "m_gt532", 200}, {"PIPE_TEST_CARD_1022", "m_gt1022", 34}} {
		path := os.Getenv(test.env)
		if path == "" {
			t.Skip("private technical cards not supplied")
		}
		p, e := manager.Prepare(path, test.id, "actor")
		if e != nil {
			t.Fatal(test.id, e)
		}
		if len(p.Rows) != test.want {
			t.Fatalf("%s got %d profiles, want %d", test.id, len(p.Rows), test.want)
		}

		if test.id == "m_gt1022" && !strings.Contains(p.Rows[0].Input.Version, "2024") {
			t.Fatal("mistook a standard's year for the card year", p.Rows[0].Input.Version)
		}
		type1, type2 := 0, 0
		foundExample := false
		sourceConflict := false
		keys := []string{}
		for _, row := range p.Rows {
			keys = append(keys, row.Key)
			if test.id == "m_gt532" && row.Input.DiameterMm == 530 {
				t.Fatal("offered factory range absent from supplied tooling")
			}
			if test.id == "m_gt532" && row.Input.DiameterMm == 219 && row.Input.MinPipeLengthMm != nil && *row.Input.MinPipeLengthMm == 3850 {
				sourceConflict = len(row.Warnings) > 0
			}
			if test.id == "m_gt1022" && row.Input.DiameterMm == 720 {
				if row.Input.BendType == "1ГО" {
					type1++
				}
				if row.Input.BendType == "2ГО" {
					type2++
				}
			}
			if test.id == "m_gt1022" && row.Input.DiameterMm == 530 && row.Input.TargetAngle == "9" {
				foundExample = row.Input.MinPipeLengthMm != nil && *row.Input.MinPipeLengthMm == 11580
				for _, param := range row.Input.Parameters {
					if strings.HasPrefix(param.Name, "Передвижка") && param.Value != "180" {
						t.Fatal("misaligned merged columns", param)
					}
				}
				if row.Input.Notes == "" || !strings.Contains(row.SourceJSON, "2500*") {
					t.Fatal("lost conditions or stars")
				}
			}
		}
		if test.id == "m_gt532" && !sourceConflict {
			t.Fatal("source length inconsistency was hidden or corrected")
		}
		if test.id == "m_gt1022" && (type1 != 9 || type2 != 3 || !foundExample) {
			t.Fatal("incorrect grid spans/type2 boundary", type1, type2, foundExample)
		}
		n, e := manager.Commit(p.ID, keys, "actor")
		if e != nil || n != test.want {
			t.Fatal(test.id, n, e)
		}
		t.Logf("%s: %d profiles", test.id, n)
	}
}

func TestHorizontalMergedTypesAndPreservedSourceCells(t *testing.T) {
	path := cardFile(t, `<w:p><w:r><w:t>ГТ1022 — вымышленный документ</w:t></w:r></w:p><w:p><w:r><w:t>Таблица 3 - Режимы изготовления отводов из труб Ø530 мм</w:t></w:r></w:p><w:tbl>`+
		`<w:tr><w:tc><w:p><w:r><w:t>Тип отвода</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>тип 1</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>тип 2</w:t></w:r></w:p></w:tc></w:tr>`+
		xmlRow("Суммарный угол отвода, град", "1", "2", "3")+xmlRow("Единичный угол гибки, град", "0,1", "0,2", "0,3")+xmlRow("Передвижка, мм", "101*", "202", "303")+xmlRow("Минимальная длина трубы, мм", "4001", "4002", "4003")+xmlRow("*Вымышленная сноска", "", "", "")+`</w:tbl>`)
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	rows, _, e := ParseTechnicalCard(f, Machine{ID: "m_gt1022", Name: "ГТ1022", Model: "ГТ1022"})
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 3 || rows[0].Input.BendType != "1ГО" || rows[1].Input.BendType != "1ГО" || rows[2].Input.BendType != "2ГО" {
		t.Fatal("merged cells shifted mode columns", rows)
	}
	if !strings.Contains(rows[0].SourceJSON, "101*") || rows[0].Input.Notes != "*Вымышленная сноска" {
		t.Fatal("source lost")
	}
}
func TestCardXMLDepthAndMacroBounds(t *testing.T) {
	path := cardFile(t, strings.Repeat("<nested>", 150)+strings.Repeat("</nested>", 150))
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	_, _, e = docxTables(f)
	f.Close()
	assertCode(t, e, apperror.ValidationError)
	path = filepath.Join(t.TempDir(), "macro.docx")
	f, e = os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	w, _ := z.Create("word/vbaProject.bin")
	w.Write([]byte("fictional"))
	z.Close()
	f.Close()
	f, e = os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	_, _, e = docxTables(f)
	f.Close()
	assertCode(t, e, apperror.ValidationError)
}
