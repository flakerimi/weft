// Command static serves a folder the plain way, for the benchmark in bench/:
// net/http's FileServer, with text gzipped on the fly for a client that
// accepts gzip. It is the server Astro's static output gets besides
// `astro preview`, so Astro is not measured only behind its preview server.
//
// Usage: static -dir bench/astro/dist [-addr 127.0.0.1:4302]
package main

import (
	"compress/gzip"
	"flag"
	"log"
	"mime"
	"net/http"
	"strings"
	"sync"
)

func handler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			files.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		files.ServeHTTP(gw, r)
	})
}

func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			return strings.ReplaceAll(params, " ", "") != "q=0"
		}
	}
	return false
}

// compressible is a Content-Type worth gzipping: text, scripts, JSON and SVG.
func compressible(contentType string) bool {
	t, _, _ := mime.ParseMediaType(contentType)
	return strings.HasPrefix(t, "text/") || t == "application/javascript" || t == "application/json" ||
		t == "image/svg+xml" || strings.HasSuffix(t, "+json")
}

var gzipWriters = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

// gzipWriter gzips a 200 response of a compressible type; anything else (a
// range, a redirect, a 304, an image) passes through as the FileServer wrote it.
type gzipWriter struct {
	http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
}

func (g *gzipWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	h := g.Header()
	if code == http.StatusOK && h.Get("Content-Encoding") == "" && compressible(h.Get("Content-Type")) {
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		g.zw = gzipWriters.Get().(*gzip.Writer)
		g.zw.Reset(g.ResponseWriter)
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.zw != nil {
		return g.zw.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) close() {
	if g.zw != nil {
		g.zw.Close()
		gzipWriters.Put(g.zw)
	}
}

func main() {
	dir := flag.String("dir", "", "the folder to serve")
	addr := flag.String("addr", "127.0.0.1:4302", "the address to listen on")
	flag.Parse()
	if *dir == "" {
		log.Fatal("static: -dir is required")
	}
	log.Printf("static: %s on http://%s", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, handler(*dir)))
}
