//go:build !windows

package windows

import "pheebkit/internal/core"

func detectGPUs() []core.GPU { return []core.GPU{} }
