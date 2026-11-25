package warp

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"reflect"
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
	if want := []string{"start:a", "start:b", "stop:a"}; !reflect.DeepEqual(log[len(log)-3:], want) {
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
	if want := []string{"register:a", "register:b", "register:c", "migrate:a", "migrate:b"}; !reflect.DeepEqual(log, want) {
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
		"warp: Use was called after the app started serving; list every module before Serve.",
		`warp: route "GET /x" was registered after the app started serving; register every route before Serve.`,
		"warp: Wrap was called after the app started serving; wrap before Serve.",
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

func TestServerTimesOutSlowHeaders(t *testing.T) {
	if got := newServer(http.NotFoundHandler()).ReadHeaderTimeout; got != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout %v, want 10s", got)
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
	t.Run("a failing start stops the earlier ones", func(t *testing.T) {
		var log []string
		b := mod("b", &log)
		b.startErr = errors.New("boom")
		app := New(Config{})
		app.Use(mod("a", &log), b, mod("c", &log))
		if err := app.Start(ctx); errText(err) != `warp: module "b" could not start: boom` {
			t.Fatalf("Start = %q", errText(err))
		}
		if want := []string{"start:a", "start:b", "stop:a"}; !reflect.DeepEqual(log[len(log)-3:], want) {
			t.Fatalf("log %v", log)
		}
		n := len(log)
		if err := app.Stop(ctx); err != nil || len(log) != n {
			t.Fatalf("Stop after a failed Start: %v, log %v", err, log)
		}
	})
	t.Run("a failing migrate starts nothing", func(t *testing.T) {
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
		if want := []string{"register:a", "register:b", "migrate:a", "migrate:b"}; !reflect.DeepEqual(log, want) {
			t.Fatalf("log %v", log)
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
