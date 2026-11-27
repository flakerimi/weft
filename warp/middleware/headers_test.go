package middleware

import (
	"context"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/warptest"
)

// securityHeaders are the headers every answer carries over HTTP.
var securityHeaders = map[string]string{
	"X-Content-Type-Options":     "nosniff",
	"Referrer-Policy":            "same-origin",
	"X-Frame-Options":            "DENY",
	"Content-Security-Policy":    "frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'",
	"Cross-Origin-Opener-Policy": "same-origin",
}

// site is a page build of a single-page app: index.html and a file.
var site = fstest.MapFS{"index.html": {Data: []byte("<p>page</p>")}, "robots.txt": {Data: []byte("User-agent: *\n")}}

// run serves app and dist through Run on a port of its own until the test
// ends, and returns its address.
func run(t *testing.T, app *warp.App, dist fs.FS) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, ln, dist) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Run did not return")
		}
	})
	return "http://" + ln.Addr().String()
}

// get sends a request and returns the answer, its body read.
func get(t *testing.T, method, url string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp
}

// apiApp is an app with the middleware module and a route at /api/x.
func apiApp(vars map[string]string, opts ...Option) *warp.App {
	app := warp.New(warp.ConfigFrom(vars))
	app.Use(Module(opts...))
	app.Handle("GET /x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "x") }))
	app.Handle("GET /framed", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		io.WriteString(w, "framed")
	}))
	return app
}

func TestHeadersOnEveryAnswer(t *testing.T) {
	url := run(t, apiApp(nil), site)
	for _, c := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200}, {"GET", "/robots.txt", 200}, {"GET", "/api/x", 200},
		{"GET", "/api/nothing", 404}, {"POST", "/api/x", 405},
	} {
		resp := get(t, c.method, url+c.path)
		if resp.StatusCode != c.status {
			t.Errorf("%s %s answered %d, want %d", c.method, c.path, resp.StatusCode, c.status)
		}
		for k, v := range securityHeaders {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("%s %s has %s %q, want %q", c.method, c.path, k, got, v)
			}
		}
		if hsts := resp.Header.Get("Strict-Transport-Security"); hsts != "" {
			t.Errorf("%s %s over HTTP has Strict-Transport-Security %q", c.method, c.path, hsts)
		}
	}
}

func TestHSTSOnlyOverHTTPS(t *testing.T) {
	plain := warptest.New(t, nil, Module())
	if h := plain.Get("/api/nothing").Header.Get("Strict-Transport-Security"); h != "" {
		t.Errorf("over HTTP the answer has Strict-Transport-Security %q", h)
	}
	// Without TRUST_PROXY, X-Forwarded-Proto is not believed.
	r := httptest.NewRequest("GET", "http://warptest.local/api/nothing", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	if h := plain.Do(r).Header.Get("Strict-Transport-Security"); h != "" {
		t.Errorf("an X-Forwarded-Proto not trusted gives Strict-Transport-Security %q", h)
	}

	proxied := warptest.New(t, map[string]string{"TRUST_PROXY": "on"}, Module())
	r = httptest.NewRequest("GET", "http://warptest.local/api/nothing", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	if h := proxied.Do(r).Header.Get("Strict-Transport-Security"); h != "max-age=31536000" {
		t.Errorf("over HTTPS through a trusted proxy Strict-Transport-Security is %q, want max-age=31536000", h)
	}
}

func TestHandlerMayReplaceAHeader(t *testing.T) {
	url := run(t, apiApp(nil), nil)
	resp := get(t, "GET", url+"/api/framed")
	if got := resp.Header.Values("X-Frame-Options"); len(got) != 1 || got[0] != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options is %q, want the handler's SAMEORIGIN alone", got)
	}
	if got := resp.Header.Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy is %q beside the handler's header", got)
	}
}

func TestOffCostsNothing(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Handle("GET /x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "x") }))
	url := run(t, app, site)
	for _, path := range []string{"/", "/robots.txt", "/api/x", "/api/nothing"} {
		resp := get(t, "GET", url+path)
		for k := range securityHeaders {
			if got := resp.Header.Get(k); got != "" {
				t.Errorf("without the module %s has %s %q", path, k, got)
			}
		}
	}
}

func TestModuleName(t *testing.T) {
	if n := Module().Name(); n != "middleware" {
		t.Errorf("Name = %q, want middleware", n)
	}
}
