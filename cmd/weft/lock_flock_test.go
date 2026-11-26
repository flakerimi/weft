//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A command that opened the lock's file just as its holder let go of it and
// removed it, while a third command took the new one, does not hold the lock
// too: the file it locked is no longer the one at the path.
func TestLockFileRemovedMeanwhile(t *testing.T) {
	app := t.TempDir()
	first, err := lockWeft(app)
	if err != nil {
		t.Fatal(err)
	}
	var third func()
	done := false
	t.Cleanup(func() { lockOpened = func() {} })
	lockOpened = func() {
		if done {
			return
		}
		done = true
		first() // the holder lets go, removing the file this one opened
		if third, err = lockWeft(app); err != nil {
			t.Fatal(err)
		}
	}
	unlock, err := tryLock(filepath.Join(app, ".weft.lock"))
	if !errors.Is(err, errLockHeld) {
		t.Errorf("tryLock = %v, want errLockHeld while another command holds the new file", err)
		if unlock != nil {
			unlock()
		}
	}
	third()
}

// On a file system with no flock, the lock is the file itself, made with an
// exclusive create, as it was before flock: a second command waits for the
// first, and a file older than lockWait is taken over.
func TestLockWithoutFlock(t *testing.T) {
	app := t.TempDir()
	lock := filepath.Join(app, ".weft.lock")
	was, wait := flockFile, lockWait
	t.Cleanup(func() { flockFile, lockWait = was, wait })
	flockFile = func(int) error { return syscall.ENOTSUP }
	lockWait = 300 * time.Millisecond

	first, err := lockWeft(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tryLock(lock); !errors.Is(err, errLockHeld) {
		t.Fatalf("a second command: %v, want errLockHeld while the first holds it", err)
	}
	first()
	if exists(t, lock) {
		t.Fatal("the lock is left after unlock")
	}
	second, err := lockWeft(app)
	if err != nil {
		t.Fatalf("after the first let go: %v", err)
	}
	second()

	os.WriteFile(lock, []byte("12345\n"), 0o644)
	old := time.Now().Add(-time.Minute)
	os.Chtimes(lock, old, old)
	third, err := lockWeft(app)
	if err != nil {
		t.Fatalf("a lock older than lockWait: %v", err)
	}
	third()
}
