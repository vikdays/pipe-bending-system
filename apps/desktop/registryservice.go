package main

import (
	"sync"

	"changeme/internal/apperror"
	"changeme/internal/auth"
	"changeme/internal/documents"
	"changeme/internal/registry"
)

// DesktopRegistryService checks the current native session for every operation.
type DesktopRegistryService struct {
	auth           *auth.Manager
	repository     *registry.Repository
	documents      *documents.Store
	imports        *registry.ImportManager
	technicalCards *registry.TechnicalCardManager
	chooseFile     func(title, pattern string) (string, error)
	openFile       func(path string) error
	saveFile       func(filename string) (string, error)
	writeExport    func(path string, data []byte) error
	exportMu       sync.Mutex
}

func (s *DesktopRegistryService) requireAdmin(access string) (auth.User, error) {
	user, err := s.auth.CurrentUser(access)
	if err != nil {
		return auth.User{}, apperror.Public(err)
	}
	if user.Role != "admin" {
		return auth.User{}, auth.ErrForbidden
	}
	return user, nil
}

func (s *DesktopRegistryService) ListPipes(access string) ([]registry.Pipe, error) {
	if _, err := s.requireAdmin(access); err != nil {
		return nil, err
	}
	return publicResult(s.repository.ListPipes())
}
func (s *DesktopRegistryService) ListCertificates(access string) ([]registry.Certificate, error) {
	if _, err := s.requireAdmin(access); err != nil {
		return nil, err
	}
	return publicResult(s.repository.ListCertificates())
}
func (s *DesktopRegistryService) CreatePipe(access string, input registry.PipeInput) (registry.Pipe, error) {
	user, err := s.requireAdmin(access)
	if err != nil {
		return registry.Pipe{}, err
	}
	return publicResult(s.repository.CreatePipe(input, user.ID))
}
func (s *DesktopRegistryService) UpdatePipe(access, id string, revision int, input registry.PipeInput) (registry.Pipe, error) {
	user, err := s.requireAdmin(access)
	if err != nil {
		return registry.Pipe{}, err
	}
	return publicResult(s.repository.UpdatePipe(id, revision, input, user.ID))
}
func (s *DesktopRegistryService) ListTasks(access string) ([]registry.Task, error) {
	if _, err := s.requireAdmin(access); err != nil {
		return nil, err
	}
	return publicResult(s.repository.ListTasks())
}
func (s *DesktopRegistryService) CreateTask(access string, input registry.TaskInput) (registry.Task, error) {
	user, err := s.requireAdmin(access)
	if err != nil {
		return registry.Task{}, err
	}
	return publicResult(s.repository.CreateTask(input, user.ID))
}
