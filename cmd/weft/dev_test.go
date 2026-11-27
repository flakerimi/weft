//go:build unix

package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

var apiFlagRe = regexp.MustCompile(`--api http://127\.0\.0\.1:(\d+)`)

// findGoPort is the Go server's port, from the --api flag weft dev gives the
// page server it started on pagePort, or "" while there is none.
func findGoPort(t *testing.T, pagePort int) string {
	t.Helper()
	out, err := exec.Command("ps", "-A", "-ww", "-o", "command=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "weft-dev --root web --port "+strconv.Itoa(pagePort)+" ") {
			if m := apiFlagRe.FindStringSubmatch(line); m != nil {
				return m[1]
			}
		}
	}
	return ""
}

// goPort is the Go server's port.
func (r *devRun) goPort(t *testing.T) string {
	t.Helper()
	port, _ := strconv.Atoi(r.url[strings.LastIndexByte(r.url, ':')+1:])
	p := findGoPort(t, port)
	if p == "" {
		t.Fatalf("found no page server on port %d", port)
	}
	return p
}

// otherAddress is an address of this machine that is not a loopback one, or
// "" when it has none.
func otherAddress(t *testing.T) string {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && !n.IP.IsLinkLocalUnicast() && n.IP.To4() != nil {
			return n.IP.String()
		}
	}
	return ""
}

// appGroups returns the commands, by process group, other than the test's
// own, of the processes whose command line names the app's folder (the Go
// server) or the page port (pnpm and node, which run in the app's folder and
// name it relatively).
func appGroups(t *testing.T, app string, port int) map[int][]string {
	t.Helper()
	names := []string{app, "weft-dev --root web --port " + strconv.Itoa(port) + " "}
	if real, err := filepath.EvalSymlinks(app); err == nil && real != app {
		names = append(names, real)
	}
	out, err := exec.Command("ps", "-A", "-ww", "-o", "pgid=,command=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	groups := map[int][]string{}
	own := syscall.Getpgrp()
	for _, line := range strings.Split(string(out), "\n") {
		pgidText, command, ok := strings.Cut(strings.TrimSpace(line), " ")
		pgid, err := strconv.Atoi(pgidText)
		if !ok || err != nil || pgid == own {
			continue
		}
		for _, n := range names {
			if strings.Contains(command, n) {
				groups[pgid] = append(groups[pgid], command)
				break
			}
		}
	}
	return groups
}

// groupAlive says whether any process of the group is left.
func groupAlive(pgid int) bool {
	return !errors.Is(syscall.Kill(-pgid, 0), syscall.ESRCH)
}

// eventually polls url until its body has want, for up to within.
func eventually(t *testing.T, url, want string, within time.Duration) {
	t.Helper()
	last := ""
	for deadline := time.Now().Add(within); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		res, err := http.Get(url)
		if err != nil {
			last = err.Error()
			continue
		}
		var b bytes.Buffer
		b.ReadFrom(res.Body)
		res.Body.Close()
		last = b.String()
		if strings.Contains(last, want) {
			return
		}
	}
	t.Fatalf("GET %s did not have %q within %s; last %q", url, want, within, last)
}

