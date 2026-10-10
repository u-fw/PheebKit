package main

import (
	"embed"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"kompeg/internal/app"
	"kompeg/internal/core"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[core.Media]("media:probed")
	application.RegisterEvent[core.Job]("job:updated")
	application.RegisterEvent[string]("bridge:ready")
	application.RegisterEvent[[]string]("media:drop")
	application.RegisterEvent[bool]("window:maximised")
}

func main() {
	releaseGuard, err := holdInstallerGuard()
	if err != nil {
		log.Fatal(err)
	}
	defer releaseGuard()
	service := app.NewService()
	a := application.New(application.Options{
		Name:        "PheebKit",
		Description: "菲比工具箱 · 本地媒体处理工具",
		Services:    []application.Service{application.NewService(service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows:     windowsDebugOptions(),
	})
	service.Attach(a)
	a.OnShutdown(service.Close)
	window := a.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "菲比工具箱 · PheebKit",
		Width: 1280, Height: 860, MinWidth: 360, MinHeight: 620,
		Frameless:        runtime.GOOS == "windows",
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGB(15, 22, 23),
		URL:              "/",
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		a.Event.Emit("media:drop", event.Context().DroppedFiles())
	})
	for _, eventType := range []events.WindowEventType{events.Common.WindowMaximise, events.Common.WindowUnMaximise} {
		window.OnWindowEvent(eventType, func(*application.WindowEvent) {
			a.Event.Emit("window:maximised", window.IsMaximised())
		})
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// Debugging is opt-in for local WebView verification; normal runs expose no port.
func windowsDebugOptions() application.WindowsOptions {
	value := os.Getenv("KOMPEG_DEBUG_PORT")
	if value == "" {
		return application.WindowsOptions{}
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1024 || port > 65535 {
		log.Fatal("KOMPEG_DEBUG_PORT must be an integer from 1024 to 65535")
	}
	return application.WindowsOptions{AdditionalBrowserArgs: []string{"--remote-debugging-port=" + strconv.Itoa(port)}}
}
