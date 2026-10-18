package buildinfo

import (
	"regexp"
	"runtime/debug"
)

// Stable Wails v2 disables Go's automatic VCS stamping. The reviewed Windows
// build script supplies these values through the supported linker flags.
var Revision = "unknown"
var Modified = "true"

var revisionPattern = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)

func Provenance() (string, bool) {
	var settings []debug.BuildSetting
	if info, ok := debug.ReadBuildInfo(); ok {
		settings = info.Settings
	}
	return provenance(Revision, Modified, settings)
}

func provenance(revision, modified string, settings []debug.BuildSetting) (string, bool) {
	if revisionPattern.MatchString(revision) {
		return revision, modified != "false"
	}
	revision, modified = "unknown", "true"
	for _, setting := range settings {
		if setting.Key == "vcs.revision" && revisionPattern.MatchString(setting.Value) {
			revision = setting.Value
		}
		if setting.Key == "vcs.modified" {
			modified = setting.Value
		}
	}
	return revision, revision == "unknown" || modified != "false"
}
