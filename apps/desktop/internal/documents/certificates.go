package documents

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"path/filepath"
	"time"

	"changeme/internal/apperror"
)

type Attachment struct {
	File
	DisplayName string `json:"displayName"`
}

func (s *Store) CertificateExists(id string) error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM certificates WHERE id=?", id).Scan(&count); err != nil {
		return errStorage
	}
	if count != 1 {
		return apperror.New(apperror.NotFound, "Сертификат не найден.")
	}
	return nil
}
func (s *Store) AttachPDF(certificateID, source, actorID string) (Attachment, error) {
	if err := s.CertificateExists(certificateID); err != nil {
		return Attachment{}, err
	}
	file, err := s.Ingest(source, "pdf", actorID, nil)
	if err != nil {
		return Attachment{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Attachment{}, errStorage
	}
	defer tx.Rollback()
	var active bool
	err = tx.QueryRow("SELECT active FROM certificate_documents WHERE certificate_id=? AND document_id=?", certificateID, file.ID).Scan(&active)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, errStorage
	}
	name := safeName(filepath.Base(source))
	if !active {
		if _, err = tx.Exec("INSERT INTO certificate_documents(certificate_id,document_id,display_name,active) VALUES(?,?,?,1) ON CONFLICT(certificate_id,document_id) DO UPDATE SET active=1,display_name=excluded.display_name", certificateID, file.ID, name); err != nil {
			return Attachment{}, errStorage
		}
		if _, err = tx.Exec("INSERT INTO document_link_events(id,certificate_id,document_id,action,actor_id,created_at) VALUES(?,?,?,'attach',?,?)", "de_"+rand.Text(), certificateID, file.ID, actorID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return Attachment{}, errStorage
		}
	}
	if err = tx.Commit(); err != nil {
		return Attachment{}, errStorage
	}
	return Attachment{File: file, DisplayName: name}, nil
}
func (s *Store) ListAttachments(certificateID string) ([]Attachment, error) {
	if err := s.CertificateExists(certificateID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT f.id,f.sha256,f.kind,f.original_name,f.size_bytes,f.page_count,f.created_at,c.display_name FROM document_files f JOIN certificate_documents c ON c.document_id=f.id WHERE c.certificate_id=? AND c.active=1 ORDER BY f.created_at,f.id", certificateID)
	if err != nil {
		return nil, errStorage
	}
	defer rows.Close()
	files := []Attachment{}
	for rows.Next() {
		var a Attachment
		if err = rows.Scan(&a.ID, &a.SHA256, &a.Kind, &a.OriginalName, &a.SizeBytes, &a.PageCount, &a.CreatedAt, &a.DisplayName); err != nil {
			return nil, errStorage
		}
		files = append(files, a)
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	return files, nil
}
func (s *Store) CertificatePDFPath(certificateID, documentID string) (string, error) {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM certificate_documents c JOIN document_files f ON f.id=c.document_id WHERE c.certificate_id=? AND c.document_id=? AND c.active=1 AND f.kind='pdf'", certificateID, documentID).Scan(&count); err != nil {
		return "", errStorage
	}
	if count != 1 {
		return "", apperror.New(apperror.NotFound, "PDF не связан с этим сертификатом.")
	}
	return s.ManagedPath(documentID)
}
func (s *Store) Detach(certificateID, documentID, actorID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return errStorage
	}
	defer tx.Rollback()
	result, err := tx.Exec("UPDATE certificate_documents SET active=0 WHERE certificate_id=? AND document_id=? AND active=1", certificateID, documentID)
	if err != nil {
		return errStorage
	}
	count, err := result.RowsAffected()
	if err != nil {
		return errStorage
	}
	if count != 1 {
		return apperror.New(apperror.NotFound, "Активная привязка документа не найдена.")
	}
	if _, err = tx.Exec("INSERT INTO document_link_events(id,certificate_id,document_id,action,actor_id,created_at) VALUES(?,?,?,'detach',?,?)", "de_"+rand.Text(), certificateID, documentID, actorID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return errStorage
	}
	if err = tx.Commit(); err != nil {
		return errStorage
	}
	return nil
}
