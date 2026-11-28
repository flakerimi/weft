package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/flakerimi/weft/warp"
)

// lockedBuffer is a buffer the server and a test may use at once.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// keepLogging puts slog's default and the standard logger back as they were
// when the test ends. A test that calls it does not run in parallel.
func keepLogging(t *testing.T) {
	d, w, f := slog.Default(), log.Writer(), log.Flags()
	t.Cleanup(func() {
		slog.SetDefault(d)
		log.SetOutput(w)
		log.SetFlags(f)
	})
}

// requestLines are the lines of out that are request lines.
func requestLines(out string) []string {
	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if strings.Contains(l, "msg=request ") || strings.Contains(l, `"msg":"request"`) {
			lines = append(lines, l)
		}
	}
	return lines
}

// logApp is an app with the middleware module logging to out, with a route
// that answers 200, one that answers 500, and one that flushes.
func logApp(t *testing.T, vars map[string]string, out io.Writer) *warp.App {
	keepLogging(t)
	app := warp.New(warp.ConfigFrom(vars))
	app.Use(Module(LogTo(out)))
	app.Handle("GET /x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "hello") }))
	app.Handle("GET /boom", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		warp.WriteError(w, r, io.ErrUnexpectedEOF)
	}))
	return app
}

// sendThree sends a page, an API 200 and an API 500, each with a query.
func sendThree(t *testing.T, url string) {
	for _, p := range []string{"/?q=secret", "/api/x?search=secret", "/api/boom?q=secret"} {
		get(t, "GET", url+p)
	}
}

func TestLogText(t *testing.T) {
	out := &lockedBuffer{}
	sendThree(t, run(t, logApp(t, nil, out), site))
	lines := requestLines(out.String())
	if len(lines) != 3 {
		t.Fatalf("the log has %d request lines, want 3:\n%s", len(lines), out)
	}
	for i, want := range []string{
		"level=INFO msg=request method=GET path=/ status=200 bytes=11 ms=",
		"level=INFO msg=request method=GET path=/api/x status=200 bytes=5 ms=",
		"level=ERROR msg=request method=GET path=/api/boom status=500 bytes=",
	} {
		if !strings.Contains(lines[i], want) || !strings.HasSuffix(lines[i], " ip=127.0.0.1") {
			t.Errorf("line %d is\n%s\nwant it to have\n%s … ip=127.0.0.1", i+1, lines[i], want)
		}
	}
	if strings.Contains(out.String(), "secret") {
		t.Errorf("the log has a query string:\n%s", out)
	}
}

func TestLogJSON(t *testing.T) {
	out := &lockedBuffer{}
	sendThree(t, run(t, logApp(t, map[string]string{"LOG": "json"}, out), site))
	lines := requestLines(out.String())
	if len(lines) != 3 {
		t.Fatalf("the log has %d request lines, want 3:\n%s", len(lines), out)
	}
	for i, want := range []struct {
		level, path string
		status      float64
	}{{"INFO", "/", 200}, {"INFO", "/api/x", 200}, {"ERROR", "/api/boom", 500}} {
		var line map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &line); err != nil {
			t.Fatalf("line %d is not JSON: %v\n%s", i+1, err, lines[i])
		}
		if line["level"] != want.level || line["method"] != "GET" || line["path"] != want.path || line["status"] != want.status || line["ip"] != "127.0.0.1" {
			t.Errorf("line %d is %v, want %s %s %v", i+1, line, want.level, want.path, want.status)
		}
		if _, ok := line["bytes"].(float64); !ok {
			t.Errorf("line %d has no bytes: %v", i+1, line)
		}
		if _, ok := line["ms"].(float64); !ok {
			t.Errorf("line %d has no ms: %v", i+1, line)
		}
	}
	if strings.Contains(out.String(), "secret") {
		t.Errorf("the log has a query string:\n%s", out)
	}
}

func TestLogOff(t *testing.T) {
	out := &lockedBuffer{}
	before := slog.Default()
	sendThree(t, run(t, logApp(t, map[string]string{"LOG": "off"}, out), site))
	if out.String() != "" {
		t.Errorf("LOG=off wrote\n%s", out)
	}
	if slog.Default() != before {
		t.Error("LOG=off changed slog's default")
	}
}

