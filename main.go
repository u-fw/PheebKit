package main

import (
	"context"
	"embed"
	"log"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"pheebkit/internal/app"
	"pheebkit/internal/buildinfo"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--build-info" {
		if err := buildinfo.WriteDescription(os.Stdout); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}
	releaseGuard, err := holdInstallerGuard()
	if err != nil {
		log.Fatal(err)
	}
	defer releaseGuard()
	service := app.NewService()
	defer service.Close()
	var host *wailsHost
	err = wails.Run(&options.App{
		Title: "菲比工具箱 · PheebKit",
		Width: 1280, Height: 860, MinWidth: 360, MinHeight: 620,
		Frameless:        runtime.GOOS == "windows",
		BackgroundColour: options.NewRGB(15, 22, 23),
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: service.AssetHandler},
		Bind:             []interface{}{&app.Desktop{DesktopAPI: service}},
		OnStartup: func(ctx context.Context) {
			host = newWailsHost(ctx, service.Locale)
			if err := wailsruntime.InitializeNotifications(ctx); err != nil {
				log.Printf("initialize notifications: %v", err)
			}
			wailsruntime.OnNotificationResponse(ctx, func(result wailsruntime.NotificationResult) {
				if result.Error != nil || result.Response.ID == "" {
					return
				}
				wailsruntime.WindowUnminimise(ctx)
				wailsruntime.WindowShow(ctx)
				wailsruntime.EventsEmit(ctx, "job:activated", result.Response.ID)
			})
			service.Attach(host)
		},
		OnShutdown: func(ctx context.Context) {
			wailsruntime.CleanupNotifications(ctx)
			if host != nil {
				host.Close()
			}
		},
		OnBeforeClose: func(context.Context) bool { return !service.CanClose() },
		DragAndDrop:   &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
		Windows:       &windows.Options{WindowClassName: "PheebKitWindow", DLLSearchPaths: windows.DLLSearchApplicationDir | windows.DLLSearchSystem32},
	})
	if err != nil {
		log.Fatal(err)
	}
}
