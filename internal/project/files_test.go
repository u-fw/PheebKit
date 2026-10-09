package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffmpegstudio/internal/core"
)

func TestRoundTripAndFutureSchemaRejection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "项目.ffproject")
	input := core.Project{SchemaVersion: 1, ID: "test", Inputs: []core.Source{{URI: "content://media/123", Kind: core.SourceContentURI}}, Settings: core.DefaultSettings()}
	if err := Save(path, input); err != nil {
		t.Fatal(err)
	}
	result, err := Load(path)
	if err != nil || result.Inputs[0].URI != input.Inputs[0].URI || result.Settings.Codec != "hevc" {
		t.Fatalf("round trip lost intent: %+v %v", result, err)
	}
	for _, raw := range []string{`{"schemaVersion":2}`, `{"schemaVersion":1} {}`, `{"schemaVersion":1,"unexpected":true}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("accepted unsupported project: %s", raw)
		}
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1}`+strings.Repeat(" ", maxProjectSize)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("oversized project with trailing whitespace bypassed the limit")
	}
}
