// Package warp is the Go side of a Weft app: its modules, typed routes and serving.
package warp

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
)

// appsWithRoutes counts the Apps in this process that have a route, which
// the package-level Serve cannot describe.
var appsWithRoutes atomic.Int64

// describeErr is where Serve says why it cannot describe the app; a test
// puts another in its place.
var describeErr io.Writer = os.Stderr

const (
	cacheImmutable = "public, max-age=31536000, immutable"
	cacheNone      = "no-cache"
)

// Serve returns a handler for a Weft application. Paths under /api/ go to api.
//
// When dist is a page build (it has routes.json, and the manifest.json that
// the page build, weft-build-pages, writes), the whole build is read into
// memory once, here, and each request is answered from it:
//   - a page at its path, with or without a trailing slash;
//   - a file the manifest lists, at its path;
//   - a path under the app's prefix, except under its assets/, gets the
//     app's index.html;
//   - a path that a runtime page's pattern matches (routes.json lists it as
//     "/products/:id" with "runtime": true) gets that page's one shell, with
//     status 200: the first such page in routes.json's order. ":name" matches
//     one segment, ":...name" the rest of the path or none of it, and a
//     segment of the pattern is compared byte for byte;
//   - anything else gets the not-found page with status 404, or a plain 404.
//
// Each answer is the stored brotli or gzip variant the client accepts best,
// or the file as it is, with the manifest's ETag (a variant's with -br or -gz
// added), so If-None-Match gets a 304. Hashed files are cached for a year and
// the rest revalidated. A page with Early Hints first gets a 103 with its Link
// headers. The build's own files (manifest.json, routes.json, .weft-build and
// the page files under /pages/) are never served as they are. Serve panics,
// naming the file, if the build is incomplete.
//
// Otherwise dist is a single-page app, and a manifest.json in it is just a
// file. Files in dist are served as they are, and every other path gets
// dist's index.html so the Vue router can handle it. Files under /assets/ are
// content-hashed by Vite, so they are cached for a year; everything else is
// revalidated. A missing file under /assets/ is a 404, never index.html.
// Serve panics if dist has no index.html, or is nil.
//
// Options add to what is served: WithFiles serves a folder on disk beside dist.
//
// With WARP_DESCRIBE=1 in the environment, as weft gen, weft routes and weft
// build run an app, Serve writes the description of an App with no routes to
// standard output and ends the process with status 0, so an app that serves
// without a warp.App describes itself as having no typed routes and never
// listens while weft waits. Once an App in the process has a route, which
// Serve cannot see, it says to call the App's own Serve instead and ends the
// process with status 1, as a registration mistake does. An App's own Serve,
// Run and Start describe it themselves.
func Serve(dist fs.FS, api http.Handler, opts ...ServeOption) http.Handler {
	if os.Getenv("WARP_DESCRIBE") == "1" {
		if appsWithRoutes.Load() > 0 {
			fmt.Fprintln(describeErr, "warp: Serve was called from the package and the app's routes are on an App; call the app's own Serve so they can be described.")
			exit(1)
		}
		if err := New(Config{}).describeTo(describeOut); err != nil {
			fmt.Fprintln(describeErr, err)
			exit(1)
		}
		exit(0)
	}
	return serveBuild(dist, api, opts...)
}

// serveBuild is Serve without its WARP_DESCRIBE check, for Run.
func serveBuild(dist fs.FS, api http.Handler, opts ...ServeOption) http.Handler {
	if dist == nil {
		panic("warp.Serve: dist is nil; pass the page build, or serve the API alone with app.Handler().")
	}
	if api == nil {
		api = http.HandlerFunc(noAPI)
	}
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	var pages http.Handler
	var taken func(p string) string
	if manifest, ok := pageManifest(dist); ok {
		s := loadSite(dist, manifest)
		pages, taken = s, s.serves
	} else {
		pages, taken = spa(dist), func(p string) string {
			if fi, err := fs.Stat(dist, p[1:]); err == nil && !fi.IsDir() {
				return "a file"
			}
			return ""
		}
	}
	files := loadFolders(o.folders, taken)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		// A folder's file is only its exact path, as a build's file is.
		if len(files) > 0 && !strings.HasSuffix(r.URL.Path, "/") {
			if f := files[path.Clean("/"+r.URL.Path)]; f != nil && f.serve(w, r) {
				return
			}
		}
		pages.ServeHTTP(w, r)
	})
}

// spa serves a single-page app's dist, as Serve describes.
func spa(dist fs.FS) http.Handler {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		panic("warp.Serve: dist has no index.html")
	}
	files := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		isAsset := strings.HasPrefix(name, "assets/")
		if name != "" && name != "index.html" {
			if fi, err := fs.Stat(dist, name); err == nil && !fi.IsDir() {
				if isAsset {
					w.Header().Set("Cache-Control", cacheImmutable)
				} else {
					w.Header().Set("Cache-Control", cacheNone)
				}
				files.ServeHTTP(w, r)
				return
			}
			if isAsset {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheNone)
		w.Write(index)
	})
}

func noAPI(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, NotFound("This app has no API."))
}
