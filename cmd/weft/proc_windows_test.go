//go:build windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// openTrackedProcess opens a SYNCHRONIZE|PROCESS_TERMINATE handle on pid while
// the process is known to be alive, pinning its process object so PID reuse
// cannot cause a false alive check or kill an unrelated process, and registers
// a t.Cleanup that terminates the process through the handle if it is still
// alive when the test ends and closes the handle.
func openTrackedProcess(t *testing.T, pid int) syscall.Handle {
	t.Helper()
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		t.Fatalf("OpenProcess(%d): %v", pid, err)
	}
	t.Cleanup(func() {
		if handleAlive(h) {
			syscall.TerminateProcess(h, 1)
		}
		syscall.CloseHandle(h)
	})
	return h
}

// handleAlive reports whether the process referred to by handle h is still
// running (WaitForSingleObject with timeout 0 returns WAIT_TIMEOUT).
func handleAlive(h syscall.Handle) bool {
	if h == 0 {
		return false
	}
	ev, err := syscall.WaitForSingleObject(h, 0)
	return err == nil && ev == syscall.WAIT_TIMEOUT
}

// waitHandlesGone polls until every process handle in hs is no longer running,
// for up to within.
func waitHandlesGone(t *testing.T, hs []syscall.Handle, within time.Duration) {
	t.Helper()
	for _, h := range hs {
		for deadline := time.Now().Add(within); handleAlive(h); time.Sleep(25 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Errorf("process handle %v is still alive after %s", h, within)
				break
			}
		}
	}
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

// openActiveChildHandles opens tracked process handles on the active *exec.Cmd
// processes currently in jobs while they are still running.
func openActiveChildHandles(t *testing.T) []syscall.Handle {
	t.Helper()
	mu.Lock()
	var pids []int
	for cmd := range jobs {
		if cmd.Process != nil {
			pids = append(pids, cmd.Process.Pid)
		}
	}
	mu.Unlock()
	hs := make([]syscall.Handle, 0, len(pids))
	for _, pid := range pids {
		hs = append(hs, openTrackedProcess(t, pid))
	}
	return hs
}

// findProcessByExe finds the PID of a running process whose executable base
// name equals exeName (e.g. "app.exe").
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

// terminateHandleHard terminates the process referred to by h immediately via
// TerminateProcess, as a hard kill does.
func terminateHandleHard(t *testing.T, h syscall.Handle) {
	t.Helper()
	if err := syscall.TerminateProcess(h, 1); err != nil {
		t.Fatalf("TerminateProcess(%v): %v", h, err)
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
	grandH := openTrackedProcess(t, grandPID)
	if !handleAlive(grandH) {
		c.stop()
		t.Fatalf("grandchild %d is not running before stop()", grandPID)
	}
	// Stopping the exited child unpins and closes its Job Object (with
	// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE) and waits for the job to empty.
	c.stop()
	waitHandlesGone(t, []syscall.Handle{grandH}, time.Second)
}

// Also test stopGroup while the child is still running after spawning a
// grandchild: c.stop() waits until the whole job is gone before returning.
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
	grandH := openTrackedProcess(t, grandPID)
	if !handleAlive(grandH) {
		c.stop()
		t.Fatalf("grandchild %d is not running before stop()", grandPID)
	}
	c.stop()
	waitHandlesGone(t, []syscall.Handle{grandH}, time.Second)
}

