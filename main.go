package main

import (
	"embed"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var frontend embed.FS

func main() {
	app := &App{}
	applicationMenu := buildMenu(app)
	err := wails.Run(&options.App{
		Title: "Paddle JSON Editor", Width: 1440, Height: 900, MinWidth: 960, MinHeight: 640,
		AssetServer: &assetserver.Options{Assets: frontend, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/session-assets/") || app.doc == nil {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimPrefix(r.URL.Path, "/session-assets/")
			path, mimeType, err := app.doc.Asset(id)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Cache-Control", "no-store")
			http.ServeFile(w, r, path)
		})},
		Menu: applicationMenu, EnableDefaultContextMenu: true, BackgroundColour: &options.RGBA{R: 246, G: 245, B: 242, A: 1},
		OnStartup: app.startup, OnShutdown: app.shutdown, OnBeforeClose: app.beforeClose,
		Bind: []interface{}{app}, ErrorFormatter: func(err error) any { return formatError(err) },
		SingleInstanceLock: &options.SingleInstanceLock{UniqueId: "com.nemuboshi.paddle-json-editor", OnSecondInstanceLaunch: func(options.SecondInstanceData) {
			if app.ctx != nil {
				runtime.WindowShow(app.ctx)
				runtime.WindowUnminimise(app.ctx)
			}
		}},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

func buildMenu(app *App) *menu.Menu {
	root := menu.NewMenu()
	file := root.AddSubmenu("File")
	file.AddText("Import JSON…", keys.CmdOrCtrl("o"), func(*menu.CallbackData) { runtime.EventsEmit(app.ctx, "menu:command", "import") })
	app.exportItems = []*menu.MenuItem{
		file.AddText("Export JSON…", keys.CmdOrCtrl("s"), func(*menu.CallbackData) { runtime.EventsEmit(app.ctx, "menu:command", "export-json") }).Disable(),
		file.AddText("Export Markdown…", nil, func(*menu.CallbackData) { runtime.EventsEmit(app.ctx, "menu:command", "export-markdown") }).Disable(),
	}
	file.AddSeparator()
	file.AddText("Quit", nil, func(*menu.CallbackData) { runtime.Quit(app.ctx) })
	find := root.AddSubmenu("Find")
	find.AddText("Search…", keys.CmdOrCtrl("f"), func(*menu.CallbackData) { runtime.EventsEmit(app.ctx, "menu:command", "search") })
	view := root.AddSubmenu("View")
	app.pageToolsItem = view.AddCheckbox("Page Tools", false, nil, func(*menu.CallbackData) { runtime.EventsEmit(app.ctx, "menu:command", "page-tools") })
	return root
}
