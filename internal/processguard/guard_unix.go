//go:build !windows

package processguard

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type unixBackend struct {
	mu        sync.Mutex
	process   *os.Process
	attached  bool
	closed    bool
	cancelled bool
}

func prepare(cmd *exec.Cmd) (backend, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return &unixBackend{}, nil
}

func (g *unixBackend) Attach(process *os.Process) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return errors.New("process guard is closed")
	}
	g.process = process
	g.attached = true
	if g.cancelled {
		return unix.Kill(-process.Pid, unix.SIGKILL)
	}
	return nil
}

func (g *unixBackend) Terminate() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return os.ErrProcessDone
	}
	g.cancelled = true
	if !g.attached || g.process == nil {
		return nil
	}
	if err := unix.Kill(-g.process.Pid, unix.SIGKILL); err != nil && !errors.Is(err, unix.ESRCH) {
		return g.process.Kill()
	}
	return nil
}

func (g *unixBackend) Close() {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	g.closed = true
	process, attached := g.process, g.attached
	g.mu.Unlock()
	if attached && process != nil {
		_ = unix.Kill(-process.Pid, unix.SIGKILL)
	}
}
