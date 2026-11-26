//go:build windows

package main

import (
	"os"
	"syscall"
	"testing"
)

// makeUnreadable makes the file at path unreadable to this process until the
// test ends. Windows has no mode bit that stops a read, so the file is held
// open with no sharing, and with the right to delete it, as .weft.lock is
// (tryLock): every other open, which does not share deleting, fails.
func makeUnreadable(t *testing.T, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|fileDelete, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.CloseHandle(h) })
	if b, err := os.ReadFile(path); err == nil {
		t.Fatalf("holding %s open with no sharing did not stop a read of it (%q)", path, b)
	}
}
