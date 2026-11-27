//go:build windows

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
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

// withoutSetupLink is a log without the line auth writes at its start while
// no user exists, Set up the owner at …/setup#token=….
func withoutSetupLink(log string) string {
	var kept []string
	for _, l := range strings.SplitAfter(log, "\n") {
		if !strings.Contains(l, "Set up the owner at ") {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "")
}

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

var (
	sharedOnce sync.Once
	sharedDir  string
)

// sharedDevApp runs the test in one copy of the installed app, made once for
// this test binary, for the tests that change none of its files.
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

// startDev runs weft dev with args in the current folder; it returns once
// weft dev prints its URL line, and stops it when the test ends.
func startDev(t *testing.T, app string, args ...string) *devRun {
	t.Helper()
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
	lineRe := regexp.MustCompile(`(?m)^weft dev: (http://localhost:\d+)\n`)
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if m := lineRe.FindStringSubmatch(r.stdout.String()); m != nil {
			r.url = m[1]
			return r
		}
		select {
		case code := <-r.done:
			r.done = nil
			t.Fatalf("weft dev exited %d before its URL line: %s%s", code, r.stdout, r.stderr)
		default:
		}
	}
	t.Fatalf("weft dev printed no URL line within 60 s: %s%s", r.stdout, r.stderr)
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

// processAlive reports whether a process with pid is currently running on
// Windows (OpenProcess succeeds and WaitForSingleObject with timeout 0
// returns WAIT_TIMEOUT).
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	ev, err := syscall.WaitForSingleObject(h, 0)
	return err == nil && ev == syscall.WAIT_TIMEOUT
}

// activeChildJobs returns a snapshot of the open Job Object handles currently
// tracked in jobs.
func activeChildJobs() []syscall.Handle {
	mu.Lock()
	defer mu.Unlock()
	out := make([]syscall.Handle, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, j)
	}
	return out
}

// activeChildPIDs returns the PIDs of the active *exec.Cmd processes currently
// tracked in jobs.
func activeChildPIDs() []int {
	mu.Lock()
	defer mu.Unlock()
	out := make([]int, 0, len(jobs))
	for cmd := range jobs {
		if cmd.Process != nil {
			out = append(out, cmd.Process.Pid)
		}
	}
	return out
}

// findProcessByExe finds the PID of a running process whose executable base
// name equals exeName and whose parent PID (or ancestor) is in jobs, or by
// matching exeName (e.g. "app.exe").
func findProcessByExe(t *testing.T, exeName string) int {
	t.Helper()
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		t.Fatalf("CreateToolhelp32Snapshot: %v", err)
	}
	defer syscall.CloseHandle(snap)
	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := syscall.Process32First(snap, &pe); err != nil {
		return 0
	}
	for {
		name := syscall.UTF16ToString(pe.ExeFile[:])
		if strings.EqualFold(name, exeName) {
			return int(pe.ProcessID)
		}
		pe.Size = uint32(unsafe.Sizeof(pe))
		if err := syscall.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return 0
}

// terminateProcessHard terminates pid immediately via TerminateProcess, as a
// hard kill does.
func terminateProcessHard(t *testing.T, pid int) {
	t.Helper()
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE|syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatalf("OpenProcess(%d): %v", pid, err)
	}
	defer syscall.CloseHandle(h)
	if err := syscall.TerminateProcess(h, 1); err != nil {
		t.Fatalf("TerminateProcess(%d): %v", pid, err)
	}
}

