package main

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// stored is one file held in memory: its bytes, and the brotli and gzip
// that bench/tools/precompress.mjs left beside it, when it did.
type stored struct {
	contentType   string
	plain, br, gz []byte
}

// precompressed serves a folder that bench/tools/precompress.mjs prepared,
// the way weft.Serve serves a Weft build: every file is read into memory
// once, here, and a request is answered with the stored brotli or gzip the
// client accepts, or the file as it is. Nothing is compressed per request.
// It is how the benchmark puts every framework's static output on the same
// serving terms as Weft. A folder's index.html answers at the folder's path,
// with or without the slash.
func precompressed(dir string) (http.Handler, error) {
	files := map[string]*stored{}
	variants := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		name := "/" + filepath.ToSlash(rel)
		if strings.HasSuffix(name, ".br") || strings.HasSuffix(name, ".gz") {
			variants[name] = b
			return nil
		}
		t := mime.TypeByExtension(path.Ext(name))
		if t == "" {
			t = http.DetectContentType(b)
		}
		files[name] = &stored{contentType: t, plain: b}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for name, f := range files {
		f.br, f.gz = variants[name+".br"], variants[name+".gz"]
		delete(variants, name+".br")
		delete(variants, name+".gz")
	}
	// A .br or .gz with no file beside it is a file of its own.
	for name, b := range variants {
		files[name] = &stored{contentType: "application/octet-stream", plain: b}
	}
	for name, f := range files {
		if folder, ok := strings.CutSuffix(name, "/index.html"); ok {
			files[folder+"/"] = f
			if folder != "" {
				files[folder] = f
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := files[r.URL.Path]
		if f == nil {
			http.NotFound(w, r)
			return
		}
		h := w.Header()
		body := f.plain
		if f.br != nil || f.gz != nil {
			h.Set("Vary", "Accept-Encoding")
			br, gz := accepts(r.Header.Get("Accept-Encoding"))
			switch {
			case br && f.br != nil:
				body = f.br
				h.Set("Content-Encoding", "br")
			case gz && f.gz != nil:
				body = f.gz
				h.Set("Content-Encoding", "gzip")
			}
		}
		h.Set("Content-Type", f.contentType)
		h.Set("Content-Length", strconv.Itoa(len(body)))
		if r.Method != http.MethodHead {
			w.Write(body)
		}
	}), nil
}

// accepts reports whether an Accept-Encoding header takes brotli and gzip.
func accepts(header string) (br, gz bool) {
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		ok := strings.ReplaceAll(params, " ", "") != "q=0"
		switch strings.ToLower(strings.TrimSpace(coding)) {
		case "br":
			br = ok
		case "gzip":
			gz = ok
		}
	}
	return br, gz
}
