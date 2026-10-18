package buildinfo

import (
	"runtime/debug"
	"strings"
	"testing"
)

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
