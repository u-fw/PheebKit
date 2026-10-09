//go:build windows

package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

var (
	thumbnailOLE                = syscall.NewLazyDLL("ole32.dll")
	thumbnailShell              = syscall.NewLazyDLL("shell32.dll")
	thumbnailGDI                = syscall.NewLazyDLL("gdi32.dll")
	coInitializeEx              = thumbnailOLE.NewProc("CoInitializeEx")
	coUninitialize              = thumbnailOLE.NewProc("CoUninitialize")
	shCreateItemFromParsingName = thumbnailShell.NewProc("SHCreateItemFromParsingName")
	getObjectW                  = thumbnailGDI.NewProc("GetObjectW")
	getDIBits                   = thumbnailGDI.NewProc("GetDIBits")
	deleteObject                = thumbnailGDI.NewProc("DeleteObject")
	createCompatibleDC          = thumbnailGDI.NewProc("CreateCompatibleDC")
	deleteDC                    = thumbnailGDI.NewProc("DeleteDC")
	thumbnailSlots              = make(chan struct{}, 2)
)

type imageFactoryVTable struct{ QueryInterface, AddRef, Release, GetImage uintptr }
type shellImageFactory struct{ VTable *imageFactoryVTable }
type nativeBitmap struct {
	Type, Width, Height, WidthBytes int32
	Planes, BitsPixel               uint16
	Bits                            unsafe.Pointer
}
type bitmapInfoHeader struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}

// Shell providers can be slow or broken. Bound callers and workers separately:
// COM cannot safely be interrupted while it owns a provider's interface.
func platformThumbnail(ctx context.Context, path string) ([]byte, error) {
	select {
	case thumbnailSlots <- struct{}{}:
	default:
		return nil, errors.New("Windows 缩略图服务正在忙碌。")
	}
	type result struct {
		data []byte
		err  error
	}
	completed := make(chan result, 1)
	go func() {
		defer func() { <-thumbnailSlots }()
		data, err := extractShellThumbnail(path)
		completed <- result{data, err}
	}()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	select {
	case value := <-completed:
		return value.data, value.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("Windows 缩略图读取超时。")
	}
}

func extractShellThumbnail(path string) ([]byte, error) {
	if strconv.IntSize != 64 {
		return nil, errors.New("Windows 缩略图暂不支持 32 位进程。")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// This background thread has no window/message loop, so use MTA. COM calls,
	// interface release and uninitialization all stay on the locked OS thread.
	hresult, _, _ := coInitializeEx.Call(0, 4) // COINIT_MULTITHREADED | COINIT_DISABLE_OLE1DDE
	if int32(hresult) < 0 {
		return nil, fmt.Errorf("initialize thumbnail COM: 0x%08x", uint32(hresult))
	}
	defer coUninitialize.Call()
	widePath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	iid := syscall.GUID{Data1: 0xbcc18b79, Data2: 0xba16, Data3: 0x442f, Data4: [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b}}
	var factory *shellImageFactory
	hresult, _, _ = shCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(widePath)), 0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&factory)))
	if int32(hresult) < 0 || factory == nil {
		return nil, fmt.Errorf("create thumbnail factory: 0x%08x", uint32(hresult))
	}
	defer syscall.SyscallN(factory.VTable.Release, uintptr(unsafe.Pointer(factory)))
	var bitmap uintptr
	// SIZE is passed by value as two packed LONGs on Windows 64-bit ABIs.
	size := uintptr(uint64(512) | uint64(512)<<32)
	hresult, _, _ = syscall.SyscallN(factory.VTable.GetImage, uintptr(unsafe.Pointer(factory)), size, 0x08, uintptr(unsafe.Pointer(&bitmap))) // SIIGBF_THUMBNAILONLY; preserve aspect ratio.
	if int32(hresult) < 0 || bitmap == 0 {
		return nil, fmt.Errorf("get Windows thumbnail: 0x%08x", uint32(hresult))
	}
	defer deleteObject.Call(bitmap)
	var dimensions nativeBitmap
	count, _, _ := getObjectW.Call(bitmap, unsafe.Sizeof(dimensions), uintptr(unsafe.Pointer(&dimensions)))
	if count == 0 || dimensions.Width <= 0 || dimensions.Height <= 0 || dimensions.Width > 1024 || dimensions.Height > 1024 {
		return nil, errors.New("invalid shell thumbnail dimensions")
	}
	dc, _, _ := createCompatibleDC.Call(0)
	if dc == 0 {
		return nil, errors.New("create thumbnail device context failed")
	}
	defer deleteDC.Call(dc)
	width, height := int(dimensions.Width), int(dimensions.Height)
	pixels := make([]byte, width*height*4)
	header := bitmapInfoHeader{Size: 40, Width: dimensions.Width, Height: -dimensions.Height, Planes: 1, BitCount: 32, SizeImage: uint32(len(pixels))}
	lines, _, _ := getDIBits.Call(dc, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0)
	if lines != uintptr(height) {
		return nil, errors.New("read shell thumbnail bitmap failed")
	}
	output := image.NewNRGBA(image.Rect(0, 0, width, height))
	for offset := 0; offset < len(pixels); offset += 4 {
		output.Pix[offset], output.Pix[offset+1], output.Pix[offset+2], output.Pix[offset+3] = pixels[offset+2], pixels[offset+1], pixels[offset], 255
	}
	var encoded limitedBuffer
	encoded.limit = thumbnailLimit
	if err := png.Encode(&encoded, output); err != nil {
		return nil, err
	}
	return bytes.Clone(encoded.Bytes()), nil
}
