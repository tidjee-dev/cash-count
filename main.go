package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/tidjee-dev/cash-count/backend/counts"
	"github.com/tidjee-dev/cash-count/backend/denominations"
	"github.com/tidjee-dev/cash-count/backend/settings"
	"github.com/tidjee-dev/cash-count/backend/store"
)

// appVersion is shown in Help > About. Keep in sync with info.version in
// build/config.yml (that file is not embedded, so it cannot be read here).
const appVersion = "0.1.0"

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

	help := menu.AddSubmenu("Help")
	help.Add("About Cash Count").OnClick(func(_ *application.Context) {
		showAbout(app, countsSvc.Store.Dir)
	})

	app.Menu.Set(menu)
}

// showAbout opens the app info panel: version, platform, and the live data
// paths (database + exports) with filesystem stats. Stats are best-effort:
// failures render as "?" rather than blocking the dialog.
func showAbout(app *application.App, dataDir string) {
	dbPath := filepath.Join(dataDir, "cashcount.db")
	exportsDir := filepath.Join(dataDir, "exports")

	dbSize := "?"
	if fi, err := os.Stat(dbPath); err == nil {
		dbSize = formatBytes(fi.Size())
	}
	exportCount := "?"
	if files, err := filepath.Glob(filepath.Join(exportsDir, "*.csv")); err == nil {
		exportCount = fmt.Sprintf("%d CSV file(s)", len(files))
	}

	msg := fmt.Sprintf(
		"Cash Count %s\nPOS cash counting — local-first, single register.\n(c) 2026, Cash Count · MIT\n\nPlatform: %s / %s\nData folder: %s\nDatabase: %s (%s)\nExports: %s (%s)",
		appVersion, runtime.GOOS, runtime.GOARCH,
		dataDir, dbPath, dbSize, exportsDir, exportCount,
	)

	dlg := app.Dialog.Info().SetTitle("About Cash Count").SetMessage(msg)
	dlg.AddButton("Open data folder").OnClick(func() {
		_ = app.Env.OpenFileManager(dataDir, false)
	})
	dlg.AddButton("Open exports folder").OnClick(func() {
		_ = app.Env.OpenFileManager(exportsDir, false)
	})
	dlg.AddButton("Close").SetAsDefault().OnClick(func() {})
	dlg.Show()
}

// formatBytes renders a byte count for the About panel (e.g. 84 KB).
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	v := float64(n)
	for _, u := range []string{"KB", "MB", "GB"} {
		v /= unit
		if v < unit {
			return fmt.Sprintf("%.0f %s", v, u)
		}
	}
	return fmt.Sprintf("%.0f TB", v/unit)
}
