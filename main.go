package main

import (
	"context"
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"kompeg/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	releaseGuard, err := holdInstallerGuard()
	if err != nil {
		log.Fatal(err)
	}
	defer releaseGuard()
	service := app.NewService()
	defer service.Close()
	err = wails.Run(&options.App{
		Title: "菲比工具箱 · PheebKit",
		Width: 1280, Height: 860, MinWidth: 360, MinHeight: 620,
		Frameless:        runtime.GOOS == "windows",
		BackgroundColour: options.NewRGB(15, 22, 23),
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: service.AssetHandler},
		Bind:             []interface{}{&app.Desktop{DesktopAPI: service}},
		OnStartup:        func(ctx context.Context) { service.Attach(&wailsHost{ctx: ctx}) },
		OnBeforeClose:    func(context.Context) bool { return !service.CanClose() },
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
		Windows:          &windows.Options{WindowClassName: "PheebKitWindow", DLLSearchPaths: windows.DLLSearchApplicationDir | windows.DLLSearchSystem32},
	})
	if err != nil {
		log.Fatal(err)
	}
}
