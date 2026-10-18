package buildinfo

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildDescriptionUsesActualProvenance(t *testing.T) {
	var buffer bytes.Buffer
	if err := WriteDescription(&buffer); err != nil {
		t.Fatal(err)
	}
	var value Description
	if err := json.Unmarshal(buffer.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	revision, modified := Provenance()
	if value.SchemaVersion != 1 || value.Version != Version || value.SourceRevision != revision || value.Modified != modified {
		t.Fatalf("incorrect build description: %+v", value)
	}
}

func TestProvenanceNeverCallsUnknownBuildClean(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, test := range []struct {
		revision, modified string
		settings           []debug.BuildSetting
		wantRevision       string
		wantModified       bool
	}{
		{"unknown", "false", nil, "unknown", true},
		{sha, "false", nil, sha, false},
		{sha, "unknown", nil, sha, true},
		{"unknown", "false", []debug.BuildSetting{{Key: "vcs.revision", Value: sha}}, sha, true},
		{"unknown", "true", []debug.BuildSetting{{Key: "vcs.revision", Value: sha}, {Key: "vcs.modified", Value: "false"}}, sha, false},
	} {
		revision, modified := provenance(test.revision, test.modified, test.settings)
		if revision != test.wantRevision || modified != test.wantModified {
			t.Fatalf("got %q/%v, want %q/%v", revision, modified, test.wantRevision, test.wantModified)
		}
	}
}
