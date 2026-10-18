package main

import (
	"pheebkit/internal/app"
	"testing"
)

func TestNativeDialogLocaleDoesNotTranslateUserPathsOrFilenames(t *testing.T) {
	s := &app.Service{}
	host := &wailsHost{locale: s.Locale}
	input := app.FileDialog{Title: "选择媒体文件（可多选）", FilterName: "视频与音频", Filename: "设置.mp4", Directory: "C:/用户/工作台"}
	if err := s.SetLocale("en-US"); err != nil {
		t.Fatal(err)
	}
	output := host.localized(input)
	if output.Title != "Choose media files (multiple selection)" || output.FilterName != "Video and audio" {
		t.Fatal("dialog did not use selected language", output)
	}
	if output.Filename != input.Filename || output.Directory != input.Directory {
		t.Fatal("user data was translated")
	}
	if err := s.SetLocale("zh-CN"); err != nil {
		t.Fatal(err)
	}
	if host.localized(input) != input {
		t.Fatal("Chinese dialog changed")
	}
}
