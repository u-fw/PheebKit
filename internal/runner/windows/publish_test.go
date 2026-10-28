package windows

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pheebkit/internal/core"
)

func TestPublicationNeverReplacesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "scratch.mp4"), filepath.Join(dir, "output.mp4")
	if err := os.WriteFile(from, []byte("encoded media"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("other process output"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(from, to); err == nil {
		t.Fatal("publication overwrote an existing destination")
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "other process output" {
		t.Fatalf("destination was changed: %q %v", data, err)
	}
}

func TestScratchCleanupRejectsOutsideParent(t *testing.T) {
	parent := t.TempDir()
	other := t.TempDir()
	if err := cleanupScratch(other, parent); err == nil {
		t.Fatal("cleanup accepted an unrelated directory")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("cleanup affected unrelated directory: %v", err)
	}
}

func TestScratchCleanupFailureAfterPublicationIsAWarning(t *testing.T) {
	result := core.Result{Output: core.Source{URI: "C:/output.mp4"}}
	var resultErr error
	var messages []string
	applyScratchCleanupOutcome(&result, &resultErr, func(message string) {
		messages = append(messages, message)
	}, errors.New("file is busy"))
	if resultErr != nil {
		t.Fatalf("successful publication became failed: %v", resultErr)
	}
	if result.Verification == nil || len(result.Verification.Warnings) != 1 ||
		!strings.Contains(result.Verification.Warnings[0], "file is busy") {
		t.Fatalf("cleanup warning was not retained: %+v", result.Verification)
	}
	if len(messages) != 1 || messages[0] != result.Verification.Warnings[0] {
		t.Fatalf("visible cleanup warning missing: %#v", messages)
	}
}
