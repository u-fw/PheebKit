package app

import "fmt"

// rgbHexFromARGB drops the alpha channel used by Windows colorization values.
func rgbHexFromARGB(value uint32) string {
	return fmt.Sprintf("#%06x", value&0x00ffffff)
}
