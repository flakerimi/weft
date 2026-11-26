package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	// devPoll is how often weft dev looks for a changed .go file or .env.
	devPoll = 300 * time.Millisecond
	// devGrace is how long a child that stopped may wait for weft dev's own
	// stop: a signal sent to every process, as at a shutdown, may reach a
	// child before weft dev. A Ctrl-C at a terminal reaches weft dev alone,
	// since each child is in a process group of its own.
	devGrace = 500 * time.Millisecond
	// pageURLLine starts the line weft-dev prints once it listens; weft dev
	// prints its own URL line instead.
	pageURLLine = "weft: dev server at "
)

// devReadyWithin is how long the Go server and the page server have to
// answer once started; tests shorten it.
var devReadyWithin = 30 * time.Second

func devCmd(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), stopSignals...)
	defer stop()
	return dev(ctx, args, stdout, stderr)
}

// dev runs the app's Go server, built with the dev tag, and the page server
// (weft-dev), which proxies /api to it, until ctx is done: then it stops both
// and returns 0. When a .go file or the app's .env changes it rebuilds the Go
// server and, if the build succeeds, replaces the running one. A child that
// stops by itself stops the other and makes it return 1.
func dev(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft dev [--port n]."
	stdout, stderr = locked(stdout), locked(stderr)
	fl := flag.NewFlagSet("dev", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	port := fl.Int("port", 5173, "the port of the page server")
	if code, done := flagsFailed(fl.Parse(args), "dev", usage, stdout, stderr); done {
		return code
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "weft dev: %q is not a flag; %s\n", fl.Arg(0), usage)
		return 2
	}
	if *port < 0 || *port > 65535 {
		fmt.Fprintf(stderr, "weft dev: --port %d is not a port number; give one from 0 to 65535.\n", *port)
		return 2
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintln(stderr, "weft dev: "+errNoApp)
		return 1
	}
	// The page port is checked first, before pnpm install or any child, so
	// a port in use is one sentence and nothing is left running.
	if *port == 0 {
		if *port, err = freePort(); err != nil {
			return fail(stderr, "dev", err)
		}
	} else if !portFree(*port) {
		fmt.Fprintf(stderr, "weft dev: port %d is in use; stop what is using it or pass --port.\n", *port)
		return 1
	}
	cfg, err := ensure(appDir, stdout, stderr)
	if err != nil {
		if ctx.Err() != nil {
			return 0
		}
		return fail(stderr, "dev", err)
	}
	goPort, err := freePort()
	if err != nil {
		return fail(stderr, "dev", err)
	}
	// Each child's standard input is the read end of a pipe whose write end
	// weft dev holds, and never writes to, for its whole life. When weft dev
	// dies, however it dies, SIGKILL too, the kernel closes the write end and
	// each child reads the end of its input and exits: the Go server under
	// WEFT_DEV_PARENT=1, the page server under --exit-with-parent. An
	// *os.File as Stdin is handed to the child as it is, with no goroutine
	// copying into it.
	parentIn, parentOut, err := os.Pipe()
	if err != nil {
		return fail(stderr, "dev", fmt.Errorf("cannot make a pipe for the servers (%v); run weft dev again", err))
	}
	defer parentIn.Close()
	defer parentOut.Close()

	// say prints a sentence about watching once, however often it recurs.
	said := map[string]bool{}
	say := func(sentence string) {
		if !said[sentence] {
			said[sentence] = true
			fmt.Fprintln(stderr, sentence)
		}
	}
	// files returns the app's .go files, or false when it cannot look.
	files := func() (map[string]time.Time, bool) {
		now, err := goFiles(appDir, func(dir string) {
			say(fmt.Sprintf("weft dev: cannot read %s, so a change in it does not restart the Go server; make it readable if it holds Go code.", dir))
		})
		if err != nil {
			say(fmt.Sprintf("weft dev: cannot look for changed .go files (%v), so a save does not restart the Go server; fix that and run weft dev again.", err))
			return nil, false
		}
		return now, true
	}

	bin := devBinary(appDir)
	last, _ := files()
	if !goBuild(ctx, "dev", appDir, bin, stderr) {
		if ctx.Err() != nil {
			return 0
		}
		fmt.Fprintln(stderr, "weft dev: go build failed; fix what its output above says and run weft dev again.")
		return 1
	}
	// gen writes web/.weft-types/api.ts from the build just made. An app that
	// does not describe itself is said, and weft dev goes on with the file as
	// it was; the Go server has devReadyWithin to describe itself, as to
	// answer.
	gen := func() {
		_, err := regenerate(ctx, appDir, bin, devReadyWithin, stderr)
		switch {
		case err == nil, ctx.Err() != nil:
		case errors.Is(err, errNoDescription), errors.Is(err, errDescribeTimeout), errors.As(err, new(outputError)):
			if errors.As(err, new(outputError)) {
				fail(stderr, "dev", err)
			}
			fmt.Fprintln(stderr, "weft dev: the app did not describe itself, so web/.weft-types/api.ts is as it was; fix what its output above says.")
		default:
			fail(stderr, "dev", err)
		}
	}
	gen()
	if ctx.Err() != nil {
		return 0
	}

	goServer := func() (*child, error) {
		cmd := exec.Command(bin)
		cmd.Dir = appDir
		cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(goPort), "HOST=127.0.0.1", "WEFT_DEV_PARENT=1")
		cmd.Stdin = parentIn
		cmd.Stdout, cmd.Stderr = stdout, stderr
		return startChild("the Go server", cmd)
	}
	api, err := goServer()
	if err != nil {
		return fail(stderr, "dev", fmt.Errorf("cannot start %s (%v); run weft dev again", bin, err))
	}
	pageArgs := []string{"exec", "weft-dev", "--root", "web", "--port", strconv.Itoa(*port), "--api", "http://127.0.0.1:" + strconv.Itoa(goPort)}
	if cfg.Theme != "" {
		pageArgs = append(pageArgs, "--theme", cfg.Theme)
	}
	pageArgs = append(pageArgs, "--exit-with-parent")
	cmd := exec.Command("pnpm", pageArgs...)
	cmd.Dir = appDir
	// pnpm exec hands its standard input to node as it is.
	cmd.Stdin = parentIn
	pageOut := &dropLine{w: stdout, prefix: []byte(pageURLLine)}
	cmd.Stdout = pageOut
	cmd.Stderr = stderr
	cmd.WaitDelay = time.Second
	pages, err := startChild("the page server", cmd)
	if err != nil {
		api.stop()
		return fail(stderr, "dev", fmt.Errorf("cannot run pnpm exec weft-dev (%v); run pnpm install in %s and run weft dev again", err, appDir))
	}
	stopAll := func() {
		var wg sync.WaitGroup
		for _, c := range []*child{api, pages} {
			if c != nil {
				wg.Go(c.stop)
			}
		}
		wg.Wait()
	}
	// stopAsked stops both as asked. What the page server prints then is
	// left out, such as pnpm's line saying that node was stopped by a signal,
	// which it prints when the signal comes before node has run weft-dev.
	stopAsked := func() int {
		pageOut.quiet.Store(true)
		stopAll()
		return 0
	}
	// exited ends weft dev after c stopped by itself, unless it was a stop
	// weft dev is about to be asked for too.
	exited := func(c *child) int {
		select {
		case <-ctx.Done():
			return stopAsked()
		case <-time.After(devGrace):
		}
		stopAll()
		fmt.Fprintf(stderr, "weft dev: %s stopped; see its output above.\n", c.name)
		return 1
	}

	// Both answer before the URL is printed. The page server answers on
	// localhost, so it is asked on localhost, whichever address that is.
	pageURL := "http://localhost:" + strconv.Itoa(*port) + "/"
	apiURL := "http://127.0.0.1:" + strconv.Itoa(goPort) + "/"
	// Both, started just before, have devReadyWithin to answer.
	ready := time.NewTicker(50 * time.Millisecond)
	deadline := time.After(devReadyWithin)
	apiReady := false
	for {
		if !apiReady && answers(apiURL) {
			apiReady = true
		}
		if apiReady && answers(pageURL) {
			break
		}
		select {
		case <-ctx.Done():
			return stopAsked()
		case <-api.done:
			return exited(api)
		case <-pages.done:
			return exited(pages)
		case <-deadline:
			pageOut.quiet.Store(true)
			stopAll()
			if !apiReady {
				fmt.Fprintf(stderr, "weft dev: the Go server did not answer within %d s; check that main.go serves the app (warp's Serve reads PORT), and run weft dev again.\n", int(devReadyWithin.Seconds()))
			} else {
				fmt.Fprintf(stderr, "weft dev: the page server did not answer within %d s; see its output above.\n", int(devReadyWithin.Seconds()))
			}
			return 1
		case <-ready.C:
		}
	}
	ready.Stop()
	fmt.Fprintf(stdout, "weft dev: http://localhost:%d\n", *port)

	// After a restart a Go server that stops by itself, as one that panics at
	// start, is like a compile error: weft dev waits for the next save, with
	// api nil until then. Only the first one stopping ends weft dev.
	restarted := false
	apiDone := func() <-chan struct{} {
		if api == nil {
			return nil
		}
		return api.done
	}
	tick := time.NewTicker(devPoll)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return stopAsked()
		case <-apiDone():
			if !restarted {
				return exited(api)
			}
			select {
			case <-ctx.Done():
				return stopAsked()
			case <-time.After(devGrace):
			}
			api = nil
			fmt.Fprintln(stderr, "weft dev: the Go server stopped after the restart; fix it and save again.")
			continue
		case <-pages.done:
			return exited(pages)
		case <-tick.C:
		}
		now, ok := files()
		if !ok || sameFiles(now, last) {
			continue
		}
		last = now
		if !goBuild(ctx, "dev", appDir, bin, stderr) {
			if ctx.Err() == nil {
				fmt.Fprintln(stderr, "weft dev: go build failed; fix what its output above says, and until you save the fix the last build keeps serving /api.")
			}
			continue
		}
		gen()
		if ctx.Err() != nil {
			return stopAsked()
		}
		if api != nil {
			api.stop()
		}
		if api, err = goServer(); err != nil {
			pages.stop()
			return fail(stderr, "dev", fmt.Errorf("cannot start %s (%v); run weft dev again", bin, err))
		}
		restarted = true
	}
}

