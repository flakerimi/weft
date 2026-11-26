package main

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// A child that has exited and been waited for is not stopped: its process
// id may by then be another program's (Windows gives it out again at once),
// so weft dev sends nothing to it. A child that runs is stopped.
func TestChildStoppedOnlyWhileItRuns(t *testing.T) {
	var stopped []string
	prev := stopChild
	stopChild = func(cmd *exec.Cmd) {
		stopped = append(stopped, cmd.Args[0])
		prev(cmd)
	}
	t.Cleanup(func() { stopChild = prev })

	quick, err := startChild("quick", exec.Command("go", "version"))
	if err != nil {
		t.Fatal(err)
	}
	<-quick.done
	quick.stop()
	if len(stopped) != 0 {
		t.Errorf("a child that had exited was stopped: %v", stopped)
	}

	in, out, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	defer out.Close()
	long := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperReadsInput$")
	long.Env = append(os.Environ(), "WEFT_TEST_HELPER=read")
	long.Stdin = in
	running, err := startChild("running", long)
	if err != nil {
		t.Fatal(err)
	}
	stopDone := make(chan struct{})
	go func() { running.stop(); close(stopDone) }()
	select {
	case <-stopDone:
	case <-time.After(10 * time.Second):
		t.Fatal("stop did not return within 10 s")
	}
	if len(stopped) != 1 || stopped[0] != os.Args[0] {
		t.Errorf("stopped %v, want only the running child", stopped)
	}
}

// TestHelperReadsInput is the running child of TestChildStoppedOnlyWhileItRuns:
// it reads its input until it ends, or until it is stopped.
func TestHelperReadsInput(t *testing.T) {
	if os.Getenv("WEFT_TEST_HELPER") != "read" {
		t.Skip("the child TestChildStoppedOnlyWhileItRuns starts")
	}
	io.Copy(io.Discard, os.Stdin)
}
