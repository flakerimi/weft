package weft

import (
	"io"
	"net/http"
	"net/http/httptest"
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
