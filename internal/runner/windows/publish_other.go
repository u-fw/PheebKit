//go:build !windows

package windows

import "os"

func publishFile(from, to string) error {
	// A hard link atomically creates a name only when the destination is absent.
	// Scratch files are siblings of the output, so they share a filesystem.
	if err := os.Link(from, to); err != nil {
		return err
	}
	return os.Remove(from)
}
