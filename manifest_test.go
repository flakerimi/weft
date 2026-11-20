package weft

import (
	"encoding/json"
	"io/fs"
	"maps"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

// siteFile is one file of a test build: its URL path, its bytes, and what the
// manifest says about it. A file with variants gets fake .br and .gz siblings
// ("br:" and "gz:" before its bytes): the server never decodes them.
type siteFile struct {
	path, body, etag, typ string
	variants, immutable   bool
}

var siteFiles = []siteFile{
	{"/pages/index/index.html", "<p>home</p>", `"home"`, "text/html; charset=utf-8", true, false},
	{"/pages/about/index.html", "<p>about us</p>", `"about"`, "text/html; charset=utf-8", true, false},
	{"/pages/counter/index.html", "<p>counter</p>", `"counter"`, "text/html; charset=utf-8", true, false},
	{"/pages/404/index.html", "<p>not here</p>", `"lost"`, "text/html; charset=utf-8", true, false},
	{"/assets/client-abc.js", "export const a=1", `"client"`, "text/javascript; charset=utf-8", true, true},
	{"/app/index.html", `<div id="app"></div>`, `"appindex"`, "text/html; charset=utf-8", true, false},
	{"/app/assets/main-abc.js", "app()", `"appmain"`, "text/javascript; charset=utf-8", true, false},
	{"/robots.txt", "User-agent: *", `"robots"`, "text/plain; charset=utf-8", true, false},
	{"/icon.bin", "\x00\x01\x02", `"icon"`, "application/octet-stream", false, false},
	{"/routes.json", "{}", `"routes"`, "application/json", true, false},
}

const siteRoutes = `{
  "pages": [
    {"path": "/", "html": "pages/index/index.html", "islands": []},
    {"path": "/about", "html": "pages/about/index.html", "islands": []},
    {"path": "/counter", "html": "pages/counter/index.html", "islands": ["Counter"]}
  ],
  "notFound": "pages/404/index.html",
  "app": {"prefix": "/app", "index": "app/index.html"}
}
`

var siteHints = map[string][]string{
	"/counter": {"</assets/client-abc.js>; rel=modulepreload", "</assets/Counter-x.js>; rel=modulepreload"},
	"404":      {"</assets/client-abc.js>; rel=modulepreload"},
}

// pageBuild returns a build as weft build writes it: the files with their
// variants, routes.json, manifest.json and the .weft-build marker.
func pageBuild(t testing.TB) fstest.MapFS {
	t.Helper()
	dist := fstest.MapFS{}
	files := map[string]any{}
	for _, f := range siteFiles {
		name := f.path[1:]
		dist[name] = &fstest.MapFile{Data: []byte(f.body)}
		if f.variants {
			dist[name+".br"] = &fstest.MapFile{Data: []byte("br:" + f.body)}
			dist[name+".gz"] = &fstest.MapFile{Data: []byte("gz:" + f.body)}
		}
		files[f.path] = map[string]any{"etag": f.etag, "type": f.typ, "br": f.variants, "gz": f.variants, "immutable": f.immutable}
	}
	dist["routes.json"] = &fstest.MapFile{Data: []byte(siteRoutes)}
	manifest, err := json.Marshal(map[string]any{"files": files, "hints": siteHints})
	if err != nil {
		t.Fatal(err)
	}
	dist["manifest.json"] = &fstest.MapFile{Data: manifest}
	dist[".weft-build"] = &fstest.MapFile{Data: []byte(`{"weft":1,"files":[]}`)}
	return dist
}

// editManifest returns dist with its manifest.json changed by edit.
func editManifest(t *testing.T, dist fstest.MapFS, edit func(m map[string]any)) fstest.MapFS {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(dist["manifest.json"].Data, &m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	out := maps.Clone(dist)
	out["manifest.json"] = &fstest.MapFile{Data: b}
	return out
}

func without(dist fstest.MapFS, names ...string) fstest.MapFS {
	out := maps.Clone(dist)
	for _, n := range names {
		delete(out, n)
	}
	return out
}

func wantPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != want {
			t.Fatalf("panic %v\nwant  %s", r, want)
		}
	}()
	f()
}

func TestManifestSiteLoads(t *testing.T) {
	// The .weft-build marker is in dist but not in files: it is ignored.
	Serve(pageBuild(t), nil)
}

