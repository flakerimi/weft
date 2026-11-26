//go:build !unix

package main

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

// stopSignals are the signals weft dev stops its children on and exits 0.
var stopSignals = []os.Signal{os.Interrupt}

// startGroup starts cmd. Outside Unix there are no process groups to make;
// stopGroup finds what cmd started by its parent instead.
func startGroup(cmd *exec.Cmd) error {
	return cmd.Start()
}

// stopGroup kills cmd and, on Windows, what it started too, such as the node
// that pnpm exec runs, with taskkill /T, which every Windows has; cmd itself
// is killed even if taskkill fails.
func stopGroup(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
	cmd.Process.Kill()
}

// interrupt stops p. Windows has no interrupt to send another process, so p
// is killed.
func interrupt(p *os.Process) error {
	return p.Kill()
}

// setAside moves bin, a binary weft dev's Go server may be running from, to
// bin.old, so the next build can write bin: Windows lets a running binary be
// renamed, but not written over. The bin.old of the build before is no
// longer running by then, so it is replaced. A bin that is not there is
// left so.
func setAside(bin string) {
	os.Rename(bin, bin+".old")
}
