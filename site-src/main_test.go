package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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
// at most 6 KB; with every script the client build made (the island runtime,
// Vue's Vapor runtime and the island's chunk) at most 26 KB. The CSS is the
// theme, base.css and the stylesheets of the components the page renders,
// so adding components to @weft/ui does not grow it.
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
		if static > 6*1024 {
			t.Errorf("home page without its island is %d B brotli, over 6 KB", static)
		}
		if static+scripts > 26*1024 {
			t.Errorf("home page with its island is %d B brotli, over 26 KB", static+scripts)
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

// stagedReleases writes a folder like the one scripts/stage-releases.sh
// stages into the image: the installer and one release.
func stagedReleases(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range map[string]string{
		"install":                                "#!/bin/sh\necho weft\n",
		"releases/latest":                        "0.0.9\n",
		"releases/0.0.9/SHA256SUMS":              "abc  weft_linux_amd64.tar.gz\n",
		"releases/0.0.9/weft_linux_amd64.tar.gz": "tarball",
	} {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func siteHandler(t *testing.T, dir string) http.Handler {
	t.Helper()
	releases, err := releaseFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	return handler(releases)
}

func TestSiteServes(t *testing.T) {
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
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

	for _, p := range []string{"/routes.json", "/manifest.json", "/pages/index/index.html", "/nothing", "/releases/", "/releases/0.0.8/SHA256SUMS"} {
		if res, body = get(t, srv, p); res.StatusCode != 404 || !strings.Contains(body, "Not found") {
			t.Errorf("GET %s: %d, want the 404 page", p, res.StatusCode)
		}
	}
}

// The installer and the releases come from the folder on disk, not the binary.
func TestSiteServesReleasesFromDisk(t *testing.T) {
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	for p, want := range map[string]string{
		"/install":                                "#!/bin/sh\necho weft\n",
		"/releases/latest":                        "0.0.9\n",
		"/releases/0.0.9/SHA256SUMS":              "abc  weft_linux_amd64.tar.gz\n",
		"/releases/0.0.9/weft_linux_amd64.tar.gz": "tarball",
	} {
		res, body := get(t, srv, p)
		if res.StatusCode != 200 || body != want {
			t.Fatalf("GET %s: %d %q", p, res.StatusCode, body)
		}
		etag := res.Header.Get("Etag")
		if etag == "" {
			t.Fatalf("GET %s: no ETag", p)
		}
		if res, _ = get(t, srv, p, "If-None-Match", etag); res.StatusCode != 304 {
			t.Fatalf("GET %s with If-None-Match %s: %d, want 304", p, etag, res.StatusCode)
		}
	}
}

// The folder holds the installer and releases/, as scripts/stage-releases.sh
// writes them, and nothing else: anything else would be served at the root.
func TestReleaseFilesRefuses(t *testing.T) {
	dir := stagedReleases(t)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644)
	if _, err := releaseFiles(dir); err == nil || err.Error() != "the releases folder "+dir+" holds index.html; it holds only install and releases/, as scripts/stage-releases.sh stages them" {
		t.Fatalf("an extra file: %v", err)
	}
	empty := t.TempDir()
	if _, err := releaseFiles(empty); err == nil || err.Error() != "the releases folder "+empty+" has no install; stage it with scripts/stage-releases.sh, or set WEFT_RELEASES_DIR" {
		t.Fatalf("no installer: %v", err)
	}
	missing := filepath.Join(empty, "none")
	if _, err := releaseFiles(missing); err == nil || !strings.HasPrefix(err.Error(), "the releases folder "+missing+" cannot be read (") {
		t.Fatalf("no folder: %v", err)
	}
	// Where site/releases is empty, as in CI, only the installer is staged.
	only := t.TempDir()
	os.WriteFile(filepath.Join(only, "install"), []byte("#!/bin/sh\n"), 0o644)
	if _, err := releaseFiles(only); err != nil {
		t.Fatalf("the installer alone: %v", err)
	}
}

func TestReleasesDirDefault(t *testing.T) {
	t.Setenv("WEFT_RELEASES_DIR", "")
	if got := releasesDir(); got != "/data/releases" {
		t.Fatalf("default %q", got)
	}
	t.Setenv("WEFT_RELEASES_DIR", "/srv/r")
	if got := releasesDir(); got != "/srv/r" {
		t.Fatalf("from the environment %q", got)
	}
}

// The binary embeds the pages only: no installer and no release, so it no
// longer grows by every release's tarballs.
func TestBinaryHoldsNoReleases(t *testing.T) {
	var m builtFiles
	readJSON(t, dist(t), "manifest.json", &m)
	for key := range m.Files {
		if key == "/install" || strings.HasPrefix(key, "/releases/") {
			t.Errorf("the build embeds %s; releases are served from WEFT_RELEASES_DIR", key)
		}
	}
	if testing.Short() {
		t.Skip("builds the binary")
	}
	out := filepath.Join(t.TempDir(), "weft-site")
	// As the Dockerfile scripts/publish-site.sh writes builds it.
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", out, ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, b)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("site binary: %.1f MB", float64(fi.Size())/(1<<20))
	if fi.Size() >= 15<<20 {
		t.Fatalf("the site binary is %.1f MB, not under 15 MB", float64(fi.Size())/(1<<20))
	}
}

// The folder is opened as an os.Root: a file swapped for a symlink that
// leads out of it after the start is not followed, even one whose target
// has the size and time the file had.
func TestSiteDoesNotFollowASymlinkOut(t *testing.T) {
	dir := stagedReleases(t)
	srv := httptest.NewServer(siteHandler(t, dir))
	defer srv.Close()
	latest := filepath.Join(dir, "releases/latest")
	fi, err := os.Stat(latest)
	if err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(secret, []byte("SECRET"), 0o644)
	os.Chtimes(secret, fi.ModTime(), fi.ModTime())
	os.Rename(latest, filepath.Join(t.TempDir(), "latest"))
	if err := os.Symlink(secret, latest); err != nil {
		t.Skip(err)
	}
	res, body := get(t, srv, "/releases/latest")
	if res.StatusCode == 200 || strings.Contains(body, "SECRET") {
		t.Fatalf("GET /releases/latest through a symlink out: %d %q", res.StatusCode, body)
	}
}