// A child started with startChild/startGroup that spawns a grandchild and
// exits immediately leaves the grandchild inside its Job Object: calling
// child.stop() (or stopGroup) terminates the grandchild even though the
// intermediate parent has already exited (which taskkill /T /PID missed).
func TestJobObjectStopsOrphanedGrandchild(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	cmd := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperSpawnGrandchildAndExit$")
	cmd.Env = append(os.Environ(), "WEFT_TEST_HELPER=spawn-grandchild", "WEFT_TEST_PID_FILE="+pidFile)
	c, err := startChild("spawner", cmd)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the child itself to exit after writing the grandchild's PID.
	select {
	case <-c.done:
	case <-time.After(10 * time.Second):
		c.stop()
		t.Fatal("spawner child did not exit within 10 s")
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		c.stop()
		t.Fatalf("read grandchild pid: %v", err)
	}
	grandPID, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || grandPID <= 0 {
		c.stop()
		t.Fatalf("bad grandchild pid %q: %v", raw, err)
	}
	t.Cleanup(func() {
		if processAlive(grandPID) {
			if h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(grandPID)); err == nil {
				syscall.TerminateProcess(h, 1)
				syscall.CloseHandle(h)
			}
		}
	})
	if !processAlive(grandPID) {
		c.stop()
		t.Fatalf("grandchild %d is not running before stop()", grandPID)
	}
	// Stopping the exited child unpins and closes its Job Object (with
	// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE), terminating the orphaned grandchild.
	c.stop()
	for deadline := time.Now().Add(3 * time.Second); processAlive(grandPID); time.Sleep(25 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d is still alive 3 s after child.stop()", grandPID)
		}
	}
}

// Also test stopGroup directly while the child is still running after spawning
// a grandchild.
func TestJobObjectStopGroupEndsTree(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	in, out, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	defer out.Close()
	cmd := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperSpawnGrandchildAndWait$")
	cmd.Env = append(os.Environ(), "WEFT_TEST_HELPER=spawn-grandchild-wait", "WEFT_TEST_PID_FILE="+pidFile)
	cmd.Stdin = in
	c, err := startChild("spawner-wait", cmd)
	if err != nil {
		t.Fatal(err)
	}
	var grandPID int
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(25 * time.Millisecond) {
		if raw, err := os.ReadFile(pidFile); err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && p > 0 {
				grandPID = p
				break
			}
		}
		if time.Now().After(deadline) {
			c.stop()
			t.Fatal("grandchild pid file was not written within 10 s")
		}
	}
	t.Cleanup(func() {
		if processAlive(grandPID) {
			if h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(grandPID)); err == nil {
				syscall.TerminateProcess(h, 1)
				syscall.CloseHandle(h)
			}
		}
	})
	if !processAlive(grandPID) {
		c.stop()
		t.Fatalf("grandchild %d is not running before stop()", grandPID)
	}
	c.stop()
	for deadline := time.Now().Add(3 * time.Second); processAlive(grandPID); time.Sleep(25 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d is still alive 3 s after child.stop()", grandPID)
		}
	}
}

// When the parent process holding the Job Objects is killed hard via
// TerminateProcess, Windows closes its Job Object handles and terminates every
// child and grandchild in them within a second or two, even if the children
// ignore stdin EOF.
func TestJobObjectKillsTreeWhenHolderTerminatedHard(t *testing.T) {
	treeFile := filepath.Join(t.TempDir(), "tree.pids")
	holder := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperJobHolder$")
	holder.Env = append(os.Environ(), "WEFT_TEST_HELPER=job-holder", "WEFT_TEST_PID_FILE="+treeFile)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	holderDone := make(chan error, 1)
	go func() { holderDone <- holder.Wait() }()
	t.Cleanup(func() {
		select {
		case <-holderDone:
		default:
			holder.Process.Kill()
			<-holderDone
		}
	})

	var childPID, grandPID int
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(25 * time.Millisecond) {
		if raw, err := os.ReadFile(treeFile); err == nil {
			fields := strings.Fields(string(raw))
			if len(fields) == 2 {
				c, err1 := strconv.Atoi(fields[0])
				g, err2 := strconv.Atoi(fields[1])
				if err1 == nil && err2 == nil && c > 0 && g > 0 {
					childPID, grandPID = c, g
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("holder did not write child and grandchild pids within 10 s")
		}
	}
	t.Cleanup(func() {
		for _, pid := range []int{childPID, grandPID} {
			if processAlive(pid) {
				if h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid)); err == nil {
					syscall.TerminateProcess(h, 1)
					syscall.CloseHandle(h)
				}
			}
		}
	})
	if !processAlive(childPID) || !processAlive(grandPID) {
		t.Fatalf("before TerminateProcess: child %d alive=%v, grandchild %d alive=%v", childPID, processAlive(childPID), grandPID, processAlive(grandPID))
	}

	terminateProcessHard(t, holder.Process.Pid)
	select {
	case <-holderDone:
	case <-time.After(5 * time.Second):
		t.Fatal("holder did not exit after TerminateProcess")
	}

	for deadline := time.Now().Add(2 * time.Second); processAlive(childPID) || processAlive(grandPID); time.Sleep(25 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("2 s after holder TerminateProcess: child %d alive=%v, grandchild %d alive=%v", childPID, processAlive(childPID), grandPID, processAlive(grandPID))
		}
	}
}

