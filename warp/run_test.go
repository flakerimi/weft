package warp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

// running is an app that Run is serving in the background.
type running struct {
	url    string
	cancel context.CancelFunc
	done   chan error
}

func listenLocal(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func startApp(t *testing.T, app *App, dist fs.FS) *running {
	t.Helper()
	ln := listenLocal(t)
	ctx, cancel := context.WithCancel(context.Background())
	r := &running{url: "http://" + ln.Addr().String(), cancel: cancel, done: make(chan error, 1)}
	go func() { r.done <- app.Run(ctx, ln, dist) }()
	t.Cleanup(cancel)
	return r
}

// stop ends Run's context and returns what Run returned.
func (r *running) stop(t *testing.T) error {
	t.Helper()
	r.cancel()
	select {
	case err := <-r.done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context ended")
		return nil
	}
}

func fetch(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func closed(addr string) bool {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return true
	}
	c.Close()
	return false
}

func TestRunLifecycle(t *testing.T) {
	var log []string
	app := New(Config{})
	app.Use(mod("a", &log), mod("b", &log), mod("c", &log))
	r := startApp(t, app, nil)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"register:a", "register:b", "register:c",
		"migrate:a", "migrate:b", "migrate:c",
		"start:a", "start:b", "start:c",
		"stop:c", "stop:b", "stop:a",
	}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v\nwant %v", log, want)
	}
}

func TestRunStartFailureStopsEarlier(t *testing.T) {
	var log []string
	b := mod("b", &log)
	b.startErr = errors.New("boom")
	app := New(Config{})
	app.Use(mod("a", &log), b, mod("c", &log))
	ln := listenLocal(t)
	err := app.Run(context.Background(), ln, nil)
	if want := `warp: module "b" could not start: boom`; errText(err) != want {
		t.Fatalf("Run = %q, want %q", errText(err), want)
	}
	// Every module migrated before any started, so every one is stopped.
	if want := []string{"start:a", "start:b", "stop:c", "stop:b", "stop:a"}; !reflect.DeepEqual(log[len(log)-5:], want) {
		t.Fatalf("log %v", log)
	}
	if !closed(ln.Addr().String()) {
		t.Fatal("the listener is still open")
	}
}

func TestRunMigrateFailure(t *testing.T) {
	var log []string
	b := mod("b", &log)
	b.migrateErr = errors.New("table exists")
	app := New(Config{})
	app.Use(mod("a", &log), b, mod("c", &log))
	err := app.Run(context.Background(), listenLocal(t), nil)
	if want := `warp: module "b" could not migrate: table exists`; errText(err) != want {
		t.Fatalf("Run = %q, want %q", errText(err), want)
	}
	if want := []string{"register:a", "register:b", "register:c", "migrate:a", "migrate:b", "stop:a"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v", log)
	}
}

func TestRunStopFailuresAreJoined(t *testing.T) {
	var log []string
	a, c := mod("a", &log), mod("c", &log)
	a.stopErr, c.stopErr = errors.New("a stuck"), errors.New("c stuck")
	app := New(Config{})
	app.Use(a, mod("b", &log), c)
	err := startApp(t, app, nil).stop(t)
	want := "warp: module \"c\" could not stop: c stuck\nwarp: module \"a\" could not stop: a stuck"
	if errText(err) != want {
		t.Fatalf("Run = %q, want %q", errText(err), want)
	}
	if want := []string{"stop:c", "stop:b", "stop:a"}; !reflect.DeepEqual(log[len(log)-3:], want) {
		t.Fatalf("log %v", log)
	}
}

// failingListener accepts one connection, then fails its next Accept once
// fail is closed.
type failingListener struct {
	net.Listener
	accepted int
	fail     chan struct{}
}

var errListenerBroke = errors.New("the listener broke")

func (l *failingListener) Accept() (net.Conn, error) {
	if l.accepted++; l.accepted > 1 {
		<-l.fail
		return nil, errListenerBroke
	}
	return l.Listener.Accept()
}

