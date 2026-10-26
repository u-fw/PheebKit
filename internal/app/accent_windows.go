package app

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

func systemAccentColor() (string, error) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\DWM`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetIntegerValue("ColorizationColor")
	if err != nil {
		return "", err
	}
	if value > 0xffffffff {
		return "", errors.New("Windows accent color is outside the ARGB range")
	}
	return rgbHexFromARGB(uint32(value)), nil
}
