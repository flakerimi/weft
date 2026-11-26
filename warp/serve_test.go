package warp

import (
	"context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

var dist = fstest.MapFS{
	"index.html":        {Data: []byte(`<!doctype html><div id="app"></div>`)},
	"assets/app-abc.js": {Data: []byte(`console.log(1)`)},
	"favicon.svg":       {Data: []byte(`<svg/>`)},
}

func get(t *testing.T, h http.Handler, path string) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Result()
}

func body(r *http.Response) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}

func TestServeRoutesAPI(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "api "+r.URL.Path) })
	r := get(t, Serve(dist, api), "/api/ping")
	if got := body(r); got != "api /api/ping" {
		t.Fatalf("body %q", got)
	}
}

func TestServeAssetsImmutable(t *testing.T) {
	r := get(t, Serve(dist, nil), "/assets/app-abc.js")
	if r.StatusCode != 200 {
		t.Fatalf("status %d", r.StatusCode)
	}
	if cc := r.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("cache-control %q", cc)
	}
	if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("content-type %q", ct)
	}
}

func TestServeMissingAssetIs404(t *testing.T) {
	r := get(t, Serve(dist, nil), "/assets/gone.js")
	if r.StatusCode != 404 {
		t.Fatalf("status %d", r.StatusCode)
	}
	if strings.Contains(body(r), `id="app"`) {
		t.Fatal("served index.html for a missing asset")
	}
}

func TestServeClientRoutesGetIndex(t *testing.T) {
	for _, path := range []string{"/", "/customers/42", "/reports/2026.11"} {
		r := get(t, Serve(dist, nil), path)
		if r.StatusCode != 200 || !strings.Contains(body(r), `id="app"`) {
			t.Fatalf("%s: status %d", path, r.StatusCode)
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("%s: content-type %q", path, ct)
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("%s: cache-control %q", path, cc)
		}
	}
}

func TestServeOtherFilesNoCache(t *testing.T) {
	r := get(t, Serve(dist, nil), "/favicon.svg")
	if r.StatusCode != 200 || r.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("status %d, cache-control %q", r.StatusCode, r.Header.Get("Cache-Control"))
	}
}

