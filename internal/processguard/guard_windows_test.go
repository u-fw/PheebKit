//go:build windows

package processguard

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestProcessGuardTerminatesDescendantTree(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessGuardHelper$")
	cmd.Env = append(os.Environ(), "PHEEBKIT_PROCESS_GUARD_TEST=parent", "PHEEBKIT_PROCESS_GUARD_PID_FILE="+pidFile)
	done := make(chan error, 1)
	go func() { done <- Run(cmd) }()

	var pid int
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil && pid > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid <= 0 {
		cancel()
		<-done
		t.Fatal("helper descendant did not start")
	}
	if !processAlive(pid) {
		t.Fatal("helper descendant exited before cancellation")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("guarded process did not exit after cancellation")
	}
	deadline = time.Now().Add(5 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if processAlive(pid) {
		t.Fatal("descendant survived process guard termination")
	}
}

func TestProcessGuardCleansDescendantsAfterNormalParentExit(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestProcessGuardHelper$")
	cmd.Env = append(os.Environ(), "PHEEBKIT_PROCESS_GUARD_TEST=exiting-parent", "PHEEBKIT_PROCESS_GUARD_PID_FILE="+pidFile)
	if err := Run(cmd); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid descendant PID %q: %v", data, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if processAlive(pid) {
		t.Fatal("descendant survived normal guarded parent exit")
	}
}

func TestProcessGuardHelper(t *testing.T) {
	mode := os.Getenv("PHEEBKIT_PROCESS_GUARD_TEST")
	switch mode {
	case "descendant":
		time.Sleep(time.Hour)
	case "parent", "exiting-parent":
		child := exec.Command(os.Args[0], "-test.run=^TestProcessGuardHelper$")
		child.Env = replaceEnv(os.Environ(),
			"PHEEBKIT_PROCESS_GUARD_TEST=descendant",
			"PHEEBKIT_PROCESS_GUARD_PID_FILE=",
		)
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		path := os.Getenv("PHEEBKIT_PROCESS_GUARD_PID_FILE")
		if err := os.WriteFile(path, []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			t.Fatal(err)
		}
		if mode == "exiting-parent" {
			time.Sleep(100 * time.Millisecond)
			return
		}
		time.Sleep(time.Hour)
	default:
		t.Skip("process guard helper")
	}
}

func replaceEnv(environment []string, replacements ...string) []string {
	keys := make(map[string]struct{}, len(replacements))
	for _, replacement := range replacements {
		key, _, _ := strings.Cut(replacement, "=")
		keys[strings.ToUpper(key)] = struct{}{}
	}
	filtered := make([]string, 0, len(environment)+len(replacements))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := keys[strings.ToUpper(key)]; !replaced {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, replacements...)
}

func processAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer windows.CloseHandle(handle)
	var code uint32
	return windows.GetExitCodeProcess(handle, &code) == nil && code == 259
}
