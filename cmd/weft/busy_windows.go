//go:build windows

package main

import (
	"errors"
	"syscall"
)

// busy says whether err is Windows refusing a rename or a removal because a
// process has the file, or a file in the folder, open: access denied, a
// sharing violation or a lock violation.
var busy = func(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED) || errors.Is(err, errorSharingViolation) || errors.Is(err, errorLockViolationCode)
}
