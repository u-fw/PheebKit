//go:build windows

package taskbar

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	taskbarButtonCreated = 0x00000001
	taskbarNoProgress    = 0x00000000
	taskbarIndeterminate = 0x00000001
	taskbarNormal        = 0x00000002
	taskbarError         = 0x00000004
	taskbarPaused        = 0x00000008
	terminalDisplayTime  = 5 * time.Second
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	ole32                  = windows.NewLazySystemDLL("ole32.dll")
	procFindWindowW        = user32.NewProc("FindWindowW")
	procFindWindowExW      = user32.NewProc("FindWindowExW")
	procGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	procCoCreateInstance   = ole32.NewProc("CoCreateInstance")
)

type taskbarController struct {
	mu     sync.RWMutex
	closed bool
	queue  chan Snapshot
	done   chan struct{}
}

func New() Controller {
	controller := &taskbarController{
		queue: make(chan Snapshot, 1),
		done:  make(chan struct{}),
	}
	go controller.run()
	return controller
}

func (c *taskbarController) Set(snapshot Snapshot) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return
	}
	select {
	case c.queue <- snapshot:
	default:
		select {
		case <-c.queue:
		default:
		}
		c.queue <- snapshot
	}
}

func (c *taskbarController) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.queue)
	c.mu.Unlock()
	<-c.done
}

func (c *taskbarController) run() {
	defer close(c.done)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		return
	}
	defer windows.CoUninitialize()
	object, err := createTaskbarList()
	if err != nil {
		return
	}
	defer releaseCOM(object)
	if err := hrInit(object); err != nil {
		return
	}

	var window windows.Handle
	var reset <-chan time.Time
	for snapshot := range c.queue {
		if window == 0 {
			window = applicationWindow()
		}
		if window == 0 {
			continue
		}
		applyTaskbarState(object, window, snapshot)
		if snapshot.State == Complete || snapshot.State == Error {
			reset = time.After(terminalDisplayTime)
		} else {
			reset = nil
		}
		if reset == nil {
			continue
		}
		for waiting := true; waiting; {
			select {
			case next, open := <-c.queue:
				if !open {
					applyTaskbarState(object, window, Snapshot{State: Hidden})
					return
				}
				snapshot = next
				if window == 0 {
					window = applicationWindow()
				}
				if window != 0 {
					applyTaskbarState(object, window, snapshot)
				}
				waiting = snapshot.State == Complete || snapshot.State == Error
				if waiting {
					reset = time.After(terminalDisplayTime)
				}
			case <-reset:
				applyTaskbarState(object, window, Snapshot{State: Hidden})
				waiting = false
			}
		}
	}
	if window != 0 {
		applyTaskbarState(object, window, Snapshot{State: Hidden})
	}
}

type comGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidTaskbarList = comGUID{
		Data1: 0x56fdf344,
		Data2: 0xfd6d,
		Data3: 0x11d0,
		Data4: [8]byte{0x95, 0x8a, 0x00, 0x60, 0x97, 0xc9, 0xa0, 0x90},
	}
	iidTaskbarList3 = comGUID{
		Data1: 0xea1afb91,
		Data2: 0x9e28,
		Data3: 0x4b86,
		Data4: [8]byte{0x90, 0xe9, 0x9e, 0x9f, 0x8a, 0x5e, 0xef, 0xaf},
	}
)

func createTaskbarList() (unsafe.Pointer, error) {
	var object unsafe.Pointer
	result, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidTaskbarList)),
		0,
		windows.CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&iidTaskbarList3)),
		uintptr(unsafe.Pointer(&object)),
	)
	if err := hresultError(result); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, syscall.EINVAL
	}
	return object, nil
}

func hrInit(object unsafe.Pointer) error {
	result, err := callCOM(object, 3)
	if err != nil {
		return err
	}
	return hresultError(result)
}

func applyTaskbarState(object unsafe.Pointer, window windows.Handle, snapshot Snapshot) {
	flag := uintptr(taskbarNoProgress)
	switch snapshot.State {
	case Normal, Complete:
		flag = taskbarNormal
	case Paused:
		flag = taskbarPaused
	case Error:
		flag = taskbarError
	case Hidden:
		flag = taskbarNoProgress
	default:
		flag = taskbarIndeterminate
	}
	_, _ = callCOM(object, 10, uintptr(window), flag)
	if snapshot.State == Hidden {
		return
	}
	_, _ = callCOM(object, 9, uintptr(window), uintptr(clamp(snapshot.Value)), 100)
}

func callCOM(object unsafe.Pointer, index uintptr, args ...uintptr) (uintptr, error) {
	if object == nil {
		return 0, syscall.EINVAL
	}
	vtable := *(*unsafe.Pointer)(object)
	if vtable == nil {
		return 0, syscall.EINVAL
	}
	method := *(*uintptr)(unsafe.Add(vtable, index*unsafe.Sizeof(uintptr(0))))
	if method == 0 {
		return 0, syscall.EINVAL
	}
	callArgs := make([]uintptr, 0, len(args)+1)
	callArgs = append(callArgs, uintptr(object))
	callArgs = append(callArgs, args...)
	result, _, _ := syscall.SyscallN(method, callArgs...)
	return result, nil
}

func releaseCOM(object unsafe.Pointer) {
	if object != nil {
		_, _ = callCOM(object, 2)
	}
}

func hresultError(result uintptr) error {
	if int32(uint32(result)) >= 0 {
		return nil
	}
	return syscall.Errno(uint32(result))
}

func applicationWindow() windows.Handle {
	className, err := windows.UTF16PtrFromString("PheebKitWindow")
	if err != nil {
		return 0
	}
	window, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		0,
	)
	for window != 0 {
		var processID uint32
		procGetWindowThreadPID.Call(
			window,
			uintptr(unsafe.Pointer(&processID)),
		)
		if processID == uint32(windows.GetCurrentProcessId()) {
			return windows.Handle(window)
		}
		next, _, _ := procFindWindowExW.Call(
			0,
			window,
			uintptr(unsafe.Pointer(className)),
			0,
		)
		window = next
	}
	return 0
}
