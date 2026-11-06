package registry

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"

	"changeme/internal/apperror"
)

var errStorage = apperror.New(apperror.InternalError, "Не удалось выполнить операцию с локальными данными. Повторите попытку.")

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const pipeColumns = "id,inventory_number,factory_number,manufacturer,diameter_mm,wall_thickness_mm,length_mm,mass_kg,strength_class,steel_grade,heat_number,carbon_equivalent_pct,pipe_specification,coating_type,coating_specification,test_pressure_mpa,min_operating_temp_c,incoming_inspection_reference,allowed_category54,allowed_category74,notes,attributes_json,revision,created_at,updated_at"

type scanner interface{ Scan(...any) error }

func readPipe(row scanner) (Pipe, error) {
	pipe := Pipe{Certificates: []Certificate{}}
	err := row.Scan(&pipe.ID, &pipe.InventoryNumber, &pipe.FactoryNumber, &pipe.Manufacturer, &pipe.DiameterMm, &pipe.WallThicknessMm, &pipe.LengthMm, &pipe.MassKg, &pipe.StrengthClass, &pipe.SteelGrade, &pipe.HeatNumber, &pipe.CarbonEquivalentPct, &pipe.PipeSpecification, &pipe.CoatingType, &pipe.CoatingSpecification, &pipe.TestPressureMpa, &pipe.MinOperatingTempC, &pipe.IncomingInspectionReference, &pipe.AllowedCategory54, &pipe.AllowedCategory74, &pipe.Notes, &pipe.AttributesJSON, &pipe.Revision, &pipe.CreatedAt, &pipe.UpdatedAt)
	return pipe, err
}

func pipeValues(input PipeInput) []any {
	return []any{input.InventoryNumber, input.FactoryNumber, input.Manufacturer, input.DiameterMm, input.WallThicknessMm, input.LengthMm, input.MassKg, input.StrengthClass, input.SteelGrade, input.HeatNumber, input.CarbonEquivalentPct, input.PipeSpecification, input.CoatingType, input.CoatingSpecification, input.TestPressureMpa, input.MinOperatingTempC, input.IncomingInspectionReference, input.AllowedCategory54, input.AllowedCategory74, input.Notes, input.AttributesJSON}
}

