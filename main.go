package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/tidjee-dev/cash-count/backend/counts"
	"github.com/tidjee-dev/cash-count/backend/denominations"
	"github.com/tidjee-dev/cash-count/backend/settings"
	"github.com/tidjee-dev/cash-count/backend/store"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataDir, err := store.AppDataDir()
	if err != nil {
		log.Fatalf("app data dir: %v", err)
	}
	st, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	settingsSvc := settings.New(st)
	denominationsSvc := denominations.New(st)
	countsSvc := counts.New(st)

	app := application.New(application.Options{
		Name:        "cash-count-app",
		Description: "A cash count application",
		Services: []application.Service{
			application.NewService(settingsSvc),
			application.NewService(denominationsSvc),
			application.NewService(countsSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              "Cash Count",
		Width:              1000,
		Height:             618,
		BackgroundColour:   application.NewRGB(11, 13, 20),
		URL:                "/",
		UseApplicationMenu: true,
	})

	buildAppMenu(app, countsSvc)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func emitNavigate(app *application.App, path string) {
	w := app.Window.Current()
	if w == nil {
		return
	}
	w.EmitEvent("navigate", path)
}

func buildAppMenu(app *application.App, countsSvc *counts.Service) {
	menu := app.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}

	file := menu.AddSubmenu("File")
	file.Add("New Count…").SetAccelerator("CmdOrCtrl+N").OnClick(func(_ *application.Context) {
		emitNavigate(app, "/")
		if w := app.Window.Current(); w != nil {
			w.EmitEvent("count:new")
		}
	})
	file.Add("Export History CSV").SetAccelerator("CmdOrCtrl+E").OnClick(func(_ *application.Context) {
		path, err := countsSvc.ExportHistoryCSV()
		if err != nil {
			app.Dialog.Error().SetTitle("Export failed").SetMessage(err.Error()).Show()
			return
		}
		app.Dialog.Info().SetTitle("Export complete").SetMessage(path).Show()
		if w := app.Window.Current(); w != nil {
			w.EmitEvent("export:history-done", path)
		}
	})
	file.AddSeparator()
	file.AddRole(application.Quit)

	// menu.AddRole(application.EditMenu)

	goMenu := menu.AddSubmenu("Go")
	goMenu.Add("Count").OnClick(func(_ *application.Context) {
		emitNavigate(app, "/")
	})
	goMenu.Add("History").OnClick(func(_ *application.Context) {
		emitNavigate(app, "/history")
	})
	goMenu.Add("Settings").OnClick(func(_ *application.Context) {
		emitNavigate(app, "/settings")
	})

	menu.AddRole(application.ViewMenu)
	menu.AddRole(application.WindowMenu)
	menu.AddRole(application.HelpMenu)

	app.Menu.Set(menu)
}
