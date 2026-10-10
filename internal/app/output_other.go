//go:build !windows

package app

import "kompeg/internal/core"

func openOutput(string, bool) error { return core.ErrUnsupported }
