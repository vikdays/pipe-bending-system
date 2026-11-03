package registry

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestCachedFormulasPercentPrecisionAndFormattedIdentifiers(t *testing.T) {
	m, _, dir := managerFixture(t)
	book := excelize.NewFile()
	headers := []any{"Номер трубы", "Диаметр трубы, мм", "Эквивалент углерода, %"}
	book.SetSheetRow("Sheet1", "A1", &headers)
	book.SetCellInt("Sheet1", "A2", 7)
	format := "000000"
	style, _ := book.NewStyle(&excelize.Style{CustomNumFmt: &format})
	book.SetCellStyle("Sheet1", "A2", "A2", style)
	book.SetCellFormula("Sheet1", "B2", "1+1")
	book.SetCellValue("Sheet1", "C2", 0.003333)
	percentStyle, _ := book.NewStyle(&excelize.Style{NumFmt: 10})
	book.SetCellStyle("Sheet1", "C2", "C2", percentStyle)
	book.SetCellValue("Sheet1", "A3", "000008")
	book.SetCellFormula("Sheet1", "B3", "1+1")
	book.SetCellValue("Sheet1", "C3", "0,33%")
	book.SetCellFormula("Sheet1", "B4", "NA()")
	original, _ := book.WriteToBuffer()
	book.Close()
	archive, e := zip.NewReader(bytes.NewReader(original.Bytes()), int64(original.Len()))
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, entry := range archive.File {
		stream, e := entry.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(stream)
		stream.Close()
		if e != nil {
			t.Fatal(e)
		}
		if entry.Name == "xl/worksheets/sheet1.xml" {
			re := regexp.MustCompile(`<c\b[^>]*r="B2"[^>]*>.*?</c>`)
			data = re.ReplaceAll(data, []byte(`<c r="B2"><f>1+1</f><v>530</v></c>`))
			re = regexp.MustCompile(`<c\b[^>]*r="B3"[^>]*>.*?</c>`)
			data = re.ReplaceAll(data, []byte(`<c r="B3"><f>1+1</f><v></v></c>`))
		}
		target, _ := writer.Create(entry.Name)
		target.Write(data)
	}
	writer.Close()
	path := filepath.Join(dir, "formulas.xlsx")
	os.WriteFile(path, out.Bytes(), 0600)
	p, e := m.Prepare(path, "actor")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Rows) != 2 {
		t.Fatal("formula-only empty row became a pipe", len(p.Rows))
	}
	a, b := p.Rows[0], p.Rows[1]
	if a.Input.FactoryNumber != "000007" || a.Input.DiameterMm == nil || *a.Input.DiameterMm != 530 {
		t.Fatalf("display ID or cached result changed: %+v", a.Input)
	}
	if a.Input.CarbonEquivalentPct == nil || math.Abs(*a.Input.CarbonEquivalentPct-0.3333) > 1e-10 || b.Input.CarbonEquivalentPct == nil || *b.Input.CarbonEquivalentPct != 0.33 {
		t.Fatal("percentage precision lost or text percentage multiplied twice")
	}
	if b.Status != "invalid" || b.Input.DiameterMm != nil {
		t.Fatal("missing formula cache was calculated or turned into zero")
	}
	if sourceOf(a.Input).Cells["B2"].Formula != "1+1" {
		t.Fatal("original formula lost")
	}
}
