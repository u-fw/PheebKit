// Package processguard binds a child process and all descendants to one
// lifecycle so cancellation also closes helpers they started.
package processguard

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

type backend interface {
	Attach(*os.Process) error
	Terminate() error
	Close()
}

type Guard struct {
	backend backend
}

// Prepare configures a CommandContext before it starts. The returned guard must
// be attached immediately after Start and closed after Wait.
func Prepare(cmd *exec.Cmd) (*Guard, error) {
	if cmd == nil {
		return nil, errors.New("process guard requires a command")
	}
	implementation, err := prepare(cmd)
	if err != nil {
		return nil, err
	}
	guard := &Guard{backend: implementation}
	cmd.Cancel = guard.Terminate
	cmd.WaitDelay = 2 * time.Second
	return guard, nil
}

func (g *Guard) Attach(process *os.Process) error {
	if g == nil || g.backend == nil {
		return errors.New("process guard is not initialized")
	}
	if process == nil {
		return errors.New("process guard cannot attach a nil process")
	}
	return g.backend.Attach(process)
}

func (g *Guard) Terminate() error {
	if g == nil || g.backend == nil {
		return os.ErrProcessDone
	}
	return g.backend.Terminate()
}

func (g *Guard) Close() {
	if g != nil && g.backend != nil {
		g.backend.Close()
	}
}

// Run provides the same guarded lifecycle for short-lived inspection commands
// created with exec.CommandContext.
func Run(cmd *exec.Cmd) error {
	guard, err := Prepare(cmd)
	if err != nil {
		return err
	}
	defer guard.Close()
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := guard.Attach(cmd.Process); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	return cmd.Wait()
}
