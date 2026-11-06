package registry

import (
	"changeme/internal/apperror"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// File-export state is derived from the durable journal for this exact revision.
// Keep prepared rows immutable: exporting must not rewrite the frozen content.
const taskWasExportedSQL = "EXISTS (SELECT 1 FROM exchange_event_tasks em JOIN exchange_events ee ON ee.id=em.event_id WHERE em.task_id=manufacturing_tasks.id AND em.task_revision=manufacturing_tasks.revision AND ee.status='saved')"
const taskExportedAtSQL = "COALESCE((SELECT ee.completed_at FROM exchange_event_tasks em JOIN exchange_events ee ON ee.id=em.event_id WHERE em.task_id=manufacturing_tasks.id AND em.task_revision=manufacturing_tasks.revision AND ee.status='saved' ORDER BY ee.created_at DESC,ee.id DESC LIMIT 1),'')"
const taskColumns = "id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,CASE WHEN status='ready' AND " + taskWasExportedSQL + " THEN 'exported' ELSE status END,created_at,updated_at,revision,queue_position,bend_type,notes,COALESCE(configuration_id,''),configuration_revision,configuration_snapshot_json,snapshot_version,COALESCE(prepared_at,''),COALESCE(machine_id,''),machine_snapshot_json," + taskExportedAtSQL

func readTask(row scanner) (Task, error) {
	var task Task
	var pipe string
	var configuration sql.NullString
	var machine sql.NullString
	e := row.Scan(&task.ID, &task.Number, &task.PipeID, &task.PipeRevision, &pipe, &task.BendNumber, &task.Machine, &task.TargetAngle, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.Revision, &task.QueuePosition, &task.BendType, &task.Notes, &task.ConfigurationID, &task.ConfigurationRevision, &configuration, &task.SnapshotVersion, &task.PreparedAt, &task.MachineID, &machine, &task.ExportedAt)
	if e != nil {
		return task, e
	}
	if json.Unmarshal([]byte(pipe), &task.PipeSnapshot) != nil {
		return task, errStorage
	}
	if configuration.Valid && json.Unmarshal([]byte(configuration.String), &task.ConfigurationSnapshot) != nil {
		return task, errStorage
	}
	if machine.Valid && json.Unmarshal([]byte(machine.String), &task.MachineSnapshot) != nil {
		return task, errStorage
	}
	return task, nil
}
func getTask(tx *sql.Tx, id string) (Task, error) {
	task, e := readTask(tx.QueryRow("SELECT "+taskColumns+" FROM manufacturing_tasks WHERE id=?", id))
	if errors.Is(e, sql.ErrNoRows) {
		return task, apperror.New(apperror.NotFound, "Задание не найдено.")
	}
	if e != nil {
		return task, errStorage
	}
	return task, nil
}
func (r *Repository) GetTask(id string) (Task, error) {
	tx, e := r.db.Begin()
	if e != nil {
		return Task{}, errStorage
	}
	defer tx.Rollback()
	return getTask(tx, id)
}
func (r *Repository) ListTasks() ([]Task, error) {
	rows, e := r.db.Query("SELECT " + taskColumns + " FROM manufacturing_tasks ORDER BY queue_position,id")
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		task, e := readTask(rows)
		if e != nil {
			return nil, errStorage
		}
		out = append(out, task)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}
func taskConflict() error {
	return apperror.New(apperror.Conflict, "Задание или его исходные данные изменены. Обновите список; откройте и сохраните черновик, проверив актуальные трубу и конфигурацию.")
}
func taskSources(tx *sql.Tx, input *TaskInput) (Pipe, *Configuration, error) {
	pipe, e := getPipe(tx, input.PipeID)
	if e != nil {
		return pipe, nil, e
	}
	if pipe.Revision != input.PipeRevision {
		return pipe, nil, taskConflict()
	}
	var c *Configuration
	if input.ConfigurationID != "" {
		value, e := getConfiguration(tx, input.ConfigurationID)
		if e != nil {
			return pipe, nil, e
		}
		if value.Revision != input.ConfigurationRevision {
			return pipe, nil, taskConflict()
		}
		c = &value
		input.Machine = value.Machine
		if input.MachineID == "" {
			input.MachineID = value.MachineID
		}
		if value.MachineID != "" && input.MachineID != value.MachineID {
			return pipe, nil, invalid("Выбранная конфигурация относится к другому станку.")
		}
	}
	if input.MachineID != "" {
		m, e := getMachine(tx, input.MachineID)
		if e != nil {
			return pipe, nil, e
		}
		input.Machine = m.Name
	}
	return pipe, c, nil
}
func (r *Repository) CreateTask(raw TaskInput, actor string) (Task, error) {
	tasks, e := r.CreateTasks(TaskBatchInput{RequestID: "single_" + rand.Text(), Tasks: []TaskInput{raw}}, actor)
	if e != nil {
		return Task{}, e
	}
	return tasks[0], nil
}
func createTaskTx(tx *sql.Tx, input TaskInput, actor string) (Task, error) {
	pipe, c, e := taskSources(tx, &input)
	if e != nil {
		return Task{}, e
	}
	var position int
	if e = tx.QueryRow("SELECT COALESCE(MAX(queue_position),0)+1 FROM manufacturing_tasks").Scan(&position); e != nil {
		return Task{}, errStorage
	}
	if position <= 0 || position > 9007199254740991 {
		return Task{}, invalid("Очередь заданий исчерпана.")
	}
	if input.Number == "" {
		for n := position; ; n++ {
			candidate := fmt.Sprintf("ЗД-%04d", n)
			var count int
			if e = tx.QueryRow("SELECT COUNT(*) FROM manufacturing_tasks WHERE number=?", candidate).Scan(&count); e != nil {
				return Task{}, errStorage
			}
			if count == 0 {
				input.Number = candidate
				break
			}
		}
	}
	if e = checkTaskNumber(tx, input.Number, ""); e != nil {
		return Task{}, e
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := "t_" + rand.Text()
	var cfgID, configuration any
	if c != nil {
		cfgID = c.ID
		configuration = jsonText(c)
	}
	_, e = tx.Exec("INSERT INTO manufacturing_tasks(id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,created_at,created_by,updated_at,updated_by,queue_position,bend_type,notes,configuration_id,configuration_revision,configuration_snapshot_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, input.Number, input.PipeID, input.PipeRevision, jsonText(pipe), input.BendNumber, input.Machine, input.TargetAngle, now, actor, now, actor, position, input.BendType, input.Notes, cfgID, input.ConfigurationRevision, configuration)
	if e != nil {
		return Task{}, errStorage
	}
	if e = saveTaskMachine(tx, id, input.MachineID); e != nil {
		return Task{}, e
	}
	return getTask(tx, id)
}
func checkTaskNumber(tx *sql.Tx, number, id string) error {
	var n int
	if e := tx.QueryRow("SELECT COUNT(*) FROM manufacturing_tasks WHERE number=? AND id<>?", number, id).Scan(&n); e != nil {
		return errStorage
	}
	if n > 0 {
		return apperror.New(apperror.Conflict, "Номер задания уже занят: "+number+".")
	}
	return nil
}
func (r *Repository) UpdateTask(id string, revision int, raw TaskInput, actor string) (Task, error) {
	input, e := normalizeTask(raw)
	if e != nil {
		return Task{}, e
	}
	if input.Number == "" {
		return Task{}, invalid("У существующего задания должен быть номер.")
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
	if task.Status != "draft" {
		return Task{}, apperror.New(apperror.Conflict, "Подготовленное задание нельзя изменять.")
	}
	if revision <= 0 || task.Revision != revision {
		return Task{}, taskConflict()
	}
	pipe, c, e := taskSources(tx, &input)
	if e != nil {
		return Task{}, e
	}
	if e = checkTaskNumber(tx, input.Number, id); e != nil {
		return Task{}, e
	}
	var cfgID, configuration any
	if c != nil {
		cfgID = c.ID
		configuration = jsonText(c)
	}
	_, e = tx.Exec("UPDATE manufacturing_tasks SET number=?,pipe_id=?,pipe_revision=?,pipe_snapshot_json=?,bend_number=?,machine=?,target_angle=?,bend_type=?,notes=?,configuration_id=?,configuration_revision=?,configuration_snapshot_json=?,revision=revision+1,updated_at=?,updated_by=? WHERE id=? AND revision=? AND status='draft'", input.Number, input.PipeID, input.PipeRevision, jsonText(pipe), input.BendNumber, input.Machine, input.TargetAngle, input.BendType, input.Notes, cfgID, input.ConfigurationRevision, configuration, time.Now().UTC().Format(time.RFC3339Nano), actor, id, revision)
	if e != nil {
		return Task{}, errStorage
	}
	if e = saveTaskMachine(tx, id, input.MachineID); e != nil {
		return Task{}, e
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
func saveTaskMachine(tx *sql.Tx, id, machineID string) error {
	var mid, snapshot any
	if machineID != "" {
		m, e := getMachine(tx, machineID)
		if e != nil {
			return e
		}
		mid = m.ID
		snapshot = jsonText(m)
	}
	if _, e := tx.Exec("UPDATE manufacturing_tasks SET machine_id=?,machine_snapshot_json=? WHERE id=?", mid, snapshot, id); e != nil {
		return errStorage
	}
	return nil
}
