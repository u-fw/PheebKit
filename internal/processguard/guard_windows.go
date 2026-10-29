//go:build windows

package processguard

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsBackend struct {
	mu        sync.Mutex
	job       windows.Handle
	process   *os.Process
	attached  bool
	closed    bool
	cancelled bool
}

var getThreadTimes = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetThreadTimes")

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
	// Go's StartProcess closes the primary thread handle before returning. Keep
	// the process suspended until Attach has enrolled it in our job, then resume
	// that thread through the documented Toolhelp/thread APIs. No user code can
	// create an uncontained descendant between Start and Attach.
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW | windows.CREATE_SUSPENDED
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
	if g.cancelled {
		return windows.TerminateJobObject(g.job, 1)
	}
	if err := resumePrimaryThread(uint32(process.Pid)); err != nil {
		_ = windows.TerminateJobObject(g.job, 1)
		return fmt.Errorf("resume guarded process: %w", err)
	}
	return nil
}

// A suspended newly created process has not run its loader. Select the oldest
// thread rather than assuming Toolhelp's enumeration order, so an injected
// instrumentation thread cannot be mistaken for the primary thread.
func resumePrimaryThread(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	var primary windows.Handle
	var oldest uint64
	defer func() {
		if primary != 0 {
			_ = windows.CloseHandle(primary)
		}
	}()
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME|windows.THREAD_QUERY_LIMITED_INFORMATION, false, entry.ThreadID)
		if openErr != nil {
			return openErr
		}
		var created, exited, kernel, user windows.Filetime
		ok, _, timesErr := getThreadTimes.Call(uintptr(thread),
			uintptr(unsafe.Pointer(&created)), uintptr(unsafe.Pointer(&exited)),
			uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
		if ok == 0 {
			_ = windows.CloseHandle(thread)
			return timesErr
		}
		stamp := uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime)
		if primary == 0 || stamp < oldest {
			if primary != 0 {
				_ = windows.CloseHandle(primary)
			}
			primary, oldest = thread, stamp
		} else {
			_ = windows.CloseHandle(thread)
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return err
	}
	if primary == 0 {
		return errors.New("suspended primary thread was not found")
	}
	count, err := windows.ResumeThread(primary)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("unexpected primary thread suspend count %d", count)
	}
	return nil
}

func (g *windowsBackend) Terminate() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.job == 0 {
		return os.ErrProcessDone
	}
	g.cancelled = true
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
