package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"changeme/internal/apperror"
	"changeme/internal/auth"
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
	db, err := storage.Open(filepath.Join(configDir, "PipeBendingControl"))
	if err != nil {
		log.Fatal("Не удалось открыть локальную базу данных приложения")
	}
	defer db.Close()
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