// setMessage rewrites the API's message in app/hello/routes.go.
func setMessage(t *testing.T, app, message string) {
	t.Helper()
	file := filepath.Join(app, "app", "hello", "routes.go")
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`Message: "[^"]*"`)
	if !re.Match(src) {
		t.Fatalf("app/hello/routes.go has no message: %s", src)
	}
	if err := os.WriteFile(file, re.ReplaceAll(src, []byte(`Message: "`+message+`"`)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The folder has a space and a non-ASCII letter (Review Focus 1).
func TestDevServesPagesAndAPI(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	if want := "weft dev: http://localhost:" + strconv.Itoa(port) + "\n"; r.stdout.String() != want {
		t.Errorf("stdout %q, want only %q", r.stdout, want)
	}
	check := func() {
		t.Helper()
		if code, body := get(t, r.url+"/"); code != 200 || !strings.Contains(body, ">Welcome</h1>") {
			t.Errorf("GET / = %d %q", code, body)
		}
		if code, body := get(t, r.url+"/api/hello"); code != 200 || strings.TrimSpace(body) != `{"message":"Hello from Go"}` {
			t.Errorf("GET /api/hello = %d %q", code, body)
		}
	}
	check()
	// Each server's standard input is a pipe weft dev holds open: it reads
	// it to see weft dev go, and must stay up while it is open.
	time.Sleep(3 * time.Second)
	check()
	// The Go server prints no URL of its own, and answers on 127.0.0.1 only.
	// The owner's setup link auth logs is the page server's, and is not
	// counted.
	if got := strings.Count(r.stdout.String()+withoutSetupLink(r.stderr.String()), "http://"); got != 1 {
		t.Errorf("the output has %d URLs: %q", got, r.stdout.String()+r.stderr.String())
	}
	goPort := r.goPort(t)
	if code, _ := get(t, "http://127.0.0.1:"+goPort+"/api/hello"); code != 200 {
		t.Errorf("the Go server does not answer on 127.0.0.1: %d", code)
	}
	if addr := otherAddress(t); addr == "" {
		t.Log("this machine has no address but loopback ones")
	} else if c, err := net.DialTimeout("tcp", net.JoinHostPort(addr, goPort), time.Second); err == nil {
		c.Close()
		t.Errorf("the Go server answers on %s, not only on 127.0.0.1", addr)
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// An unreadable folder in the app, such as a database volume, is said once
// and leaves restarts working.
func TestDevRestartsGo(t *testing.T) {
	app := devApp(t)
	// Not data/, where the app keeps its database.
	data := filepath.Join(app, "private")
	unreadableDir(t, data)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	eventually(t, r.url+"/api/hello", "Hello from Go", 5*time.Second)
	setMessage(t, app, "Hello again")
	eventually(t, r.url+"/api/hello", "Hello again", 10*time.Second)
	setMessage(t, app, "Hello a third time")
	eventually(t, r.url+"/api/hello", "Hello a third time", 10*time.Second)
	if n := strings.Count(r.stderr.String(), "weft dev: cannot read "+data); n != 1 {
		t.Errorf("the unreadable folder was named %d times, want once: %s", n, r.stderr)
	}
	if strings.Contains(r.stderr.String(), "stopped") {
		t.Errorf("a restart was reported as a stop: %s", r.stderr)
	}
}

// Review Focus 3: a compile error is shown and the next save recovers.
func TestDevSurvivesCompileError(t *testing.T) {
	app := devApp(t)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	file := filepath.Join(app, "app", "hello", "routes.go")
	src, _ := os.ReadFile(file)
	os.WriteFile(file, append(src, []byte("\nfunc broken( {\n")...), 0o644)
	for deadline := time.Now().Add(15 * time.Second); !strings.Contains(r.stderr.String(), "weft dev: go build failed"); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("no build failure within 15 s: %s", r.stderr)
		}
	}
	if !strings.Contains(r.stderr.String(), "routes.go:") {
		t.Errorf("the compiler's output was not shown: %s", r.stderr)
	}
	eventually(t, r.url+"/api/hello", "Hello from Go", 5*time.Second)
	os.WriteFile(file, src, 0o644)
	setMessage(t, app, "Hello again")
	eventually(t, r.url+"/api/hello", "Hello again", 10*time.Second)
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// A restarted Go server that stops at once, as one that panics at start, is
// like a compile error: the page server stays, and the next save recovers.
func TestDevSurvivesGoServerStopAfterRestart(t *testing.T) {
	app := devApp(t)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	file := filepath.Join(app, "main.go")
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	broken := bytes.Replace(src, []byte("func main() {\n"), []byte("func main() {\n\tpanic(\"boom\")\n"), 1)
	if bytes.Equal(broken, src) {
		t.Fatalf("main.go has no func main: %s", src)
	}
	os.WriteFile(file, broken, 0o644)
	want := "weft dev: the Go server stopped after the restart; fix it and save again.\n"
	for deadline := time.Now().Add(15 * time.Second); !strings.Contains(r.stderr.String(), want); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("no %q within 15 s: %s", want, r.stderr)
		}
	}
	if !strings.Contains(r.stderr.String(), "boom") {
		t.Errorf("the panic was not shown: %s", r.stderr)
	}
	if code, body := get(t, r.url+"/"); code != 200 || !strings.Contains(body, ">Welcome</h1>") {
		t.Errorf("the page server stopped answering: GET / = %d %q", code, body)
	}
	select {
	case code := <-r.done:
		r.done = nil
		t.Fatalf("weft dev exited %d: %s", code, r.stderr)
	default:
	}
	os.WriteFile(file, src, 0o644)
	setMessage(t, app, "Hello again")
	eventually(t, r.url+"/api/hello", "Hello again", 10*time.Second)
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// childGroups returns the process groups of pid's children.
func childGroups(t *testing.T, pid int) []int {
	t.Helper()
	out, err := exec.Command("ps", "-A", "-o", "ppid=,pgid=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	var groups []int
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == strconv.Itoa(pid) {
			pgid, _ := strconv.Atoi(f[1])
			groups = append(groups, pgid)
		}
	}
	return groups
}

// buildWeft builds weft from this package, for a test that runs weft dev as
// a process of its own: a signal to the test binary would stop the test.
func buildWeft(t *testing.T) string {
	t.Helper()
	pkg, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "weft")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = pkg
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v: %s", err, out)
	}
	return bin
}

