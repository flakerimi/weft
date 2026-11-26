//go:build !unix && !windows

package main

import (
	"os"
	"os/exec"
)

// stopSignals are the signals weft dev stops its children on and exits 0.
var stopSignals = []os.Signal{os.Interrupt}

// startGroup starts cmd. Outside Unix and Windows (weft is released for
// neither) there are no process groups to make.
func startGroup(cmd *exec.Cmd) error {
	return cmd.Start()
}

// stopGroup kills cmd, but not what it started.
func stopGroup(cmd *exec.Cmd) {
	cmd.Process.Kill()
}

// interrupt kills cmd.
func interrupt(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func pin(*exec.Cmd)   {}
func unpin(*exec.Cmd) {}

// setAside does nothing.
func setAside(string) {}
