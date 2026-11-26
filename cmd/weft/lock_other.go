//go:build !(darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || windows)

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// tryLock takes the lock at path without waiting: errLockHeld when the file
// is there. On a system with neither flock nor Windows' sharing modes (weft
// is released for neither), a lock left by a command that did not finish is
// taken over once it is older than lockWait.
func tryLock(path string) (unlock func(), err error) {
	for {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if !takeOverOld(path) {
			return nil, errLockHeld
		}
	}
}
