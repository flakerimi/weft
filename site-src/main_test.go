package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// The parts of the build's manifest.json and routes.json these tests read.
type builtFiles struct {
	Files map[string]struct {
		Br bool `json:"br"`
	} `json:"files"`
}

type builtRoutes struct {
	Pages []struct {
		Path    string   `json:"path"`
		HTML    string   `json:"html"`
		Islands []string `json:"islands"`
	} `json:"pages"`
	NotFound *string `json:"notFound"`
}

func dist(t *testing.T) fs.FS {
	t.Helper()
	d, err := fs.Sub(siteDist, "dist")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func readJSON(t *testing.T, d fs.FS, name string, v any) {
	t.Helper()
	b, err := fs.ReadFile(d, name)
	if err != nil {
		t.Fatalf("%s: %v (run scripts/site.sh build)", name, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

// brotliSize is what the server sends for the manifest's file key to a client
// that accepts brotli: the stored .br variant.
func brotliSize(t *testing.T, d fs.FS, m builtFiles, key string) int {
	t.Helper()
	f, ok := m.Files[key]
	if !ok || !f.Br {
		t.Fatalf("manifest.json lists no brotli variant of %s", key)
	}
	b, err := fs.ReadFile(d, strings.TrimPrefix(key, "/")+".br")
	if err != nil {
		t.Fatal(err)
	}
	return len(b)
}

var (
	stylesheetLink = regexp.MustCompile(`<link rel="stylesheet" href="([^"]+)">`)
	moduleScript   = regexp.MustCompile(`<script type="module" src="([^"]+)">`)
	anyScript      = regexp.MustCompile(`<script\b[^>]*>`)
)

// The home page has one island, the copy button. Brotli, its HTML and CSS are
// at most 8 KB; with every script the client build made (the island runtime,
// Vue's Vapor runtime and the island's chunk) at most 28 KB. The CSS is the
// whole of weft.css, every component's rules whether the page uses them or
// not (2026-11-21: the form controls took it from 6 KB and 26 KB to these),
// until the page build sends only the CSS of the components a page renders.
func TestHomePageBudgets(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)

	var html string
	for _, p := range r.Pages {
		if p.Path != "/" {
			continue
		}
		if strings.Join(p.Islands, ",") != "CopyCommand" {
			t.Fatalf("home page islands %v, want [CopyCommand]", p.Islands)
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html = string(b)
		if n := strings.Count(html, "<weft-island"); n != 1 {
			t.Fatalf("home page HTML has %d islands, want 1", n)
		}
		if n := len(moduleScript.FindAllString(html, -1)); n != 1 {
			t.Fatalf("home page HTML has %d module scripts, want 1", n)
		}
		static := brotliSize(t, d, m, "/"+p.HTML)
		for _, link := range stylesheetLink.FindAllStringSubmatch(html, -1) {
			static += brotliSize(t, d, m, link[1])
		}
		scripts := 0
		for key := range m.Files {
			if strings.HasSuffix(key, ".js") {
				scripts += brotliSize(t, d, m, key)
			}
		}
		t.Logf("home page, brotli: HTML+CSS %d B, scripts %d B, total %d B", static, scripts, static+scripts)
		if static > 8*1024 {
			t.Errorf("home page without its island is %d B brotli, over 8 KB", static)
		}
		if static+scripts > 28*1024 {
			t.Errorf("home page with its island is %d B brotli, over 28 KB", static+scripts)
		}
	}
	if html == "" {
		t.Fatal("routes.json has no page at /")
	}
}

// A page with no island loads no JavaScript; JSON script types are allowed.
func TestNotFoundPageLoadsNoJavaScript(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	if r.NotFound == nil {
		t.Fatal("routes.json has no not-found page")
	}
	b, err := fs.ReadFile(d, *r.NotFound)
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range anyScript.FindAllString(string(b), -1) {
		if !strings.Contains(tag, `type="speculationrules"`) && !strings.Contains(tag, `type="application/json"`) {
			t.Errorf("not-found page has %s", tag)
		}
	}
}

func get(t *testing.T, srv *httptest.Server, path string, header ...string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func TestSiteServes(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()

	res, body := get(t, srv, "/", "Accept-Encoding", "identity")
	if res.StatusCode != 200 || !strings.Contains(body, `<weft-island data-id="CopyCommand"`) ||
		!strings.Contains(body, "<title>Weft — one UI system for Go and Vue apps</title>") {
		t.Fatalf("GET /: %d %.300s", res.StatusCode, body)
	}

	res, _ = get(t, srv, "/", "Accept-Encoding", "br, gzip")
	etag := res.Header.Get("Etag")
	if res.Header.Get("Content-Encoding") != "br" || etag == "" {
		t.Fatalf("GET / with br: Content-Encoding %q, ETag %q", res.Header.Get("Content-Encoding"), etag)
	}
	if res, _ = get(t, srv, "/", "Accept-Encoding", "br, gzip", "If-None-Match", etag); res.StatusCode != 304 {
		t.Fatalf("GET / with If-None-Match %s: %d, want 304", etag, res.StatusCode)
	}

	// The installer is staged from site/install by scripts/site.sh build.
	if res, body = get(t, srv, "/install"); res.StatusCode != 200 || !strings.HasPrefix(body, "#!/bin/sh") {
		t.Fatalf("GET /install: %d %.80q", res.StatusCode, body)
	}

	for _, p := range []string{"/routes.json", "/manifest.json", "/pages/index/index.html", "/nothing"} {
		if res, body = get(t, srv, p); res.StatusCode != 404 || !strings.Contains(body, "Not found") {
			t.Errorf("GET %s: %d, want the 404 page", p, res.StatusCode)
		}
	}

	// Releases are staged only where site/releases has them (not in CI).
	var m builtFiles
	readJSON(t, dist(t), "manifest.json", &m)
	if _, ok := m.Files["/releases/latest"]; ok {
		res, body = get(t, srv, "/releases/latest")
		v := strings.TrimSpace(body)
		if res.StatusCode != 200 || v == "" {
			t.Fatalf("GET /releases/latest: %d %q", res.StatusCode, body)
		}
		if res, _ = get(t, srv, "/releases/"+v+"/SHA256SUMS"); res.StatusCode != 200 {
			t.Fatalf("GET /releases/%s/SHA256SUMS: %d", v, res.StatusCode)
		}
	}
}
