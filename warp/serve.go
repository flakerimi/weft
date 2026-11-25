// Package warp is the Go side of a Weft app: its modules, typed routes and serving.
package warp

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

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
// Serve panics if dist has no index.html.
//
// Options add to what is served: WithFiles serves a folder on disk beside dist.
func Serve(dist fs.FS, api http.Handler, opts ...ServeOption) http.Handler {
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
