package registry

import (
	"changeme/internal/apperror"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// The request ID makes retries of the same batch safe, including after a restart.
func (r *Repository) CreateTasks(batch TaskBatchInput, actor string) ([]Task, error) {
	batch.RequestID = strings.TrimSpace(batch.RequestID)
	if batch.RequestID == "" || !validText(batch.RequestID, 128) || len(batch.Tasks) == 0 || len(batch.Tasks) > 50 {
		return nil, invalid("Нужны идентификатор операции и от 1 до 50 заданий.")
	}
	inputs := make([]TaskInput, len(batch.Tasks))
	for i, raw := range batch.Tasks {
		input, e := normalizeTask(raw)
		if e != nil {
			return nil, taskRowError(i, e)
		}
		inputs[i] = input
	}
	hash := sha256.Sum256([]byte(jsonText(inputs)))
	digest := hex.EncodeToString(hash[:])
	tx, e := r.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	var owner, prior, payload string
	e = tx.QueryRow("SELECT actor_id,payload_hash,task_ids_json FROM task_creation_requests WHERE id=?", batch.RequestID).Scan(&owner, &payload, &prior)
	if e == nil {
		if owner != actor || payload != digest {
			return nil, apperror.New(apperror.Conflict, "Идентификатор операции уже использован с другими данными. Закройте форму и откройте её снова.")
		}
		var ids []string
		if json.Unmarshal([]byte(prior), &ids) != nil {
			return nil, errStorage
		}
		out := []Task{}
		for _, id := range ids {
			task, e := getTask(tx, id)
			if e != nil {
				return nil, e
			}
			out = append(out, task)
		}
		return out, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, errStorage
	}
	out := []Task{}
	ids := []string{}
	for i, input := range inputs {
		task, e := createTaskTx(tx, input, actor)
		if e != nil {
			return nil, taskRowError(i, e)
		}
		out = append(out, task)
		ids = append(ids, task.ID)
	}
	if _, e = tx.Exec("INSERT INTO task_creation_requests(id,actor_id,payload_hash,task_ids_json,created_at) VALUES(?,?,?,?,?)", batch.RequestID, actor, digest, jsonText(ids), time.Now().UTC().Format(time.RFC3339Nano)); e != nil {
		return nil, errStorage
	}
	if e = tx.Commit(); e != nil {
		return nil, errStorage
	}
	return out, nil
}
func taskRowError(index int, err error) error {
	payload := apperror.Describe(err)
	return apperror.New(payload.Code, fmt.Sprintf("Задание %d: %s", index+1, payload.Message))
}
func (r *Repository) PrepareTask(id string, revision int, actor string) (Task, error) {
	if revision <= 0 {
		return Task{}, invalid("Обновите карточку задания перед подготовкой.")
	}
	tx, e := r.db.Begin()
	if e != nil {
		return Task{}, errStorage
	}
	defer tx.Rollback()
	task, e := getTask(tx, id)
	if e != nil {
		return task, e
	}
	if (task.Status == "ready" || task.Status == "exported") && (task.Revision == revision || task.Revision == revision+1) {
		return task, nil
	}
	if task.Status != "draft" || revision <= 0 || task.Revision != revision {
		return Task{}, taskConflict()
	}
	input, e := normalizeTask(task.TaskInput)
	if e != nil {
		return Task{}, e
	}
	missing := []string{}
	for _, field := range []struct{ value, label string }{{input.Number, "номер задания"}, {input.BendNumber, "номер отвода"}, {input.BendType, "тип отвода"}, {input.TargetAngle, "целевой угол"}, {input.ConfigurationID, "технологическая конфигурация"}} {
		if field.value == "" {
			missing = append(missing, field.label)
		}
	}
	if len(missing) > 0 {
		return Task{}, invalid("Перед подготовкой заполните: " + strings.Join(missing, ", ") + ".")
	}
	pipe, c, e := taskSources(tx, &input)
	if e != nil {
		return Task{}, e
	}
	if c == nil {
		return Task{}, invalid("Выберите технологическую конфигурацию.")
	}
	if input.MachineID == "" || c.MachineID == "" || input.MachineID != c.MachineID {
		return Task{}, invalid("Выберите станок и связанную с ним конфигурацию.")
	}
	if c.NeedsReview {
		return Task{}, invalid("Проверьте импортированную конфигурацию по оригиналу ТК и подтвердите проверку в её карточке.")
	}
	if c.BendType != "" && c.BendType != input.BendType {
		return Task{}, invalid("Тип отвода отличается от табличного варианта конфигурации.")
	}
	if c.TargetAngle != "" && c.TargetAngle != input.TargetAngle {
		return Task{}, invalid("Целевой угол отличается от табличного варианта. Интерполяция режимов автоматически не выполняется.")
	}
	if _, e = normalizeConfiguration(c.ConfigurationInput); e != nil {
		return Task{}, e
	}
	if pipe.DiameterMm == nil || pipe.WallThicknessMm == nil || pipe.LengthMm == nil {
		return Task{}, invalid("В журнале трубы должны быть заполнены диаметр, толщина стенки и длина.")
	}
	if *pipe.DiameterMm != c.DiameterMm {
		return Task{}, invalid("Диаметр трубы не совпадает с диаметром выбранной конфигурации.")
	}
	if c.MinPipeLengthMm != nil && *pipe.LengthMm < *c.MinPipeLengthMm {
		return Task{}, invalid("Длина трубы меньше указанной в выбранном табличном режиме ТК.")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if e = saveTaskMachine(tx, id, input.MachineID); e != nil {
		return Task{}, e
	}
	_, e = tx.Exec("UPDATE manufacturing_tasks SET pipe_snapshot_json=?,configuration_snapshot_json=?,machine=?,snapshot_version=1,status='ready',revision=revision+1,prepared_at=?,prepared_by=?,updated_at=?,updated_by=? WHERE id=? AND revision=? AND status='draft'", jsonText(pipe), jsonText(c), c.Machine, now, actor, now, actor, id, revision)
	if e != nil {
		return Task{}, errStorage
	}
	task, e = getTask(tx, id)
	if e != nil {
		return task, e
	}
	if e = tx.Commit(); e != nil {
		return Task{}, errStorage
	}
	return task, nil
}
