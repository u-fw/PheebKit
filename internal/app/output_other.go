//go:build !windows

package app

import "pheebkit/internal/core"

func openOutput(string, bool) error { return core.ErrUnsupported }