func TestHelperSpawnGrandchildAndExit(t *testing.T) {
	if os.Getenv("WEFT_TEST_HELPER") != "spawn-grandchild" {
		t.Skip("helper process")
	}
	grand := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperSleepForever$")
	grand.Env = append(os.Environ(), "WEFT_TEST_HELPER=sleep-forever")
	if err := grand.Start(); err != nil {
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("WEFT_TEST_PID_FILE"), []byte(strconv.Itoa(grand.Process.Pid)+"\n"), 0o644); err != nil {
		grand.Process.Kill()
		os.Exit(3)
	}
	os.Exit(0)
}

func TestHelperSpawnGrandchildAndWait(t *testing.T) {
	if os.Getenv("WEFT_TEST_HELPER") != "spawn-grandchild-wait" {
		t.Skip("helper process")
	}
	grand := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperSleepForever$")
	grand.Env = append(os.Environ(), "WEFT_TEST_HELPER=sleep-forever")
	if err := grand.Start(); err != nil {
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("WEFT_TEST_PID_FILE"), []byte(strconv.Itoa(grand.Process.Pid)+"\n"), 0o644); err != nil {
		grand.Process.Kill()
		os.Exit(3)
	}
	time.Sleep(time.Minute)
}

func TestHelperJobHolder(t *testing.T) {
	if os.Getenv("WEFT_TEST_HELPER") != "job-holder" {
		t.Skip("helper process")
	}
	subFile := os.Getenv("WEFT_TEST_PID_FILE") + ".grand"
	cmd := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperSpawnGrandchildAndWait$")
	cmd.Env = append(os.Environ(), "WEFT_TEST_HELPER=spawn-grandchild-wait", "WEFT_TEST_PID_FILE="+subFile)
	if err := startGroup(cmd); err != nil {
		os.Exit(2)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if raw, err := os.ReadFile(subFile); err == nil {
			grandPID := strings.TrimSpace(string(raw))
			if grandPID != "" {
				_ = os.WriteFile(os.Getenv("WEFT_TEST_PID_FILE"), []byte(fmt.Sprintf("%d %s\n", cmd.Process.Pid, grandPID)), 0o644)
				time.Sleep(time.Minute)
			}
		}
	}
	os.Exit(3)
}

func TestHelperSleepForever(t *testing.T) {
	if os.Getenv("WEFT_TEST_HELPER") != "sleep-forever" {
		t.Skip("helper process")
	}
	time.Sleep(time.Minute)
}

