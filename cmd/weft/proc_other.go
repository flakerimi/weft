//go:build !unix

package main

import (
	"os"
	"os/exec"
)

// stopSignals are the signals weft dev stops its children on and exits 0.
var stopSignals = []os.Signal{os.Interrupt}

// startGroup starts cmd. Outside Unix there are no process groups to make:
// weft is released for darwin and linux only.
func startGroup(cmd *exec.Cmd) error {
	return cmd.Start()
}

// stopGroup kills cmd, but not what it started.
func stopGroup(cmd *exec.Cmd) {
	cmd.Process.Kill()
}