// child is a process weft dev started in a group of its own; done is closed
// once it has exited.
type child struct {
	name string
	cmd  *exec.Cmd
	done chan struct{}
}

func startChild(name string, cmd *exec.Cmd) (*child, error) {
	if err := startGroup(cmd); err != nil {
		return nil, err
	}
	c := &child{name: name, cmd: cmd, done: make(chan struct{})}
	go func() {
		cmd.Wait()
		close(c.done)
	}()
	return c, nil
}

// stop stops the child's group and waits for the child.
func (c *child) stop() {
	stopGroup(c.cmd)
	<-c.done
}

// goBuild builds the app with the dev tag into bin, passing the compiler's
// output through; it says whether the build succeeded. command names the
// weft command in its sentence.
func goBuild(ctx context.Context, command, appDir, bin string, stderr io.Writer) bool {
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		fmt.Fprintf(stderr, "weft %s: cannot make %s (%v); make the app's folder writable.\n", command, filepath.Dir(bin), err)
		return false
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "dev", "-o", bin, ".")
	cmd.Dir, cmd.Env = appDir, hostBuildEnv(os.Environ())
	cmd.Stdout, cmd.Stderr = stderr, stderr
	// Stopped as a Ctrl-C would, so go stops the compilers it started.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 3 * time.Second
	return cmd.Run() == nil
}