func TestLogMistake(t *testing.T) {
	app := logApp(t, map[string]string{"LOG": "xml"}, io.Discard)
	err := app.Start(context.Background())
	if want := `warp: module "middleware" could not start: LOG is "xml"; set it to text, json or off in .env.`; err == nil || err.Error() != want {
		t.Errorf("Start = %v, want %q", err, want)
	}
}

func TestLogKeepsFlusher(t *testing.T) {
	out := &lockedBuffer{}
	app := logApp(t, nil, out)
	flushed := make(chan error, 1)
	var unwrapped bool
	app.Handle("GET /stream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, unwrapped = w.(interface{ Unwrap() http.ResponseWriter })
		io.WriteString(w, "part")
		flushed <- http.NewResponseController(w).Flush()
	}))
	get(t, "GET", run(t, app, nil)+"/api/stream")
	if err := <-flushed; err != nil {
		t.Errorf("Flush through the log's writer: %v", err)
	}
	if !unwrapped {
		t.Error("the log's writer has no Unwrap")
	}
	if lines := requestLines(out.String()); len(lines) != 1 || !strings.Contains(lines[0], "status=200 bytes=4 ") {
		t.Errorf("a flushed answer is logged as %q", lines)
	}
}

func TestDefaultLoggerFollows(t *testing.T) {
	out := &lockedBuffer{}
	app := logApp(t, map[string]string{"LOG": "json"}, out)
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer app.Stop(context.Background())
	log.Printf("hello %d", 1)
	slog.Warn("careful")
	var line map[string]any
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("the log has %q, want two lines", lines)
	}
	if err := json.Unmarshal([]byte(lines[0]), &line); err != nil || line["msg"] != "hello 1" || line["level"] != "INFO" {
		t.Errorf("log.Printf after Start writes %q, want JSON with msg hello 1", lines[0])
	}
	if err := json.Unmarshal([]byte(lines[1]), &line); err != nil || line["msg"] != "careful" || line["level"] != "WARN" {
		t.Errorf("slog.Warn after Start writes %q", lines[1])
	}
}

// TestAppsStoppedInAnyOrder starts two apps, A and then B, and stops them in
// each order: the default is the newest app's still running, and the
// process's own once both have stopped.
func TestAppsStoppedInAnyOrder(t *testing.T) {
	for _, order := range []string{"AB", "BA"} {
		t.Run(order, func(t *testing.T) {
			keepLogging(t)
			before := slog.Default()
			outs := map[byte]*lockedBuffer{'A': {}, 'B': {}}
			apps := map[byte]*warp.App{}
			for _, name := range []byte("AB") {
				apps[name] = logApp(t, map[string]string{"LOG": "json"}, outs[name])
				if err := apps[name].Start(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			// says writes a line and answers which app's log it went to.
			says := func(msg string) string {
				slog.Info(msg)
				var in string
				for name, out := range outs {
					if strings.Contains(out.String(), msg) {
						in += string(name)
					}
				}
				return in
			}
			if got := says("both running"); got != "B" {
				t.Fatalf("with A and B running, slog's default writes to %q, want B", got)
			}
			first, second := order[0], order[1]
			apps[first].Stop(context.Background())
			if got := says("one stopped"); got != string(second) {
				t.Errorf("after %c stopped, slog's default writes to %q, want %c, which still runs", first, got, second)
			}
			apps[second].Stop(context.Background())
			if slog.Default() != before {
				t.Errorf("after both stopped, slog's default is not the one before; a line goes to %q", says("both stopped"))
			}
			var b bytes.Buffer
			log.SetOutput(&b)
			log.Print("plain")
			if !strings.HasSuffix(b.String(), "plain\n") || strings.Contains(b.String(), "{") {
				t.Errorf("after both stopped the standard logger writes %q", b.String())
			}
		})
	}
}

func TestStopPutsTheLoggerBack(t *testing.T) {
	keepLogging(t)
	before := slog.Default()
	app := logApp(t, map[string]string{"LOG": "json"}, io.Discard)
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if slog.Default() == before {
		t.Fatal("Start did not set slog's default")
	}
	app.Stop(context.Background())
	if slog.Default() != before {
		t.Error("Stop did not put slog's default back")
	}
	var b bytes.Buffer
	log.SetOutput(&b)
	log.Print("plain")
	if !strings.HasSuffix(b.String(), "plain\n") || strings.Contains(b.String(), "{") {
		t.Errorf("after Stop the standard logger writes %q", b.String())
	}
}
