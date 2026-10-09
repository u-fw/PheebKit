//go:build !windows

package windows

import "kompeg/internal/core"

func detectGPUs() []core.GPU { return []core.GPU{} }
