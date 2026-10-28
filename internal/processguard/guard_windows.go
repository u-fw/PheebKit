//go:build windows

package processguard

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsBackend struct {
	mu       sync.Mutex
	job      windows.Handle
	process  *os.Process
	attached bool
	closed   bool
}

func prepare(cmd *exec.Cmd) (backend, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	ret, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil || ret == 0 {
		windows.CloseHandle(job)
		if err == nil {
			err = errors.New("configure process guard job object failed")
		}
		return nil, err
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
	return &windowsBackend{job: job}, nil
}

func (g *windowsBackend) Attach(process *os.Process) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.job == 0 {
		return errors.New("process guard is closed")
	}
	handle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(process.Pid),
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(g.job, handle); err != nil {
		return err
	}
	g.process = process
	g.attached = true
	return nil
}

func (g *windowsBackend) Terminate() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.job == 0 {
		return os.ErrProcessDone
	}
	if !g.attached {
		return nil
	}
	if err := windows.TerminateJobObject(g.job, 1); err != nil &&
		!errors.Is(err, windows.ERROR_ACCESS_DENIED) &&
		!errors.Is(err, windows.ERROR_INVALID_HANDLE) {
		return err
	}
	if g.process != nil {
		_ = g.process.Kill()
	}
	return nil
}

func (g *windowsBackend) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	g.closed = true
	if g.job != 0 {
		_ = windows.CloseHandle(g.job)
		g.job = 0
	}
}