// devProc is weft dev running as a process of its own; done is closed once
// it has exited, and err is then what Wait returned.
type devProc struct {
	cmd       *exec.Cmd
	out, errb *syncBuffer
	done      chan struct{}
	err       error
	groups    []int
}

// startDevProc runs bin dev --port port in app and returns once it prints its
// URL line, with the process groups of its two children. Whatever the test
// finds, nothing it started is left: a weft dev still running is sent
// SIGTERM, so it stops its children, and killed only if it does not exit,
// and then the groups it had are killed.
func startDevProc(t *testing.T, bin, app string, port int) *devProc {
	t.Helper()
	p := &devProc{out: &syncBuffer{}, errb: &syncBuffer{}, done: make(chan struct{})}
	p.cmd = exec.Command(bin, "dev", "--port", strconv.Itoa(port))
	p.cmd.Dir = app
	p.cmd.Stdout, p.cmd.Stderr = p.out, p.errb
	// Wait returns once weft dev has exited, even when a child it left holds
	// its output open.
	p.cmd.WaitDelay = time.Second
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		p.err = p.cmd.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		select {
		case <-p.done:
		default:
			p.cmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-p.done:
			case <-time.After(10 * time.Second):
				p.cmd.Process.Kill()
				<-p.done
			}
		}
		for _, pgid := range p.groups {
			if groupAlive(pgid) {
				syscall.Kill(-pgid, syscall.SIGKILL)
			}
		}
	})
	for deadline := time.Now().Add(60 * time.Second); !strings.Contains(p.out.String(), "weft dev: http://localhost:"); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("no URL line within 60 s: %s%s", p.out.String(), p.errb.String())
		}
	}
	p.groups = childGroups(t, p.cmd.Process.Pid)
	if len(p.groups) != 2 {
		t.Fatalf("want the groups of the Go server and the page server, found %v", p.groups)
	}
	return p
}

// Closing the terminal (SIGHUP) or Ctrl-\ (SIGQUIT) stops the children too.
func TestDevStopsOnHangUpAndQuit(t *testing.T) {
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	bin := buildWeft(t)
	app := sharedDevApp(t)
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGQUIT} {
		port := freeLocalPort(t)
		p := startDevProc(t, bin, app, port)
		goPort := findGoPort(t, port)
		if goPort == "" {
			t.Fatalf("%v: found no page server on port %d", sig, port)
		}
		p.cmd.Process.Signal(sig)
		select {
		case <-p.done:
			if p.err != nil {
				t.Errorf("%v: weft dev: %v: %s", sig, p.err, p.errb.String())
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%v: weft dev did not exit within 10 s", sig)
		}
		for _, addr := range []string{"localhost:" + strconv.Itoa(port), "127.0.0.1:" + goPort} {
			if !refuses(addr) {
				t.Errorf("%v: %s still answers", sig, addr)
			}
		}
		for _, pgid := range p.groups {
			if groupAlive(pgid) {
				t.Errorf("%v: process group %d is left", sig, pgid)
			}
		}
	}
}

// weft dev killed so that it cannot stop its children (SIGKILL, as a
// harness stops a command it ran in the background) leaves neither server
// running: each reads the end of the pipe weft dev held as its standard
// input, and exits.
func TestDevStopsWhenKilled(t *testing.T) {
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	bin := buildWeft(t)
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	p := startDevProc(t, bin, app, port)
	goPort := findGoPort(t, port)
	if goPort == "" {
		t.Fatalf("found no page server on port %d", port)
	}
	p.cmd.Process.Kill()
	select {
	case <-p.done:
	case <-time.After(10 * time.Second):
		t.Fatal("weft dev did not die of SIGKILL within 10 s")
	}
	addrs := []string{"localhost:" + strconv.Itoa(port), "127.0.0.1:" + goPort}
	left := func() []string {
		var what []string
		for _, addr := range addrs {
			if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
				c.Close()
				what = append(what, addr+" answers")
			}
		}
		for _, pgid := range p.groups {
			if groupAlive(pgid) {
				what = append(what, "group "+strconv.Itoa(pgid)+" is alive")
			}
		}
		return what
	}
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		what := left()
		if len(what) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("5 s after weft dev was killed: %v", what)
		}
	}
}

