//go:build !windows

package app

import (
	"context"
	"pheebkit/internal/core"
)

func platformThumbnail(context.Context, string) ([]byte, error) { return nil, core.ErrUnsupported }