func (r *Repository) ListPipes() ([]Pipe, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	rows, err := tx.Query("SELECT " + pipeColumns + " FROM pipes ORDER BY created_at DESC,id DESC")
	if err != nil {
		return nil, errStorage
	}
	pipes := []Pipe{}
	positions := map[string]int{}
	for rows.Next() {
		pipe, err := readPipe(rows)
		if err != nil {
			rows.Close()
			return nil, errStorage
		}
		positions[pipe.ID] = len(pipes)
		pipes = append(pipes, pipe)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, errStorage
	}
	rows, err = tx.Query("SELECT pc.pipe_id,c.id,c.reference,c.number,c.issued_on,c.page_count,c.issuer FROM pipe_certificates pc JOIN certificates c ON c.id=pc.certificate_id ORDER BY c.reference,c.id")
	if err != nil {
		return nil, errStorage
	}
	for rows.Next() {
		var id string
		var certificate Certificate
		if err = rows.Scan(&id, &certificate.ID, &certificate.Reference, &certificate.Number, &certificate.IssuedOn, &certificate.PageCount, &certificate.Issuer); err != nil {
			rows.Close()
			return nil, errStorage
		}
		if index, ok := positions[id]; ok {
			pipes[index].Certificates = append(pipes[index].Certificates, certificate)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, errStorage
	}
	if err = tx.Commit(); err != nil {
		return nil, errStorage
	}
	return pipes, nil
}

func (r *Repository) ListCertificates() ([]Certificate, error) {
	rows, err := r.db.Query("SELECT id,reference,number,issued_on,page_count,issuer FROM certificates ORDER BY reference,id")
	if err != nil {
		return nil, errStorage
	}
	defer rows.Close()
	certificates := []Certificate{}
	for rows.Next() {
		var certificate Certificate
		if err = rows.Scan(&certificate.ID, &certificate.Reference, &certificate.Number, &certificate.IssuedOn, &certificate.PageCount, &certificate.Issuer); err != nil {
			return nil, errStorage
		}
		certificates = append(certificates, certificate)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return certificates, nil
}

func getPipe(tx *sql.Tx, id string) (Pipe, error) {
	pipe, err := readPipe(tx.QueryRow("SELECT "+pipeColumns+" FROM pipes WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Pipe{}, apperror.New(apperror.NotFound, "Запись трубы не найдена.")
	}
	if err != nil {
		return Pipe{}, errStorage
	}
	rows, err := tx.Query("SELECT c.id,c.reference,c.number,c.issued_on,c.page_count,c.issuer FROM certificates c JOIN pipe_certificates pc ON pc.certificate_id=c.id WHERE pc.pipe_id=? ORDER BY c.reference,c.id", id)
	if err != nil {
		return Pipe{}, errStorage
	}
	defer rows.Close()
	for rows.Next() {
		var certificate Certificate
		if err = rows.Scan(&certificate.ID, &certificate.Reference, &certificate.Number, &certificate.IssuedOn, &certificate.PageCount, &certificate.Issuer); err != nil {
			return Pipe{}, errStorage
		}
		pipe.Certificates = append(pipe.Certificates, certificate)
	}
	if rows.Err() != nil {
		return Pipe{}, errStorage
	}
	return pipe, nil
}

func (r *Repository) CreatePipe(input PipeInput, actorID string) (Pipe, error) {
	return r.savePipe("", 0, input, actorID)
}

func (r *Repository) UpdatePipe(id string, revision int, input PipeInput, actorID string) (Pipe, error) {
	if id == "" || revision <= 0 {
		return Pipe{}, invalid("Обновите список труб и повторите редактирование.")
	}
	return r.savePipe(id, revision, input, actorID)
}

func (r *Repository) savePipe(id string, revision int, raw PipeInput, actorID string) (Pipe, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return Pipe{}, errStorage
	}
	defer tx.Rollback()
	pipe, err := savePipeTx(tx, id, revision, raw, actorID)
	if err != nil {
		return Pipe{}, err
	}
	if err = tx.Commit(); err != nil {
		return Pipe{}, errStorage
	}
	return pipe, nil
}

// Used by manual edits and the atomic import; both share identical validation.
func savePipeTx(tx *sql.Tx, id string, revision int, raw PipeInput, actorID string) (Pipe, error) {
	input, err := normalizePipe(raw)
	if err != nil {
		return Pipe{}, err
	}
	if id != "" {
		current, err := getPipe(tx, id)
		if err != nil {
			return Pipe{}, err
		}
		if current.Revision != revision {
			return Pipe{}, apperror.New(apperror.Conflict, "Труба уже изменена. Обновите журнал и повторите редактирование.")
		}
	}
	var count int
	if err = tx.QueryRow("SELECT COUNT(*) FROM pipes WHERE inventory_number=? AND id<>?", input.InventoryNumber, id).Scan(&count); err != nil {
		return Pipe{}, errStorage
	}
	if count != 0 {
		return Pipe{}, apperror.New(apperror.Conflict, "Инвентарный номер записи уже занят. Заводские номера разных труб могут совпадать — задайте разные инвентарные номера.")
	}
	for _, certificateID := range input.CertificateIDs {
		if err = tx.QueryRow("SELECT COUNT(*) FROM certificates WHERE id=?", certificateID).Scan(&count); err != nil {
			return Pipe{}, errStorage
		}
		if count == 0 {
			return Pipe{}, apperror.New(apperror.NotFound, "Выбранный сертификат не найден. Обновите журнал.")
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if id == "" {
		id = "p_" + rand.Text()
		values := append([]any{id}, pipeValues(input)...)
		values = append(values, now, now, actorID, actorID)
		_, err = tx.Exec("INSERT INTO pipes(id,inventory_number,factory_number,manufacturer,diameter_mm,wall_thickness_mm,length_mm,mass_kg,strength_class,steel_grade,heat_number,carbon_equivalent_pct,pipe_specification,coating_type,coating_specification,test_pressure_mpa,min_operating_temp_c,incoming_inspection_reference,allowed_category54,allowed_category74,notes,attributes_json,created_at,updated_at,created_by,updated_by) VALUES("+strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")+")", values...)
	} else {
		values := append(pipeValues(input), now, actorID, id, revision)
		_, err = tx.Exec("UPDATE pipes SET inventory_number=?,factory_number=?,manufacturer=?,diameter_mm=?,wall_thickness_mm=?,length_mm=?,mass_kg=?,strength_class=?,steel_grade=?,heat_number=?,carbon_equivalent_pct=?,pipe_specification=?,coating_type=?,coating_specification=?,test_pressure_mpa=?,min_operating_temp_c=?,incoming_inspection_reference=?,allowed_category54=?,allowed_category74=?,notes=?,attributes_json=?,revision=revision+1,updated_at=?,updated_by=? WHERE id=? AND revision=?", values...)
	}
	if err != nil {
		return Pipe{}, errStorage
	}
	if _, err = tx.Exec("DELETE FROM pipe_certificates WHERE pipe_id=?", id); err != nil {
		return Pipe{}, errStorage
	}
	ids := append([]string(nil), input.CertificateIDs...)
	if certificate := input.NewCertificate; certificate != nil {
		certificateID := "c_" + rand.Text()
		if _, err = tx.Exec("INSERT INTO certificates(id,reference,number,issued_on,page_count,issuer,created_at,created_by) VALUES(?,?,?,?,?,?,?,?)", certificateID, certificate.Reference, certificate.Number, certificate.IssuedOn, certificate.PageCount, certificate.Issuer, now, actorID); err != nil {
			return Pipe{}, errStorage
		}
		ids = append(ids, certificateID)
	}
	for _, certificateID := range ids {
		if _, err = tx.Exec("INSERT INTO pipe_certificates(pipe_id,certificate_id) VALUES(?,?)", id, certificateID); err != nil {
			return Pipe{}, errStorage
		}
	}
	pipe, err := getPipe(tx, id)
	if err != nil {
		return Pipe{}, err
	}
	return pipe, nil
}
