package main

import (
	"context"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"os"
	"pheebkit/internal/app"
	"pheebkit/internal/taskbar"
	"pheebkit/locales"
	"sync"
)

type wailsHost struct {
	ctx       context.Context
	locale    func() string
	taskbar   taskbar.Controller
	closeOnce sync.Once
}

func newWailsHost(ctx context.Context, locale func() string) *wailsHost {
	return &wailsHost{
		ctx:     ctx,
		locale:  locale,
		taskbar: taskbar.New(),
	}
}

func (h *wailsHost) Close() {
	h.closeOnce.Do(h.taskbar.Close)
}

func (h *wailsHost) localized(o app.FileDialog) app.FileDialog {
	if h.locale != nil {
		o.Title = locales.Translate(o.Title, h.locale())
		o.FilterName = locales.Translate(o.FilterName, h.locale())
	}
	return o
}

func (h *wailsHost) Emit(name string, data ...interface{}) {
	wailsruntime.EventsEmit(h.ctx, name, data...)
}
func (h *wailsHost) Quit() { wailsruntime.Quit(h.ctx) }

func (h *wailsHost) SetTaskbarProgress(progress app.TaskbarProgress) {
	state := taskbar.Hidden
	switch progress.State {
	case "running":
		state = taskbar.Normal
	case "paused":
		state = taskbar.Paused
	case "completed":
		state = taskbar.Complete
	case "error":
		state = taskbar.Error
	}
	h.taskbar.Set(taskbar.Snapshot{State: state, Value: progress.Value})
}

func (h *wailsHost) NotifyJob(notification app.JobNotification) {
	if !wailsruntime.WindowIsMinimised(h.ctx) && applicationIsForeground() {
		return
	}
	if !wailsruntime.IsNotificationAvailable(h.ctx) {
		return
	}
	notification = app.SanitizeJobNotification(notification)
	if err := wailsruntime.SendNotification(h.ctx, wailsruntime.NotificationOptions{
		ID:    notification.JobID,
		Title: notification.Title,
		Body:  notification.Body,
		Data:  map[string]interface{}{"jobID": notification.JobID},
	}); err != nil {
		log.Printf("send job notification: %v", err)
	}
}
func openOptions(o app.FileDialog) wailsruntime.OpenDialogOptions {
	options := wailsruntime.OpenDialogOptions{Title: o.Title, DefaultDirectory: dialogDirectory(o.Directory), DefaultFilename: o.Filename, CanCreateDirectories: true}
	if o.Pattern != "" {
		options.Filters = []wailsruntime.FileFilter{{DisplayName: o.FilterName, Pattern: o.Pattern}}
	}
	return options
}
func (h *wailsHost) OpenFile(o app.FileDialog) (string, error) {
	return wailsruntime.OpenFileDialog(h.ctx, openOptions(h.localized(o)))
}
func (h *wailsHost) OpenFiles(o app.FileDialog) ([]string, error) {
	return wailsruntime.OpenMultipleFilesDialog(h.ctx, openOptions(h.localized(o)))
}
func (h *wailsHost) OpenDirectory(o app.FileDialog) (string, error) {
	return wailsruntime.OpenDirectoryDialog(h.ctx, openOptions(h.localized(o)))
}
func (h *wailsHost) SaveFile(o app.FileDialog) (string, error) {
	o = h.localized(o)
	options := wailsruntime.SaveDialogOptions{Title: o.Title, DefaultDirectory: dialogDirectory(o.Directory), DefaultFilename: o.Filename, CanCreateDirectories: true}
	if o.Pattern != "" {
		options.Filters = []wailsruntime.FileFilter{{DisplayName: o.FilterName, Pattern: o.Pattern}}
	}
	return wailsruntime.SaveFileDialog(h.ctx, options)
}

// A remembered folder can disappear between runs. Still let the user choose a
// new location instead of failing Wails' default-directory preflight.
func dialogDirectory(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	return ""
}
