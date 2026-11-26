//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// tryLock takes the lock at path, a file it makes if need be, without
// waiting: errLockHeld when a running process holds it. The lock is the
// kernel's (flock), which it lets go of when the process holding it ends,
// however it ends, so a file a killed weft command left is taken over at
// once, whichever process now has the id written in it. The file is removed
// while it is held and then let go of, so a command that opened the old file
// meanwhile sees it is not the one at path and opens that.
//
// On a file system that has no flock (some network ones), the lock is the
// file itself, as on a system without flock (lock_other.go): this command
// holds it when its exclusive create made it, and a file older than lockWait
// is taken over.
func tryLock(path string) (unlock func(), err error) {
	for {
		made := true
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			made = false
			if f, err = os.OpenFile(path, os.O_RDWR, 0); errors.Is(err, fs.ErrNotExist) {
				continue // let go of meanwhile
			}
		}
		if err != nil {
			return nil, err
		}
		lockOpened()
		err = flockFile(int(f.Fd()))
		switch {
		case errors.Is(err, syscall.EINTR):
			f.Close()
			continue
		case errors.Is(err, syscall.EWOULDBLOCK):
			f.Close()
			return nil, errLockHeld
		case errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP), errors.Is(err, syscall.ENOLCK):
			if !made {
				f.Close()
				if takeOverOld(path) {
					continue
				}
				return nil, errLockHeld
			}
		case err != nil:
			f.Close()
			return nil, err
		default:
			held, err1 := f.Stat()
			now, err2 := os.Stat(path)
			if err1 != nil || err2 != nil || !os.SameFile(held, now) {
				f.Close() // removed by the command that held it: open the new one
				continue
			}
		}
		// The process's id, for a person who finds the file.
		f.Truncate(0)
		fmt.Fprintf(f, "%d\n", os.Getpid())
		return func() {
			os.Remove(path)
			f.Close()
		}, nil
	}
}

// flockFile takes an exclusive flock on fd without waiting; a test puts a
// file system with no flock in its place.
var flockFile = func(fd int) error { return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) }

// lockOpened is called between opening the lock's file and locking it; a
// test lets another command finish there.
var lockOpened = func() {}
