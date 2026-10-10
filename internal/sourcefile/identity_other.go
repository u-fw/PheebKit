//go:build !windows && !linux && !android

package sourcefile

import (
	"os"
)

func platformIdentity(_ *os.File, info os.FileInfo) (string, error) {
	// Other targets retain bounded content/stat checks until they gain a native
	// media adapter. Never include last-access time, which a read can change.
	return info.Name(), nil
}
