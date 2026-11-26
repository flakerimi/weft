//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

// stopSignals are the signals weft dev stops its children on and exits 0.
var stopSignals = []os.Signal{os.Interrupt}

// processQueryLimitedInformation is PROCESS_QUERY_LIMITED_INFORMATION.
const processQueryLimitedInformation = 0x1000

// pinned holds a handle on each process weft started and has not let go of.
// While a handle on a process is open, Windows gives its id to no other
// process, even once it has exited, so a taskkill by that id reaches only
// it and what it started. mu is held across each taskkill, so unpin cannot
// let go of the id meanwhile.
var (
	mu     sync.Mutex
	pinned = map[*exec.Cmd]syscall.Handle{}
)

// startGroup starts cmd and pins its process. There are no process groups:
// stopGroup finds what cmd started by its parent instead.
func startGroup(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	pin(cmd)
	return nil
}

// pin opens a handle on cmd's process, just started and not yet waited for,
// so its id is not given out again until unpin.
func pin(cmd *exec.Cmd) {
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|processQueryLimitedInformation, false, uint32(cmd.Process.Pid))
	if err != nil {
		return
	}
	mu.Lock()
	pinned[cmd] = h
	mu.Unlock()
}

// unpin closes the handle pin opened, if any.
func unpin(cmd *exec.Cmd) {
	mu.Lock()
	defer mu.Unlock()
	if h, ok := pinned[cmd]; ok {
		syscall.CloseHandle(h)
		delete(pinned, cmd)
	}
}

// killTree kills cmd and, while it is pinned, what it started, such as the
// node that pnpm exec runs or the compilers go build runs, with taskkill /T,
// which every Windows has. An unpinned id may be another program's by now,
// so then only cmd itself is killed, through its os.Process, which knows
// whether it was waited for.
func killTree(cmd *exec.Cmd) {
	mu.Lock()
	if _, ok := pinned[cmd]; ok {
		exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
	mu.Unlock()
	cmd.Process.Kill()
}

// stopGroup kills cmd and what it started.
func stopGroup(cmd *exec.Cmd) {
	killTree(cmd)
}

// interrupt kills cmd and what it started: Windows has no interrupt to send
// another process.
func interrupt(cmd *exec.Cmd) error {
	killTree(cmd)
	return nil
}

// setAside moves bin, a binary weft dev's Go server may be running from, to
// bin.old, so the next build can write bin: Windows lets a running binary be
// renamed, but not written over. The bin.old of the build before is no
// longer running by then, so it is replaced. A bin that is not there is
// left so.
func setAside(bin string) {
	os.Rename(bin, bin+".old")
}