// When the parent process holding the Job Objects is killed hard via
// TerminateProcess, Windows closes its Job Object handles and terminates every
// child and grandchild in them within a second or two, even if the children
// ignore stdin EOF.
func TestJobObjectKillsTreeWhenHolderTerminatedHard(t *testing.T) {
	treeFile := filepath.Join(t.TempDir(), "tree.pids")
	holder := exec.Command(os.Args[0], "-test.short", "-test.run=^TestHelperJobHolder$")
	holder.Env = append(os.Environ(), "WEFT_TEST_HELPER=job-holder", "WEFT_TEST_PID_FILE="+treeFile)
	var holderOut syncBuffer
	holder.Stdout, holder.Stderr = &holderOut, &holderOut
	holder.WaitDelay = time.Second
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	holderH := openTrackedProcess(t, holder.Process.Pid)
	var holderErr error
	holderDone := make(chan struct{})
	go func() {
		holderErr = holder.Wait()
		close(holderDone)
	}()
	t.Cleanup(func() {
		select {
		case <-holderDone:
		default:
			holder.Process.Kill()
			select {
			case <-holderDone:
			case <-time.After(5 * time.Second):
				t.Errorf("holder (pid %d) did not exit within 5 s of Kill in cleanup", holder.Process.Pid)
			}
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
		select {
		case <-holderDone:
			t.Fatalf("holder exited early (%v) before writing pids: %s", holderErr, holderOut.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("holder did not write child and grandchild pids within 10 s (alive=%v): %s", handleAlive(holderH), holderOut.String())
		}
	}
	childH := openTrackedProcess(t, childPID)
	grandH := openTrackedProcess(t, grandPID)
	if !handleAlive(childH) || !handleAlive(grandH) {
		t.Fatalf("before TerminateProcess: child %d alive=%v, grandchild %d alive=%v", childPID, handleAlive(childH), grandPID, handleAlive(grandH))
	}

	terminateHandleHard(t, holderH)
	select {
	case <-holderDone:
	case <-time.After(5 * time.Second):
		t.Fatal("holder did not exit after TerminateProcess")
	}

	for deadline := time.Now().Add(5 * time.Second); handleAlive(childH) || handleAlive(grandH); time.Sleep(25 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("5 s after holder TerminateProcess: child %d alive=%v, grandchild %d alive=%v", childPID, handleAlive(childH), grandPID, handleAlive(grandH))
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
		fmt.Fprintf(os.Stderr, "grand.Start: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("WEFT_TEST_PID_FILE"), []byte(strconv.Itoa(grand.Process.Pid)+"\n"), 0o644); err != nil {
		grand.Process.Kill()
		fmt.Fprintf(os.Stderr, "WriteFile: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "grand.Start: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("WEFT_TEST_PID_FILE"), []byte(strconv.Itoa(grand.Process.Pid)+"\n"), 0o644); err != nil {
		grand.Process.Kill()
		fmt.Fprintf(os.Stderr, "WriteFile: %v\n", err)
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
	// Do not pass the holder's own os.Stdout/os.Stderr pipe handles to cmd:
	// when the test kills the holder with TerminateProcess, cmd and its
	// grandchild are still alive for a few milliseconds until Windows runs
	// down the holder's job handle, and if they inherited the holder's stdout
	// pipe they would keep go's pipe-reader goroutine in holder.Wait() open.
	if err := startGroup(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "startGroup: %v\n", err)
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
	fmt.Fprintf(os.Stderr, "timed out waiting for %s\n", subFile)
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
	hs := openActiveChildHandles(t)
	if len(hs) != 2 {
		t.Fatalf("want 2 active child jobs (Go server and page server), found %d", len(hs))
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
	if !refuses("localhost:" + strconv.Itoa(port)) {
		t.Errorf("localhost:%d still answers after stop", port)
	}
	for _, h := range hs {
		if handleAlive(h) {
			t.Errorf("child process handle %v is still alive", h)
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
	for deadline := time.Now().Add(30 * time.Second); len(activeChildJobs()) < 2; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the page server did not start: %s%s", out.String(), errb.String())
		}
	}
	hs := openActiveChildHandles(t)
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
	waitHandlesGone(t, hs, 2*time.Second)
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
	goH := openTrackedProcess(t, goPID)
	hs := openActiveChildHandles(t)
	terminateHandleHard(t, goH)
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
	waitHandlesGone(t, hs, 2*time.Second)
}

// When weft dev itself is killed hard via TerminateProcess on Windows, its
// Job Objects close and terminate both servers and any grandchild they spawned
// — including a grandchild that ignores stdin EOF and would otherwise outlive
// weft dev.
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
	app := devApp(t)
	// Have the app's main.go spawn a detached grandchild ( running the test
	// binary's TestHelperSleepForever, which ignores stdin EOF ) and write its
	// PID to grand.pid before serving. Without JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	// on the Go server's Job Object, this grandchild survives when weft.exe is
	// killed hard via TerminateProcess.
	grandPIDFile := filepath.Join(app, "grand.pid")
	mainPath := filepath.Join(app, "main.go")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	inject := fmt.Sprintf(`func main() {
	if os.Getenv("WARP_DESCRIBE") == "" {
		g := exec.Command(%q, "-test.short", "-test.run=^TestHelperSleepForever$")
		g.Env = append(os.Environ(), "WEFT_TEST_HELPER=sleep-forever")
		if err := g.Start(); err == nil {
			_ = os.WriteFile(%q, []byte(strconv.Itoa(g.Process.Pid)), 0o644)
		}
	}
`, os.Args[0], grandPIDFile)
	withImports := bytes.Replace(src, []byte("import (\n"), []byte("import (\n\t\"os\"\n\t\"os/exec\"\n\t\"strconv\"\n"), 1)
	patched := bytes.Replace(withImports, []byte("func main() {\n"), []byte(inject), 1)
	if bytes.Equal(patched, withImports) {
		t.Fatalf("main.go has no func main(): %s", src)
	}
	if err := os.WriteFile(mainPath, patched, 0o644); err != nil {
		t.Fatal(err)
	}

	port := freeLocalPort(t)
	out, errb := &syncBuffer{}, &syncBuffer{}
	cmd := exec.Command(bin, "dev", "--port", strconv.Itoa(port))
	cmd.Dir = app
	cmd.Stdout, cmd.Stderr = out, errb
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	weftH := openTrackedProcess(t, cmd.Process.Pid)
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			cmd.Process.Kill()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Errorf("weft.exe (pid %d) did not exit within 5 s of Kill in cleanup", cmd.Process.Pid)
			}
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
	goH := openTrackedProcess(t, goPID)
	rawGrand, err := os.ReadFile(grandPIDFile)
	if err != nil {
		t.Fatalf("read grand.pid: %v", err)
	}
	grandPID, err := strconv.Atoi(strings.TrimSpace(string(rawGrand)))
	if err != nil || grandPID <= 0 {
		t.Fatalf("bad grand.pid %q: %v", rawGrand, err)
	}
	grandH := openTrackedProcess(t, grandPID)
	if !handleAlive(grandH) {
		t.Fatalf("grandchild %d is not running before TerminateProcess", grandPID)
	}

	terminateHandleHard(t, weftH)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not die of TerminateProcess within 10 s")
	}

	addr := "localhost:" + strconv.Itoa(port)
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		c, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			c.Close()
		}
		if err != nil && !handleAlive(goH) && !handleAlive(grandH) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("5 s after weft dev was killed with TerminateProcess: addr err=%v, goPID(%d) alive=%v, grandPID(%d) alive=%v", err, goPID, handleAlive(goH), grandPID, handleAlive(grandH))
		}
	}
}
