package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"changeme/internal/apperror"
	"changeme/internal/auth"
	"changeme/internal/documents"
	"changeme/internal/registry"
	"changeme/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Не удалось определить папку данных приложения")
	}
	dataDirectory := filepath.Join(configDir, "PipeBendingControl")
	db, err := storage.Open(dataDirectory)
	if err != nil {
		log.Fatal("Не удалось открыть локальную базу данных приложения")
	}
	defer db.Close()
	documentStore, err := documents.New(db, dataDirectory)
	if err != nil {
		log.Fatal("Не удалось открыть хранилище оригиналов документов")
	}
	manager, err := auth.New(db)
	if err != nil {
		log.Fatal("Не удалось запустить проверку учётных записей")
	}
	authService := newDesktopAuthService(manager)
	defer authService.Logout()

	app := application.New(application.Options{
		Name:        "pipe-bending-desktop",
		Description: "Цифровой контроль изготовления отводов холодного гнутья",
		Services: []application.Service{
			application.NewServiceWithOptions(authService, application.ServiceOptions{
				MarshalError: apperror.Marshal,
			}),
			application.NewServiceWithOptions(&DesktopRegistryService{auth: manager, repository: registry.New(db), documents: documentStore, imports: registry.NewImportManager(db, documentStore), technicalCards: registry.NewTechnicalCardManager(db, documentStore),
				chooseFile: func(title, pattern string) (string, error) {
					return application.Get().Dialog.OpenFile().SetTitle(title).AddFilter("Документы", pattern).PromptForSingleSelection()
				},
				saveFile: func(filename string) (string, error) {
					dialog := application.Get().Dialog.SaveFile()
					dialog.SetOptions(&application.SaveFileDialogOptions{Title: "Выгрузить задания", Filename: filename, CanCreateDirectories: true, Filters: []application.FileFilter{{DisplayName: "Пакет заданий JSON", Pattern: "*.json"}}})
					return dialog.PromptForSingleSelection()
				},
				openFile: func(path string) error { return application.Get().Browser.OpenFile(path) },
			}, application.ServiceOptions{
				MarshalError: apperror.Marshal,
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Контроль гибки — рабочее место руководителя",
		Width:  1440,
		Height: 900,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
