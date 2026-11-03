package registry

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"changeme/internal/apperror"
	"changeme/internal/documents"
)

type ImportManager struct {
	db        *sql.DB
	documents *documents.Store
	mu        sync.Mutex
}

func NewImportManager(db *sql.DB, store *documents.Store) *ImportManager {
	return &ImportManager{db: db, documents: store}
}
func conflict() error {
	return apperror.New(apperror.Conflict, "Предпросмотр или журнал изменён. Обновите предпросмотр и проверьте строки ещё раз.")
}
func jsonText(value any) string { b, _ := json.Marshal(value); return string(b) }

func (m *ImportManager) Prepare(path, actor string) (*ImportPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, e := m.documents.Ingest(path, "xlsx", actor, validateWorkbook)
	if e != nil {
		return nil, e
	}
	// Resume an unfinished draft. A committed file can be selected again to import
	// previously skipped rows; provenance prevents already imported rows being added twice.
	var id string
	e = m.db.QueryRow("SELECT id FROM registry_imports WHERE document_id=? AND owner_id=? AND status='prepared' ORDER BY created_at DESC LIMIT 1", file.ID, actor).Scan(&id)
	if e == nil {
		return m.get(id, actor)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, errStorage
	}
	managed, e := m.documents.ManagedPath(file.ID)
	if e != nil {
		return nil, e
	}
	book, e := openWorkbook(managed)
	if e != nil {
		return nil, e
	}
	defer book.Close()
	c := defaultConfiguration(book)
	rows := []ImportRow{}
	if c.Columns["factoryNumber"] > 0 {
		rows, _, _, e = parseRows(book, file, c)
		if e != nil {
			return nil, e
		}
	}
	id = "i_" + rand.Text()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT INTO registry_imports(id,document_id,original_name,owner_id,configuration_json,created_at) VALUES(?,?,?,?,?,?)", id, file.ID, filepath.Base(path), actor, jsonText(c), now); e != nil {
		return nil, errStorage
	}
	if e = replaceImportRows(tx, id, rows); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	return m.get(id, actor)
}
func replaceImportRows(tx *sql.Tx, id string, rows []ImportRow) error {
	if _, e := tx.Exec("DELETE FROM registry_import_rows WHERE import_id=?", id); e != nil {
		return errStorage
	}
	for _, row := range rows {
		if _, e := tx.Exec("INSERT INTO registry_import_rows(id,import_id,source_row,input_json,issues_json) VALUES(?,?,?,?,?)", "ir_"+rand.Text(), id, row.SourceRow, jsonText(row.Input), jsonText(row.Issues)); e != nil {
			return errStorage
		}
	}
	return nil
}
func (m *ImportManager) Get(id, actor string) (*ImportPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.get(id, actor)
}
func loadPreview(tx *sql.Tx, id, actor string) (*ImportPreview, error) {
	p := &ImportPreview{Rows: []ImportRow{}, Fields: []ImportField{}, Sheets: []string{}, Columns: []ImportColumn{}, Warnings: []string{}}
	var owner, configuration string
	var result sql.NullString
	e := tx.QueryRow("SELECT document_id,original_name,owner_id,configuration_json,revision,status,created_at,result_json FROM registry_imports WHERE id=?", id).Scan(&p.DocumentID, &p.OriginalName, &owner, &configuration, &p.Revision, &p.Status, &p.CreatedAt, &result)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, apperror.New(apperror.NotFound, "Импорт не найден.")
	}
	if e != nil {
		return nil, errStorage
	}
	if owner != actor {
		return nil, apperror.New(apperror.Forbidden, "Этот черновик принадлежит другой учётной записи.")
	}
	p.ID = id
	if json.Unmarshal([]byte(configuration), &p.Configuration) != nil {
		return nil, errStorage
	}
	if result.Valid {
		if json.Unmarshal([]byte(result.String), &p.Result) != nil {
			return nil, errStorage
		}
	}
	rows, e := tx.Query("SELECT id,source_row,input_json,issues_json,COALESCE(pipe_id,'') FROM registry_import_rows WHERE import_id=? ORDER BY source_row", id)
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	for rows.Next() {
		var row ImportRow
		var input, issues string
		if e = rows.Scan(&row.ID, &row.SourceRow, &input, &issues, &row.ExistingPipeID); e != nil {
			return nil, errStorage
		}
		if json.Unmarshal([]byte(input), &row.Input) != nil || json.Unmarshal([]byte(issues), &row.Issues) != nil {
			return nil, errStorage
		}
		p.Rows = append(p.Rows, row)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return p, nil
}
func existingPipes(tx *sql.Tx) ([]Pipe, error) {
	rows, e := tx.Query("SELECT " + pipeColumns + " FROM pipes ORDER BY id")
	if e != nil {
		return nil, errStorage
	}
	pipes := []Pipe{}
	positions := map[string]int{}
	for rows.Next() {
		p, e := readPipe(rows)
		if e != nil {
			rows.Close()
			return nil, errStorage
		}
		positions[p.ID] = len(pipes)
		pipes = append(pipes, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, errStorage
	}
	rows, e = tx.Query("SELECT pc.pipe_id,c.id,c.reference,c.number,c.issued_on,c.page_count,c.issuer FROM pipe_certificates pc JOIN certificates c ON c.id=pc.certificate_id")
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var c Certificate
		if e = rows.Scan(&id, &c.ID, &c.Reference, &c.Number, &c.IssuedOn, &c.PageCount, &c.Issuer); e != nil {
			return nil, errStorage
		}
		if pos, ok := positions[id]; ok {
			pipes[pos].Certificates = append(pipes[pos].Certificates, c)
		}
	}
	return pipes, rows.Err()
}
func (m *ImportManager) get(id, actor string) (*ImportPreview, error) {
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	p, e := loadPreview(tx, id, actor)
	if e != nil {
		return nil, e
	}
	imported := map[string]bool{}
	for _, row := range p.Rows {
		if row.ExistingPipeID != "" {
			imported[row.ID] = true
		}
	}
	pipes, e := existingPipes(tx)
	if e != nil {
		return nil, e
	}
	classifyRows(p.Rows, pipes)
	if p.Status == "committed" {
		for i := range p.Rows {
			if imported[p.Rows[i].ID] {
				p.Rows[i].Status = "imported"
			}
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	file, e := m.documents.Get(p.DocumentID)
	if e != nil {
		return nil, e
	}
	p.SHA256 = file.SHA256
	path, e := m.documents.ManagedPath(p.DocumentID)
	if e != nil {
		return nil, e
	}
	book, e := openWorkbook(path)
	if e != nil {
		return nil, e
	}
	defer book.Close()
	p.Sheets = book.GetSheetList()
	p.Columns, e = sheetHeaders(book, p.Configuration.Sheet, p.Configuration.HeaderRow)
	if e != nil {
		return nil, e
	}
	for _, spec := range importFields {
		p.Fields = append(p.Fields, spec.ImportField)
	}
	p.Warnings = append(p.Warnings, "Формулы не пересчитываются. Проверьте столбцы, единицы, реквизиты и выбранные строки. Исходные значения и оригинал сохраняются.")
	p.Warnings = append(p.Warnings, "Пустые строки, содержащие только вспомогательные значения или формулы без номера трубы и исходных характеристик, пропускаются.")
	for _, spec := range importFields {
		if p.Configuration.Columns[spec.Key] == 0 {
			p.Warnings = append(p.Warnings, "Не сопоставлено: "+spec.Label)
		}
		matches := 0
		for _, column := range p.Columns {
			for _, alias := range spec.aliases {
				if heading(alias) == heading(column.Label) {
					matches++
					break
				}
			}
		}
		if matches > 1 {
			p.Warnings = append(p.Warnings, "Несколько подходящих столбцов: "+spec.Label+". Проверьте выбранный столбец; вспомогательный справочник не должен заменять характеристику трубы.")
		}
	}
	return p, nil
}
func (m *ImportManager) Configure(id, actor string, revision int, c ImportConfiguration) (*ImportPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, e := m.get(id, actor)
	if e != nil {
		return nil, e
	}
	if current.Status != "prepared" || current.Revision != revision {
		return nil, conflict()
	}
	file, e := m.documents.Get(current.DocumentID)
	if e != nil {
		return nil, e
	}
	path, e := m.documents.ManagedPath(file.ID)
	if e != nil {
		return nil, e
	}
	book, e := openWorkbook(path)
	if e != nil {
		return nil, e
	}
	defer book.Close()
	if len(c.Columns) == 0 {
		columns, err := sheetHeaders(book, c.Sheet, c.HeaderRow)
		if err != nil {
			return nil, err
		}
		c.Columns = suggestMapping(columns)
		c.Units = suggestUnits(columns, c.Columns)
	}
	rows, _, _, e := parseRows(book, file, c)
	if e != nil {
		return nil, e
	}
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	res, e := tx.Exec("UPDATE registry_imports SET configuration_json=?,revision=revision+1 WHERE id=? AND owner_id=? AND revision=? AND status='prepared'", jsonText(c), id, actor, revision)
	if e != nil {
		return nil, errStorage
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, conflict()
	}
	if e = replaceImportRows(tx, id, rows); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	return m.get(id, actor)
}
func (m *ImportManager) EditRow(id, actor string, revision int, rowID string, input PipeInput) (*ImportPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	p, e := loadPreview(tx, id, actor)
	if e != nil {
		return nil, e
	}
	if p.Revision != revision || p.Status != "prepared" {
		return nil, conflict()
	}
	found := false
	for _, row := range p.Rows {
		if row.ID != rowID {
			continue
		}
		found = true
		// The source identity/cells come from the native parser, never from edited WebView data.
		var old, edited map[string]json.RawMessage
		if json.Unmarshal([]byte(input.AttributesJSON), &edited) != nil || edited == nil {
			return nil, invalid("Некорректные дополнительные сведения.")
		}
		_ = json.Unmarshal([]byte(row.Input.AttributesJSON), &old)
		edited["source"] = old["source"]
		input.AttributesJSON = jsonText(edited)
	}
	if !found {
		return nil, apperror.New(apperror.NotFound, "Строка импорта не найдена.")
	}
	input, e = normalizePipe(input)
	if e != nil {
		return nil, e
	}
	if _, e = tx.Exec("UPDATE registry_import_rows SET input_json=?,issues_json='[]' WHERE id=? AND import_id=?", jsonText(input), rowID, id); e != nil {
		return nil, errStorage
	}
	if _, e = tx.Exec("UPDATE registry_imports SET revision=revision+1 WHERE id=?", id); e != nil {
		return nil, errStorage
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	return m.get(id, actor)
}
func (m *ImportManager) Commit(id, actor string, revision int, selection []string) (*ImportResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(selection) == 0 || len(selection) > maxImportRows {
		return nil, invalid("Выберите строки для добавления.")
	}
	selected := append([]string{}, selection...)
	sort.Strings(selected)
	for i := 1; i < len(selected); i++ {
		if selected[i] == selected[i-1] {
			return nil, invalid("Строки выбора не должны повторяться.")
		}
	}
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	p, e := loadPreview(tx, id, actor)
	if e != nil {
		return nil, e
	}
	if p.Status == "committed" {
		var prior string
		if e = tx.QueryRow("SELECT selection_json FROM registry_imports WHERE id=?", id).Scan(&prior); e != nil {
			return nil, errStorage
		}
		if prior == jsonText(selected) {
			return p.Result, nil
		}
		return nil, conflict()
	}
	if p.Revision != revision {
		return nil, conflict()
	}
	pipes, e := existingPipes(tx)
	if e != nil {
		return nil, e
	}
	classifyRows(p.Rows, pipes)
	wanted := map[string]bool{}
	for _, s := range selected {
		wanted[s] = true
	}
	found := 0
	existing := 0
	for _, row := range p.Rows {
		if row.Status == "existing" {
			existing++
		}
		if wanted[row.ID] {
			found++
			if row.Status != "new" {
				return nil, conflict()
			}
		}
	}
	if found != len(selected) {
		return nil, invalid("Выбрана неизвестная строка импорта.")
	}
	certificates := map[string]string{}
	for _, row := range p.Rows {
		if !wanted[row.ID] {
			continue
		}
		input := row.Input
		if c := input.NewCertificate; c != nil {
			key := jsonText([]string{strings.ToLower(strings.TrimSpace(input.Manufacturer)), jsonText(c)})
			certID := certificates[key]
			if certID == "" {
				certID, e = findImportCertificate(tx, *c, input.Manufacturer)
				if e != nil {
					return nil, e
				}
			}
			if certID == "" {
				certID = "c_" + rand.Text()
				if _, e = tx.Exec("INSERT INTO certificates(id,reference,number,issued_on,page_count,issuer,created_at,created_by) VALUES(?,?,?,?,?,?,?,?)", certID, c.Reference, c.Number, c.IssuedOn, c.PageCount, c.Issuer, time.Now().UTC().Format(time.RFC3339Nano), actor); e != nil {
					return nil, errStorage
				}
			}
			input.NewCertificate = nil
			present := false
			for _, id := range input.CertificateIDs {
				if id == certID {
					present = true
				}
			}
			if !present {
				input.CertificateIDs = append(input.CertificateIDs, certID)
			}
			pipe, e := savePipeTx(tx, "", 0, input, actor)
			if e != nil {
				return nil, e
			}
			certificates[key] = certID
			if _, e = tx.Exec("UPDATE registry_import_rows SET pipe_id=? WHERE id=?", pipe.ID, row.ID); e != nil {
				return nil, errStorage
			}
		} else {
			pipe, e := savePipeTx(tx, "", 0, input, actor)
			if e != nil {
				return nil, e
			}
			if _, e = tx.Exec("UPDATE registry_import_rows SET pipe_id=? WHERE id=?", pipe.ID, row.ID); e != nil {
				return nil, errStorage
			}
		}
	}
	result := &ImportResult{id, len(selected), existing, len(p.Rows) - len(selected) - existing, time.Now().UTC().Format(time.RFC3339Nano)}
	if _, e = tx.Exec("UPDATE registry_imports SET status='committed',revision=revision+1,result_json=?,selection_json=?,committed_at=? WHERE id=?", jsonText(result), jsonText(selected), result.CommittedAt, id); e != nil {
		return nil, errStorage
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	return result, nil
}
func findImportCertificate(tx *sql.Tx, c CertificateDraft, manufacturer string) (string, error) {
	if c.Number == "" || c.IssuedOn == "" || strings.TrimSpace(manufacturer) == "" {
		return "", nil
	}
	rows, e := tx.Query("SELECT c.id,p.manufacturer FROM certificates c JOIN pipe_certificates pc ON pc.certificate_id=c.id JOIN pipes p ON p.id=pc.pipe_id WHERE c.reference=? AND c.number=? AND c.issued_on=? AND c.page_count IS ?", c.Reference, c.Number, c.IssuedOn, c.PageCount)
	if e != nil {
		return "", errStorage
	}
	defer rows.Close()
	ids := map[string]bool{}
	incompatible := map[string]bool{}
	for rows.Next() {
		var id, m string
		if rows.Scan(&id, &m) != nil {
			return "", errStorage
		}
		if strings.EqualFold(strings.TrimSpace(m), strings.TrimSpace(manufacturer)) {
			ids[id] = true
		} else {
			incompatible[id] = true
		}
	}
	if rows.Err() != nil {
		return "", errStorage
	}
	if len(ids) == 1 {
		for id := range ids {
			if !incompatible[id] {
				return id, nil
			}
		}
	}
	return "", nil
}
func (m *ImportManager) List(actor string) ([]ImportSummary, error) {
	rows, e := m.db.Query("SELECT id,original_name,configuration_json,status,created_at,result_json FROM registry_imports WHERE owner_id=? ORDER BY created_at DESC LIMIT 100", actor)
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	out := []ImportSummary{}
	for rows.Next() {
		var s ImportSummary
		var c string
		var result sql.NullString
		if rows.Scan(&s.ID, &s.OriginalName, &c, &s.Status, &s.CreatedAt, &result) != nil {
			return nil, errStorage
		}
		var config ImportConfiguration
		if json.Unmarshal([]byte(c), &config) != nil {
			return nil, errStorage
		}
		s.Sheet = config.Sheet
		if result.Valid && json.Unmarshal([]byte(result.String), &s.Result) != nil {
			return nil, errStorage
		}
		out = append(out, s)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}
func (m *ImportManager) OriginalPath(id, actor string) (string, error) {
	tx, e := m.db.Begin()
	if e != nil {
		return "", errStorage
	}
	p, e := loadPreview(tx, id, actor)
	tx.Rollback()
	if e != nil {
		return "", e
	}
	return m.documents.ManagedPath(p.DocumentID)
}
