package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// get asks a real server, so the 103 Early Hints a page gets first is read
// as the client reads it, before the final response.
func get(t *testing.T, srv *httptest.Server, path, encoding string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if encoding != "" {
		req.Header.Set("Accept-Encoding", encoding)
	}
	res, err := srv.Client().Transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// The landing the benchmark measures: the counter island, the image, and the
// stored brotli variant for a client that accepts it.
func TestLanding(t *testing.T) {
	h := httptest.NewServer(handler())
	defer h.Close()

	res := get(t, h, "/", "")
	body, _ := io.ReadAll(res.Body)
	html := string(body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d, want 200 (run bench/run.sh or pnpm --filter weft-bench build)", res.StatusCode)
	}
	for _, want := range []string{`<weft-island data-id="Counter" data-when="idle"`, `src="/hero.webp"`, `width="1600"`, "<h1", "Enterprise"} {
		if !strings.Contains(html, want) {
			t.Errorf("GET / has no %q", want)
		}
	}

	if res := get(t, h, "/", "br, gzip"); res.Header.Get("Content-Encoding") != "br" {
		t.Errorf("GET / with brotli accepted: Content-Encoding %q, want br", res.Header.Get("Content-Encoding"))
	}

	res = get(t, h, "/hero.webp", "")
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/webp" {
		t.Errorf("GET /hero.webp = %d %q, want 200 image/webp", res.StatusCode, res.Header.Get("Content-Type"))
	}
}
