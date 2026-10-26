package buildinfo

import (
	"encoding/json"
	"io"
	"regexp"
	"runtime/debug"

	"pheebkit/internal/bundledffmpeg"
)

type Description struct {
	SchemaVersion  int                             `json:"schemaVersion"`
	Version        string                          `json:"version"`
	SourceRevision string                          `json:"sourceRevision"`
	Modified       bool                            `json:"modified"`
	Framework      string                          `json:"framework"`
	BundledFFmpeg  *bundledffmpeg.BuildDescription `json:"bundledFFmpeg,omitempty"`
}

// WriteDescription supports artifact verification without creating a window,
// reading user configuration or starting a media process.
func WriteDescription(writer io.Writer) error {
	revision, modified := Provenance()
	value := Description{SchemaVersion: 1, Version: Version, SourceRevision: revision, Modified: modified, Framework: "unknown"}
	bundle := bundledffmpeg.DescribeBuild()
	if bundle.Kind != "external" {
		value.BundledFFmpeg = &bundle
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dependency := range info.Deps {
			if dependency.Path == "github.com/wailsapp/wails/v2" {
				value.Framework = "wails-" + dependency.Version
			}
		}
	}
	return json.NewEncoder(writer).Encode(value)
}

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
