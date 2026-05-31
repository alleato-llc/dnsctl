// Command gui is the Wails (Go -> TypeScript) desktop frontend for dnsctl.
//
// It binds guiapi.App, which sits on the shared service facade. Privileged
// operations are forwarded to the dnsctl-helper daemon (the GUI runs
// unprivileged), so install the helper first: `make install-helper`.
//
// Build/run requires the Wails toolchain — see gui/README.md.
package main

import (
	"embed"
	"log"

	"github.com/nycjv321/dnsctl/guiapi"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// assets is the built frontend. `wails build`/`wails dev` populate frontend/dist.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := guiapi.NewApp()

	if err := wails.Run(&options.App{
		Title:  "dnsctl",
		Width:  900,
		Height: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