func TestRunServeFailureStopsModules(t *testing.T) {
	var log []string
	entered, release := make(chan struct{}), make(chan struct{})
	app := New(Config{})
	app.Use(mod("a", &log))
	app.Handle("GET /slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		note(&log, "request:done")
		io.WriteString(w, "done")
	}))
	ln := &failingListener{Listener: listenLocal(t), fail: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- app.Run(context.Background(), ln, nil) }()
	got := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/api/slow")
		if err != nil {
			got <- 0
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		got <- resp.StatusCode
	}()
	<-entered
	close(ln.fail)
	select {
	case err := <-done:
		t.Fatalf("Run returned %v with a request in flight", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if status := <-got; status != 200 {
		t.Fatalf("status %d", status)
	}
	err := <-done
	if want := "warp: the server stopped serving (the listener broke); start the app again."; errText(err) != want || !errors.Is(err, errListenerBroke) {
		t.Fatalf("Run = %q, want %q", errText(err), want)
	}
	if got := notes(&log); !reflect.DeepEqual(got[len(got)-2:], []string{"request:done", "stop:a"}) {
		t.Fatalf("log %v", got)
	}
}

func TestRunReturnsRegistrationMistakes(t *testing.T) {
	var log []string
	app := New(Config{})
	app.Use(mod("a", &log), mod("a", &log))
	ln := listenLocal(t)
	err := app.Run(context.Background(), ln, nil)
	if want := `warp: two modules are named "a"; give one another name.`; errText(err) != want {
		t.Fatalf("Run = %q, want %q", errText(err), want)
	}
	if want := []string{"register:a"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v", log)
	}
	if !closed(ln.Addr().String()) {
		t.Fatal("the listener is still open")
	}
}

func TestRunFinishesRequestInFlight(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	app := New(Config{})
	app.Handle("GET /slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		io.WriteString(w, "done")
	}))
	r := startApp(t, app, nil)
	type answer struct {
		status int
		body   string
		err    error
	}
	got := make(chan answer, 1)
	go func() {
		resp, err := http.Get(r.url + "/api/slow")
		if err != nil {
			got <- answer{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		got <- answer{resp.StatusCode, string(b), err}
	}()
	<-entered
	r.cancel()
	// Shutdown closes the listener first; wait for that, so the request is
	// still in flight once the server is stopping.
	for deadline := time.Now().Add(5 * time.Second); !closed(strings.TrimPrefix(r.url, "http://")); {
		if time.Now().After(deadline) {
			t.Fatal("the server did not start shutting down")
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case err := <-r.done:
		t.Fatalf("Run returned %v with a request in flight", err)
	default:
	}
	close(release)
	if a := <-got; a.err != nil || a.status != 200 || a.body != "done" {
		t.Fatalf("answer %+v", a)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestRunServesPagesAndAPI(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /ping", text("pong"))
	r := startApp(t, app, fstest.MapFS{"index.html": {Data: []byte("<p>page</p>")}})
	if status, body := fetch(t, r.url+"/"); status != 200 || body != "<p>page</p>" {
		t.Fatalf("/: %d %q", status, body)
	}
	if status, body := fetch(t, r.url+"/api/ping"); status != 200 || body != "pong" {
		t.Fatalf("/api/ping: %d %q", status, body)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

// noRedirects is a client that answers a redirect as it is.
var noRedirects = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// headerOn fetches url and returns its status and the header key.
func headerOn(t *testing.T, url, key string) (int, string) {
	t.Helper()
	resp, err := noRedirects.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode, resp.Header.Get(key)
}

// pagesAndFile is a page build of a single-page app: index.html and a file.
var pagesAndFile = fstest.MapFS{"index.html": {Data: []byte("<p>page</p>")}, "robots.txt": {Data: []byte("User-agent: *\n")}}

func TestWrapAllCoversPagesFilesAndAPI(t *testing.T) {
	app := New(Config{})
	app.WrapAll(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-All", "on")
			next.ServeHTTP(w, r)
		})
	})
	app.Handle("GET /x", text("x"))
	app.Handle("GET /files/{rest...}", text("files"))
	r := startApp(t, app, pagesAndFile)
	for path, status := range map[string]int{"/": 200, "/robots.txt": 200, "/somewhere": 200, "/api/x": 200, "/api/nothing": 404, "/api/a//x": 307, "/api/files": 307} {
		if got, h := headerOn(t, r.url+path, "X-All"); got != status || h != "on" {
			t.Errorf("%s answers %d with X-All %q, want %d with on", path, got, h, status)
		}
	}
	r.stop(t)
}

func TestWrapAllOnce(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	app := New(Config{})
	app.WrapAll(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			calls++
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	})
	app.Handle("GET /x", text("x"))
	r := startApp(t, app, pagesAndFile)
	for _, path := range []string{"/", "/api/x"} {
		mu.Lock()
		calls = 0
		mu.Unlock()
		fetch(t, r.url+path)
		mu.Lock()
		if calls != 1 {
			t.Errorf("one request to %s ran WrapAll %d times, want once", path, calls)
		}
		mu.Unlock()
	}
	r.stop(t)
}

func TestWrapStillAPIOnly(t *testing.T) {
	app := New(Config{})
	app.Wrap(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Api", "on")
			next.ServeHTTP(w, r)
		})
	})
	app.Handle("GET /x", text("x"))
	r := startApp(t, app, pagesAndFile)
	if _, h := headerOn(t, r.url+"/api/x", "X-Api"); h != "on" {
		t.Errorf("Wrap is not on the API")
	}
	for _, path := range []string{"/", "/robots.txt"} {
		if _, h := headerOn(t, r.url+path, "X-Api"); h != "" {
			t.Errorf("Wrap is on %s, a page", path)
		}
	}
	r.stop(t)
}

