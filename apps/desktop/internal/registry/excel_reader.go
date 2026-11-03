package registry

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"changeme/internal/documents"
	"github.com/xuri/excelize/v2"
)

const maxImportRows = 10000
const maxSheetRows = 50000
const maxSheetColumns = 128

// Bound decompression and cell coordinates before Excelize allocates worksheet data.
func validateWorkbook(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return invalid("Не удалось прочитать Excel.")
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return invalid("Нужна книга .xlsx без пароля.")
	}
	var total uint64
	if len(archive.File) > 4096 {
		return invalid("В книге слишком много внутренних файлов.")
	}
	for _, entry := range archive.File {
		if entry.UncompressedSize64 > 256<<20 || total > (256<<20)-entry.UncompressedSize64 {
			return invalid("Распакованная книга превышает 256 МБ.")
		}
		total += entry.UncompressedSize64
		if strings.Contains(strings.ToLower(entry.Name), "vbaproject") {
			return invalid("Книги с макросами не поддерживаются.")
		}
		if !strings.HasPrefix(entry.Name, "xl/worksheets/") || !strings.HasSuffix(entry.Name, ".xml") {
			continue
		}
		stream, e := entry.Open()
		if e != nil {
			return invalid("Книга Excel повреждена.")
		}
		decoder := xml.NewDecoder(io.LimitReader(stream, 256<<20))
		cells := 0
		rows := 0
		for {
			token, e := decoder.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				stream.Close()
				return invalid("Повреждён XML листа Excel.")
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			if start.Name.Local == "row" {
				rows++
				if rows > maxSheetRows {
					stream.Close()
					return invalid("На листе больше 50 000 строк.")
				}
			}
			if start.Name.Local != "c" && start.Name.Local != "row" {
				continue
			}
			for _, a := range start.Attr {
				if a.Name.Local != "r" {
					continue
				}
				if start.Name.Local == "row" {
					n, e := strconv.Atoi(a.Value)
					if e != nil || n < 1 || n > maxSheetRows {
						stream.Close()
						return invalid("Номер строки Excel вне допустимого диапазона.")
					}
				} else {
					c, r, e := excelize.CellNameToCoordinates(a.Value)
					if e != nil || r > maxSheetRows || c > maxSheetColumns {
						stream.Close()
						return invalid("Поддерживаются листы до 50 000 строк и 128 столбцов.")
					}
				}
			}
			if start.Name.Local == "c" {
				cells++
				if cells > maxSheetRows*maxSheetColumns {
					stream.Close()
					return invalid("Слишком много ячеек на листе.")
				}
			}
		}
		stream.Close()
	}
	return nil
}
func openWorkbook(path string) (*excelize.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, invalid("Оригинал Excel недоступен.")
	}
	defer file.Close()
	if err = validateWorkbook(file); err != nil {
		return nil, err
	}
	file.Seek(0, io.SeekStart)
	book, err := excelize.OpenReader(file, excelize.Options{UnzipSizeLimit: 256 << 20, UnzipXMLSizeLimit: 16 << 20, TmpDir: os.TempDir()})
	if err != nil {
		return nil, invalid("Не удалось открыть .xlsx. Проверьте формат и отсутствие пароля.")
	}
	return book, nil
}
func sheetHeaders(book *excelize.File, sheet string, row int) ([]ImportColumn, error) {
	out := []ImportColumn{}
	iterator, err := book.Rows(sheet)
	if err != nil {
		return nil, invalid("Лист Excel не найден.")
	}
	defer iterator.Close()
	for n := 1; iterator.Next(); n++ {
		if n != row {
			continue
		}
		values, e := iterator.Columns()
		if e != nil {
			return nil, invalid("Не удалось прочитать заголовки.")
		}
		for col, value := range values {
			letter, _ := excelize.ColumnNumberToName(col + 1)
			out = append(out, ImportColumn{col + 1, letter, strings.TrimSpace(value)})
		}
		return out, nil
	}
	return out, nil
}
func suggestMapping(columns []ImportColumn) map[string]int {
	result := map[string]int{}
	for _, spec := range importFields {
		matches := []int{}
		for _, col := range columns {
			for _, alias := range spec.aliases {
				if heading(col.Label) == heading(alias) {
					matches = append(matches, col.Index)
					break
				}
			}
		}
		if len(matches) > 0 {
			result[spec.Key] = matches[0]
		}
	}
	return result
}
func defaultConfiguration(book *excelize.File) ImportConfiguration {
	sheets := book.GetSheetList()
	config := ImportConfiguration{Columns: map[string]int{}, Units: map[string]string{}, HeaderRow: 1}
	best := -1
	for _, sheet := range sheets {
		for row := 1; row <= 10; row++ {
			columns, e := sheetHeaders(book, sheet, row)
			if e != nil {
				continue
			}
			mapping := suggestMapping(columns)
			score := len(mapping)
			if mapping["factoryNumber"] > 0 {
				score += 20
			}
			if score > best {
				best = score
				config.Sheet = sheet
				config.HeaderRow = row
				config.Columns = mapping
			}
		}
	}
	columns, _ := sheetHeaders(book, config.Sheet, config.HeaderRow)
	config.Units = suggestUnits(columns, config.Columns)
	return config
}
func suggestUnits(columns []ImportColumn, mapping map[string]int) map[string]string {
	units := map[string]string{}
	for _, spec := range importFields {
		if len(spec.Units) > 0 {
			units[spec.Key] = spec.Units[0]
		}
	}
	for _, col := range columns {
		for key, index := range mapping {
			if index != col.Index {
				continue
			}
			h := strings.ToLower(strings.ReplaceAll(col.Label, " ", ""))
			if key == "lengthMm" && strings.HasSuffix(h, ",м") {
				units[key] = "m"
			}
			if key == "massKg" && strings.HasSuffix(h, ",т") {
				units[key] = "t"
			}
		}
	}
	return units
}
func validateConfiguration(book *excelize.File, c ImportConfiguration) error {
	found := false
	for _, s := range book.GetSheetList() {
		if s == c.Sheet {
			found = true
		}
	}
	if !found || c.HeaderRow < 1 || c.HeaderRow > maxSheetRows {
		return invalid("Выберите лист и корректную строку заголовков.")
	}
	known := map[string]bool{}
	used := map[int]bool{}
	for _, spec := range importFields {
		known[spec.Key] = true
		col := c.Columns[spec.Key]
		if spec.Required && col == 0 {
			return invalid("Укажите столбец заводского номера трубы.")
		}
		if col < 0 || col > maxSheetColumns || col > 0 && used[col] {
			return invalid("Каждому полю нужен отдельный столбец от 1 до 128.")
		}
		if col > 0 {
			used[col] = true
		}
		if col > 0 && len(spec.Units) > 0 {
			valid := false
			for _, u := range spec.Units {
				if c.Units[spec.Key] == u {
					valid = true
				}
			}
			if !valid {
				return invalid("Проверьте единицы измерения числовых столбцов.")
			}
		}
	}
	for key := range c.Columns {
		if !known[key] {
			return invalid("Неизвестное поле сопоставления.")
		}
	}
	return nil
}