// Review Focus 3: stopping leaves no child process.
func TestDevStopsChildren(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	if code, body := get(t, r.url+"/api/hello"); code != 200 || strings.TrimSpace(body) != `{"message":"Hello from Go"}` {
		t.Fatalf("GET /api/hello = %d %q", code, body)
	}
	pids := activeChildPIDs()
	if len(pids) != 2 {
		t.Fatalf("want 2 active child jobs (Go server and page server), found %v", pids)
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
	if !refuses("localhost:" + strconv.Itoa(port)) {
		t.Errorf("localhost:%d still answers after stop", port)
	}
	for _, pid := range pids {
		if processAlive(pid) {
			t.Errorf("child process %d is still alive", pid)
		}
	}
	if left := activeChildJobs(); len(left) != 0 {
		t.Errorf("jobs are still open: %v", left)
	}
	if strings.Contains(r.stderr.String(), "stopped") {
		t.Errorf("a stop as asked was reported: %s", r.stderr)
	}
}

// A Ctrl-C while the page server starts is a clean stop too.
func TestDevStopsWhileStarting(t *testing.T) {
	_ = sharedDevApp(t)
	port := freeLocalPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	var out, errb syncBuffer
	var code int
	finished := make(chan struct{})
	go func() {
		code = dev(ctx, []string{"--port", strconv.Itoa(port)}, &out, &errb)
		close(finished)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("weft dev did not stop within 10 s of being cancelled")
		}
	})
	for deadline := time.Now().Add(30 * time.Second); len(activeChildPIDs()) < 2; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the page server did not start: %s%s", out.String(), errb.String())
		}
	}
	pids := activeChildPIDs()
	cancel()
	select {
	case <-finished:
		if code != 0 {
			t.Errorf("exit %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not stop within 10 s")
	}
	if got := out.String(); got != "" && !regexp.MustCompile(`^weft dev: http://localhost:\d+\n$`).MatchString(got) {
		t.Errorf("stdout %q", got)
	}
	if got := withoutSetupLink(errb.String()); got != "" {
		t.Errorf("stderr %q, want nothing", got)
	}
	for _, pid := range pids {
		for deadline := time.Now().Add(2 * time.Second); processAlive(pid); time.Sleep(25 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Errorf("child process %d is still alive", pid)
				break
			}
		}
	}
	if left := activeChildJobs(); len(left) != 0 {
		t.Errorf("jobs are still open: %v", left)
	}
}

// A child that stops when nobody asked stops the other, and weft dev fails.
func TestDevChildStopsByItself(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	goPID := findProcessByExe(t, "app.exe")
	if goPID == 0 {
		t.Fatal("found no Go server (app.exe)")
	}
	pids := activeChildPIDs()
	terminateProcessHard(t, goPID)
	select {
	case code := <-r.done:
		r.done = nil
		if code != 1 {
			t.Errorf("exit %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not stop within 10 s of its Go server")
	}
	if want := "weft dev: the Go server stopped; see its output above.\n"; !strings.HasSuffix(r.stderr.String(), want) {
		t.Errorf("stderr ends %q, want %q", r.stderr, want)
	}
	if !refuses("localhost:" + strconv.Itoa(port)) {
		t.Error("the page server still answers")
	}
	for _, pid := range pids {
		for deadline := time.Now().Add(2 * time.Second); processAlive(pid); time.Sleep(25 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Errorf("child process %d is still alive", pid)
				break
			}
		}
	}
}

// When weft dev itself is killed hard via TerminateProcess on Windows, its
// Job Objects close and both the Go server and the page server (pnpm and node)
// stop within a couple of seconds.
func TestDevStopsWhenKilled(t *testing.T) {
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	pkg, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "weft.exe")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = pkg
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v: %s", err, out)
	}
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	out, errb := &syncBuffer{}, &syncBuffer{}
	cmd := exec.Command(bin, "dev", "--port", strconv.Itoa(port))
	cmd.Dir = app
	cmd.Stdout, cmd.Stderr = out, errb
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			cmd.Process.Kill()
			<-done
		}
	})
	for deadline := time.Now().Add(60 * time.Second); !strings.Contains(out.String(), "weft dev: http://localhost:"); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("no URL line within 60 s: %s%s", out.String(), errb.String())
		}
	}
	url := "http://localhost:" + strconv.Itoa(port)
	if code, body := get(t, url+"/api/hello"); code != 200 || strings.TrimSpace(body) != `{"message":"Hello from Go"}` {
		t.Fatalf("GET /api/hello = %d %q", code, body)
	}
	goPID := findProcessByExe(t, "app.exe")
	if goPID == 0 {
		t.Fatal("found no Go server (app.exe)")
	}

	terminateProcessHard(t, cmd.Process.Pid)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not die of TerminateProcess within 10 s")
	}

	addr := "localhost:" + strconv.Itoa(port)
	for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		c, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			c.Close()
		}
		if err != nil && !processAlive(goPID) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("2 s after weft dev was killed with TerminateProcess: addr err=%v, goPID(%d) alive=%v", err, goPID, processAlive(goPID))
		}
	}
	_ = errors.New // keep errors import if needed
}