// Review Focus 3: stopping leaves no child process.
func TestDevStopsChildren(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	goPort := r.goPort(t)
	groups := appGroups(t, app, port)
	// The Go server alone; pnpm and the node it runs.
	if len(groups) != 2 {
		t.Fatalf("want the groups of the Go server and the page server, found %v", groups)
	}
	for pgid, commands := range groups {
		if n := len(commands); n != 1 && n != 2 {
			t.Errorf("group %d: %q", pgid, commands)
		}
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
	for _, addr := range []string{"localhost:" + strconv.Itoa(port), "127.0.0.1:" + goPort} {
		if !refuses(addr) {
			t.Errorf("%s still answers", addr)
		}
	}
	for pgid := range groups {
		if groupAlive(pgid) {
			t.Errorf("process group %d is left", pgid)
		}
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
	if strings.Contains(r.stderr.String(), "stopped") {
		t.Errorf("a stop as asked was reported: %s", r.stderr)
	}
}

// A signal sent to every process, as at a shutdown, reaches the children too,
// so a child may stop before weft dev hears of it: that is still a clean stop.
// (A Ctrl-C at a terminal reaches weft dev alone: each child is in a process
// group of its own.)
func TestDevChildStopsFirstOnCtrlC(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	groups := appGroups(t, app, port)
	if len(groups) != 2 {
		t.Fatalf("want the groups of the Go server and the page server, found %v", groups)
	}
	for pgid := range groups {
		syscall.Kill(-pgid, syscall.SIGINT)
	}
	// The Go server stops at once on SIGINT; weft dev hears of it after.
	for deadline := time.Now().Add(5 * time.Second); len(appGroups(t, app, port)) == len(groups); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("no child stopped on SIGINT")
		}
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
	if strings.Contains(r.stderr.String(), "stopped") {
		t.Errorf("a child stopped by the signal was reported as a stop: %s", r.stderr)
	}
}

// A Ctrl-C while the page server starts is a clean stop too.
func TestDevStopsWhileStarting(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	var out, errb syncBuffer
	var code int
	finished := make(chan struct{})
	go func() {
		code = dev(ctx, []string{"--port", strconv.Itoa(port)}, &out, &errb)
		close(finished)
	}()
	// On every path, a failure's too, weft dev is stopped and waited for, so
	// it stops its children.
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("weft dev did not stop within 10 s of being cancelled")
		}
	})
	// The page server starts right after the Go server and takes a second to
	// answer.
	for deadline := time.Now().Add(30 * time.Second); findGoPort(t, port) == ""; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the page server did not start: %s%s", out.String(), errb.String())
		}
	}
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
	// Nothing but the owner's setup link, which auth logs at its start.
	if got := withoutSetupLink(errb.String()); got != "" {
		t.Errorf("stderr %q, want nothing", got)
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
}

// A child that stops when nobody asked stops the other, and weft dev fails.
func TestDevChildStopsByItself(t *testing.T) {
	app := sharedDevApp(t)
	port := freeLocalPort(t)
	r := startDev(t, app, "--port", strconv.Itoa(port))
	goPort := r.goPort(t)
	killed := false
	for pgid, commands := range appGroups(t, app, port) {
		if strings.HasSuffix(commands[0], filepath.Join(".weft", "dev", "app")) {
			syscall.Kill(-pgid, syscall.SIGKILL)
			killed = true
		}
	}
	if !killed {
		t.Fatal("found no Go server")
	}
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
	if !refuses("localhost:"+strconv.Itoa(port)) || !refuses("127.0.0.1:"+goPort) {
		t.Error("a server still answers")
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
}

// devWithin runs weft dev, which is expected to fail, and stops it if it is
// still running after within, failing the test.
func devWithin(t *testing.T, within time.Duration, stdout, stderr *syncBuffer, args ...string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()
	code := dev(ctx, args, stdout, stderr)
	if ctx.Err() != nil {
		t.Errorf("weft dev was still running after %s: %s%s", within, stdout, stderr)
	}
	return code
}

// binTime is when weft dev last built the app's Go server, or zero.
func binTime(t *testing.T, app string) time.Time {
	t.Helper()
	info, err := os.Stat(filepath.Join(app, ".weft", "dev", "app"))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// Review Focus 3: a taken port is one sentence, and no child is started.
func TestDevPortTaken(t *testing.T) {
	app := sharedDevApp(t)
	// "" is the wildcard: on macOS a socket may bind 127.0.0.1 or ::1 beside it.
	for _, host := range []string{"127.0.0.1", "::1", ""} {
		l, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("%s: %v", host, err)
			continue
		}
		port := l.Addr().(*net.TCPAddr).Port
		var out, errb syncBuffer
		before := binTime(t, app)
		code := devWithin(t, 30*time.Second, &out, &errb, "--port", strconv.Itoa(port))
		l.Close()
		if code != 1 {
			t.Errorf("%s: exit %d", host, code)
		}
		if want := "weft dev: port " + strconv.Itoa(port) + " is in use; stop what is using it or pass --port.\n"; errb.String() != want || out.String() != "" {
			t.Errorf("%s: stdout %q, stderr %q, want only %q", host, out.String(), errb.String(), want)
		}
		if left := appGroups(t, app, port); len(left) != 0 {
			t.Errorf("%s: processes are left: %v", host, left)
		}
		if !binTime(t, app).Equal(before) {
			t.Errorf("%s: the Go server was built", host)
		}
	}
}

// The port is checked before anything else: in a fresh clone, before pnpm
// install.
func TestDevPortTakenBeforeInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("weft dev runs the app's servers")
	}
	inTempDir(t, "work")
	if code, _, errs := create(t, "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	t.Chdir("myapp")
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	var out, errb syncBuffer
	code := devWithin(t, 30*time.Second, &out, &errb, "--port", strconv.Itoa(port))
	if want := "weft dev: port " + strconv.Itoa(port) + " is in use; stop what is using it or pass --port.\n"; code != 1 || errb.String() != want || out.String() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, want only %q", code, out.String(), errb.String(), want)
	}
	if exists(t, "node_modules") {
		t.Error("pnpm install ran before the port was checked")
	}
}

