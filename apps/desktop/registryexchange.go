package main

import (
	"path/filepath"

	"changeme/internal/apperror"
	"changeme/internal/exchange"
	"changeme/internal/registry"
)

type TaskExportResult struct {
	Outcome       string                         `json:"outcome"` // cancelled, saved, written_unlogged
	FileName      string                         `json:"fileName"`
	TaskCount     int                            `json:"taskCount"`
	Message       string                         `json:"message"`
	ExportedTasks []registry.TaskExportSelection `json:"exportedTasks"`
}

func (s *DesktopRegistryService) ExportTasks(access string, tasks []registry.TaskExportSelection) (TaskExportResult, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return TaskExportResult{}, e
	}
	if !s.exportMu.TryLock() {
		return TaskExportResult{}, apperror.New(apperror.Conflict, "Дождитесь завершения текущей выгрузки.")
	}
	defer s.exportMu.Unlock()
	p, data, e := s.repository.BuildTasksExport(tasks)
	if e != nil {
		return TaskExportResult{}, apperror.Public(e)
	}
	if s.saveFile == nil {
		return TaskExportResult{}, apperror.New(apperror.InternalError, "Диалог сохранения недоступен.")
	}
	path, e := s.saveFile(p.Filename())
	if e != nil {
		return TaskExportResult{}, apperror.New(apperror.InternalError, "Не удалось открыть диалог сохранения.")
	}
	if path == "" {
		return TaskExportResult{Outcome: "cancelled"}, nil
	}
	// A native dialog may stay open longer than the user's session.
	user, e := s.requireAdmin(access)
	if e != nil {
		return TaskExportResult{}, e
	}
	eventID, e := s.repository.BeginTasksExport(user.ID, filepath.Base(path), p, data)
	if e != nil {
		return TaskExportResult{}, apperror.New(apperror.InternalError, "Файл не записан: не удалось создать запись журнала обмена.")
	}
	write := s.writeExport
	if write == nil {
		write = exchange.WriteJSON
	}
	if e = write(path, data); e != nil {
		logError := s.repository.FinishTaskExport(eventID, false)
		message := "Не удалось записать файл пакета заданий. Проверьте носитель, свободное место и права доступа, затем повторите выгрузку."
		if logError != nil {
			message += " Запись журнала осталась неподтверждённой."
		}
		return TaskExportResult{}, apperror.New(apperror.InternalError, message)
	}
	result := TaskExportResult{Outcome: "saved", FileName: filepath.Base(path), TaskCount: len(p.Tasks)}
	if e = s.repository.FinishTaskExport(eventID, true); e != nil {
		result.Outcome = "written_unlogged"
		result.Message = "Пакет записан, но подтвердить запись в журнале обмена не удалось. Получение файла планшетом не подтверждено."
	} else {
		result.ExportedTasks = append([]registry.TaskExportSelection{}, tasks...)
	}
	return result, nil
}
func (s *DesktopRegistryService) ListExchangeEvents(access string) ([]registry.ExchangeEvent, error) {
	if _, e := s.requireAdmin(access); e != nil {
		return nil, e
	}
	return publicResult(s.repository.ListExchangeEvents())
}
