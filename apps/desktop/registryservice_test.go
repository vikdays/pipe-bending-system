package main

import (
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/auth"
	"changeme/internal/documents"
	"changeme/internal/registry"
	"changeme/internal/storage"
)

func TestRegistryServiceRequiresCurrentAdminSession(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manager, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	const password = "Fictional registry test phrase 2026!"
	if _, err = manager.Bootstrap("supervisor", password); err != nil {
		t.Fatal(err)
	}
	admin, err := manager.Login("supervisor", password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manager.CreateUser(admin.Session.AccessToken, "operator.one", password, "operator"); err != nil {
		t.Fatal(err)
	}
	operator, err := manager.Login("operator.one", password)
	if err != nil {
		t.Fatal(err)
	}
	service := &DesktopRegistryService{auth: manager, repository: registry.New(db)}
	store, err := documents.New(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	service.documents = store
	service.imports = registry.NewImportManager(db, store)
	service.technicalCards = registry.NewTechnicalCardManager(db, store)
	chooserCalls := 0
	service.saveFile = func(string) (string, error) { chooserCalls++; return "", nil }
	service.chooseFile = func(string, string) (string, error) { chooserCalls++; return "", nil }
	for _, test := range []struct {
		access string
		code   apperror.Code
	}{{"", apperror.Unauthenticated}, {operator.Session.AccessToken, apperror.Forbidden}} {
		_, err = service.ListPipes(test.access)
		requireError(t, err, test.code)
		_, err = service.ListCertificates(test.access)
		requireError(t, err, test.code)
		_, err = service.ExportTasks(test.access, []registry.TaskExportSelection{{ID: "missing", Revision: 1}})
		requireError(t, err, test.code)
		_, err = service.ListExchangeEvents(test.access)
		requireError(t, err, test.code)
		_, err = service.ListTasks(test.access)
		requireError(t, err, test.code)
		_, err = service.CreatePipe(test.access, registry.PipeInput{})
		requireError(t, err, test.code)
		_, err = service.UpdatePipe(test.access, "missing", 1, registry.PipeInput{})
		requireError(t, err, test.code)
		_, err = service.CreateTask(test.access, registry.TaskInput{})
		requireError(t, err, test.code)
		_, err = service.GetTask(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.CreateTasks(test.access, registry.TaskBatchInput{})
		requireError(t, err, test.code)
		_, err = service.UpdateTask(test.access, "missing", 1, registry.TaskInput{})
		requireError(t, err, test.code)
		_, err = service.PrepareTask(test.access, "missing", 1)
		requireError(t, err, test.code)

		_, err = service.ListMachines(test.access)
		requireError(t, err, test.code)
		_, err = service.PrepareTechnicalCardImport(test.access, "m_gt532")
		requireError(t, err, test.code)
		_, err = service.GetTechnicalCard(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.ListTechnicalCards(test.access)
		requireError(t, err, test.code)
		_, err = service.CommitTechnicalCardImport(test.access, "missing", []string{"row"})
		requireError(t, err, test.code)
		err = service.OpenTechnicalCard(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.ListConfigurations(test.access)
		requireError(t, err, test.code)
		_, err = service.CreateConfiguration(test.access, registry.ConfigurationInput{})
		requireError(t, err, test.code)
		_, err = service.UpdateConfiguration(test.access, "missing", 1, registry.ConfigurationInput{})
		requireError(t, err, test.code)
		_, err = service.PrepareExcelImport(test.access)
		requireError(t, err, test.code)
		_, err = service.GetExcelImport(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.ListExcelImports(test.access)
		requireError(t, err, test.code)
		_, err = service.ConfigureExcelImport(test.access, "missing", 1, registry.ImportConfiguration{})
		requireError(t, err, test.code)
		_, err = service.EditExcelImportRow(test.access, "missing", 1, "row", registry.PipeInput{})
		requireError(t, err, test.code)
		_, err = service.CommitExcelImport(test.access, "missing", 1, []string{"row"})
		requireError(t, err, test.code)
		err = service.OpenExcelImportOriginal(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.ListCertificateAttachments(test.access, "missing")
		requireError(t, err, test.code)
		_, err = service.AttachCertificatePDF(test.access, "missing")
		requireError(t, err, test.code)
		err = service.OpenCertificatePDF(test.access, "missing", "missing")
		requireError(t, err, test.code)
		err = service.DetachCertificatePDF(test.access, "missing", "missing")
		requireError(t, err, test.code)
		_, err = service.CreateCertificate(test.access, registry.CertificateDraft{})
		requireError(t, err, test.code)
	}
	if chooserCalls != 0 {
		t.Fatal("file dialog opened without admin authorization")
	}
	if p, e := service.PrepareExcelImport(admin.Session.AccessToken); e != nil || p != nil {
		t.Fatal("cancelled Excel selection should be a no-op", e)
	}

	if p, e := service.PrepareTechnicalCardImport(admin.Session.AccessToken, "m_gt532"); e != nil || p != nil {
		t.Fatal("cancelled DOCX selection should be a no-op", e)
	}
	if machines, e := service.ListMachines(admin.Session.AccessToken); e != nil || len(machines) != 2 {
		t.Fatal("machines not seeded", e)
	}
	c, e := service.CreateCertificate(admin.Session.AccessToken, registry.CertificateDraft{Reference: "Fictional certificate"})
	if e != nil {
		t.Fatal(e)
	}
	if a, e := service.AttachCertificatePDF(admin.Session.AccessToken, c.ID); e != nil || a != nil {
		t.Fatal("cancelled PDF selection should be a no-op", e)
	}
	var files int
	if e = db.QueryRow("SELECT COUNT(*) FROM document_files").Scan(&files); e != nil || files != 0 {
		t.Fatal("cancel persisted a document", e)
	}
	if _, err = service.CreatePipe(admin.Session.AccessToken, registry.PipeInput{PipeData: registry.PipeData{InventoryNumber: "TEST-1", FactoryNumber: "0001"}}); err != nil {
		t.Fatal(err)
	}
	if err = manager.Logout(admin.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err = service.ListPipes(admin.Session.AccessToken)
	requireError(t, err, apperror.Unauthenticated)
}