func TestRunWithoutDist(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /ping", text("pong"))
	r := startApp(t, app, nil)
	if status, body := fetch(t, r.url+"/api/ping"); status != 200 || body != "pong" {
		t.Fatalf("/api/ping: %d %q", status, body)
	}
	status, body := fetch(t, r.url+"/")
	want := decode(t, `{"error":{"code":"not_found","message":"Nothing is at /."}}`)
	if got := decode(t, body); status != 404 || !reflect.DeepEqual(got, want) {
		t.Fatalf("/: %d %v", status, got)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

// Run has passed its Err() check by then, so each mistake made after the
// app started is logged too.
func TestRegisteringAfterStartIsAMistake(t *testing.T) {
	var log []string
	logs := captureLog(t)
	app := New(Config{})
	r := startApp(t, app, nil)
	if status, _ := fetch(t, r.url+"/api/x"); status != 404 {
		t.Fatalf("status %d", status)
	}
	app.Use(mod("late", &log))
	app.Handle("GET /x", text("x"))
	app.Wrap(func(next http.Handler) http.Handler { return next })
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	mistakes := []string{
		"warp: Use was called after the app started; list every module before Serve or Start.",
		`warp: route "GET /x" was registered after the app started; register every route before Serve or Start.`,
		"warp: Wrap was called after the app started; wrap before Serve or Start.",
	}
	if want := strings.Join(mistakes, "\n"); errText(app.Err()) != want {
		t.Fatalf("Err() = %q\nwant %q", errText(app.Err()), want)
	}
	for _, m := range mistakes {
		if !strings.Contains(logs.String(), strconv.Quote(m)) {
			t.Errorf("the log lacks %s:\n%s", m, logs)
		}
	}
	if len(app.wraps) != 0 {
		t.Fatal("a Wrap after the start was kept")
	}
	if len(log) != 0 || len(app.Routes()) != 0 {
		t.Fatalf("log %v, routes %v", log, app.Routes())
	}
}

func TestServePortTaken(t *testing.T) {
	ln := listenLocal(t)
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	app := New(ConfigFrom(map[string]string{"HOST": "127.0.0.1", "PORT": port}))
	err := app.Serve(nil)
	if want := "warp: port " + port + " is taken; stop what uses it or set PORT in .env."; errText(err) != want {
		t.Fatalf("Serve = %q, want %q", errText(err), want)
	}
}

func TestUntilEOF(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := untilEOF(context.Background(), pr)
	defer cancel()
	pw.Write([]byte("still here"))
	select {
	case <-ctx.Done():
		t.Fatal("the context ended before standard input closed")
	case <-time.After(20 * time.Millisecond):
	}
	pw.Close()
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context did not end when standard input closed")
	}
}

// stopCheck is a module whose Stop, which runs after the drain, records
// whether stopped was closed by then.
type stopCheck struct {
	stopped <-chan struct{}
	early   chan bool
}

func (stopCheck) Name() string            { return "check" }
func (stopCheck) Register(app *App) error { return nil }
func (m stopCheck) Stop(ctx context.Context) error {
	select {
	case <-m.stopped:
		m.early <- true
	case <-time.After(2 * time.Second):
		m.early <- false
	}
	return nil
}

// Once the first signal ends Serve's context, Serve stops taking the signals,
// so a second Ctrl-C during the drain ends the process the default way. A
// test cannot send itself SIGINT, so notifyContext stands in for
// signal.NotifyContext, and the test checks that its stop runs as soon as the
// context ends, before the modules stop.
func TestServeStopsTakingSignalsOnceTold(t *testing.T) {
	signal, stopped := make(chan struct{}), make(chan struct{})
	var once sync.Once
	was := notifyContext
	notifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		go func() { <-signal; cancel() }()
		return ctx, func() { once.Do(func() { close(stopped) }); cancel() }
	}
	t.Cleanup(func() { notifyContext = was })
	captureLog(t)
	check := stopCheck{stopped: stopped, early: make(chan bool, 1)}
	app := New(Config{Host: "127.0.0.1", Port: "0"})
	app.Use(check)
	done := make(chan error, 1)
	go func() { done <- app.Serve(nil) }()
	close(signal)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after its context ended")
	}
	if !<-check.early {
		t.Fatal("the signals were still taken while the app stopped")
	}
}

