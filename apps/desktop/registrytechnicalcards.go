package main

import (
	"changeme/internal/apperror"
	"changeme/internal/registry"
)

func (s *DesktopRegistryService) ListMachines(access string) ([]registry.Machine, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.repository.ListMachines())
}
func (s *DesktopRegistryService) PrepareTechnicalCardImport(access, machineID string) (*registry.TechnicalCardPreview, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	path, e := s.chooseFile("Выберите технологическую карту DOCX", "*.docx")
	if e != nil {
		return nil, apperror.New(apperror.InternalError, "Не удалось открыть диалог файла.")
	}
	if path == "" {
		return nil, nil
	}
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.technicalCards.Prepare(path, machineID, user.ID))
}
func (s *DesktopRegistryService) GetTechnicalCard(access, id string) (*registry.TechnicalCardPreview, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.technicalCards.Get(id))
}
func (s *DesktopRegistryService) ListTechnicalCards(access string) ([]registry.TechnicalCardPreview, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.technicalCards.List())
}
func (s *DesktopRegistryService) CommitTechnicalCardImport(access, id string, keys []string) (int, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return 0, e
	}
	return publicResult(s.technicalCards.Commit(id, keys, user.ID))
}
func (s *DesktopRegistryService) OpenTechnicalCard(access, id string) error {
	if _, e := s.requireAdmin(access); e != nil {
		return e
	}
	path, e := s.technicalCards.OriginalPath(id)
	if e != nil {
		return apperror.Public(e)
	}
	return s.openManagedFile(path)
}
