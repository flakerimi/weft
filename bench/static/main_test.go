package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func site(t *testing.T) (dir string, page []byte) {
	t.Helper()
	dir = t.TempDir()
	page = []byte("<!doctype html><title>t</title>" + strings.Repeat("<p>hello</p>", 200))
	if err := os.WriteFile(filepath.Join(dir, "index.html"), page, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hero.webp"), []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, page
}

func serve(t *testing.T, h http.Handler, path string, header map[string]string) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	return res, body
}

func TestGzipsTextWhenAccepted(t *testing.T) {
	dir, page := site(t)
	res, body := serve(t, handler(dir), "/", map[string]string{"Accept-Encoding": "br, gzip"})
	if res.StatusCode != 200 || res.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("GET / = %d, Content-Encoding %q; want 200 gzip", res.StatusCode, res.Header.Get("Content-Encoding"))
	}
	if res.Header.Get("Content-Length") != "" {
		t.Errorf("gzipped response keeps the file's Content-Length %s", res.Header.Get("Content-Length"))
	}
	if !strings.Contains(res.Header.Get("Vary"), "Accept-Encoding") {
		t.Errorf("Vary %q does not name Accept-Encoding", res.Header.Get("Vary"))
	}
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil || !bytes.Equal(plain, page) {
		t.Fatalf("gunzipped body differs from index.html (err %v)", err)
	}
	if len(body) >= len(page) {
		t.Errorf("gzipped body %d B is not smaller than the page %d B", len(body), len(page))
	}
}

func TestPlainWhenGzipNotAccepted(t *testing.T) {
	dir, page := site(t)
	res, body := serve(t, handler(dir), "/", nil)
	if res.Header.Get("Content-Encoding") != "" || !bytes.Equal(body, page) {
		t.Fatalf("GET / without Accept-Encoding: Content-Encoding %q, body %d B; want the file as it is", res.Header.Get("Content-Encoding"), len(body))
	}
}

func TestImagesAndRangesAreNotGzipped(t *testing.T) {
	dir, _ := site(t)
	h := handler(dir)
	if res, _ := serve(t, h, "/hero.webp", map[string]string{"Accept-Encoding": "gzip"}); res.Header.Get("Content-Encoding") != "" {
		t.Errorf("hero.webp got Content-Encoding %q", res.Header.Get("Content-Encoding"))
	}
	res, body := serve(t, h, "/", map[string]string{"Accept-Encoding": "gzip", "Range": "bytes=0-9"})
	if res.StatusCode != http.StatusPartialContent || res.Header.Get("Content-Encoding") != "" || string(body) != "<!doctype " {
		t.Errorf("range request = %d, Content-Encoding %q, body %q; want 206, none, the first 10 bytes", res.StatusCode, res.Header.Get("Content-Encoding"), body)
	}
}

// packed is a site as bench/tools/precompress.mjs leaves it: each text file
// with a .br and a .gz beside it. The "brotli" here is a marker, since Go's
// standard library cannot make or read brotli; the server only sends it.
func packed(t *testing.T) (dir string, page, br, gz []byte) {
	t.Helper()
	dir, page = site(t)
	br = []byte("BR:" + string(page[:20]))
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(page)
	zw.Close()
	gz = buf.Bytes()
	for name, b := range map[string][]byte{"index.html.br": br, "index.html.gz": gz, "about/index.html": []byte("<p>about</p>")} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, page, br, gz
}

func TestPrecompressedSendsTheStoredVariant(t *testing.T) {
	dir, page, br, gz := packed(t)
	h, err := precompressed(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		accept, encoding string
		body             []byte
	}{
		{"br, gzip", "br", br},
		{"gzip, deflate, br", "br", br},
		{"gzip", "gzip", gz},
		{"br;q=0, gzip", "gzip", gz},
		{"", "", page},
	} {
		res, body := serve(t, h, "/", map[string]string{"Accept-Encoding": c.accept})
		if res.StatusCode != 200 || res.Header.Get("Content-Encoding") != c.encoding || !bytes.Equal(body, c.body) {
			t.Errorf("Accept-Encoding %q: %d, Content-Encoding %q, %d B; want 200 %q %d B", c.accept, res.StatusCode, res.Header.Get("Content-Encoding"), len(body), c.encoding, len(c.body))
		}
		if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
			t.Errorf("Accept-Encoding %q: Content-Type %q", c.accept, got)
		}
		if got := res.Header.Get("Content-Length"); got != strconv.Itoa(len(c.body)) {
			t.Errorf("Accept-Encoding %q: Content-Length %q, want %d", c.accept, got, len(c.body))
		}
		if !strings.Contains(res.Header.Get("Vary"), "Accept-Encoding") {
			t.Errorf("Accept-Encoding %q: Vary %q", c.accept, res.Header.Get("Vary"))
		}
	}
}

func TestPrecompressedPaths(t *testing.T) {
	dir, _, _, _ := packed(t)
	h, err := precompressed(dir)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{
		"/": 200, "/index.html": 200, "/about/": 200, "/about": 200, "/hero.webp": 200,
		"/nope": 404, "/index.html.br": 404, "/index.html.gz": 404,
	} {
		if res, _ := serve(t, h, path, map[string]string{"Accept-Encoding": "br"}); res.StatusCode != want {
			t.Errorf("GET %s = %d, want %d", path, res.StatusCode, want)
		}
	}
	// A file with no variants beside it goes out as it is.
	res, body := serve(t, h, "/about/", map[string]string{"Accept-Encoding": "br, gzip"})
	if res.Header.Get("Content-Encoding") != "" || string(body) != "<p>about</p>" {
		t.Errorf("GET /about/: Content-Encoding %q, body %q", res.Header.Get("Content-Encoding"), body)
	}
	if res, _ := serve(t, h, "/hero.webp", map[string]string{"Accept-Encoding": "br"}); res.Header.Get("Content-Type") != "image/webp" {
		t.Errorf("hero.webp Content-Type %q", res.Header.Get("Content-Type"))
	}
}

func TestPrecompressedReadsTheFolderOnce(t *testing.T) {
	dir, _, br, _ := packed(t)
	h, err := precompressed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html.br"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, body := serve(t, h, "/", map[string]string{"Accept-Encoding": "br"}); !bytes.Equal(body, br) {
		t.Errorf("the body followed the file on disk: %q", body)
	}
}