func TestServerTimeouts(t *testing.T) {
	srv := newServer(http.NotFoundHandler())
	if srv.ReadHeaderTimeout != 10*time.Second || srv.ReadTimeout != 30*time.Second || srv.IdleTimeout != 120*time.Second || srv.WriteTimeout != 0 {
		t.Fatalf("ReadHeaderTimeout %v, ReadTimeout %v, IdleTimeout %v, WriteTimeout %v; want 10s, 30s, 2m0s and none",
			srv.ReadHeaderTimeout, srv.ReadTimeout, srv.IdleTimeout, srv.WriteTimeout)
	}
}

// swapTimeouts sets the server's read and idle timeouts until the test ends.
func swapTimeouts(t *testing.T, read, idle time.Duration) {
	r, i := readTimeout, idleTimeout
	readTimeout, idleTimeout = read, idle
	t.Cleanup(func() { readTimeout, idleTimeout = r, i })
}

// bodyApp serves POST /api/body, which reads the whole body and answers how
// many bytes it read; whole is sent true when a handler read all 100.
func bodyApp(t *testing.T, whole chan<- bool) *running {
	app := New(Config{})
	app.Handle("POST /body", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		whole <- err == nil && len(b) == 100
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		io.WriteString(w, strconv.Itoa(len(b)))
	}))
	return startApp(t, app, nil)
}

// postHead is the head of a POST of a 100-byte body to /api/body.
const postHead = "POST /api/body HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/octet-stream\r\nContent-Length: 100\r\n\r\n"

