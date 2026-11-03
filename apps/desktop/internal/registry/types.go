package registry

// PipeData contains the source-pipe register fields, not the bend's control results.
// Missing numeric values are null; zero and missing must not be conflated.
type PipeData struct {
	InventoryNumber             string   `json:"inventoryNumber"`
	FactoryNumber               string   `json:"factoryNumber"`
	Manufacturer                string   `json:"manufacturer"`
	DiameterMm                  *float64 `json:"diameterMm"`
	WallThicknessMm             *float64 `json:"wallThicknessMm"`
	LengthMm                    *float64 `json:"lengthMm"`
	MassKg                      *float64 `json:"massKg"`
	StrengthClass               string   `json:"strengthClass"`
	SteelGrade                  string   `json:"steelGrade"`
	HeatNumber                  string   `json:"heatNumber"`
	CarbonEquivalentPct         *float64 `json:"carbonEquivalentPct"`
	PipeSpecification           string   `json:"pipeSpecification"`
	CoatingType                 string   `json:"coatingType"`
	CoatingSpecification        string   `json:"coatingSpecification"`
	TestPressureMpa             *float64 `json:"testPressureMpa"`
	MinOperatingTempC           *float64 `json:"minOperatingTempC"`
	IncomingInspectionReference string   `json:"incomingInspectionReference"`
	AllowedCategory54           string   `json:"allowedCategory54"`
	AllowedCategory74           string   `json:"allowedCategory74"`
	Notes                       string   `json:"notes"`
	AttributesJSON              string   `json:"attributesJson"`
}

type CertificateDraft struct {
	Reference string `json:"reference"`
	Number    string `json:"number"`
	IssuedOn  string `json:"issuedOn"`
	PageCount *int   `json:"pageCount"`
	Issuer    string `json:"issuer"`
}

type Certificate struct {
	CertificateDraft
	ID string `json:"id"`
}

type PipeInput struct {
	PipeData
	CertificateIDs []string          `json:"certificateIds"`
	NewCertificate *CertificateDraft `json:"newCertificate"`
}

type Pipe struct {
	PipeData
	ID           string        `json:"id"`
	Revision     int           `json:"revision"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    string        `json:"createdAt"`
	UpdatedAt    string        `json:"updatedAt"`
}

type TaskInput struct {
	Number       string `json:"number"`
	PipeID       string `json:"pipeId"`
	PipeRevision int    `json:"pipeRevision"`
	BendNumber   string `json:"bendNumber"`
	Machine      string `json:"machine"`
	TargetAngle  string `json:"targetAngle"`
}

type Task struct {
	TaskInput
	ID           string `json:"id"`
	PipeSnapshot Pipe   `json:"pipeSnapshot"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}
