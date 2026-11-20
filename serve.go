package weft

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

// Serve returns a handler for a Weft application: paths under /api/ go to api,
// files in dist are served as they are, and every other path gets dist's
// index.html so the Vue router can handle it. Files under /assets/ are
// content-hashed by Vite, so they are cached for a year; everything else is
// revalidated. A missing file under /assets/ is a 404, never index.html.
//
// It panics if dist has no index.html.
func Serve(dist fs.FS, api http.Handler) http.Handler {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		panic("weft.Serve: dist has no index.html")
	}
	if api == nil {
		api = http.HandlerFunc(noAPI)
	}
	files := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
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

func noAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":{"code":"not_found","message":"no API"}}` + "\n"))
}