// closedWithin reads c until it is closed, and says whether that was within
// d.
func closedWithin(c net.Conn, d time.Duration) bool {
	begun := time.Now()
	c.SetReadDeadline(begun.Add(d + time.Second))
	io.Copy(io.Discard, c)
	return time.Since(begun) < d
}

func TestSlowBodyIsCut(t *testing.T) {
	swapTimeouts(t, 200*time.Millisecond, 120*time.Second)
	whole := make(chan bool, 1)
	r := bodyApp(t, whole)
	c, err := net.Dial("tcp", strings.TrimPrefix(r.url, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	io.WriteString(c, postHead+strings.Repeat("x", 10))
	if !closedWithin(c, time.Second) {
		t.Error("a body sent slowly kept its connection open past a second, with a read timeout of 200 ms")
	}
	select {
	case ok := <-whole:
		if ok {
			t.Error("the handler saw a whole body that was never sent")
		}
	case <-time.After(time.Second):
		t.Error("the handler was not ended by the read timeout")
	}
}

func TestQuickBodyIsServed(t *testing.T) {
	swapTimeouts(t, 200*time.Millisecond, 120*time.Second)
	whole := make(chan bool, 1)
	r := bodyApp(t, whole)
	resp, err := http.Post(r.url+"/api/body", "application/octet-stream", strings.NewReader(strings.Repeat("x", 100)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(b) != "100" || !<-whole {
		t.Errorf("a body sent at once is answered %d %q", resp.StatusCode, b)
	}
}

func TestIdleConnectionClosed(t *testing.T) {
	swapTimeouts(t, 30*time.Second, 200*time.Millisecond)
	app := New(Config{})
	app.Handle("GET /ping", text("pong"))
	r := startApp(t, app, nil)
	c, err := net.Dial("tcp", strings.TrimPrefix(r.url, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	io.WriteString(c, "GET /api/ping HTTP/1.1\r\nHost: localhost\r\n\r\n")
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.Close {
		t.Fatalf("the first request is answered %d with Close %v, want a kept-alive 200", resp.StatusCode, resp.Close)
	}
	begun := time.Now()
	c.SetReadDeadline(begun.Add(2 * time.Second))
	io.Copy(io.Discard, br)
	if took := time.Since(begun); took > time.Second {
		t.Errorf("an idle connection stayed open %v, with an idle timeout of 200 ms", took)
	}
}

func TestRunNeedsAListener(t *testing.T) {
	app := New(ConfigFrom(map[string]string{"PORT": "abc"}))
	if err := app.Run(context.Background(), nil, nil); errText(err) != "warp: Run needs a listener; call Serve, or pass one from net.Listen." {
		t.Fatalf("Run = %q", errText(err))
	}
}

func TestStartAndStop(t *testing.T) {
	ctx := context.Background()
	t.Run("order", func(t *testing.T) {
		var log []string
		app := New(Config{})
		app.Use(mod("a", &log), mod("b", &log), mod("c", &log))
		app.Handle("GET /ping", text("pong"))
		if err := app.Start(ctx); err != nil {
			t.Fatal(err)
		}
		if w := serve(t, app.Handler(), "GET", "/api/ping"); w.Body.String() != "pong" {
			t.Fatalf("body %q", w.Body.String())
		}
		if err := app.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		want := []string{
			"register:a", "register:b", "register:c",
			"migrate:a", "migrate:b", "migrate:c",
			"start:a", "start:b", "start:c",
			"stop:c", "stop:b", "stop:a",
		}
		if !reflect.DeepEqual(log, want) {
			t.Fatalf("log %v\nwant %v", log, want)
		}
		if err := app.Stop(ctx); err != nil || len(log) != len(want) {
			t.Fatalf("a second Stop: %v, log %v", err, log)
		}
		if err := app.Start(ctx); errText(err) != "warp: the app was already started." {
			t.Fatalf("a second Start = %q", errText(err))
		}
	})
	t.Run("a failing start stops every module that migrated", func(t *testing.T) {
		var log []string
		b := mod("b", &log)
		b.startErr = errors.New("boom")
		app := New(Config{})
		app.Use(mod("a", &log), b, mod("c", &log))
		if err := app.Start(ctx); errText(err) != `warp: module "b" could not start: boom` {
			t.Fatalf("Start = %q", errText(err))
		}
		if want := []string{"start:a", "start:b", "stop:c", "stop:b", "stop:a"}; !reflect.DeepEqual(log[len(log)-5:], want) {
			t.Fatalf("log %v", log)
		}
		n := len(log)
		if err := app.Stop(ctx); err != nil || len(log) != n {
			t.Fatalf("Stop after a failed Start: %v, log %v", err, log)
		}
	})
	t.Run("a failing migrate starts nothing and stops what migrated", func(t *testing.T) {
		var log []string
		b := mod("b", &log)
		b.migrateErr = errors.New("table exists")
		app := New(Config{})
		app.Use(mod("a", &log), b)
		if err := app.Start(ctx); errText(err) != `warp: module "b" could not migrate: table exists` {
			t.Fatalf("Start = %q", errText(err))
		}
		if err := app.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		if want := []string{"register:a", "register:b", "migrate:a", "migrate:b", "stop:a"}; !reflect.DeepEqual(log, want) {
			t.Fatalf("log %v", log)
		}
	})
	t.Run("a module that only migrates is stopped when a later one fails", func(t *testing.T) {
		for _, fail := range []string{"migrate", "start"} {
			var log []string
			db := &migrateOnly{name: "db", log: &log}
			later := mod("later", &log)
			if fail == "migrate" {
				later.migrateErr = errors.New("boom")
			} else {
				later.startErr = errors.New("boom")
			}
			plain := &stopOnly{name: "plain", log: &log} // neither migrates nor started: left alone
			app := New(Config{})
			app.Use(db, plain, later)
			if err := app.Start(ctx); err == nil {
				t.Fatalf("%s: Start succeeded", fail)
			}
			want := []string{"migrate:db", "migrate:later", "stop:db"}
			if fail == "start" {
				want = []string{"migrate:db", "migrate:later", "start:later", "stop:later", "stop:db"}
			}
			if got := notes(&log)[1:]; !reflect.DeepEqual(got, want) { // after register:later
				t.Fatalf("%s fails: log %v, want %v", fail, got, want)
			}
		}
	})
	t.Run("stop failures are joined", func(t *testing.T) {
		var log []string
		a, c := mod("a", &log), mod("c", &log)
		a.stopErr, c.stopErr = errors.New("a stuck"), errors.New("c stuck")
		app := New(Config{})
		app.Use(a, mod("b", &log), c)
		if err := app.Start(ctx); err != nil {
			t.Fatal(err)
		}
		want := "warp: module \"c\" could not stop: c stuck\nwarp: module \"a\" could not stop: a stuck"
		if err := app.Stop(ctx); errText(err) != want || !errors.Is(err, a.stopErr) {
			t.Fatalf("Stop = %q, want %q", errText(err), want)
		}
	})
	t.Run("registration mistakes come first", func(t *testing.T) {
		var log []string
		app := New(Config{})
		app.Use(mod("a", &log), mod("a", &log))
		if err := app.Start(ctx); errText(err) != `warp: two modules are named "a"; give one another name.` {
			t.Fatalf("Start = %q", errText(err))
		}
		if want := []string{"register:a"}; !reflect.DeepEqual(log, want) {
			t.Fatalf("log %v", log)
		}
	})
	t.Run("Run after Start", func(t *testing.T) {
		app := New(Config{})
		if err := app.Start(ctx); err != nil {
			t.Fatal(err)
		}
		ln := listenLocal(t)
		if err := app.Run(ctx, ln, nil); errText(err) != "warp: the app was already started." {
			t.Fatalf("Run = %q", errText(err))
		}
		if !closed(ln.Addr().String()) {
			t.Fatal("the listener is still open")
		}
	})
	t.Run("Start after Run", func(t *testing.T) {
		var log []string
		app := New(Config{})
		app.Use(mod("a", &log))
		app.Handle("GET /ping", text("pong"))
		r := startApp(t, app, nil)
		if status, _ := fetch(t, r.url+"/api/ping"); status != 200 {
			t.Fatalf("status %d", status)
		}
		if err := app.Start(ctx); errText(err) != "warp: the app was already started." {
			t.Fatalf("Start = %q", errText(err))
		}
		if err := r.stop(t); err != nil {
			t.Fatal(err)
		}
		if want := []string{"register:a", "migrate:a", "start:a", "stop:a"}; !reflect.DeepEqual(notes(&log), want) {
			t.Fatalf("log %v", notes(&log))
		}
	})
}

// migrateOnly migrates and stops, and has nothing to start, as data's module
// opens its database in Migrate.
type migrateOnly struct {
	name string
	log  *[]string
}

func (m *migrateOnly) Name() string                  { return m.name }
func (m *migrateOnly) Register(*App) error           { return nil }
func (m *migrateOnly) Migrate(context.Context) error { note(m.log, "migrate:"+m.name); return nil }
func (m *migrateOnly) Stop(context.Context) error    { note(m.log, "stop:"+m.name); return nil }

// stopOnly has only a Stop, which no failed Start reaches.
type stopOnly struct {
	name string
	log  *[]string
}

func (m *stopOnly) Name() string               { return m.name }
func (m *stopOnly) Register(*App) error        { return nil }
func (m *stopOnly) Stop(context.Context) error { note(m.log, "stop:"+m.name); return nil }

// Serve listens once the app has started: nothing answers on the port while
// the modules migrate, and a start that fails never says it is listening.
func TestServeListensOnceStarted(t *testing.T) {
	logs := captureLog(t)
	ln := listenLocal(t)
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	var log []string
	m := mod("probe", &log)
	answered := make(chan bool, 1)
	m.migrateErr = errors.New("stop here")
	m.register = func(app *App) {}
	probe := &dialAtMigrate{testMod: m, addr: net.JoinHostPort("127.0.0.1", port), answered: answered}
	app := New(ConfigFrom(map[string]string{"HOST": "127.0.0.1", "PORT": port}))
	app.Use(probe)
	err := app.Serve(nil)
	if err == nil || !strings.Contains(err.Error(), "stop here") {
		t.Fatalf("Serve = %v", err)
	}
	if <-answered {
		t.Error("the port answered while the modules migrated")
	}
	if strings.Contains(logs.String(), "listening on") {
		t.Errorf("a start that failed said it was listening: %s", logs)
	}
}

// dialAtMigrate dials addr as it migrates, and says whether anything answered.
type dialAtMigrate struct {
	*testMod
	addr     string
	answered chan bool
}

func (m *dialAtMigrate) Migrate(ctx context.Context) error {
	c, err := net.DialTimeout("tcp", m.addr, 200*time.Millisecond)
	if err == nil {
		c.Close()
	}
	m.answered <- err == nil
	return m.testMod.Migrate(ctx)
}

// A module whose Stop ignores its context does not hold the app past its
// time: Stop says which one it left, and the modules before it still stop.
func TestStopGoesOnPastAModuleThatDoesNotStop(t *testing.T) {
	var log []string
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	app := New(Config{})
	app.Use(mod("first", &log), stuckStop{release})
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	began := time.Now()
	err := app.Stop(ctx)
	if took := time.Since(began); took > 3*time.Second {
		t.Fatalf("Stop took %v", took)
	}
	want := `warp: module "stuck" did not stop before the app's time to stop ran out, so the app went on without it; make its Stop return when its context ends.`
	if errText(err) != want {
		t.Errorf("Stop = %q, want %q", errText(err), want)
	}
	if !slices.Contains(notes(&log), "stop:first") {
		t.Errorf("steps %v: the module before it was not stopped", notes(&log))
	}
}

type stuckStop struct{ release chan struct{} }

func (stuckStop) Name() string            { return "stuck" }
func (stuckStop) Register(app *App) error { return nil }
func (m stuckStop) Stop(context.Context) error {
	<-m.release
	return nil
}

// waitsInMigrate is a module whose Migrate waits for its context to end, as
// opening a database does when the app is stopped meanwhile, and says so.
type waitsInMigrate struct{ migrating chan struct{} }

func (waitsInMigrate) Name() string        { return "slow" }
func (waitsInMigrate) Register(*App) error { return nil }
func (m waitsInMigrate) Migrate(ctx context.Context) error {
	close(m.migrating)
	<-ctx.Done()
	return errors.Join(errors.New("the database cannot be opened"), ctx.Err())
}

// Asked to stop while its modules start, as weft dev's Go server is when
// weft dev goes at once, Serve returns nil: the start did not fail, it was
// stopped.
func TestServeStoppedWhileStarting(t *testing.T) {
	signal := make(chan struct{})
	was := notifyContext
	notifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		go func() { <-signal; cancel() }()
		return ctx, cancel
	}
	t.Cleanup(func() { notifyContext = was })
	captureLog(t)
	m := waitsInMigrate{migrating: make(chan struct{})}
	app := New(Config{Host: "127.0.0.1", Port: "0"})
	app.Use(m)
	done := make(chan error, 1)
	go func() { done <- app.Serve(nil) }()
	<-m.migrating
	close(signal)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after its context ended")
	}
}

// Stopped while starting, a module already migrated that cannot be stopped
// is still said: only the start's own context error is a clean stop.
func TestServeStoppedWhileStartingSaysWhatCouldNotStop(t *testing.T) {
	signal := make(chan struct{})
	was := notifyContext
	notifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		go func() { <-signal; cancel() }()
		return ctx, cancel
	}
	t.Cleanup(func() { notifyContext = was })
	captureLog(t)
	var log []string
	first := mod("first", &log)
	first.stopErr = errors.New("the database cannot be closed")
	m := waitsInMigrate{migrating: make(chan struct{})}
	app := New(Config{Host: "127.0.0.1", Port: "0"})
	app.Use(first, m)
	done := make(chan error, 1)
	go func() { done <- app.Serve(nil) }()
	<-m.migrating
	close(signal)
	select {
	case err := <-done:
		if err == nil || !errors.Is(err, first.stopErr) || strings.Contains(err.Error(), "could not migrate") {
			t.Fatalf("Serve = %v, want only first's stop error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after its context ended")
	}
}

// unbuiltDist is web/dist as weft create leaves it before the first build:
// only an empty marker, which the embed pattern matches and the page build
// replaces.
var unbuiltDist = fstest.MapFS{".weft-build": {Data: []byte(`{"weft":1,"files":[]}` + "\n")}}

const unbuiltSentence = "warp: the pages are not built (dist holds only the empty marker weft create writes); run weft build, which builds them into the binary."

// A binary built without its pages refuses to start, saying why, rather
// than serve nothing; nothing is started.
func TestServeRefusesUnbuiltPages(t *testing.T) {
	captureLog(t)
	var log []string
	app := New(Config{Host: "127.0.0.1", Port: "0"})
	app.Use(mod("a", &log))
	if err := app.Serve(unbuiltDist); errText(err) != unbuiltSentence {
		t.Fatalf("Serve = %q, want %q", errText(err), unbuiltSentence)
	}
	if want := []string{"register:a"}; !reflect.DeepEqual(log, want) {
		t.Errorf("log %v, want only the registration", log)
	}
}

// warp.Serve, the handler alone, panics with the same sentence.
func TestServeHandlerRefusesUnbuiltPages(t *testing.T) {
	defer func() {
		if r := recover(); r != unbuiltSentence {
			t.Fatalf("recovered %v, want %q", r, unbuiltSentence)
		}
	}()
	Serve(unbuiltDist, nil)
}
