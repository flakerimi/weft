package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
// at most 6 KB; with every script it loads (the island runtime with Vue's
// Vapor runtime, the chunks that imports, and the island's chunk) at most
// 26 KB. The CSS is the theme, base.css and the stylesheets of the
// components the page renders, so adding components to @weft/ui does not
// grow it; nor do the gallery's islands grow its runtime, which they do not
// share (the gallery has a runtime of its own, --separate-runtime).
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
		for _, key := range pageScripts(t, d, m, html, p.Islands) {
			scripts += brotliSize(t, d, m, key)
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

var modulePreload = regexp.MustCompile(`<link rel="modulepreload" href="([^"]+)">`)

// pageScripts are every script a page loads: its module script (the runtime
// entry), the modules it preloads (what the entry imports), and its
// islands' chunks with the chunks they import.
func pageScripts(t *testing.T, d fs.FS, m builtFiles, html string, islands []string) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, s := range moduleScript.FindAllStringSubmatch(html, -1) {
		seen[s[1]] = true
	}
	for _, s := range modulePreload.FindAllStringSubmatch(html, -1) {
		seen[s[1]] = true
	}
	for _, c := range islandChunks(t, d, m, islands) {
		seen[c] = true
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
	// The binary is the Go program (about 6 MB) and the build it embeds, which
	// grows with the site's pages; a release in it would show as more.
	var built int64
	fs.WalkDir(dist(t), ".", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			if info, err := e.Info(); err == nil {
				built += info.Size()
			}
		}
		return nil
	})
	t.Logf("site binary: %.1f MB, of which the build %.1f MB", float64(fi.Size())/(1<<20), float64(built)/(1<<20))
	if fi.Size()-built >= 8<<20 {
		t.Fatalf("the site binary is %.1f MB, %.1f MB more than the build it embeds, not under 8 MB", float64(fi.Size())/(1<<20), float64(fi.Size()-built)/(1<<20))
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

// docsPages are the routes.json entries of the docs, by path.
func docsPages(t *testing.T, r builtRoutes) map[string]string {
	t.Helper()
	pages := map[string]string{}
	for _, p := range r.Pages {
		if p.Path == "/docs" || strings.HasPrefix(p.Path, "/docs/") {
			pages[p.Path] = p.HTML
		}
	}
	return pages
}

// A doc as content/docs/ has it: its route, title and order.
type doc struct {
	path, title string
	order       int
}

var (
	frontTitle = regexp.MustCompile(`(?m)^title: (.+)$`)
	frontOrder = regexp.MustCompile(`(?m)^order: (\d+)$`)
)

// contentDocs reads content/docs/**/*.md, in the sidebar's order.
func contentDocs(t *testing.T) []doc {
	t.Helper()
	var docs []doc
	err := filepath.WalkDir("content/docs", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		slug := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(p), "content/docs/"), ".md")
		slug = strings.TrimSuffix(strings.TrimSuffix(slug, "index"), "/")
		route := "/docs"
		if slug != "" {
			route += "/" + slug
		}
		title, order := frontTitle.FindSubmatch(b), frontOrder.FindSubmatch(b)
		if title == nil || order == nil {
			t.Fatalf("%s has no title or no order; every doc has both, so the sidebar can place it", p)
		}
		n := 0
		fmt.Sscan(string(order[1]), &n)
		docs = append(docs, doc{route, string(title[1]), n})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// As the build orders them: by order, then by slug.
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].order != docs[j].order {
			return docs[i].order < docs[j].order
		}
		return docs[i].path < docs[j].path
	})
	return docs
}

var chunkImport = regexp.MustCompile(`(?:from|import)\s*"\./([^"]+\.js)"`)