// A Go server that never answers on PORT stops weft dev after the deadline.
func TestDevGoServerIgnoresPort(t *testing.T) {
	app := devApp(t)
	main := "package main\n\nimport \"net/http\"\n\nfunc main() { http.ListenAndServe(\"127.0.0.1:0\", nil) }\n"
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	was := devReadyWithin
	devReadyWithin = 2 * time.Second
	t.Cleanup(func() { devReadyWithin = was })
	port := freeLocalPort(t)
	var out, errb syncBuffer
	code := devWithin(t, 30*time.Second, &out, &errb, "--port", strconv.Itoa(port))
	want := "weft dev: the Go server did not answer within 2 s; check that main.go serves the app (warp's Serve reads PORT), and run weft dev again.\n"
	if code != 1 || !strings.HasSuffix(errb.String(), want) || out.String() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, want it to end %q", code, out.String(), errb.String(), want)
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
}

// A page server that never answers, here one whose Vite config never
// finishes loading, stops weft dev after the same deadline, in a sentence
// that points at its output.
func TestDevPageServerNeverAnswers(t *testing.T) {
	app := devApp(t)
	config := "await new Promise((done) => setTimeout(done, 60_000))\nexport default {}\n"
	if err := os.WriteFile(filepath.Join(app, "web", "vite.config.mjs"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	was := devReadyWithin
	devReadyWithin = 5 * time.Second
	t.Cleanup(func() { devReadyWithin = was })
	port := freeLocalPort(t)
	var out, errb syncBuffer
	code := devWithin(t, 30*time.Second, &out, &errb, "--port", strconv.Itoa(port))
	want := "weft dev: the page server did not answer within 5 s; see its output above.\n"
	if code != 1 || !strings.HasSuffix(errb.String(), want) || out.String() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, want it to end %q", code, out.String(), errb.String(), want)
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
}

func TestDevBuildFails(t *testing.T) {
	app := devApp(t)
	os.WriteFile(filepath.Join(app, "app", "hello", "broken.go"), []byte("package hello\n\nfunc broken( {\n"), 0o644)
	var out, errb syncBuffer
	port := freeLocalPort(t)
	code := devWithin(t, 30*time.Second, &out, &errb, "--port", strconv.Itoa(port))
	if code != 1 {
		t.Errorf("exit %d", code)
	}
	if want := "weft dev: go build failed; fix what its output above says and run weft dev again.\n"; !strings.HasSuffix(errb.String(), want) || !strings.Contains(errb.String(), "broken.go:") {
		t.Errorf("stderr %q, want the compiler's output, then %q", errb.String(), want)
	}
	if left := appGroups(t, app, port); len(left) != 0 {
		t.Errorf("processes are left: %v", left)
	}
}

func TestDevUsage(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"dev", "extra"}, {"dev", "--nope"}, {"dev", "--port", "eighty"}, {"dev", "--port", "70000"}} {
		code, _, errs := runCmd(args...)
		if code != 2 || !strings.HasPrefix(errs, "weft dev: ") || strings.Count(strings.TrimSpace(errs), "\n") != 0 {
			t.Errorf("%v: exit %d, stderr %q", args, code, errs)
		}
	}
	code, out, errs := runCmd("dev")
	if want := "weft dev: no Weft app here; run it in a folder made by weft create.\n"; code != 1 || errs != want || out != "" {
		t.Errorf("outside an app: exit %d, stderr %q, want %q", code, errs, want)
	}
}

