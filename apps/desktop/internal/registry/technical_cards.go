package registry

import (
	"changeme/internal/apperror"
	"changeme/internal/documents"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"
)

type TechnicalCardManager struct {
	db    *sql.DB
	store *documents.Store
}

func NewTechnicalCardManager(db *sql.DB, store *documents.Store) *TechnicalCardManager {
	return &TechnicalCardManager{db, store}
}
func (m *TechnicalCardManager) Prepare(path, machineID, actor string) (*TechnicalCardPreview, error) {
	tx, e := m.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	machine, e := getMachine(tx, machineID)
	tx.Rollback()
	if e != nil {
		return nil, e
	}
	var parsed []TechnicalCardRow
	var warnings []string
	file, e := m.store.Ingest(path, "docx", actor, func(f *os.File) error {
		var err error
		parsed, warnings, err = ParseTechnicalCard(f, machine)
		return err
	})
	if e != nil {
		return nil, e
	}
	for i := range parsed {
		parsed[i].SourceJSON = cardSource(file.OriginalName, file.SHA256, parsed[i].SourceJSON)
		parsed[i].Input.TechnicalCardReference = file.OriginalName + " · " + parsed[i].Input.TechnicalCardReference
	}
	preview := TechnicalCardPreview{ID: "tc_" + rand.Text(), DocumentID: file.ID, OriginalName: file.OriginalName, SHA256: file.SHA256, Machine: machine, Rows: parsed, Warnings: warnings}
	_, e = m.db.Exec("INSERT INTO technical_cards(id,document_id,machine_id,owner_id,parsed_json,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(document_id,machine_id) DO NOTHING", preview.ID, file.ID, machine.ID, actor, jsonText(preview), time.Now().UTC().Format(time.RFC3339Nano))
	if e != nil {
		return nil, errStorage
	}
	var id string
	if e = m.db.QueryRow("SELECT id FROM technical_cards WHERE document_id=? AND machine_id=?", file.ID, machine.ID).Scan(&id); e != nil {
		return nil, errStorage
	}
	return m.Get(id)
}
func (m *TechnicalCardManager) Get(id string) (*TechnicalCardPreview, error) {
	var raw string
	e := m.db.QueryRow("SELECT parsed_json FROM technical_cards WHERE id=?", id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, apperror.New(apperror.NotFound, "Техкарта не найдена.")
	}
	if e != nil {
		return nil, errStorage
	}
	var p TechnicalCardPreview
	if json.Unmarshal([]byte(raw), &p) != nil {
		return nil, errStorage
	}
	p.ID = id
	rows, e := m.db.Query("SELECT source_key,id FROM technological_configurations WHERE technical_card_id=?", id)
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	existing := map[string]string{}
	for rows.Next() {
		var key, cid string
		if rows.Scan(&key, &cid) != nil {
			return nil, errStorage
		}
		existing[key] = cid
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	for i := range p.Rows {
		p.Rows[i].ExistingID = existing[p.Rows[i].Key]
	}
	return &p, nil
}
func (m *TechnicalCardManager) List() ([]TechnicalCardPreview, error) {
	rows, e := m.db.Query("SELECT id,parsed_json FROM technical_cards ORDER BY created_at DESC LIMIT 100")
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	out := []TechnicalCardPreview{}
	for rows.Next() {
		var id, raw string
		if rows.Scan(&id, &raw) != nil {
			return nil, errStorage
		}
		var p TechnicalCardPreview
		if json.Unmarshal([]byte(raw), &p) != nil {
			return nil, errStorage
		}
		p.ID = id
		p.Rows = []TechnicalCardRow{}
		out = append(out, p)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}
func (m *TechnicalCardManager) Commit(id string, keys []string, actor string) (int, error) {
	if len(keys) == 0 || len(keys) > 1000 {
		return 0, invalid("Выберите конфигурации для импорта.")
	}
	p, e := m.Get(id)
	if e != nil {
		return 0, e
	}
	selected := map[string]bool{}
	for _, key := range keys {
		if key == "" || !validText(key, 160) {
			return 0, invalid("Некорректный идентификатор строки ТК.")
		}
		if selected[key] {
			return 0, invalid("Строки выбора повторяются.")
		}
		selected[key] = true
	}
	tx, e := m.db.Begin()
	if e != nil {
		return 0, errStorage
	}
	defer tx.Rollback()
	found, added := 0, 0
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, row := range p.Rows {
		if !selected[row.Key] {
			continue
		}
		found++
		input, e := normalizeConfiguration(row.Input)
		if e != nil {
			return 0, e
		}
		var existing int
		if e = tx.QueryRow("SELECT COUNT(*) FROM technological_configurations WHERE technical_card_id=? AND source_key=?", id, row.Key).Scan(&existing); e != nil {
			return 0, errStorage
		}
		if existing > 0 {
			continue
		}
		_, e = tx.Exec("INSERT INTO technological_configurations(id,name,version,machine,technical_card_reference,diameter_mm,parameters_json,notes,created_at,updated_at,created_by,updated_by,machine_id,technical_card_id,source_key,source_json,needs_review,bend_type,target_angle,min_pipe_length_mm) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?)", "cfg_"+rand.Text(), input.Name, input.Version, input.Machine, input.TechnicalCardReference, input.DiameterMm, jsonText(input.Parameters), input.Notes, now, now, actor, actor, p.Machine.ID, id, row.Key, row.SourceJSON, input.BendType, input.TargetAngle, input.MinPipeLengthMm)
		if e != nil {
			return 0, errStorage
		}
		added++
	}
	if found != len(keys) {
		return 0, invalid("Выбрана неизвестная строка карты.")
	}
	if e = tx.Commit(); e != nil {
		return 0, errStorage
	}
	return added, nil
}
func (m *TechnicalCardManager) OriginalPath(id string) (string, error) {
	p, e := m.Get(id)
	if e != nil {
		return "", e
	}
	return m.store.ManagedPath(p.DocumentID)
}