// islandChunks are the scripts a page loads for its islands, the shared
// runtime (client-*.js) aside: each island's chunk, /assets/<id>-<hash>.js,
// and the chunks it imports.
func islandChunks(t *testing.T, d fs.FS, m builtFiles, islands []string) []string {
	t.Helper()
	seen := map[string]bool{}
	var walk func(key string)
	walk = func(key string) {
		if seen[key] || strings.HasPrefix(key, "/assets/client-") {
			return
		}
		seen[key] = true
		b, err := fs.ReadFile(d, strings.TrimPrefix(key, "/"))
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range chunkImport.FindAllStringSubmatch(string(b), -1) {
			walk("/assets/" + imp[1])
		}
	}
	for _, id := range islands {
		found := false
		for key := range m.Files {
			if strings.HasPrefix(key, "/assets/"+id+"-") && strings.HasSuffix(key, ".js") {
				walk(key)
				found = true
			}
		}
		if !found {
			t.Fatalf("the build has no chunk for the island %s", id)
		}
	}
	var out []string
	for key := range seen {
		out = append(out, key)
	}
	return out
}

// Each docs page is at most 10 KB brotli: its HTML, with its CSS inlined, any
// stylesheet it links, and the scripts of its own islands (each island's
// chunk and the chunks it imports). The shared runtime, client-*.js, is not
// counted: it is cached once for the whole site and loads only on pages with
// an island. Nor are fonts: they are cached and preloaded. A docs page loads
// no JavaScript, except one with a command marked copy, whose only island is
// the copy button.
func TestDocsPageBudgets(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	islands := map[string][]string{}
	for _, p := range r.Pages {
		islands[p.Path] = p.Islands
	}
	pages := docsPages(t, r)
	if len(pages) == 0 {
		t.Fatal("routes.json has no docs pages")
	}
	for path, file := range pages {
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		size := brotliSize(t, d, m, "/"+file)
		for _, link := range stylesheetLink.FindAllStringSubmatch(html, -1) {
			size += brotliSize(t, d, m, link[1])
		}
		for _, chunk := range islandChunks(t, d, m, islands[path]) {
			size += brotliSize(t, d, m, chunk)
		}
		t.Logf("%s: %d B brotli", path, size)
		if size > 10*1024 {
			t.Errorf("%s is %d B brotli, over 10 KB", path, size)
		}
		switch got := strings.Join(islands[path], ","); got {
		case "":
			for _, tag := range anyScript.FindAllString(html, -1) {
				if !strings.Contains(tag, `type="speculationrules"`) && !strings.Contains(tag, `type="application/json"`) {
					t.Errorf("%s has no island but has %s", path, tag)
				}
			}
		case "CopyCommand":
		default:
			t.Errorf("%s has the islands %s; a docs page has none, or only CopyCommand", path, got)
		}
	}
}

// Every doc in content/docs/ is a page, and each page has the whole sidebar
// in the docs' order with itself current, its title as the h1, its table of
// contents, and links to the docs before and after it.
func TestDocsLayout(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	pages := docsPages(t, r)
	docs := contentDocs(t)
	if len(pages) != len(docs) {
		t.Errorf("%d docs pages built for %d files in content/docs", len(pages), len(docs))
	}
	sidebar := regexp.MustCompile(`(?s)<aside id="docs-nav".*?</aside>`)
	navLink := regexp.MustCompile(`<a class="weft-link weft-nav__item" href="([^"]+)"( aria-current="page")?`)
	for i, doc := range docs {
		file, ok := pages[doc.path]
		if !ok {
			t.Errorf("content/docs has %s, but no page was built at %s", doc.title, doc.path)
			continue
		}
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		var order, current []string
		for _, l := range navLink.FindAllStringSubmatch(sidebar.FindString(html), -1) {
			order = append(order, l[1])
			if l[2] != "" {
				current = append(current, l[1])
			}
		}
		var want []string
		for _, o := range docs {
			want = append(want, o.path)
		}
		if strings.Join(order, " ") != strings.Join(want, " ") {
			t.Errorf("%s: the sidebar is %v, want %v", doc.path, order, want)
		}
		if len(current) != 1 || current[0] != doc.path {
			t.Errorf("%s: the sidebar marks %v current", doc.path, current)
		}
		if !strings.Contains(html, `<h1 class="docs-title">`+doc.title+`</h1>`) {
			t.Errorf("%s: no h1 %q", doc.path, doc.title)
		}
		if strings.Count(html, `<h2 id="`) > 0 && !strings.Contains(html, `<p id="docs-toc-title">On this page</p>`) {
			t.Errorf("%s has headings but no On this page", doc.path)
		}
		if i > 0 && !strings.Contains(html, `href="`+docs[i-1].path+`" rel="prev"`) {
			t.Errorf("%s: no link back to %s", doc.path, docs[i-1].path)
		}
		if i+1 < len(docs) && !strings.Contains(html, `href="`+docs[i+1].path+`" rel="next"`) {
			t.Errorf("%s: no link on to %s", doc.path, docs[i+1].path)
		}
		// The header marks the docs as the section, and only /docs itself as the page.
		mark := `aria-current="true"`
		if doc.path == "/docs" {
			mark = `aria-current="page"`
		}
		header := regexp.MustCompile(`(?s)<nav class="site-nav".*?</nav>`).FindString(html)
		if !regexp.MustCompile(`<a [^>]*href="/docs"[^>]*` + mark).MatchString(header) {
			t.Errorf("%s: the header's Docs link is not %s", doc.path, mark)
		}
		// A keyboard reaches the text first.
		if !strings.Contains(html, `<a class="site-skip" href="#main">`) || !regexp.MustCompile(`<main [^>]*id="main"`).MatchString(html) {
			t.Errorf("%s: no skip link to the main content", doc.path)
		}
		// A table keeps its semantics and scrolls inside a container of its own.
		if strings.Count(html, "<table>") != strings.Count(html, `<div class="docs-table"><table>`) {
			t.Errorf("%s: a table is not in a scroll container", doc.path)
		}
		// The narrow screen's drawer needs no script: the Menu button opens the sidebar as a popover.
		if !strings.Contains(html, `popovertarget="docs-nav"`) || !strings.Contains(html, `<aside id="docs-nav" class="docs-nav" popover>`) {
			t.Errorf("%s: the sidebar is not a popover the Menu button opens", doc.path)
		}
	}
}

