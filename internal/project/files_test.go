package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kompeg/internal/core"
)

func TestSaveSizeBoundaryIncludesNewlineAndPreservesOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "boundary.ffproject")
	p := core.Project{SchemaVersion: 1, ID: "x"}
	base, _ := json.MarshalIndent(p, "", "  ")
	p.ID = strings.Repeat("x", maxProjectSize-len(base))
	if err := Save(path, p); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Size() != maxProjectSize {
		t.Fatal("boundary fixture has wrong size")
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	p.ID += "x"
	if err := Save(path, p); err == nil {
		t.Fatal("save produced a file too large to reopen")
	}
	if _, err := Load(path); err != nil {
		t.Fatal("rejected save damaged existing project", err)
	}
}

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
	if result.Settings.Trim != nil {
		t.Fatal("legacy whole-video project acquired a trim range")
	}
	input.Settings.Trim = &core.TimeRange{StartSeconds: 2.5, EndSeconds: 7.75}
	if err := Save(path, input); err != nil {
		t.Fatal(err)
	}
	result, err = Load(path)
	if err != nil || result.Settings.Trim == nil || *result.Settings.Trim != *input.Settings.Trim {
		t.Fatalf("trim range was lost in persistence: %+v %v", result.Settings.Trim, err)
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
