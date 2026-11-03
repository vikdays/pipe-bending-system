package registry

import (
	"crypto/rand"
	"time"
)

func (r *Repository) CreateCertificate(input CertificateDraft, actor string) (Certificate, error) {
	// Reuse precisely the same metadata validation as manual pipe entry.
	normalized, e := normalizePipe(PipeInput{PipeData: PipeData{InventoryNumber: "validation", FactoryNumber: "validation"}, NewCertificate: &input})
	if e != nil {
		return Certificate{}, e
	}
	input = *normalized.NewCertificate
	id := "c_" + rand.Text()
	_, e = r.db.Exec("INSERT INTO certificates(id,reference,number,issued_on,page_count,issuer,created_at,created_by) VALUES(?,?,?,?,?,?,?,?)", id, input.Reference, input.Number, input.IssuedOn, input.PageCount, input.Issuer, time.Now().UTC().Format(time.RFC3339Nano), actor)
	if e != nil {
		return Certificate{}, errStorage
	}
	return Certificate{CertificateDraft: input, ID: id}, nil
}
