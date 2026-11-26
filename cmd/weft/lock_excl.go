package main

import (
	"os"
	"time"
)

// takeOverOld removes the lock file at path when it is older than lockWait,
// longer than any unpack takes, so it was left by a command that did not
// finish, and says whether it did; for a lock that is the file itself, where
// there is no flock.
func takeOverOld(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return true // let go of meanwhile: try again
	}
	if time.Since(fi.ModTime()) <= lockWait {
		return false
	}
	os.Remove(path)
	return true
}
