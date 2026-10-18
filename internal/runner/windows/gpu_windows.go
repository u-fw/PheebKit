//go:build windows

package windows

import (
	"strings"
	"syscall"
	"unsafe"

	"pheebkit/internal/core"
)

type displayDevice struct {
	Size         uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

func detectGPUs() []core.GPU {
	enumerate := syscall.NewLazyDLL("user32.dll").NewProc("EnumDisplayDevicesW")
	result := []core.GPU{}
	seen := map[string]bool{}
	for index := uint32(0); index < 64; index++ {
		device := displayDevice{}
		device.Size = uint32(unsafe.Sizeof(device))
		found, _, _ := enumerate.Call(0, uintptr(index), uintptr(unsafe.Pointer(&device)), 0)
		if found == 0 {
			break
		}
		name := syscall.UTF16ToString(device.DeviceString[:])
		identifier := strings.ToUpper(syscall.UTF16ToString(device.DeviceID[:]))
		if name == "" || seen[name] {
			continue
		}
		vendor := "other"
		switch {
		case strings.Contains(identifier, "VEN_10DE"):
			vendor = "NVIDIA"
		case strings.Contains(identifier, "VEN_1002"):
			vendor = "AMD"
		case strings.Contains(identifier, "VEN_8086"):
			vendor = "Intel"
		}
		seen[name] = true
		result = append(result, core.GPU{Name: name, Vendor: vendor})
	}
	return result
}
