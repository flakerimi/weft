//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package main

import (
	"errors"
	"path/filepath"
	"testing"
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
