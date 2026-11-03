package main

import (
	"changeme/internal/apperror"
	"changeme/internal/documents"
	"changeme/internal/registry"
)

func (s *DesktopRegistryService) PrepareExcelImport(access string) (*registry.ImportPreview, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	path, e := s.chooseFile("Выберите трубный журнал Excel", "*.xlsx")
	if e != nil {
		return nil, apperror.New(apperror.InternalError, "Не удалось открыть диалог выбора файла.")
	}
	if path == "" {
		return nil, nil
	}
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.Prepare(path, user.ID))
}
func (s *DesktopRegistryService) GetExcelImport(access, id string) (*registry.ImportPreview, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.Get(id, user.ID))
}
func (s *DesktopRegistryService) ConfigureExcelImport(access, id string, revision int, c registry.ImportConfiguration) (*registry.ImportPreview, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.Configure(id, user.ID, revision, c))
}
func (s *DesktopRegistryService) EditExcelImportRow(access, id string, revision int, rowID string, input registry.PipeInput) (*registry.ImportPreview, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.EditRow(id, user.ID, revision, rowID, input))
}
func (s *DesktopRegistryService) CommitExcelImport(access, id string, revision int, selection []string) (*registry.ImportResult, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.Commit(id, user.ID, revision, selection))
}
func (s *DesktopRegistryService) ListExcelImports(access string) ([]registry.ImportSummary, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.imports.List(user.ID))
}
func (s *DesktopRegistryService) OpenExcelImportOriginal(access, id string) error {
	user, e := s.requireAdmin(access)
	if e != nil {
		return e
	}
	path, e := s.imports.OriginalPath(id, user.ID)
	if e != nil {
		return apperror.Public(e)
	}
	return s.openManagedFile(path)
}
func (s *DesktopRegistryService) ListCertificateAttachments(access, id string) ([]documents.Attachment, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.documents.ListAttachments(id))
}
func (s *DesktopRegistryService) AttachCertificatePDF(access, id string) (*documents.Attachment, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	if e := s.documents.CertificateExists(id); e != nil {
		return nil, apperror.Public(e)
	}
	path, e := s.chooseFile("Выберите PDF сертификата", "*.pdf")
	if e != nil {
		return nil, apperror.New(apperror.InternalError, "Не удалось открыть диалог выбора файла.")
	}
	if path == "" {
		return nil, nil
	}
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	attachment, e := s.documents.AttachPDF(id, path, user.ID)
	if e != nil {
		return nil, apperror.Public(e)
	}
	return &attachment, nil
}
func (s *DesktopRegistryService) OpenCertificatePDF(access, certificateID, documentID string) error {
	if _, e := s.requireAdmin(access); e != nil {
		return e
	}
	path, e := s.documents.CertificatePDFPath(certificateID, documentID)
	if e != nil {
		return apperror.Public(e)
	}
	return s.openManagedFile(path)
}
func (s *DesktopRegistryService) DetachCertificatePDF(access, certificateID, documentID string) error {
	user, e := s.requireAdmin(access)
	if e != nil {
		return e
	}
	return apperror.Public(s.documents.Detach(certificateID, documentID, user.ID))
}
func (s *DesktopRegistryService) CreateCertificate(access string, input registry.CertificateDraft) (registry.Certificate, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return registry.Certificate{}, e
	}
	return publicResult(s.repository.CreateCertificate(input, user.ID))
}
func (s *DesktopRegistryService) openManagedFile(path string) error {
	if s.openFile(path) != nil {
		return apperror.New(apperror.InternalError, "Не удалось открыть документ. Проверьте, что установлено приложение для этого формата.")
	}
	return nil
}
