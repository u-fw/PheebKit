package locales

import "testing"

func TestNativeDialogTranslationPreservesUnknownAndChineseValues(t *testing.T) {
	const title = "选择媒体文件（可多选）"
	if Translate(title, "en-US") != "Choose media files (multiple selection)" {
		t.Fatal("missing native dialog translation")
	}
	if Translate(title, "zh-CN") != title {
		t.Fatal("Chinese title changed")
	}
	const filename = "C:/用户/设置.mp4"
	if Translate(filename, "en-US") != filename {
		t.Fatal("user path was translated")
	}
}