// hostBuildEnv is environ with GOOS and GOARCH set to this machine's, and
// every variable that picks a CPU level reset to its default, since the dev
// build runs here even when the environment asks weft build for another
// system.
func hostBuildEnv(environ []string) []string {
	return append(slices.Clone(environ), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "GOAMD64=", "GOARM=", "GOARM64=",
		"GO386=", "GOMIPS=", "GOMIPS64=", "GOPPC64=", "GORISCV64=", "GOWASM=")
}

// goFiles returns the modification time of every .go file in the app, and of
// its .env at the top, by path, outside the folders at its top that hold no
// Go of the app's: .weft, .git, node_modules and web. A folder it may not
// read, such as a database volume of another user, is skipped and passed to
// unreadable, if not nil.
func goFiles(appDir string, unreadable func(dir string)) (map[string]time.Time, error) {
	files := map[string]time.Time{}
	err := filepath.WalkDir(appDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// A file or folder removed while the walk ran is no error.
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if errors.Is(err, fs.ErrPermission) && d != nil && d.IsDir() && p != appDir {
				if unreadable != nil {
					unreadable(p)
				}
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			if filepath.Dir(p) == appDir {
				switch d.Name() {
				case ".weft", ".git", "node_modules", "web":
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") && !(d.Name() == ".env" && filepath.Dir(p) == appDir) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		files[p] = info.ModTime()
		return nil
	})
	return files, err
}

func sameFiles(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for p, t := range a {
		if u, ok := b[p]; !ok || !t.Equal(u) {
			return false
		}
	}
	return true
}

// portFree says whether port is free at localhost: whether it can be
// listened on at 127.0.0.1 and at ::1, as localhost may be either, and
// nothing answers there. The dial finds a server on the wildcard address,
// beside which macOS lets a socket bind 127.0.0.1 or ::1.
func portFree(port int) bool {
	for _, host := range []string{"127.0.0.1", "::1"} {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		l, err := net.Listen("tcp", addr)
		if err == nil {
			l.Close()
		} else if errors.Is(err, syscall.EADDRINUSE) {
			return false
		}
		if c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
			c.Close()
			return false
		}
	}
	return true
}

// freePort returns a port that is free at localhost.
func freePort() (int, error) {
	for range 20 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, fmt.Errorf("cannot listen on 127.0.0.1 (%v); check the machine's network settings", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		if portFree(port) {
			return port, nil
		}
	}
	return 0, errors.New("found no free port on localhost; stop some servers and run weft dev again")
}

var probe = &http.Client{Timeout: time.Second}

// answers says whether a server answers at url. OPTIONS renders no page:
// Vite answers it itself.
func answers(url string) bool {
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	if err != nil {
		return false
	}
	res, err := probe.Do(req)
	if err != nil {
		return false
	}
	res.Body.Close()
	return true
}

// dropLine writes every line to w but those that start with prefix, and
// nothing once quiet is set.
type dropLine struct {
	w      io.Writer
	prefix []byte
	buf    []byte
	quiet  atomic.Bool
}

func (d *dropLine) Write(p []byte) (int, error) {
	if d.quiet.Load() {
		return len(p), nil
	}
	d.buf = append(d.buf, p...)
	for {
		i := bytes.IndexByte(d.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		if line := d.buf[:i+1]; !bytes.HasPrefix(line, d.prefix) {
			if _, err := d.w.Write(line); err != nil {
				return len(p), err
			}
		}
		d.buf = d.buf[i+1:]
	}
}

// lockedWriter lets the children's output and weft dev's own share a writer.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// locked returns w safe for writes from several goroutines. A file is left as
// it is, so a child given it writes to it directly and sees a terminal.
func locked(w io.Writer) io.Writer {
	if _, ok := w.(*os.File); ok {
		return w
	}
	return &lockedWriter{w: w}
}