type sourceCell struct {
	Raw     string `json:"raw"`
	Display string `json:"display"`
	Formula string `json:"formula,omitempty"`
}
type importSource struct {
	Kind     string                `json:"kind"`
	FileName string                `json:"fileName"`
	SHA256   string                `json:"sha256"`
	Sheet    string                `json:"sheet"`
	Row      int                   `json:"row"`
	Cells    map[string]sourceCell `json:"cells"`
}

func sourceOf(input PipeInput) importSource {
	var a struct {
		Source importSource `json:"source"`
	}
	_ = json.Unmarshal([]byte(input.AttributesJSON), &a)
	return a.Source
}

var certificateReference = regexp.MustCompile(`(?i)^\s*№\s*(\S+)\s+от\s+(\d{2}\.\d{2}\.\d{4})\s+на\s+(\d+)\s+л\.?\s*$`)

func parseCertificate(reference string) *CertificateDraft {
	if reference == "" {
		return nil
	}
	draft := &CertificateDraft{Reference: reference}
	parts := certificateReference.FindStringSubmatch(reference)
	if parts == nil {
		return draft
	}
	date, e := time.Parse("02.01.2006", parts[2])
	pages, e2 := strconv.Atoi(parts[3])
	if e == nil && e2 == nil && pages > 0 {
		draft.Number = parts[1]
		draft.IssuedOn = date.Format("2006-01-02")
		draft.PageCount = &pages
	}
	return draft
}
func parseNumber(value, unit string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "—" || value == "-" {
		return nil, nil
	}
	if unit == "percent" {
		value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	}
	if strings.ContainsAny(value, " \u00a0\u202f") || strings.Contains(value, ",") && strings.Contains(value, ".") {
		return nil, fmt.Errorf("Неоднозначный формат числа: %s", value)
	}
	value = strings.ReplaceAll(value, ",", ".")
	n, e := strconv.ParseFloat(value, 64)
	if e != nil || !finite(n) {
		return nil, fmt.Errorf("Ожидалось число, получено: %s", value)
	}
	if unit == "m" || unit == "t" {
		n *= 1000
	}
	return &n, nil
}
func parseRows(book *excelize.File, file documents.File, c ImportConfiguration) ([]ImportRow, []ImportColumn, []string, error) {
	if e := validateConfiguration(book, c); e != nil {
		return nil, nil, nil, e
	}
	columns, e := sheetHeaders(book, c.Sheet, c.HeaderRow)
	if e != nil {
		return nil, nil, nil, e
	}
	warnings := []string{"Проверьте соответствие столбцов и единицы. Формулы не вычисляются: используются сохранённые Excel результаты. Неотображаемые столбцы сохраняются в источнике записи."}
	for _, spec := range importFields {
		if !spec.Required && c.Columns[spec.Key] == 0 {
			warnings = append(warnings, "Не сопоставлено: "+spec.Label)
		}
	}
	rows := []ImportRow{}
	ignoredFormulaRows := 0
	iterator, e := book.Rows(c.Sheet)
	if e != nil {
		return nil, nil, nil, invalid("Не удалось прочитать лист.")
	}
	defer iterator.Close()
	for row := 1; iterator.Next(); row++ {
		if row <= c.HeaderRow {
			continue
		}
		values, e := iterator.Columns(excelize.Options{RawCellValue: true})
		if e != nil {
			return nil, nil, nil, invalid("Не удалось прочитать строку Excel.")
		}
		populated := false
		for _, col := range c.Columns {
			if col > 0 && col <= len(values) && strings.TrimSpace(values[col-1]) != "" {
				axis, _ := excelize.CoordinatesToCellName(col, row)
				formula, _ := book.GetCellFormula(c.Sheet, axis)
				if formula == "" || col == c.Columns["factoryNumber"] {
					populated = true
				}
			}
		}
		if !populated {
			if len(values) > 0 {
				ignoredFormulaRows++
			}
			continue
		}
		if len(rows) >= maxImportRows {
			return nil, nil, nil, invalid("За один импорт можно подготовить до 10 000 труб.")
		}
		item := ImportRow{SourceRow: row, Input: PipeInput{CertificateIDs: []string{}}, Issues: []ImportIssue{}}
		source := importSource{"excel", file.OriginalName, file.SHA256, c.Sheet, row, map[string]sourceCell{}}
		for col, raw := range values {
			if raw == "" {
				continue
			}
			axis, _ := excelize.CoordinatesToCellName(col+1, row)
			display, _ := book.GetCellValue(c.Sheet, axis)
			formula, _ := book.GetCellFormula(c.Sheet, axis)
			source.Cells[axis] = sourceCell{raw, display, formula}
		}
		for key, col := range c.Columns {
			if col <= 0 {
				continue
			}
			axis, _ := excelize.CoordinatesToCellName(col, row)
			value, e := book.GetCellValue(c.Sheet, axis)
			if e != nil {
				item.Issues = append(item.Issues, ImportIssue{"error", key, "Не удалось прочитать ячейку " + axis})
				continue
			}
			value = strings.TrimSpace(value)
			formula, _ := book.GetCellFormula(c.Sheet, axis)
			cellType, _ := book.GetCellType(c.Sheet, axis)
			if cellType == excelize.CellTypeError || formula != "" && value == "" {
				item.Issues = append(item.Issues, ImportIssue{"error", key, "Нет пригодного сохранённого результата в " + axis + ". Откройте и сохраните книгу в Excel либо исправьте строку вручную."})
				continue
			}
			if target, ok := textFields(&item.Input)[key]; ok {
				*target = value
			} else if target, ok := numberFields(&item.Input)[key]; ok {
				raw, _ := book.GetCellValue(c.Sheet, axis, excelize.Options{RawCellValue: true})
				n, e := parseNumber(raw, c.Units[key])
				if e == nil && n != nil && c.Units[key] == "percent" && cellType != excelize.CellTypeSharedString && cellType != excelize.CellTypeInlineString && strings.Contains(value, "%") {
					*n *= 100
				}
				if e != nil {
					item.Issues = append(item.Issues, ImportIssue{"error", key, axis + ": " + e.Error()})
				} else {
					*target = n
				}
			} else if key == "certificateReference" {
				item.Input.NewCertificate = parseCertificate(value)
			}
		}
		if item.Input.InventoryNumber == "" {
			item.Input.InventoryNumber = item.Input.FactoryNumber
		}
		attributes, _ := json.Marshal(map[string]any{"schemaVersion": 1, "source": source, "customFields": []any{}})
		item.Input.AttributesJSON = string(attributes)
		if item.Input.NewCertificate != nil && item.Input.NewCertificate.Number == "" {
			item.Issues = append(item.Issues, ImportIssue{"warning", "certificateReference", "Реквизиты сохранены целиком. Проверьте номер и дату сертификата вручную."})
		}
		rows = append(rows, item)
	}
	if e = iterator.Error(); e != nil {
		return nil, nil, nil, invalid("Ошибка чтения листа Excel.")
	}
	if ignoredFormulaRows > 0 {
		warnings = append(warnings, fmt.Sprintf("Пропущено пустых строк с формулами/вспомогательными ячейками: %d. Они не являются записями труб.", ignoredFormulaRows))
	}
	return rows, columns, warnings, nil
}