// unreadableDir makes dir unreadable until the test ends, as a database
// volume of another user is.
func unreadableDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	// Restored, so the temp folder can be removed.
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("this user can read a folder with mode 000")
	}
}

func TestGoFilesSkipsFolders(t *testing.T) {
	dir := t.TempDir()
	write := func(p string) {
		t.Helper()
		p = filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{
		"main.go", "dist_dev.go", "app/modules.go", "app/hello/routes.go", "app/notes.txt",
		".env", ".env.sample", "app/.env", "web/.env", ".weft/.env",
		".weft/go/warp/serve.go", ".git/hooks/x.go", "node_modules/x/y.go", "web/pages/x.go",
		"db/x.go",
	} {
		write(p)
	}
	unreadableDir(t, filepath.Join(dir, "db"))
	var unreadable []string
	files, err := goFiles(dir, func(d string) { unreadable = append(unreadable, d) })
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadable) != 1 || unreadable[0] != filepath.Join(dir, "db") {
		t.Errorf("unreadable folders %q, want only db", unreadable)
	}
	var got []string
	for p := range files {
		rel, _ := filepath.Rel(dir, p)
		got = append(got, filepath.ToSlash(rel))
	}
	want := map[string]bool{"main.go": true, "dist_dev.go": true, "app/modules.go": true, "app/hello/routes.go": true, ".env": true}
	if len(got) != len(want) {
		t.Errorf("goFiles = %v, want %v", got, want)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("goFiles has %s", p)
		}
	}
	info, _ := os.Stat(filepath.Join(dir, "main.go"))
	if !files[filepath.Join(dir, "main.go")].Equal(info.ModTime()) {
		t.Error("goFiles does not keep the modification time")
	}
}

// goServerPID is the process id of the app's Go server, the binary weft dev
// built into app/.weft/dev/app, or 0 while there is none.
func goServerPID(t *testing.T, app string) int {
	t.Helper()
	out, err := exec.Command("ps", "-A", "-ww", "-o", "pid=,command=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	bin := filepath.Join(app, ".weft", "dev", "app")
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), f[0])) == bin {
			pid, _ := strconv.Atoi(f[0])
			return pid
		}
	}
	return 0
}

// processEnv is the environment of process pid, as one string.
func processEnv(t *testing.T, pid int) string {
	t.Helper()
	if b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ"); err == nil {
		return strings.ReplaceAll(string(b), "\x00", "\n")
	}
	out, err := exec.Command("ps", "-E", "-ww", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		t.Fatalf("ps -E: %v", err)
	}
	return strings.ReplaceAll(string(out), " ", "\n")
}

