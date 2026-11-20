package weft

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
	if got := strings.TrimSpace(body(r)); got != `{"error":{"code":"not_found","message":"no API"}}` {
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
		if r := recover(); r != "weft.Serve: dist has no index.html" {
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
	r := do(t, h, "GET", "/app/assets/main-abc.js")
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
		{"/app/assets/main-abc.js", "app()", "no-cache"},
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
	if r := do(t, Serve(pageBuild(t), nil), "GET", "/api/x"); r.StatusCode != 404 || !strings.Contains(r.body, "no API") {
		t.Fatalf("nil api: status %d body %q", r.StatusCode, r.body)
	}
}

func TestManifestEarlyHintsHeaders(t *testing.T) {
	h := Serve(pageBuild(t), nil)
	links := siteHints["/counter"]
	r := do(t, h, "GET", "/counter")
	if len(r.early) != 1 || !slices.Equal(r.early[0].Values("Link"), links) {
		t.Fatalf("103s %v", r.early)
	}
	if !slices.Equal(r.Header.Values("Link"), links) || r.body != "<p>counter</p>" {
		t.Fatalf("200: Link %q body %q", r.Header.Values("Link"), r.body)
	}
	// The not-found page's hints are under "404".
	r = do(t, h, "GET", "/nope")
	if len(r.early) != 1 || !slices.Equal(r.Header.Values("Link"), siteHints["404"]) || r.StatusCode != 404 {
		t.Fatalf("404: 103s %v, Link %q, status %d", r.early, r.Header.Values("Link"), r.StatusCode)
	}
	// A page without hints sends no 103.
	if r := do(t, h, "GET", "/about"); len(r.early) != 0 || r.Header.Get("Link") != "" {
		t.Fatalf("about: 103s %v Link %q", r.early, r.Header.Get("Link"))
	}
}

// RFC 9110 §15.2: no 1xx to an HTTP/1.0 client. The Link headers stay on the 200.
func TestManifestNoEarlyHintsForHTTP10(t *testing.T) {
	req := httptest.NewRequest("GET", "/counter", nil)
	req.Proto, req.ProtoMajor, req.ProtoMinor = "HTTP/1.0", 1, 0
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

func TestManifestEarlyHintsOverTheWire(t *testing.T) {
	srv := httptest.NewServer(Serve(pageBuild(t), nil))
	defer srv.Close()
	var codes []int
	var hinted [][]string
	trace := &httptrace.ClientTrace{
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			codes = append(codes, code)
			hinted = append(hinted, header.Values("Link"))
			return nil
		},
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), "GET", srv.URL+"/counter", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "identity")
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	links := siteHints["/counter"]
	if !slices.Equal(codes, []int{http.StatusEarlyHints}) || !slices.Equal(hinted[0], links) {
		t.Fatalf("1xx codes %v, Link %q", codes, hinted)
	}
	if res.StatusCode != 200 || !slices.Equal(res.Header.Values("Link"), links) || body(res) != "<p>counter</p>" {
		t.Fatalf("status %d, Link %q", res.StatusCode, res.Header.Values("Link"))
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

func servePage(tb testing.TB) func() {
	h := Serve(pageBuild(tb), nil)
	req := httptest.NewRequest("GET", "/counter/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br;q=0.9")
	w := &discard{h: http.Header{}}
	return func() {
		clear(w.h)
		h.ServeHTTP(w, req)
	}
}

func TestManifestServePageAllocs(t *testing.T) {
	if n := testing.AllocsPerRun(100, servePage(t)); n > 3 {
		t.Fatalf("%v allocs per page, want at most 3", n)
	}
}

func BenchmarkServePage(b *testing.B) {
	serve := servePage(b)
	b.ReportAllocs()
	for b.Loop() {
		serve()
	}
}
