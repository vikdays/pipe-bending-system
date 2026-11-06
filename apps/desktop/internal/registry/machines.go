package registry

import (
	"changeme/internal/apperror"
	"database/sql"
	"errors"
)

func (r *Repository) ListMachines() ([]Machine, error) {
	rows, e := r.db.Query("SELECT id,name,model,serial_number,inventory_number FROM machines ORDER BY model,id")
	if e != nil {
		return nil, errStorage
	}
	defer rows.Close()
	out := []Machine{}
	for rows.Next() {
		var m Machine
		if e = rows.Scan(&m.ID, &m.Name, &m.Model, &m.SerialNumber, &m.InventoryNumber); e != nil {
			return nil, errStorage
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return out, nil
}
func getMachine(tx *sql.Tx, id string) (Machine, error) {
	var m Machine
	e := tx.QueryRow("SELECT id,name,model,serial_number,inventory_number FROM machines WHERE id=?", id).Scan(&m.ID, &m.Name, &m.Model, &m.SerialNumber, &m.InventoryNumber)
	if errors.Is(e, sql.ErrNoRows) {
		return m, apperror.New(apperror.NotFound, "Станок не найден.")
	}
	if e != nil {
		return m, errStorage
	}
	return m, nil
}