// A change to the app's .env restarts the Go server, as a change to a .go
// file does.
func TestDevRestartsOnEnvChange(t *testing.T) {
	app := devApp(t)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	before := goServerPID(t, app)
	if before == 0 {
		t.Fatal("found no Go server")
	}
	f, err := os.OpenFile(filepath.Join(app, ".env"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("# touched\n")
	f.Close()
	changed := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if pid := goServerPID(t, app); pid != 0 && pid != before {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatalf("the Go server (pid %d) was not restarted within 15 s of a .env change: %s", before, r.stderr)
	}
	eventually(t, r.url+"/api/hello", "Hello from Go", 10*time.Second)
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// The Go server weft dev runs answers on 127.0.0.1 only, and is given
// HOST=127.0.0.1 to say so.
func TestDevGoServerIsLocalOnly(t *testing.T) {
	app := sharedDevApp(t)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	goPort := r.goPort(t)
	if code, _ := get(t, "http://127.0.0.1:"+goPort+"/api/hello"); code != 200 {
		t.Errorf("the Go server does not answer on 127.0.0.1: %d", code)
	}
	if addr := otherAddress(t); addr == "" {
		t.Log("this machine has no address but loopback ones")
	} else if c, err := net.DialTimeout("tcp", net.JoinHostPort(addr, goPort), time.Second); err == nil {
		c.Close()
		t.Errorf("the Go server answers on %s, not only on 127.0.0.1", addr)
	}
	pid := goServerPID(t, app)
	if pid == 0 {
		t.Fatal("found no Go server")
	}
	if !slices.Contains(strings.Split(processEnv(t, pid), "\n"), "HOST=127.0.0.1") {
		t.Errorf("the Go server's environment has no HOST=127.0.0.1")
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// An app that hangs when asked to describe itself costs weft dev 5 s at each
// Go build, not the 30 s of weft gen; it says so, and goes on.
func TestDevDescribeLimit(t *testing.T) {
	if devDescribeWithin != 5*time.Second {
		t.Errorf("devDescribeWithin = %v, want 5s", devDescribeWithin)
	}
}

func TestDevAppThatHangsWhenDescribing(t *testing.T) {
	app := devApp(t)
	main := `package main

import (
	"net/http"
	"os"
	"time"

	"github.com/flakerimi/weft/warp"
)

func main() {
	_ = warp.New(warp.Config{})
	if os.Getenv("WARP_DESCRIBE") != "" {
		time.Sleep(time.Minute)
	}
	http.ListenAndServe(os.Getenv("HOST")+":"+os.Getenv("PORT"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("up")) }))
}
`
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	was := devDescribeWithin
	devDescribeWithin = time.Second
	t.Cleanup(func() { devDescribeWithin = was })
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	want := "weft dev: the app did not describe itself within 5 s, so web/.weft-types/api.ts is as it was; make sure main.go calls Serve before doing other work.\n"
	if !strings.Contains(r.stderr.String(), want) {
		t.Errorf("stderr %q, want it to hold %q", r.stderr, want)
	}
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// An app that makes no warp.App, as a 0.0.6 app is after weft upgrade's
// two-line change, is not run to describe itself under weft dev: it gets the
// client of an app with no routes, and nothing is said; over a client that
// has routes, the file is left and that is said once.
func TestDevAppWithoutWarpApp(t *testing.T) {
	app := devApp(t)
	main := `package main

import (
	"net/http"
	"os"
)

func main() {
	if os.Getenv("WARP_DESCRIBE") != "" {
		os.WriteFile("described.txt", nil, 0o644)
	}
	http.ListenAndServe(os.Getenv("HOST")+":"+os.Getenv("PORT"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("up")) }))
}
`
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	// weft create wrote the client of the app it made, which has routes.
	os.Remove(file)
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	src, err := os.ReadFile(file)
	if err != nil || string(src) != mustGenerate(t, noRoutes) {
		t.Errorf("api.ts is not the client of an app with no routes: %v\n%s", err, src)
	}
	if exists(t, filepath.Join(app, "described.txt")) {
		t.Error("the app was run to describe itself")
	}
	if code := r.stop(t); code != 0 || strings.Contains(r.stderr.String(), "describe") || strings.Contains(r.stderr.String(), "warp.App") {
		t.Errorf("exit %d: %s", code, r.stderr)
	}

	// Over a client with routes: left as it is, said once, through a rebuild.
	const routes = "export const api = { hello: {} }\n"
	os.WriteFile(file, []byte(routes), 0o644)
	r = startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	os.WriteFile(filepath.Join(app, "main.go"), []byte(main+"\n// saved again\n"), 0o644)
	want := "weft dev: the app has no warp.App, as no package it is built from calls warp.New, so web/.weft-types/api.ts, which has routes, was left as it was; if the app has routes, make sure its main package is built from one that calls warp.New, and if it has none, remove the file.\n"
	for deadline := time.Now().Add(10 * time.Second); !strings.Contains(r.stderr.String(), want) && time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
	}
	time.Sleep(3 * time.Second) // the rebuild after the save, which says nothing again
	if got, _ := os.ReadFile(file); string(got) != routes {
		t.Errorf("api.ts was changed: %q", got)
	}
	if code := r.stop(t); code != 0 || strings.Count(r.stderr.String(), want) != 1 {
		t.Errorf("exit %d, stderr %q, want %q once", code, r.stderr, want)
	}
}

// Under weft dev the pages are at the page server's address, so the Go
// server is told that one as APP_URL, unless weft dev was started with its own.
func TestGoServerEnv(t *testing.T) {
	base := []string{"A=1", "PORT=9", "APP_URL="}
	got := goServerEnv(base, 4000, 5173)
	for _, want := range []string{"PORT=4000", "HOST=127.0.0.1", "WEFT_DEV_PARENT=1", "APP_URL=http://localhost:5173"} {
		if !slices.Contains(got, want) {
			t.Errorf("env has no %q: %v", want, got)
		}
	}
	got = goServerEnv([]string{"APP_URL=https://x.example"}, 4000, 5173)
	if !slices.Contains(got, "APP_URL=https://x.example") || slices.Contains(got, "APP_URL=http://localhost:5173") {
		t.Errorf("a given APP_URL is not passed on: %v", got)
	}
}

func TestDevGoServerGetsAppURL(t *testing.T) {
	for _, given := range []string{"", "https://shop.example"} {
		app := devApp(t)
		t.Setenv("APP_URL", given)
		// Beside the settings the app's database and login need.
		env, _ := os.ReadFile(filepath.Join(app, ".env"))
		os.WriteFile(filepath.Join(app, ".env"), append(env, "APP_URL=http://from-dotenv\n"...), 0o600)
		r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
		want := "APP_URL=" + r.url
		if given != "" {
			want = "APP_URL=" + given
		}
		pid := goServerPID(t, app)
		if pid == 0 {
			t.Fatal("found no Go server")
		}
		if env := strings.Split(processEnv(t, pid), "\n"); !slices.Contains(env, want) {
			t.Errorf("the Go server's environment has no %q", want)
		}
		if code := r.stop(t); code != 0 {
			t.Errorf("exit %d: %s", code, r.stderr)
		}
	}
}

// weft dev writes web/.weft-types/api.ts before it starts the Go server, and
// again after each Go change that builds. One that does not describe itself
// is said, leaves the file as it was, and does not stop weft dev.
func TestDevRunsGenAfterGoChange(t *testing.T) {
	app := devApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	r := startDev(t, app, "--port", strconv.Itoa(freeLocalPort(t)))
	src, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(src), "    async hello(): Promise<Message> {\n") {
		t.Fatalf("weft dev wrote no client before it started: %v\n%s", err, src)
	}
	waitFor := func(want string) {
		t.Helper()
		for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(100 * time.Millisecond) {
			if b, _ := os.ReadFile(file); strings.Contains(string(b), want) {
				return
			}
			if time.Now().After(deadline) {
				b, _ := os.ReadFile(file)
				t.Fatalf("api.ts has no %q within 15 s: %s\n%s", want, r.stderr, b)
			}
		}
	}
	routes := filepath.Join(app, "app", "hello", "routes.go")
	goSrc, err := os.ReadFile(routes)
	if err != nil {
		t.Fatal(err)
	}
	withCount := strings.Replace(string(goSrc), "\tMessage string `json:\"message\"`\n", "\tMessage string `json:\"message\"`\n\tCount   int    `json:\"count\"`\n", 1)
	if withCount == string(goSrc) {
		t.Fatalf("app/hello/routes.go has no Message field: %s", goSrc)
	}
	os.WriteFile(routes, []byte(withCount), 0o644)
	waitFor("  count: number\n")

	// A registration mistake: the description fails, and so does the server.
	before, _ := os.ReadFile(file)
	os.WriteFile(routes, []byte(strings.Replace(withCount, `"/hello"`, `"hello"`, 1)), 0o644)
	want := "weft dev: the app did not describe itself, so web/.weft-types/api.ts is as it was; fix what its output above says.\n"
	for deadline := time.Now().Add(15 * time.Second); !strings.Contains(r.stderr.String(), want); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("no %q within 15 s: %s", want, r.stderr)
		}
	}
	if after, _ := os.ReadFile(file); !bytes.Equal(after, before) {
		t.Error("api.ts changed after a failed description")
	}
	os.WriteFile(routes, goSrc, 0o644)
	waitFor("export interface Message {\n  message: string\n}\n")
	eventually(t, r.url+"/api/hello", "Hello from Go", 10*time.Second)
	if code := r.stop(t); code != 0 {
		t.Errorf("exit %d: %s", code, r.stderr)
	}
}

// weft dev picks the Go server's port after the page server's and before the
// page server binds it, so the Go server's must never be the page server's:
// on a CI runner the system handed the same port out twice and the page
// server then found its own port in use.
func TestFreePortAvoidsWhatItIsTold(t *testing.T) {
	for range 200 {
		page, err := freePort("dev")
		if err != nil {
			t.Fatal(err)
		}
		goPort, err := freePort("dev", page)
		if err != nil {
			t.Fatal(err)
		}
		if goPort == page {
			t.Fatalf("freePort gave %d, the port it was told to avoid", page)
		}
	}
	// The check itself: a port the system hands out is skipped when it is to be avoided.
	saved := portFreeAt
	t.Cleanup(func() { portFreeAt = saved })
	var offered []int
	portFreeAt = func(port int) bool { offered = append(offered, port); return true }
	first, err := freePort("dev")
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		offered = offered[:0]
		if p, err := freePort("dev", first); err != nil || p == first || slices.Contains(offered, first) {
			t.Fatalf("freePort(\"dev\", %d) = %d, %v, having offered %v", first, p, err, offered)
		}
	}
}
