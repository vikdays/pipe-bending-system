package registry

import (
	"changeme/internal/apperror"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// These transport types intentionally omit authentication, audit actors, local
// attachment paths and parser internals. Do not marshal Task or database rows.
type TaskPackage struct {
	Format        string         `json:"format"`
	SchemaVersion int            `json:"schemaVersion"`
	Tasks         []ExportedTask `json:"tasks"`
}
type ExportedTask struct {
	ID                    string                `json:"id"`
	Revision              int                   `json:"revision"`
	Number                string                `json:"number"`
	QueuePosition         int                   `json:"queuePosition"`
	BendNumber            string                `json:"bendNumber"`
	BendType              string                `json:"bendType"`
	TargetAngleDeg        float64               `json:"targetAngleDeg"`
	Notes                 string                `json:"notes"`
	CreatedAt             string                `json:"createdAt"`
	PreparedAt            string                `json:"preparedAt"`
	SnapshotVersion       int                   `json:"snapshotVersion"`
	PipeSnapshot          ExportedPipe          `json:"pipeSnapshot"`
	ConfigurationSnapshot ExportedConfiguration `json:"configurationSnapshot"`
	MachineSnapshot       Machine               `json:"machineSnapshot"`
}
type ExportedPipe struct {
	PipeData
	SourceID     string        `json:"sourceId"`
	Revision     int           `json:"revision"`
	Certificates []Certificate `json:"certificates"`
}
type ExportedConfiguration struct {
	SourceID               string                   `json:"sourceId"`
	Revision               int                      `json:"revision"`
	Name                   string                   `json:"name"`
	Version                string                   `json:"version"`
	TechnicalCardReference string                   `json:"technicalCardReference"`
	DiameterMm             float64                  `json:"diameterMm"`
	BendType               string                   `json:"bendType"`
	TargetAngle            string                   `json:"targetAngle"`
	MinPipeLengthMm        *float64                 `json:"minPipeLengthMm"`
	Parameters             []ConfigurationParameter `json:"parameters"`
	Notes                  string                   `json:"notes"`
}
type TaskExportSelection struct {
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}
type ExchangeTask struct {
	TaskID        string `json:"taskId"`
	TaskUUID      string `json:"taskUuid"`
	TaskRevision  int    `json:"taskRevision"`
	TaskNumber    string `json:"taskNumber"`
	QueuePosition int    `json:"queuePosition"`
}
type ExchangeEvent struct {
	ID           string         `json:"id"`
	Tasks        []ExchangeTask `json:"tasks"`
	Status       string         `json:"status"`
	FileName     string         `json:"fileName"`
	SHA256       string         `json:"sha256"`
	SizeBytes    int            `json:"sizeBytes"`
	CreatedAt    string         `json:"createdAt"`
	CompletedAt  string         `json:"completedAt"`
	ErrorMessage string         `json:"errorMessage"`
}

func exportTask(t Task, externalID string) (ExportedTask, error) {
	fail := func() (ExportedTask, error) {
		return ExportedTask{}, invalid("Задание не содержит полных подготовленных снимков. Для старого задания без снимков создайте и подготовьте новый черновик.")
	}
	if t.Status != "ready" || t.SnapshotVersion != 1 || t.Revision <= 0 || t.QueuePosition <= 0 || t.Number == "" || t.BendNumber == "" || (t.BendType != "1ГО" && t.BendType != "2ГО") {
		return fail()
	}
	if _, e := time.Parse(time.RFC3339Nano, t.PreparedAt); e != nil {
		return fail()
	}
	if _, e := time.Parse(time.RFC3339Nano, t.CreatedAt); e != nil {
		return fail()
	}
	p, c, m := t.PipeSnapshot, t.ConfigurationSnapshot, t.MachineSnapshot
	if c == nil || m == nil || p.ID != t.PipeID || p.Revision != t.PipeRevision || c.ID != t.ConfigurationID || c.Revision != t.ConfigurationRevision || m.ID != t.MachineID || m.ID == "" || m.Model == "" || m.Name == "" || c.MachineID != m.ID || c.NeedsReview {
		return fail()
	}
	if p.FactoryNumber == "" || p.InventoryNumber == "" || p.DiameterMm == nil || p.WallThicknessMm == nil || p.LengthMm == nil || *p.DiameterMm <= 0 || *p.WallThicknessMm <= 0 || *p.LengthMm <= 0 || *p.DiameterMm != c.DiameterMm {
		return fail()
	}
	if _, e := normalizeConfiguration(c.ConfigurationInput); e != nil {
		return fail()
	}
	angle, e := strconv.ParseFloat(t.TargetAngle, 64)
	if e != nil || !finite(angle) || angle <= 0 || (c.BendType != "" && c.BendType != t.BendType) || (c.TargetAngle != "" && c.TargetAngle != t.TargetAngle) || (c.MinPipeLengthMm != nil && *p.LengthMm < *c.MinPipeLengthMm) {
		return fail()
	}
	var attrs map[string]json.RawMessage
	if json.Unmarshal([]byte(p.AttributesJSON), &attrs) != nil || attrs == nil {
		return fail()
	}
	certificates := append([]Certificate{}, p.Certificates...)
	parameters := append([]ConfigurationParameter{}, c.Parameters...)
	return ExportedTask{
		ID: externalID, Revision: t.Revision, Number: t.Number, QueuePosition: t.QueuePosition, BendNumber: t.BendNumber, BendType: t.BendType, TargetAngleDeg: angle, Notes: t.Notes, CreatedAt: t.CreatedAt, PreparedAt: t.PreparedAt, SnapshotVersion: t.SnapshotVersion,
		PipeSnapshot:          ExportedPipe{PipeData: p.PipeData, SourceID: p.ID, Revision: p.Revision, Certificates: certificates},
		ConfigurationSnapshot: ExportedConfiguration{SourceID: c.ID, Revision: c.Revision, Name: c.Name, Version: c.Version, TechnicalCardReference: c.TechnicalCardReference, DiameterMm: c.DiameterMm, BendType: c.BendType, TargetAngle: c.TargetAngle, MinPipeLengthMm: c.MinPipeLengthMm, Parameters: parameters, Notes: c.Notes}, MachineSnapshot: *m,
	}, nil
}

// BuildTasksExport takes a consistent snapshot of the prepared queue awaiting export.
// Expected identities/revisions bind the file to the list reviewed in the UI.
func (r *Repository) BuildTasksExport(expected []TaskExportSelection) (TaskPackage, []byte, error) {
	p := TaskPackage{Format: "pipe-bending.tasks", SchemaVersion: 1, Tasks: []ExportedTask{}}
	if len(expected) == 0 || len(expected) > 1000 {
		return p, nil, invalid("Нужно от 1 до 1000 подготовленных заданий.")
	}
	requested := map[string]int{}
	for _, v := range expected {
		if v.ID == "" || v.Revision <= 0 {
			return p, nil, invalid("Обновите список заданий перед выгрузкой.")
		}
		if _, ok := requested[v.ID]; ok {
			return p, nil, invalid("В списке выгрузки повторяется задание.")
		}
		requested[v.ID] = v.Revision
	}
	tx, e := r.db.Begin()
	if e != nil {
		return p, nil, errStorage
	}
	defer tx.Rollback()
	rows, e := tx.Query("SELECT " + taskColumns + " FROM manufacturing_tasks WHERE status='ready' AND NOT " + taskWasExportedSQL + " ORDER BY queue_position,id")
	if e != nil {
		return p, nil, errStorage
	}
	tasks := []Task{}
	for rows.Next() {
		t, err := readTask(rows)
		if err != nil {
			rows.Close()
			return p, nil, errStorage
		}
		tasks = append(tasks, t)
		if len(tasks) > 1000 {
			rows.Close()
			return p, nil, invalid("В одном пакете может быть не более 1000 заданий.")
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, nil, errStorage
	}
	conflict := func() (TaskPackage, []byte, error) {
		return p, nil, apperror.New(apperror.Conflict, "Список подготовленных заданий изменился. Обновите состав пакета и повторите выгрузку.")
	}
	if len(tasks) != len(expected) {
		return conflict()
	}
	for _, t := range tasks {
		if requested[t.ID] != t.Revision {
			return conflict()
		}
		v, err := exportTask(t, "")
		if err != nil {
			return p, nil, apperror.New(apperror.ValidationError, "Задание "+t.Number+": "+apperror.Describe(err).Message)
		}
		v.ID, err = taskExchangeUUID(tx, t.ID)
		if err != nil {
			return p, nil, err
		}
		p.Tasks = append(p.Tasks, v)
	}
	data, e := json.MarshalIndent(p, "", "  ")
	if e != nil {
		return p, nil, errStorage
	}
	data = append(data, '\n')
	if e = tx.Commit(); e != nil {
		return p, nil, errStorage
	}
	return p, data, nil
}
func taskExchangeUUID(tx *sql.Tx, id string) (string, error) {
	var externalID string
	e := tx.QueryRow("SELECT task_uuid FROM task_exchange_ids WHERE task_id=?", id).Scan(&externalID)
	if errors.Is(e, sql.ErrNoRows) {
		externalID = uuid.NewString()
		_, e = tx.Exec("INSERT INTO task_exchange_ids(task_id,task_uuid) VALUES(?,?)", id, externalID)
	}
	if e != nil {
		return "", errStorage
	}
	if _, e = uuid.Parse(externalID); e != nil {
		return "", errStorage
	}
	return externalID, nil
}
func (r *Repository) BeginTasksExport(actor, filename string, p TaskPackage, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	eventID := uuid.NewString()
	tx, e := r.db.Begin()
	if e != nil {
		return "", errStorage
	}
	defer tx.Rollback()
	_, e = tx.Exec("INSERT INTO exchange_events(id,kind,status,file_name,sha256,size_bytes,created_at,actor_id) VALUES(?,'tasks_export','pending',?,?,?,?,?)", eventID, filename, hex.EncodeToString(sum[:]), len(data), time.Now().UTC().Format(time.RFC3339Nano), actor)
	if e != nil {
		return "", errStorage
	}
	for _, v := range p.Tasks {
		_, e = tx.Exec("INSERT INTO exchange_event_tasks(event_id,task_id,task_uuid,task_revision,task_number,queue_position) VALUES(?,(SELECT task_id FROM task_exchange_ids WHERE task_uuid=?),?,?,?,?)", eventID, v.ID, v.ID, v.Revision, v.Number, v.QueuePosition)
		if e != nil {
			return "", errStorage
		}
	}
	if e = tx.Commit(); e != nil {
		return "", errStorage
	}
	return eventID, nil
}
func (r *Repository) FinishTaskExport(id string, success bool) error {
	status, message := "saved", ""
	if !success {
		status = "failed"
		message = "Не удалось записать файл пакета заданий."
	}
	result, e := r.db.Exec("UPDATE exchange_events SET status=?,completed_at=?,error_message=? WHERE id=? AND status='pending'", status, time.Now().UTC().Format(time.RFC3339Nano), message, id)
	if e != nil {
		return errStorage
	}
	n, e := result.RowsAffected()
	if e != nil || n != 1 {
		return errStorage
	}
	return nil
}
func (r *Repository) ListExchangeEvents() ([]ExchangeEvent, error) {
	tx, e := r.db.Begin()
	if e != nil {
		return nil, errStorage
	}
	defer tx.Rollback()
	rows, e := tx.Query("SELECT id,status,file_name,sha256,size_bytes,created_at,completed_at,error_message FROM exchange_events ORDER BY created_at DESC,id DESC LIMIT 100")
	if e != nil {
		return nil, errStorage
	}
	out := []ExchangeEvent{}
	indices := map[string]int{}
	for rows.Next() {
		v := ExchangeEvent{Tasks: []ExchangeTask{}}
		if e = rows.Scan(&v.ID, &v.Status, &v.FileName, &v.SHA256, &v.SizeBytes, &v.CreatedAt, &v.CompletedAt, &v.ErrorMessage); e != nil {
			rows.Close()
			return nil, errStorage
		}
		indices[v.ID] = len(out)
		out = append(out, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, errStorage
	}
	members, e := tx.Query("SELECT event_id,task_id,task_uuid,task_revision,task_number,queue_position FROM exchange_event_tasks WHERE event_id IN (SELECT id FROM exchange_events ORDER BY created_at DESC,id DESC LIMIT 100) ORDER BY event_id,queue_position")
	if e != nil {
		return nil, errStorage
	}
	defer members.Close()
	for members.Next() {
		var id string
		var v ExchangeTask
		if e = members.Scan(&id, &v.TaskID, &v.TaskUUID, &v.TaskRevision, &v.TaskNumber, &v.QueuePosition); e != nil {
			return nil, errStorage
		}
		if index, ok := indices[id]; ok {
			out[index].Tasks = append(out[index].Tasks, v)
		}
	}
	if members.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}

// Filename is deterministic for a given ordered package, including its snapshots.
func (p TaskPackage) Filename() string {
	data, _ := json.MarshalIndent(p, "", "  ")
	data = append(data, '\n')
	sum := sha256.Sum256(data)
	return "tasks-" + hex.EncodeToString(sum[:6]) + ".json"
}
