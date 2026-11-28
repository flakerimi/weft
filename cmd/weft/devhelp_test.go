package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is an output buffer that weft dev's children and the test share.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// devRun is a weft dev running in a goroutine.
type devRun struct {
	app            string
	url            string
	stdout, stderr *syncBuffer
	cancel         context.CancelFunc
	done           chan int
	code           int
}

// withoutRoutineLines is a log without the lines a created app writes in
// every run: the mail module's at its start (mail is printed, not sent …),
// auth's while no user exists (Set up the owner at …/setup#token=…), and the
// middleware's request log, one line per request (msg=request …).
func withoutRoutineLines(log string) string {
	var kept []string
	for _, l := range strings.SplitAfter(log, "\n") {
		if !strings.Contains(l, "Set up the owner at ") && !strings.Contains(l, "mail is printed, not sent;") && !strings.Contains(l, " msg=request ") {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "")
}

// freeLocalPort returns a port free on both 127.0.0.1 and ::1, since the page
// server listens on localhost, which may be either.
func freeLocalPort(t *testing.T) int {
	t.Helper()
	for range 20 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		if l6, err := net.Listen("tcp", net.JoinHostPort("::1", strconv.Itoa(port))); err == nil {
			l6.Close()
			return port
		} else if !addrInUse(err) {
			return port
		}
	}
	t.Fatal("no free port")
	return 0
}

// devApp copies the installed app into a new temp folder and runs the test in
// it, for a test that changes the app.
func devApp(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	return app
}

var (
	sharedOnce sync.Once
	sharedDir  string
)

// sharedDevApp runs the test in one copy of the installed app, made once for
// this test binary, for the tests that change none of its files: a copy
// takes seconds. Its folder has a space and a non-ASCII letter (Review
// Focus 1).
func sharedDevApp(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	inTempDir(t, "unused")
	sharedOnce.Do(func() {
		dir := filepath.Join(filepath.Dir(stagedBundle), "My Apps", "café")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		sharedDir = copyApp(t, dir)
	})
	if sharedDir == "" {
		t.Fatal("the shared app was not copied")
	}
	t.Chdir(sharedDir)
	return sharedDir
}

// logRunningProcesses logs active node, cmd, pnpm, weft and app processes on
// Windows before a dev test fails waiting for weft dev's URL line, so any hang
// shows which process is alive and with what command line.
func logRunningProcesses(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Process | Where-Object Name -Match 'node|cmd|pnpm|weft|app' | Format-Table ProcessId,ParentProcessId,Name,CommandLine -AutoSize | Out-String -Width 400")
	out, err := cmd.CombinedOutput()
	t.Logf("running processes (err=%v):\n%s", err, out)
}

// startDev runs weft dev with args in the current folder; it returns once
// weft dev prints its URL line, and stops it when the test ends.
func startDev(t *testing.T, app string, args ...string) *devRun {
	t.Helper()
	started := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	r := &devRun{app: app, stdout: &syncBuffer{}, stderr: &syncBuffer{}, cancel: cancel, done: make(chan int, 1)}
	done := r.done
	go func() { done <- dev(ctx, args, r.stdout, r.stderr) }()
	t.Cleanup(func() {
		if r.done == nil {
			return
		}
		cancel()
		select {
		case <-r.done:
		case <-time.After(10 * time.Second):
			t.Error("weft dev did not stop within 10 s of being cancelled")
		}
	})
	lineRe := regexp.MustCompile(`(?m)^weft dev: (http://localhost:\d+)\r?\n`)
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if m := lineRe.FindStringSubmatch(r.stdout.String()); m != nil {
			r.url = m[1]
			t.Logf("weft dev ready in %s", time.Since(started).Round(time.Millisecond))
			return r
		}
		select {
		case code := <-r.done:
			r.done = nil
			logRunningProcesses(t)
			t.Fatalf("weft dev exited %d after %s before its URL line: %s%s", code, time.Since(started).Round(time.Millisecond), r.stdout, r.stderr)
		default:
		}
	}
	logRunningProcesses(t)
	t.Fatalf("weft dev printed no URL line within 60 s (%s elapsed): %s%s", time.Since(started).Round(time.Millisecond), r.stdout, r.stderr)
	return nil
}

// stop cancels weft dev, as SIGINT does, and returns its exit code.
func (r *devRun) stop(t *testing.T) int {
	t.Helper()
	r.cancel()
	select {
	case r.code = <-r.done:
		r.done = nil
		return r.code
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not stop within 10 s")
		return -1
	}
}

// refuses says whether nothing answers on addr within 5 s.
func refuses(addr string) bool {
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return true
		}
		c.Close()
	}
	return false
}
