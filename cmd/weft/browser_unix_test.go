//go:build unix

package main

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A browser script that exits while what it started is still running, as a
// node script does whose Chrome is still closing its helpers, leaves nothing:
// runInGroup returns once the whole group is gone, so no process is left
// writing into the test's folders when they are removed.
func TestRunInGroupStopsWhatIsLeft(t *testing.T) {
	cmd := exec.Command("sh", "-c", "(sleep 30 >/dev/null 2>&1 &); echo ok")
	out, err := runInGroup(cmd, time.Minute)
	if err != nil || strings.TrimSpace(out) != "ok" {
		t.Fatalf("runInGroup = %q, %v", out, err)
	}
	if !errors.Is(syscall.Kill(-cmd.Process.Pid, 0), syscall.ESRCH) {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the sleep the script started was left running")
	}
}

// One that does not finish in time is stopped, group and all, and says so.
func TestRunInGroupStopsOneThatTakesTooLong(t *testing.T) {
	cmd := exec.Command("sh", "-c", "echo started; sleep 30")
	start := time.Now()
	out, err := runInGroup(cmd, 200*time.Millisecond)
	if !errors.Is(err, errBrowserTimedOut) || !strings.Contains(out, "started") {
		t.Fatalf("runInGroup = %q, %v; want the output so far and errBrowserTimedOut", out, err)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("runInGroup took %s to stop it", took)
	}
	if !errors.Is(syscall.Kill(-cmd.Process.Pid, 0), syscall.ESRCH) {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the group was left running")
	}
}
