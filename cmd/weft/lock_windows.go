//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	fileDelete             = 0x00010000 // DELETE
	fileFlagDeleteOnClose  = 0x04000000 // FILE_FLAG_DELETE_ON_CLOSE
	errorSharingViolation  = syscall.Errno(32)
	errorLockViolationCode = syscall.Errno(33)
)

// tryLock takes the lock at path without waiting: errLockHeld when a running
// process holds it. The file is opened with no sharing, so no other process
// opens it while this one has it, and deleted when it is closed, which
// Windows does when the process ends, however it ends: a lock is never left
// behind.
func tryLock(path string) (unlock func(), err error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE|fileDelete, 0, nil,
		syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL|fileFlagDeleteOnClose, 0)
	switch {
	case errors.Is(err, errorSharingViolation), errors.Is(err, errorLockViolationCode), errors.Is(err, syscall.ERROR_ACCESS_DENIED) && fileThere(path):
		// Held, or being deleted by the command that held it.
		return nil, errLockHeld
	case err != nil:
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	fmt.Fprintf(f, "%d\n", os.Getpid())
	return func() { f.Close() }, nil
}

// fileThere says whether path names something.
func fileThere(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
