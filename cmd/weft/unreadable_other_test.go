//go:build !windows

package main

import (
	"os"
	"testing"
)

// makeUnreadable makes the file at path unreadable to this process until the
// test ends: its mode is 0.
func makeUnreadable(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })
}
