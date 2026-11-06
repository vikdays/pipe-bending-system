package main

import "changeme/internal/registry"

func (s *DesktopRegistryService) GetTask(access, id string) (registry.Task, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return registry.Task{}, e
	}
	return publicResult(s.repository.GetTask(id))
}
func (s *DesktopRegistryService) CreateTasks(access string, input registry.TaskBatchInput) ([]registry.Task, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return nil, e
	}
	return publicResult(s.repository.CreateTasks(input, user.ID))
}
func (s *DesktopRegistryService) UpdateTask(access, id string, revision int, input registry.TaskInput) (registry.Task, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return registry.Task{}, e
	}
	return publicResult(s.repository.UpdateTask(id, revision, input, user.ID))
}
func (s *DesktopRegistryService) PrepareTask(access, id string, revision int) (registry.Task, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return registry.Task{}, e
	}
	return publicResult(s.repository.PrepareTask(id, revision, user.ID))
}
func (s *DesktopRegistryService) ListConfigurations(access string) ([]registry.Configuration, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.repository.ListConfigurations())
}
func (s *DesktopRegistryService) CreateConfiguration(access string, input registry.ConfigurationInput) (registry.Configuration, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return registry.Configuration{}, e
	}
	return publicResult(s.repository.SaveConfiguration("", 0, input, user.ID))
}
func (s *DesktopRegistryService) UpdateConfiguration(access, id string, revision int, input registry.ConfigurationInput) (registry.Configuration, error) {
	user, e := s.requireAdmin(access)
	if e != nil {
		return registry.Configuration{}, e
	}
	return publicResult(s.repository.SaveConfiguration(id, revision, input, user.ID))
}
