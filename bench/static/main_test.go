package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
