package registry

import (
	"changeme/internal/apperror"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const configurationColumns = "id,name,version,machine,technical_card_reference,diameter_mm,parameters_json,notes,revision,created_at,updated_at,COALESCE(machine_id,''),COALESCE(technical_card_id,''),source_json,needs_review,bend_type,target_angle,min_pipe_length_mm"

func normalizeConfiguration(input ConfigurationInput) (ConfigurationInput, error) {
	input.MachineID = strings.TrimSpace(input.MachineID)
	if input.MachineID == "" {
		return input, invalid("Выберите станок из каталога оборудования.")
	}
	if strings.TrimSpace(input.Machine) == "" && input.MachineID != "" {
		input.Machine = input.MachineID
	}
	for _, s := range []*string{&input.Name, &input.Version, &input.Machine, &input.TechnicalCardReference} {
		*s = strings.TrimSpace(*s)
		if *s == "" || !validText(*s, 1000) {
			return input, invalid("Укажите название, версию, модель станка и реквизиты технологической карты (до 1000 символов).")
		}
	}
	if !finite(input.DiameterMm) || input.DiameterMm <= 0 {
		return input, invalid("Укажите положительный диаметр трубы в миллиметрах для конфигурации.")
	}
	if !validText(input.Machine, 160) || !validText(input.Version, 160) {
		return input, invalid("Модель станка и версия конфигурации должны содержать не более 160 символов.")
	}
	input.Notes = strings.TrimSpace(input.Notes)
	input.MachineID = strings.TrimSpace(input.MachineID)
	if !validText(input.MachineID, 160) || (input.BendType != "" && input.BendType != "1ГО" && input.BendType != "2ГО") {
		return input, invalid("Проверьте станок и тип отвода конфигурации.")
	}
	if input.TargetAngle != "" {
		v, e := parseNumber(input.TargetAngle, "")
		if e != nil || v == nil || *v <= 0 {
			return input, invalid("Угол конфигурации должен быть положительным числом.")
		}
		input.TargetAngle = strconv.FormatFloat(*v, 'f', -1, 64)
	}
	if input.MinPipeLengthMm != nil && (!finite(*input.MinPipeLengthMm) || *input.MinPipeLengthMm <= 0) {
		return input, invalid("Минимальная длина должна быть положительным числом.")
	}
	if !validText(input.Notes, 4000) || len(input.Parameters) > 100 {
		return input, invalid("Примечание — до 4000 символов, параметров — не более 100.")
	}
	if input.Parameters == nil {
		input.Parameters = []ConfigurationParameter{}
	}
	seen := map[string]bool{}
	for i, p := range input.Parameters {
		p.Name = strings.TrimSpace(p.Name)
		p.Value = strings.TrimSpace(p.Value)
		p.Unit = strings.TrimSpace(p.Unit)
		key := strings.ToLower(p.Name)
		if p.Name == "" || p.Value == "" || !validText(p.Name, 160) || !validText(p.Value, 1000) || !validText(p.Unit, 80) || seen[key] {
			return input, invalid("У каждого параметра нужны уникальное название и значение из ТК. Единицу укажите по документу.")
		}
		seen[key] = true
		input.Parameters[i] = p
	}
	return input, nil
}
func readConfiguration(row scanner) (Configuration, error) {
	var c Configuration
	var p string
	e := row.Scan(&c.ID, &c.Name, &c.Version, &c.Machine, &c.TechnicalCardReference, &c.DiameterMm, &p, &c.Notes, &c.Revision, &c.CreatedAt, &c.UpdatedAt, &c.MachineID, &c.TechnicalCardID, &c.SourceJSON, &c.NeedsReview, &c.BendType, &c.TargetAngle, &c.MinPipeLengthMm)
	if e != nil {
		return c, e
	}
	if json.Unmarshal([]byte(p), &c.Parameters) != nil {
		return c, errStorage
	}
	if c.Parameters == nil {
		c.Parameters = []ConfigurationParameter{}
	}
	return c, nil
}
func getConfiguration(tx *sql.Tx, id string) (Configuration, error) {
	c, e := readConfiguration(tx.QueryRow("SELECT "+configurationColumns+" FROM technological_configurations WHERE id=?", id))
	if errors.Is(e, sql.ErrNoRows) {
		return c, apperror.New(apperror.NotFound, "Конфигурация не найдена.")
	}
	if e != nil {
		return c, errStorage
	}
	return c, nil
}
func (r *Repository) ListConfigurations() ([]Configuration, error) {
	rows, e := r.db.Query("SELECT " + configurationColumns + " FROM technological_configurations ORDER BY name,version,id")
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	out := []Configuration{}
	for rows.Next() {
		c, e := readConfiguration(rows)
		if e != nil {
			return nil, errStorage
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}
func (r *Repository) SaveConfiguration(id string, revision int, raw ConfigurationInput, actor string) (Configuration, error) {
	input, e := normalizeConfiguration(raw)
	if e != nil {
		return Configuration{}, e
	}
	if id != "" && revision <= 0 {
		return Configuration{}, invalid("Обновите конфигурацию перед изменением.")
	}
	tx, e := r.db.Begin()
	if e != nil {
		return Configuration{}, errStorage
	}
	defer tx.Rollback()
	var machineID any
	if input.MachineID != "" {
		m, e := getMachine(tx, input.MachineID)
		if e != nil {
			return Configuration{}, e
		}
		machineID = m.ID
		input.Machine = m.Name
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if id == "" {
		id = "cfg_" + rand.Text()
		_, e = tx.Exec("INSERT INTO technological_configurations(id,name,version,machine,technical_card_reference,diameter_mm,parameters_json,notes,created_at,updated_at,created_by,updated_by,machine_id,bend_type,target_angle,min_pipe_length_mm) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, input.Name, input.Version, input.Machine, input.TechnicalCardReference, input.DiameterMm, jsonText(input.Parameters), input.Notes, now, now, actor, actor, machineID, input.BendType, input.TargetAngle, input.MinPipeLengthMm)
	} else {
		c, err := getConfiguration(tx, id)
		if err != nil {
			return c, err
		}
		if c.Revision != revision {
			return Configuration{}, apperror.New(apperror.Conflict, "Конфигурация изменена. Обновите список и повторите редактирование.")
		}
		if c.TechnicalCardID != "" && input.MachineID != c.MachineID {
			return Configuration{}, invalid("Станок импортированной конфигурации определяется исходной техкартой.")
		}
		needsReview := c.TechnicalCardID != "" && !input.ConfirmSourceReview
		_, e = tx.Exec("UPDATE technological_configurations SET name=?,version=?,machine=?,technical_card_reference=?,diameter_mm=?,parameters_json=?,notes=?,revision=revision+1,updated_at=?,updated_by=?,machine_id=?,bend_type=?,target_angle=?,min_pipe_length_mm=?,needs_review=? WHERE id=? AND revision=?", input.Name, input.Version, input.Machine, input.TechnicalCardReference, input.DiameterMm, jsonText(input.Parameters), input.Notes, now, actor, machineID, input.BendType, input.TargetAngle, input.MinPipeLengthMm, needsReview, id, revision)
	}
	if e != nil {
		return Configuration{}, errStorage
	}
	c, e := getConfiguration(tx, id)
	if e != nil {
		return c, e
	}
	if e = tx.Commit(); e != nil {
		return Configuration{}, errStorage
	}
	return c, nil
}