// Every page preloads the site's two fonts, served from the build under
// hashed names and cached for good; nothing comes from another site.
func TestDocsFontsAreSelfHosted(t *testing.T) {
	d := dist(t)
	var m struct {
		Files map[string]struct {
			Immutable bool `json:"immutable"`
		} `json:"files"`
	}
	readJSON(t, d, "manifest.json", &m)
	b, err := fs.ReadFile(d, "pages/docs/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	preload := regexp.MustCompile(`<link rel="preload" href="(/assets/[^"]+\.woff2)" as="font" type="font/woff2" crossorigin>`)
	fonts := preload.FindAllStringSubmatch(html, -1)
	var names []string
	for _, f := range fonts {
		names = append(names, f[1])
		if !m.Files[f[1]].Immutable {
			t.Errorf("%s is not in the build, cached for good", f[1])
		}
	}
	if len(fonts) != 2 || !strings.Contains(names[0], "bricolage-grotesque-latin") || !strings.Contains(names[1], "jetbrains-mono-latin") {
		t.Errorf("the docs preload %v, want Bricolage Grotesque and JetBrains Mono", names)
	}
	// Nor does it link the repository while that is private (WEFT_REPO_PUBLIC=1 builds the link in).
	if regexp.MustCompile(`(?i)(src|href)="https?:|url\(https?:`).MatchString(html) {
		t.Error("the docs page links or loads something on another site")
	}
}

// The wordmark is the owner's file, byte for byte.
func TestWordmarkIsTheOwners(t *testing.T) {
	want, err := os.ReadFile("../docs/design/weft-wordmark.svg")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"public/weft.svg", "dist/weft.svg"} {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s is not docs/design/weft-wordmark.svg", name)
		}
	}
}

// The site's stylesheets draw with the theme's tokens: no colour of their own,
// so light, dark and data-weft-scheme follow the theme.
func TestSiteStylesUseTokens(t *testing.T) {
	colour := regexp.MustCompile(`#[0-9A-Fa-f]{3,8}\b|\b(?:rgba?|hsla?|oklch|lab)\(`)
	for _, name := range []string{"styles/site.css", "styles/shell.css", "styles/docs.css", "styles/gallery.css", "styles/gallery-index.css", "styles/stage.css", "components/CopyCode.vue"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if colour.MatchString(line) {
				t.Errorf("%s:%d draws with a colour of its own: %s", name, i+1, strings.TrimSpace(line))
			}
		}
	}
}
