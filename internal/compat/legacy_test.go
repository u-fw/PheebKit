package compat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrandedConfigForNewInstallAndReadOnlyLegacyReuse(t *testing.T) {
	base := t.TempDir()
	if ConfigDirectory(base) != filepath.Join(base, "PheebKit") {
		t.Fatal("new installations must use the branded directory")
	}
	old := filepath.Join(base, "kompeg")
	if err := os.Mkdir(old, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(old, "ffmpeg.json")
	if err := os.WriteFile(path, []byte("retained"), 0600); err != nil {
		t.Fatal(err)
	}
	if ConfigDirectory(base) != old {
		t.Fatal("existing configuration was abandoned")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "retained" {
		t.Fatal("legacy data was altered", err)
	}
	modern := filepath.Join(base, "PheebKit")
	if err := os.Mkdir(modern, 0700); err != nil {
		t.Fatal(err)
	}
	if ConfigDirectory(base) != modern {
		t.Fatal("explicit modern configuration must take precedence")
	}
}

func TestPublicEnvironmentWinsAndLegacyAliasesStillWork(t *testing.T) {
	values := map[string]string{"PHEEBKIT_FFMPEG": "current", "KOMPEG_FFMPEG": "legacy", "FFMPEG_PATH": "generic"}
	getenv := func(key string) string { return values[key] }
	if BinaryEnvironment("ffmpeg", getenv) != "current" {
		t.Fatal("current name lost priority")
	}
	delete(values, "PHEEBKIT_FFMPEG")
	if BinaryEnvironment("ffmpeg", getenv) != "legacy" {
		t.Fatal("old override stopped working")
	}
	delete(values, "KOMPEG_FFMPEG")
	if BinaryEnvironment("ffmpeg", getenv) != "generic" {
		t.Fatal("generic override stopped working")
	}
}