func TestManifestMissingFilePanics(t *testing.T) {
	wantPanic(t, "weft.Serve: manifest.json lists /assets/client-abc.js, but dist has no assets/client-abc.js.", func() {
		Serve(without(pageBuild(t), "assets/client-abc.js"), nil)
	})
}

func TestManifestMissingVariantPanics(t *testing.T) {
	wantPanic(t, "weft.Serve: manifest.json says /pages/about/index.html has a .br variant, but dist has no pages/about/index.html.br.", func() {
		Serve(without(pageBuild(t), "pages/about/index.html.br"), nil)
	})
	wantPanic(t, "weft.Serve: manifest.json says /robots.txt has a .gz variant, but dist has no robots.txt.gz.", func() {
		Serve(without(pageBuild(t), "robots.txt.gz"), nil)
	})
}

func TestManifestUnreadablePanics(t *testing.T) {
	dist := pageBuild(t)
	dist["manifest.json"] = &fstest.MapFile{Data: []byte(`{"files":`)}
	wantPanic(t, "weft.Serve: dist/manifest.json is not a Weft build manifest (unexpected end of JSON input).", func() {
		Serve(dist, nil)
	})
}

func TestManifestNeedsRoutes(t *testing.T) {
	wantPanic(t, "weft.Serve: dist has manifest.json but no routes.json.", func() {
		Serve(without(pageBuild(t), "routes.json"), nil)
	})
}

func TestManifestBadPathPanics(t *testing.T) {
	for _, key := range []string{"robots.txt", "/", "/.", "/../secret", "/a//b"} {
		dist := editManifest(t, pageBuild(t), func(m map[string]any) {
			m["files"].(map[string]any)[key] = map[string]any{"etag": `"x"`, "type": "text/plain"}
		})
		wantPanic(t, `weft.Serve: manifest.json lists `+key+`, which is not a path like "/assets/app.js".`, func() {
			Serve(dist, nil)
		})
	}
}

func TestManifestBadETagPanics(t *testing.T) {
	dist := editManifest(t, pageBuild(t), func(m map[string]any) {
		m["files"].(map[string]any)["/robots.txt"].(map[string]any)["etag"] = "robots"
	})
	wantPanic(t, `weft.Serve: manifest.json gives /robots.txt the ETag robots, which is not a quoted entity tag like "abc".`, func() {
		Serve(dist, nil)
	})
}

func TestManifestRouteToUnlistedFilePanics(t *testing.T) {
	dist := editManifest(t, pageBuild(t), func(m map[string]any) {
		delete(m["files"].(map[string]any), "/pages/about/index.html")
	})
	wantPanic(t, "weft.Serve: routes.json serves /about from pages/about/index.html, which manifest.json does not list.", func() {
		Serve(dist, nil)
	})
	dist = editManifest(t, pageBuild(t), func(m map[string]any) {
		delete(m["files"].(map[string]any), "/pages/404/index.html")
	})
	wantPanic(t, "weft.Serve: routes.json serves the not-found page from pages/404/index.html, which manifest.json does not list.", func() {
		Serve(dist, nil)
	})
	dist = editManifest(t, pageBuild(t), func(m map[string]any) {
		delete(m["files"].(map[string]any), "/app/index.html")
	})
	wantPanic(t, "weft.Serve: routes.json serves /app from app/index.html, which manifest.json does not list.", func() {
		Serve(dist, nil)
	})
}

// countingFS counts every Open, to show requests never touch dist.
type countingFS struct {
	fs.FS
	opens *atomic.Int64
}

func (c countingFS) Open(name string) (fs.File, error) {
	c.opens.Add(1)
	return c.FS.Open(name)
}

func TestManifestReadsEverythingAtConstruction(t *testing.T) {
	var opens atomic.Int64
	h := Serve(countingFS{pageBuild(t), &opens}, nil)
	before := opens.Load()
	if before == 0 {
		t.Fatal("construction read nothing")
	}
	for _, p := range []string{"/", "/about", "/counter", "/assets/client-abc.js", "/robots.txt", "/app/x", "/nope"} {
		for _, enc := range []string{"", "br", "gzip"} {
			do(t, h, "GET", p, "Accept-Encoding", enc)
		}
	}
	if after := opens.Load(); after != before {
		t.Fatalf("requests opened dist %d times", after-before)
	}
}
