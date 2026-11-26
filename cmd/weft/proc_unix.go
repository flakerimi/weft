//go:build unix

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// stopSignals are the signals weft dev stops its children on and exits 0:
// Ctrl-C, SIGTERM, the terminal closing (SIGHUP) and Ctrl-\ (SIGQUIT). The
// children are in groups of their own, so these reach weft dev alone.
var stopSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

// startGroup starts cmd as the leader of a process group of its own, so that
// stopGroup reaches what it starts too, such as the node that pnpm exec runs,
// and a Ctrl-C at the terminal reaches weft dev alone, which stops it.
func startGroup(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return cmd.Start()
}

// stopGroup sends SIGTERM to cmd's process group and, if any of it is left
// after 3 s, SIGKILL. It returns once the group is gone, or a second after
// SIGKILL. Someone must be waiting on cmd, as the group lasts until cmd is
// reaped.
func stopGroup(cmd *exec.Cmd) {
	pgid := cmd.Process.Pid
	if syscall.Kill(-pgid, syscall.SIGTERM) != nil {
		return
	}
	if groupGone(pgid, 3*time.Second) {
		return
	}
	syscall.Kill(-pgid, syscall.SIGKILL)
	groupGone(pgid, time.Second)
}

// groupGone polls until no process of the group is left, for up to within.
func groupGone(pgid int, within time.Duration) bool {
	for deadline := time.Now().Add(within); ; time.Sleep(20 * time.Millisecond) {
		if errors.Is(syscall.Kill(-pgid, 0), syscall.ESRCH) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
	}
}

// running says whether a process with this id is running: signal 0 reaches
// it, or it is another user's.
func running(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