func TestServeNilAPI(t *testing.T) {
	r := get(t, Serve(dist, nil), "/api/anything")
	if r.StatusCode != 404 {
		t.Fatalf("status %d", r.StatusCode)
	}
	if got := strings.TrimSpace(body(r)); got != `{"error":{"code":"not_found","message":"This app has no API."}}` {
		t.Fatalf("body %q", got)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestServePathTraversal(t *testing.T) {
	r := get(t, Serve(dist, nil), "/assets/../../etc/passwd")
	if r.StatusCode == 200 && !strings.Contains(body(r), `id="app"`) {
		t.Fatal("escaped dist")
	}
}

func TestServeNeedsIndex(t *testing.T) {
	defer func() {
		if r := recover(); r != "warp.Serve: dist has no index.html" {
			t.Fatalf("panic %v", r)
		}
	}()
	Serve(fstest.MapFS{"a.js": {}}, nil)
}

// recorder is an httptest.ResponseRecorder that keeps 1xx responses apart, as
// a server sends them, instead of taking the first as the final status.
type recorder struct {
	*httptest.ResponseRecorder
	early []http.Header
}

func (r *recorder) WriteHeader(code int) {
	if code >= 100 && code < 200 {
		r.early = append(r.early, r.Header().Clone())
		return
	}
	r.ResponseRecorder.WriteHeader(code)
}

type served struct {
	*http.Response
	body  string
	early []http.Header
}

// do sends one request to h, with header name/value pairs (an empty value is
// left out), and returns the response.
func do(t testing.TB, h http.Handler, method, path string, header ...string) served {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for i := 0; i < len(header); i += 2 {
		if header[i+1] != "" {
			req.Header.Add(header[i], header[i+1])
		}
	}
	rec := &recorder{ResponseRecorder: httptest.NewRecorder()}
	h.ServeHTTP(rec, req)
	res := rec.Result()
	return served{res, body(res), rec.early}
}

func TestManifestEncoding(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	const page = "<p>about us</p>"
	for _, c := range []struct {
		accept   []string
		encoding string
	}{
		{[]string{"br, gzip"}, "br"},
		{[]string{"gzip, br"}, "br"},
		{[]string{"gzip;q=1, br;q=0"}, "gzip"},
		{[]string{"gzip, deflate"}, "gzip"},
		{nil, ""},
		{[]string{"identity"}, ""},
		{[]string{"*"}, "br"},
		{[]string{"*;q=0"}, ""},
		{[]string{"*, br;q=0"}, "gzip"},
		{[]string{"br;q=0"}, ""},
		{[]string{"br;q=0.000, gzip ; q=0.0"}, ""},
		{[]string{"BR;Q=0.5, GZIP;q=0.9"}, "gzip"},
		{[]string{"br;q=0.9, gzip;q=0.9"}, "br"},
		{[]string{"x-gzip"}, "gzip"},
		{[]string{"deflate", "br"}, "br"},
		{[]string{"br;q=nope, gzip"}, "gzip"},
		{[]string{"br;q=2, gzip;q=0.5"}, "gzip"},
		{[]string{"compress, deflate"}, ""},
		// identity wins only when ranked above every compressed option accepted.
		{[]string{"identity;q=1, br;q=0.5"}, ""},
		{[]string{"identity;q=0.5, br;q=0.5"}, "br"},
		{[]string{"br;q=0.4, gzip;q=0.6, identity;q=0.5"}, "gzip"},
		{[]string{"identity, *;q=0.9"}, ""},
		// identity;q=0 with nothing else acceptable still gets identity (RFC 9110 §12.5.3).
		{[]string{"identity;q=0"}, ""},
		{[]string{"identity;q=0, br;q=0, gzip;q=0"}, ""},
		{[]string{"identity;q=0, gzip;q=0.1"}, "gzip"},
	} {
		req := httptest.NewRequest("GET", "/about", nil)
		for _, v := range c.accept {
			req.Header.Add("Accept-Encoding", v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		r := rec.Result()
		got := body(r)
		want, etag := page, `"about"`
		switch c.encoding {
		case "br":
			want, etag = "br:"+page, `"about-br"`
		case "gzip":
			want, etag = "gz:"+page, `"about-gz"`
		}
		if r.StatusCode != 200 || got != want {
			t.Errorf("%q: status %d body %q, want %q", c.accept, r.StatusCode, got, want)
		}
		if ce := r.Header.Get("Content-Encoding"); ce != c.encoding {
			t.Errorf("%q: Content-Encoding %q, want %q", c.accept, ce, c.encoding)
		}
		if et := r.Header.Get("ETag"); et != etag {
			t.Errorf("%q: ETag %s, want %s", c.accept, et, etag)
		}
		if cl := r.Header.Get("Content-Length"); cl != strconv.Itoa(len(want)) {
			t.Errorf("%q: Content-Length %s, want %d", c.accept, cl, len(want))
		}
		if v := r.Header.Values("Vary"); len(v) != 1 || v[0] != "Accept-Encoding" {
			t.Errorf("%q: Vary %q", c.accept, v)
		}
	}
}

func TestManifestFileWithoutVariants(t *testing.T) {
	r := do(t, Serve(pageBuild(t), nil), "GET", "/icon.bin", "Accept-Encoding", "br, gzip")
	if r.body != "\x00\x01\x02" || r.Header.Get("Content-Encoding") != "" || r.Header.Get("ETag") != `"icon"` {
		t.Fatalf("body %q, encoding %q, etag %s", r.body, r.Header.Get("Content-Encoding"), r.Header.Get("ETag"))
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestManifestVaryOnEveryResponse(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	bare := Serve(editRoutes(t, pageBuild(t), `{"pages":[],"notFound":null,"app":null}`), nil)
	for _, c := range []struct {
		name string
		r    served
	}{
		{"page", do(t, h, "GET", "/about")},
		{"asset", do(t, h, "GET", "/assets/client-abc.js", "Accept-Encoding", "gzip")},
		{"app", do(t, h, "GET", "/app/x")},
		{"not found page", do(t, h, "GET", "/nope")},
		{"plain not found", do(t, bare, "GET", "/nope")},
		{"304", do(t, h, "GET", "/about", "If-None-Match", `"about"`)},
		{"HEAD", do(t, h, "HEAD", "/about")},
		{"405", do(t, h, "POST", "/about")},
	} {
		if v := c.r.Header.Values("Vary"); len(v) != 1 || v[0] != "Accept-Encoding" {
			t.Errorf("%s: status %d, Vary %q", c.name, c.r.StatusCode, v)
		}
	}
}

func TestManifestNotModified(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for _, c := range []struct{ accept, match string }{
		{"", `"about"`},
		{"", `W/"about"`},
		{"", `"other", "about"`},
		{"", `"other",W/"about"`},
		{"", `*`},
		{"br", `"about-br"`},
		{"gzip", `W/"x", "about-gz"`},
	} {
		r := do(t, h, "GET", "/about", "Accept-Encoding", c.accept, "If-None-Match", c.match)
		if r.StatusCode != http.StatusNotModified || r.body != "" {
			t.Errorf("%q %s: status %d body %q", c.accept, c.match, r.StatusCode, r.body)
			continue
		}
		if r.Header.Get("ETag") == "" || r.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("%q %s: ETag %q Cache-Control %q", c.accept, c.match, r.Header.Get("ETag"), r.Header.Get("Cache-Control"))
		}
		if r.Header.Get("Content-Length") != "" || r.Header.Get("Content-Encoding") != "" {
			t.Errorf("%q %s: 304 with Content-Length %q, Content-Encoding %q", c.accept, c.match, r.Header.Get("Content-Length"), r.Header.Get("Content-Encoding"))
		}
	}
	// Another representation's tag is not a match.
	for _, c := range []struct{ accept, match string }{
		{"br", `"about"`},
		{"", `"about-br"`},
		{"", `"abou"`},
		{"", `"other"`},
	} {
		r := do(t, h, "GET", "/about", "Accept-Encoding", c.accept, "If-None-Match", c.match)
		if r.StatusCode != 200 || r.body == "" {
			t.Errorf("%q %s: status %d body %q, want 200", c.accept, c.match, r.StatusCode, r.body)
		}
	}
}

func TestManifestPages(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for path, want := range map[string]string{
		"/":        "<p>home</p>",
		"/about":   "<p>about us</p>",
		"/about/":  "<p>about us</p>",
		"/counter": "<p>counter</p>",
	} {
		r := do(t, h, "GET", path)
		if r.StatusCode != 200 || r.body != want {
			t.Errorf("%s: status %d body %q", path, r.StatusCode, r.body)
		}
		if ct, cc := r.Header.Get("Content-Type"), r.Header.Get("Cache-Control"); ct != "text/html; charset=utf-8" || cc != "no-cache" {
			t.Errorf("%s: content-type %q cache-control %q", path, ct, cc)
		}
	}
}

func TestManifestApp(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for _, path := range []string{"/app", "/app/", "/app/x/y", "/app/customers/42"} {
		r := do(t, h, "GET", path)
		if r.StatusCode != 200 || r.body != `<div id="app"></div>` {
			t.Errorf("%s: status %d body %q", path, r.StatusCode, r.body)
		}
	}
	// A listed file under the prefix is that file, not the app index.
	r := do(t, h, "GET", "/app/assets/main-Bx4kQ9zA.js")
	if r.StatusCode != 200 || r.body != "app()" || !strings.Contains(r.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("app asset: status %d body %q type %q", r.StatusCode, r.body, r.Header.Get("Content-Type"))
	}
	// As without a manifest, a missing asset is never the index.
	if r := do(t, h, "GET", "/app/assets/gone.js"); r.StatusCode != 404 || r.body != "<p>not here</p>" {
		t.Fatalf("missing app asset: status %d body %q", r.StatusCode, r.body)
	}
	// /apple is not under /app.
	if r := do(t, h, "GET", "/apple"); r.StatusCode != 404 {
		t.Fatalf("/apple: status %d", r.StatusCode)
	}
}

// The build puts the app in the folder its prefix names; the same rules hold.
func TestManifestAppUnderOwnPrefix(t *testing.T) {
	h := Serve(buildOf(t, []siteFile{
		{"/admin/index.html", `<div id="admin"></div>`, `"admin"`, "text/html; charset=utf-8", true, false},
		{"/admin/assets/main-abc.js", "admin()", `"adminmain"`, "text/javascript; charset=utf-8", true, false},
		{"/assets/main-abc.js", "site()", `"sitemain"`, "text/javascript; charset=utf-8", true, true},
	}, `{"pages":[],"notFound":null,"app":{"prefix":"/admin","index":"admin/index.html"}}`), nil)
	for _, c := range []struct {
		path   string
		status int
		body   string
	}{
		{"/admin", 200, `<div id="admin"></div>`},
		{"/admin/", 200, `<div id="admin"></div>`},
		{"/admin/users/7", 200, `<div id="admin"></div>`},
		{"/admin/assets/main-abc.js", 200, "admin()"},
		{"/admin/assets/gone.js", 404, "404 page not found\n"},
		{"/assets/main-abc.js", 200, "site()"},
		{"/app/x", 404, "404 page not found\n"},
		{"/administrator", 404, "404 page not found\n"},
	} {
		if r := do(t, h, "GET", c.path); r.StatusCode != c.status || r.body != c.body {
			t.Errorf("%s: status %d body %q", c.path, r.StatusCode, r.body)
		}
	}
}

func TestManifestFiles(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for _, c := range []struct{ path, body, cache string }{
		{"/assets/client-abc.js", "export const a=1", "public, max-age=31536000, immutable"},
		{"/robots.txt", "User-agent: *", "no-cache"},
		{"/app/assets/main-Bx4kQ9zA.js", "app()", "public, max-age=31536000, immutable"},
	} {
		r := do(t, h, "GET", c.path)
		if r.StatusCode != 200 || r.body != c.body || r.Header.Get("Cache-Control") != c.cache {
			t.Errorf("%s: status %d body %q cache-control %q", c.path, r.StatusCode, r.body, r.Header.Get("Cache-Control"))
		}
	}
}

func TestManifestNotFound(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for _, path := range []string{"/nope", "/about/more", "/assets/gone.js", "/robots.txt.br", "/ROBOTS.TXT", "/robots.txt/"} {
		r := do(t, h, "GET", path, "Accept-Encoding", "br")
		if r.StatusCode != 404 || r.body != "br:<p>not here</p>" || r.Header.Get("Content-Encoding") != "br" {
			t.Errorf("%s: status %d body %q", path, r.StatusCode, r.body)
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s: content-type %q", path, ct)
		}
	}
	// The not-found page is never a 304: the status is what it says.
	if r := do(t, h, "GET", "/nope", "If-None-Match", `"lost"`); r.StatusCode != 404 || r.body != "<p>not here</p>" {
		t.Fatalf("conditional: status %d body %q", r.StatusCode, r.body)
	}
}

func TestManifestPlainNotFound(t *testing.T) {
	h := Serve(editRoutes(t, pageBuild(t), `{"pages":[],"notFound":null,"app":null}`), nil)
	r := do(t, h, "GET", "/nope")
	if r.StatusCode != 404 || strings.TrimSpace(r.body) != "404 page not found" {
		t.Fatalf("status %d body %q", r.StatusCode, r.body)
	}
}

func TestManifestInternalFilesHidden(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	for _, path := range []string{
		"/manifest.json",
		"/routes.json",
		"/.weft-build",
		"/pages/about/index.html",
		"/pages/about/index.html.br",
		"/pages/404/index.html",
		"/pages/",
		"/x/../manifest.json",
		"//routes.json",
		"/pages//about/index.html",
	} {
		r := do(t, h, "GET", path)
		if r.StatusCode != 404 || r.body != "<p>not here</p>" {
			t.Errorf("%s: status %d body %q", path, r.StatusCode, r.body)
		}
	}
}

func TestManifestHEAD(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	get := do(t, h, "GET", "/about", "Accept-Encoding", "br")
	head := do(t, h, "HEAD", "/about", "Accept-Encoding", "br")
	if head.StatusCode != 200 || head.body != "" {
		t.Fatalf("status %d body %q", head.StatusCode, head.body)
	}
	for _, k := range []string{"Content-Type", "Content-Length", "Content-Encoding", "ETag", "Cache-Control", "Vary"} {
		if g, h := get.Header.Get(k), head.Header.Get(k); g != h || g == "" {
			t.Errorf("%s: GET %q, HEAD %q", k, g, h)
		}
	}
}

func TestManifestMethodNotAllowed(t *testing.T) {
	r := do(t, Serve(pageBuild(t), nil), "POST", "/about")
	if r.StatusCode != http.StatusMethodNotAllowed || r.Header.Get("Allow") != "GET, HEAD" {
		t.Fatalf("status %d allow %q", r.StatusCode, r.Header.Get("Allow"))
	}
}

func TestManifestAPI(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, r.Method+" api "+r.URL.Path) })
	h := Serve(pageBuild(t), api)
	for _, path := range []string{"/api", "/api/ping"} {
		if r := do(t, h, "POST", path); r.body != "POST api "+path {
			t.Errorf("%s: body %q", path, r.body)
		}
	}
	if r := do(t, Serve(pageBuild(t), nil), "GET", "/api/x"); r.StatusCode != 404 || !strings.Contains(r.body, "This app has no API.") {
		t.Fatalf("nil api: status %d body %q", r.StatusCode, r.body)
	}
}

// throughProxy is a browser's navigation as a proxy passes it on: the request
// headers that earn a page its 103 over HTTP/1.1.
var throughProxy = []string{"Sec-Fetch-Mode", "navigate", "Via", "1.1 proxy.example"}

func TestManifestEarlyHintsHeaders(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	links := siteHints["/counter"]
	r := do(t, h, "GET", "/counter", throughProxy...)
	if len(r.early) != 1 || !slices.Equal(r.early[0].Values("Link"), links) {
		t.Fatalf("103s %v", r.early)
	}
	if !slices.Equal(r.Header.Values("Link"), links) || r.body != "<p>counter</p>" {
		t.Fatalf("200: Link %q body %q", r.Header.Values("Link"), r.body)
	}
	// The not-found page's hints are under "404".
	r = do(t, h, "GET", "/nope", throughProxy...)
	if len(r.early) != 1 || !slices.Equal(r.Header.Values("Link"), siteHints["404"]) || r.StatusCode != 404 {
		t.Fatalf("404: 103s %v, Link %q, status %d", r.early, r.Header.Values("Link"), r.StatusCode)
	}
	// A page without hints sends no 103.
	if r := do(t, h, "GET", "/about", throughProxy...); len(r.early) != 0 || r.Header.Get("Link") != "" {
		t.Fatalf("about: 103s %v Link %q", r.early, r.Header.Get("Link"))
	}
}

// RFC 9110 §15.2: no 1xx to an HTTP/1.0 client, even a navigation through a
// proxy. The Link headers stay on the 200.
func TestManifestNoEarlyHintsForHTTP10(t *testing.T) {
	req := httptest.NewRequest("GET", "/counter", nil)
	req.Proto, req.ProtoMajor, req.ProtoMinor = "HTTP/1.0", 1, 0
	for i := 0; i < len(throughProxy); i += 2 {
		req.Header.Set(throughProxy[i], throughProxy[i+1])
	}
	rec := &recorder{ResponseRecorder: httptest.NewRecorder()}
	Serve(pageBuild(t), nil).ServeHTTP(rec, req)
	res := rec.Result()
	if len(rec.early) != 0 {
		t.Fatalf("103s to HTTP/1.0: %v", rec.early)
	}
	if res.StatusCode != 200 || !slices.Equal(res.Header.Values("Link"), siteHints["/counter"]) {
		t.Fatalf("status %d, Link %q", res.StatusCode, res.Header.Values("Link"))
	}
}

// hintsServer serves the page build on a real server: over TLS with HTTP/2
// when h2 is set, else over HTTP/1.1 in the clear.
func hintsServer(t *testing.T, h2 bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(Serve(pageBuild(t), nil))
	if h2 {
		srv.EnableHTTP2 = true
		srv.StartTLS()
	} else {
		srv.Start()
	}
	t.Cleanup(srv.Close)
	return srv
}

// wired is a response read off the wire, with the 1xx responses before it.
type wired struct {
	*http.Response
	body  string
	codes []int      // the status of each 1xx, in order
	early [][]string // the Link values of each 1xx
}

// onWire sends one request to srv with header name/value pairs, asking for
// brotli or gzip so the client leaves the body as sent, and reads the answer.
// It fails the test if the answer did not come over the protocol srv speaks.
func onWire(t *testing.T, srv *httptest.Server, method, path string, header ...string) wired {
	t.Helper()
	var w wired
	trace := &httptrace.ClientTrace{
		Got1xxResponse: func(code int, h textproto.MIMEHeader) error {
			w.codes = append(w.codes, code)
			w.early = append(w.early, h.Values("Link"))
			return nil
		},
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), method, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "br, gzip")
	for i := 0; i < len(header); i += 2 {
		req.Header.Add(header[i], header[i+1])
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	want := 1
	if srv.EnableHTTP2 {
		want = 2
	}
	if res.ProtoMajor != want {
		t.Fatalf("%s %s: answered over %s, want HTTP/%d", method, path, res.Proto, want)
	}
	w.Response, w.body = res, body(res)
	return w
}

// Early Hints go out only to a navigation that arrived over HTTP/2 or later,
// or through a proxy, which may speak HTTP/2 to the browser: a browser acts
// on a 103 over HTTP/2 and ignores one over HTTP/1.1. The final response is
// the same either way.
func TestManifestEarlyHintsOnlyWhereUsable(t *testing.T) {
	h1, h2 := hintsServer(t, false), hintsServer(t, true)
	// What each server sends for the page when no 103 can help.
	plain := map[*httptest.Server]wired{h1: onWire(t, h1, "GET", "/counter"), h2: onWire(t, h2, "GET", "/counter")}
	links := siteHints["/counter"]
	for _, c := range []struct {
		name   string
		srv    *httptest.Server
		header []string
		hinted bool
	}{
		{"HTTP/2 navigation", h2, []string{"Sec-Fetch-Mode", "navigate"}, true},
		{"HTTP/1.1 navigation with X-Forwarded-For", h1, []string{"Sec-Fetch-Mode", "navigate", "X-Forwarded-For", "203.0.113.7"}, true},
		{"HTTP/1.1 navigation with Forwarded", h1, []string{"Sec-Fetch-Mode", "navigate", "Forwarded", "for=203.0.113.7;proto=https"}, true},
		{"HTTP/1.1 navigation with Via", h1, []string{"Sec-Fetch-Mode", "navigate", "Via", "2 proxy.example"}, true},
		{"HTTP/1.1 navigation, no proxy", h1, []string{"Sec-Fetch-Mode", "navigate"}, false},
		{"HTTP/2, no Sec-Fetch-Mode", h2, nil, false},
		{"HTTP/2 cors", h2, []string{"Sec-Fetch-Mode", "cors"}, false},
		{"HTTP/2 no-cors", h2, []string{"Sec-Fetch-Mode", "no-cors"}, false},
		{"HTTP/1.1 through a proxy, no Sec-Fetch-Mode", h1, []string{"X-Forwarded-For", "203.0.113.7", "Via", "1.1 proxy.example"}, false},
		{"HTTP/1.1 cors through a proxy", h1, []string{"Sec-Fetch-Mode", "cors", "Forwarded", "for=203.0.113.7"}, false},
		{"HTTP/1.1 no-cors through a proxy", h1, []string{"Sec-Fetch-Mode", "no-cors", "Via", "1.1 proxy.example"}, false},
	} {
		r := onWire(t, c.srv, "GET", "/counter", c.header...)
		if c.hinted && (!slices.Equal(r.codes, []int{http.StatusEarlyHints}) || !slices.Equal(r.early[0], links)) {
			t.Errorf("%s: 1xx %v with Link %q, want one 103 with %q", c.name, r.codes, r.early, links)
		}
		if !c.hinted && len(r.codes) != 0 {
			t.Errorf("%s: 1xx %v with Link %q, want none", c.name, r.codes, r.early)
		}
		if r.StatusCode != 200 || !slices.Equal(r.Header.Values("Link"), links) {
			t.Errorf("%s: status %d, Link %q", c.name, r.StatusCode, r.Header.Values("Link"))
		}
		// The 200 is byte for byte the one sent with no 103.
		want := plain[c.srv]
		got, base := r.Header.Clone(), want.Header.Clone()
		got.Del("Date")
		base.Del("Date")
		if !maps.EqualFunc(got, base, slices.Equal) || r.body != want.body || r.body != "br:<p>counter</p>" {
			t.Errorf("%s: 200 %v %q, want %v %q", c.name, got, r.body, base, want.body)
		}
	}
}

// What a navigation that may have a 103 gets in every other case.
func TestManifestEarlyHintsCases(t *testing.T) {
	h1, h2 := hintsServer(t, false), hintsServer(t, true)
	nav := []string{"Sec-Fetch-Mode", "navigate"}
	for _, srv := range []*httptest.Server{h1, h2} {
		header := nav
		if srv == h1 {
			header = throughProxy
		}
		// A page with no Link headers never has a 103.
		if r := onWire(t, srv, "GET", "/about", header...); len(r.codes) != 0 || r.Header.Get("Link") != "" || r.StatusCode != 200 {
			t.Errorf("about over %s: 1xx %v, Link %q, status %d", r.Proto, r.codes, r.Header.Get("Link"), r.StatusCode)
		}
		// The not-found page has its own.
		if r := onWire(t, srv, "GET", "/nope", header...); !slices.Equal(r.codes, []int{http.StatusEarlyHints}) ||
			!slices.Equal(r.early[0], siteHints["404"]) || r.StatusCode != 404 || !slices.Equal(r.Header.Values("Link"), siteHints["404"]) {
			t.Errorf("404 over %s: 1xx %v with %q, status %d, Link %q", r.Proto, r.codes, r.early, r.StatusCode, r.Header.Values("Link"))
		}
		// A 304 is answered before any 103, so it has none and no Link.
		cond := append(slices.Clone(header), "If-None-Match", `"counter-br"`)
		if r := onWire(t, srv, "GET", "/counter", cond...); r.StatusCode != http.StatusNotModified || len(r.codes) != 0 || r.Header.Get("Link") != "" || r.body != "" {
			t.Errorf("304 over %s: status %d, 1xx %v, Link %q, body %q", r.Proto, r.StatusCode, r.codes, r.Header.Get("Link"), r.body)
		}
		// HEAD is answered as GET is, without the body.
		get, head := onWire(t, srv, "GET", "/counter", header...), onWire(t, srv, "HEAD", "/counter", header...)
		if !slices.Equal(head.codes, []int{http.StatusEarlyHints}) || !slices.Equal(head.early[0], siteHints["/counter"]) || head.body != "" {
			t.Errorf("HEAD over %s: 1xx %v with %q, body %q", head.Proto, head.codes, head.early, head.body)
		}
		for _, k := range []string{"Link", "Content-Length", "Content-Encoding", "Etag", "Cache-Control", "Vary"} {
			if g, h := get.Header.Values(k), head.Header.Values(k); !slices.Equal(g, h) || len(g) == 0 {
				t.Errorf("HEAD over %s: %s %q, GET %q", head.Proto, k, h, g)
			}
		}
		if r := onWire(t, srv, "HEAD", "/counter", "Sec-Fetch-Mode", "no-cors"); len(r.codes) != 0 {
			t.Errorf("no-cors HEAD over %s: 1xx %v", r.Proto, r.codes)
		}
	}
	// Over HTTP/1.1 a HEAD with no proxy has none either.
	if r := onWire(t, h1, "HEAD", "/counter", nav...); len(r.codes) != 0 || !slices.Equal(r.Header.Values("Link"), siteHints["/counter"]) {
		t.Errorf("HEAD over HTTP/1.1, no proxy: 1xx %v, Link %q", r.codes, r.Header.Values("Link"))
	}
}

// editRoutes returns dist with routes.json replaced.
func editRoutes(t *testing.T, dist fstest.MapFS, routes string) fstest.MapFS {
	t.Helper()
	out := maps.Clone(dist)
	out["routes.json"] = &fstest.MapFile{Data: []byte(routes)}
	return out
}

// discard is a ResponseWriter that keeps nothing, so a benchmark counts only
// what the handler allocates.
type discard struct{ h http.Header }

func (d *discard) Header() http.Header         { return d.h }
func (d *discard) Write(b []byte) (int, error) { return len(b), nil }
func (d *discard) WriteHeader(int)             {}

// servePage returns a function that asks for the page with hints over
// HTTP/proto (1 or 2), with header name/value pairs.
func servePage(tb testing.TB, proto int, header ...string) func() {
	h := Serve(pageBuild(tb), nil)
	req := httptest.NewRequest("GET", "/counter/", nil)
	if proto == 2 {
		req.Proto, req.ProtoMajor, req.ProtoMinor = "HTTP/2.0", 2, 0
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate, br;q=0.9")
	for i := 0; i < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := &discard{h: http.Header{}}
	return func() {
		clear(w.h)
		h.ServeHTTP(w, req)
	}
}

func TestManifestServePageAllocs(t *testing.T) {
	if n := testing.AllocsPerRun(100, servePage(t, 1)); n > 3 {
		t.Fatalf("%v allocs per page, want at most 3", n)
	}
}

// earlyHintsCases are requests for a page with hints, a 103 sent or not.
var earlyHintsCases = []struct {
	name   string
	proto  int
	header []string
}{
	{"plain", 1, nil},
	{"navigation", 1, []string{"Sec-Fetch-Mode", "navigate"}},
	{"navigation-proxy", 1, []string{"Sec-Fetch-Mode", "navigate", "X-Forwarded-For", "203.0.113.7"}},
	{"navigation-h2", 2, []string{"Sec-Fetch-Mode", "navigate"}},
	{"cors-proxy", 1, []string{"Sec-Fetch-Mode", "cors", "Forwarded", "for=203.0.113.7"}},
}

// Deciding whether to send a 103 allocates nothing: every page request makes
// the one slice of header values, and no more.
func TestManifestEarlyHintsDecisionAllocs(t *testing.T) {
	if raceDetector {
		t.Skip("the race detector changes allocation counts")
	}
	for _, c := range earlyHintsCases {
		if n := testing.AllocsPerRun(100, servePage(t, c.proto, c.header...)); n != 1 {
			t.Errorf("%s: %v allocs per page, want 1", c.name, n)
		}
	}
}

func BenchmarkServePage(b *testing.B) {
	serve := servePage(b, 1)
	b.ReportAllocs()
	for b.Loop() {
		serve()
	}
}

func BenchmarkServePageEarlyHints(b *testing.B) {
	for _, c := range earlyHintsCases {
		b.Run(c.name, func(b *testing.B) {
			serve := servePage(b, c.proto, c.header...)
			b.ReportAllocs()
			for b.Loop() {
				serve()
			}
		})
	}
}

// Serve with no build says what to pass, in place of a nil pointer.
func TestServeNilDist(t *testing.T) {
	defer func() {
		want := "warp.Serve: dist is nil; pass the page build, or serve the API alone with app.Handler()."
		if r := recover(); r != want {
			t.Fatalf("Serve(nil, api) panicked with %v, want %q", r, want)
		}
	}()
	Serve(nil, http.NotFoundHandler())
}
