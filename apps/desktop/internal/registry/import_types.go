package registry

type ImportConfiguration struct {
	Sheet     string            `json:"sheet"`
	HeaderRow int               `json:"headerRow"`
	Columns   map[string]int    `json:"columns"`
	Units     map[string]string `json:"units"`
}
type ImportField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Required bool     `json:"required"`
	Units    []string `json:"units"`
}
type ImportColumn struct {
	Index  int    `json:"index"`
	Letter string `json:"letter"`
	Label  string `json:"label"`
}
type ImportIssue struct {
	Severity string `json:"severity"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}
type ImportRow struct {
	ID                      string        `json:"id"`
	SourceRow               int           `json:"sourceRow"`
	Input                   PipeInput     `json:"input"`
	Issues                  []ImportIssue `json:"issues"`
	Status                  string        `json:"status"`
	ExistingPipeID          string        `json:"existingPipeId"`
	ExistingInventoryNumber string        `json:"existingInventoryNumber"`
}
type ImportPreview struct {
	ID            string              `json:"id"`
	OriginalName  string              `json:"originalName"`
	SHA256        string              `json:"sha256"`
	DocumentID    string              `json:"documentId"`
	Revision      int                 `json:"revision"`
	Status        string              `json:"status"`
	CreatedAt     string              `json:"createdAt"`
	Configuration ImportConfiguration `json:"configuration"`
	Sheets        []string            `json:"sheets"`
	Fields        []ImportField       `json:"fields"`
	Columns       []ImportColumn      `json:"columns"`
	Warnings      []string            `json:"warnings"`
	Rows          []ImportRow         `json:"rows"`
	Result        *ImportResult       `json:"result"`
}
type ImportResult struct {
	ImportID    string `json:"importId"`
	Added       int    `json:"added"`
	Existing    int    `json:"existing"`
	NotImported int    `json:"notImported"`
	CommittedAt string `json:"committedAt"`
}
type ImportSummary struct {
	ID           string        `json:"id"`
	OriginalName string        `json:"originalName"`
	Sheet        string        `json:"sheet"`
	Status       string        `json:"status"`
	CreatedAt    string        `json:"createdAt"`
	Result       *ImportResult `json:"result"`
}
