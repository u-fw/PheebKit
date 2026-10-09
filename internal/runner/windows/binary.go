package windows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveBinary bypasses Scoop's executable shim, so CommandContext cancellation
// owns the real encoder process instead of only killing its launcher.
func resolveBinary(binary string) string {
	resolved, err := exec.LookPath(binary)
	if err != nil {
		return binary // Execution reports the actionable missing-binary error.
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".exe") {
		return resolved
	}
	shim := strings.TrimSuffix(resolved, filepath.Ext(resolved)) + ".shim"
	stat, err := os.Stat(shim)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > 64*1024 {
		return resolved
	}
	data, err := os.ReadFile(shim)
	if err != nil {
		return resolved
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "path = \"") || !strings.HasSuffix(line, "\"") {
			continue
		}
		target := strings.TrimSuffix(strings.TrimPrefix(line, "path = \""), "\"")
		if !filepath.IsAbs(target) || !strings.EqualFold(filepath.Base(target), filepath.Base(resolved)) {
			return resolved
		}
		if stat, err := os.Stat(target); err == nil && stat.Mode().IsRegular() {
			return target
		}
	}
	return resolved
}
