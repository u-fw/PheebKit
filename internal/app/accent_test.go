package app

import "testing"

func TestWindowsColorizationValueMapsToRGB(t *testing.T) {
	for value, want := range map[uint32]string{
		0xc40078d4: "#0078d4",
		0xffd47800: "#d47800",
		0x00abcdef: "#abcdef",
	} {
		if got := rgbHexFromARGB(value); got != want {
			t.Fatalf("rgbHexFromARGB(%#x) = %q, want %q", value, got, want)
		}
	}
}
