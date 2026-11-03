package registry

import (
	"crypto/rand"
	"encoding/json"
	"time"

	"changeme/internal/apperror"
)

func (r *Repository) ListTasks() ([]Task, error) {
	rows, err := r.db.Query("SELECT id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,status,created_at FROM manufacturing_tasks ORDER BY created_at DESC,id DESC")
	if err != nil {
		return nil, errStorage
	}
	defer rows.Close()
	tasks := []Task{}
	for rows.Next() {
		var task Task
		var snapshot string
		if err = rows.Scan(&task.ID, &task.Number, &task.PipeID, &task.PipeRevision, &snapshot, &task.BendNumber, &task.Machine, &task.TargetAngle, &task.Status, &task.CreatedAt); err != nil {
			return nil, errStorage
		}
		if err = json.Unmarshal([]byte(snapshot), &task.PipeSnapshot); err != nil {
			return nil, errStorage
		}
		tasks = append(tasks, task)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return tasks, nil
}

func (r *Repository) CreateTask(raw TaskInput, actorID string) (Task, error) {
	input, err := normalizeTask(raw)
	if err != nil {
		return Task{}, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return Task{}, errStorage
	}
	defer tx.Rollback()
	pipe, err := getPipe(tx, input.PipeID)
	if err != nil {
		return Task{}, err
	}
	if pipe.Revision != input.PipeRevision {
		return Task{}, apperror.New(apperror.Conflict, "Характеристики трубы изменились. Обновите список и выберите трубу снова.")
	}
	if pipe.DiameterMm == nil || pipe.WallThicknessMm == nil || pipe.LengthMm == nil {
		return Task{}, invalid("Перед созданием задания заполните диаметр, толщину стенки и длину выбранной трубы в журнале.")
	}
	var count int
	if err = tx.QueryRow("SELECT COUNT(*) FROM manufacturing_tasks WHERE number=?", input.Number).Scan(&count); err != nil {
		return Task{}, errStorage
	}
	if count != 0 {
		return Task{}, apperror.New(apperror.Conflict, "Задание с таким номером уже существует.")
	}
	snapshot, err := json.Marshal(pipe)
	if err != nil {
		return Task{}, errStorage
	}
	task := Task{TaskInput: input, ID: "t_" + rand.Text(), PipeSnapshot: pipe, Status: "draft", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if _, err = tx.Exec("INSERT INTO manufacturing_tasks(id,number,pipe_id,pipe_revision,pipe_snapshot_json,bend_number,machine,target_angle,created_at,created_by) VALUES(?,?,?,?,?,?,?,?,?,?)", task.ID, task.Number, task.PipeID, task.PipeRevision, string(snapshot), task.BendNumber, task.Machine, task.TargetAngle, task.CreatedAt, actorID); err != nil {
		return Task{}, errStorage
	}
	if err = tx.Commit(); err != nil {
		return Task{}, errStorage
	}
	return task, nil
}
