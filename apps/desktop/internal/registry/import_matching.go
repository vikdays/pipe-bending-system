package registry

import "strings"

func identity(p PipeData) string {
	return jsonText([]string{strings.ToLower(strings.TrimSpace(p.Manufacturer)), strings.ToLower(strings.TrimSpace(p.FactoryNumber)), strings.ToLower(strings.TrimSpace(p.HeatNumber))})
}

// Compare populated source properties only. Neither diameter nor certificate alone identifies a pipe.
func compatible(input PipeInput, p Pipe) bool {
	other := PipeInput{PipeData: p.PipeData}
	for key, value := range textFields(&input) {
		if key == "inventoryNumber" {
			continue
		}
		if *value != "" && !strings.EqualFold(strings.TrimSpace(*value), strings.TrimSpace(*textFields(&other)[key])) {
			return false
		}
	}
	for key, value := range numberFields(&input) {
		target := *numberFields(&other)[key]
		if *value != nil && (target == nil || **value != *target) {
			return false
		}
	}
	if input.NewCertificate != nil {
		found := false
		for _, c := range p.Certificates {
			if c.Reference == input.NewCertificate.Reference {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func classifyRows(rows []ImportRow, pipes []Pipe) {
	seen := map[string]int{}
	inventories := map[string]int{}
	for i := range rows {
		row := &rows[i]
		stored := row.ExistingPipeID
		row.ExistingPipeID = ""
		row.ExistingInventoryNumber = ""
		row.Status = "new"
		// Only parser/edit issues are persisted, so matching warnings cannot accumulate across refreshes.
		for _, issue := range row.Issues {
			if issue.Severity == "error" {
				row.Status = "invalid"
			}
		}
		if _, e := normalizePipe(row.Input); e != nil {
			row.Status = "invalid"
			row.Issues = append(row.Issues, ImportIssue{"error", "", e.Error()})
		}
		if row.Status == "invalid" {
			continue
		}
		inventoryPrior, inventoryDuplicate := inventories[row.Input.InventoryNumber]
		inventories[row.Input.InventoryNumber] = i
		source := sourceOf(row.Input)
		matches := []Pipe{}
		exact := []Pipe{}
		for _, pipe := range pipes {
			previous := sourceOf(PipeInput{PipeData: pipe.PipeData})
			if pipe.ID == stored || source.SHA256 != "" && source.SHA256 == previous.SHA256 && source.Sheet == previous.Sheet && source.Row == previous.Row {
				exact = append(exact, pipe)
			}
			if strings.EqualFold(strings.TrimSpace(row.Input.FactoryNumber), strings.TrimSpace(pipe.FactoryNumber)) {
				matches = append(matches, pipe)
			}
			if pipe.InventoryNumber == row.Input.InventoryNumber && pipe.FactoryNumber != row.Input.FactoryNumber {
				row.Status = "conflict"
			}
		}
		if len(exact) > 1 {
			row.Status = "conflict"
		} else if len(exact) == 1 {
			row.Status = "existing"
			row.ExistingPipeID = exact[0].ID
			row.ExistingInventoryNumber = exact[0].InventoryNumber
		} else if len(matches) > 0 {
			candidates := []Pipe{}
			for _, p := range matches {
				if row.Input.Manufacturer != "" && row.Input.HeatNumber != "" && identity(row.Input.PipeData) == identity(p.PipeData) && compatible(row.Input, p) {
					candidates = append(candidates, p)
				}
			}
			if len(candidates) == 1 {
				row.Status = "existing"
				row.ExistingPipeID = candidates[0].ID
				row.ExistingInventoryNumber = candidates[0].InventoryNumber
			} else {
				row.Status = "conflict"
			}
		}
		key := identity(row.Input.PipeData)
		if prior, ok := seen[key]; ok {
			a, b := rows[prior].Input, row.Input
			a.InventoryNumber = ""
			b.InventoryNumber = ""
			a.AttributesJSON = ""
			b.AttributesJSON = ""
			if jsonText(a) == jsonText(b) {
				if row.Status == "new" {
					row.Status = "duplicate"
				}
			} else {
				row.Status = "conflict"
				rows[prior].Status = "conflict"
			}
		} else {
			seen[key] = i
		}
		if inventoryDuplicate && rows[inventoryPrior].Input.FactoryNumber != row.Input.FactoryNumber {
			row.Status = "conflict"
			rows[inventoryPrior].Status = "conflict"
			rows[inventoryPrior].Issues = append(rows[inventoryPrior].Issues, ImportIssue{"warning", "inventoryNumber", "Инвентарный номер повторяется у разных труб в этом файле."})
		}
		if row.Status == "conflict" {
			row.Issues = append(row.Issues, ImportIssue{"warning", "", "Совпадающий номер или противоречащие сведения. Автоматическая перезапись запрещена; проверьте существующий журнал и исходник."})
		}
		if row.Input.DiameterMm == nil || row.Input.WallThicknessMm == nil || row.Input.LengthMm == nil {
			row.Issues = append(row.Issues, ImportIssue{"warning", "", "Для создания задания позднее понадобятся диаметр, стенка и длина."})
		}
	}
}
