// Package compat contains read-only compatibility identifiers for pre-rename
// installations. New public APIs, documentation and builds use PheebKit names.
package compat

import (
	"os"
	"path/filepath"
	"strings"
)

const LegacyRunningMutex = `Local\kompeg-application-running`

func LegacyConfigDirectory(base string) string { return filepath.Join(base, "kompeg") }

// Existing installations continue to share their original configuration store;
// moving it while another old window can write would split user preferences.
// New installations create the branded directory. No data is deleted or copied.
func ConfigDirectory(base string) string {
	current := filepath.Join(base, "PheebKit")
	if info, err := os.Stat(current); err == nil && info.IsDir() {
		return current
	}
	old := LegacyConfigDirectory(base)
	if info, err := os.Stat(old); err == nil && info.IsDir() {
		return old
	}
	return current
}

func BinaryVariables(name string) []string {
	name = strings.ToUpper(name)
	return []string{"PHEEBKIT_" + name, "KOMPEG_" + name, name + "_PATH"}
}

func BinaryEnvironment(name string, getenv func(string) string) string {
	for _, variable := range BinaryVariables(name) {
		if value := strings.TrimSpace(getenv(variable)); value != "" {
			return value
		}
	}
	return ""
}

// Fingerprints are opaque persisted identifiers, not display names. Retaining
// the domain bytes avoids invalidating source-bound recovery and queued plans.
func FingerprintDomain(legacy bool) string {
	if legacy {
		return "kompeg-source-v1"
	}
	return "kompeg-source-v2"
}
