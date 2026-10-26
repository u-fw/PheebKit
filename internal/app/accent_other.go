//go:build !windows

package app

import "errors"

func systemAccentColor() (string, error) {
	return "", errors.New("system accent color is available on Windows only")
}
