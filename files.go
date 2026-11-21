package weft

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

// An Option changes what Serve serves.
type Option func(*options)

type options struct {
	folders []folder
}

// A folder is one WithFiles: the files of dir, served under prefix.
type folder struct {
	prefix string
	dir    fs.FS
}

// WithFiles serves the files of dir, a folder on disk that does not change
// while the server runs, under prefix: WithFiles("/releases/", d) serves d's
// 0.0.1/SHA256SUMS at /releases/0.0.1/SHA256SUMS. prefix starts and ends with
// "/"; "/" serves dir's files at their own paths.
//
// Serve reads every file once, at start, for its ETag (the first 16 hex
// digits of its SHA-256, as the page build gives its files) and its type, then
// reads it from disk for each request, so a large folder is not held in
// memory. A request whose If-None-Match has the ETag gets a 304. A file is
// served at its exact path only, never as a folder listing; one changed on
// disk since the start is refused with a 500 rather than sent with a stale
// ETag, and one added since is not served. Serve panics, naming the file, if
// dir holds a symlink, or if a file's path is also a page or a file of dist,
// another folder's file, or under /api/. A dot-named entry (.DS_Store, .git)
// is refused the same way: it is never meant to be published.
func WithFiles(prefix string, dir fs.FS) Option {
	return func(o *options) { o.folders = append(o.folders, folder{prefix, dir}) }
}

// A diskFile is one file of a WithFiles folder, as it was at start.
type diskFile struct {
	dir     fs.FS
	name    string // its path in dir
	etag    string
	typ     string
	size    int64
	modTime time.Time
}

// diskFiles is every WithFiles file by its URL path.
type diskFiles map[string]*diskFile

// loadFolders reads the folders' files. taken says whether dist already
// serves a path, and as what ("a page" or "a file"), or "" when it does not.
func loadFolders(folders []folder, taken func(p string) string) diskFiles {
	files := diskFiles{}
	for _, f := range folders {
		if !strings.HasPrefix(f.prefix, "/") || !strings.HasSuffix(f.prefix, "/") ||
			(f.prefix != "/" && path.Clean(f.prefix)+"/" != f.prefix) {
			panic(fmt.Sprintf(`weft.WithFiles: the prefix %q is not a path that starts and ends with "/", like "/releases/".`, f.prefix))
		}
		err := fs.WalkDir(f.dir, ".", func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			switch {
			case name != "." && strings.HasPrefix(d.Name(), "."):
				panic(fmt.Sprintf("weft.WithFiles: %s in the folder served at %s is named with a leading dot; move it out.", name, f.prefix))
			case d.Type()&fs.ModeSymlink != 0:
				panic(fmt.Sprintf("weft.WithFiles: %s in the folder served at %s is a symlink; put the file itself there.", name, f.prefix))
			case d.IsDir():
				return nil
			case !d.Type().IsRegular():
				panic(fmt.Sprintf("weft.WithFiles: %s in the folder served at %s is not a file or a folder; move it out.", name, f.prefix))
			}
			p := f.prefix + name
			switch {
			case p == "/api" || strings.HasPrefix(p, "/api/"):
				panic(fmt.Sprintf("weft.WithFiles: %s is under /api/, which goes to the API; serve the folder at another prefix.", p))
			case files[p] != nil:
				panic(fmt.Sprintf("weft.WithFiles: %s is served from two folders; serve it from one place only.", p))
			}
			if what := taken(p); what != "" {
				panic(fmt.Sprintf("weft.WithFiles: %s is %s of the build too; serve it from one place only.", p, what))
			}
			file, err := readDiskFile(f.dir, name)
			if err != nil {
				return err
			}
			files[p] = file
			return nil
		})
		if err != nil {
			panic(fmt.Sprintf("weft.WithFiles: the folder served at %s cannot be read (%v).", f.prefix, err))
		}
	}
	return files
}

// readDiskFile hashes the file name in dir and works out its type.
func readDiskFile(dir fs.FS, name string) (*diskFile, error) {
	f, err := dir.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	head := &prefixWriter{max: 512}
	if _, err := io.Copy(io.MultiWriter(h, head), f); err != nil {
		return nil, err
	}
	sum := h.Sum(nil)
	return &diskFile{
		dir:     dir,
		name:    name,
		etag:    `"` + hex.EncodeToString(sum[:8]) + `"`,
		typ:     diskType(name, head.b),
		size:    info.Size(),
		modTime: info.ModTime(),
	}, nil
}

// prefixWriter keeps the first max bytes written to it.
type prefixWriter struct {
	b   []byte
	max int
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	if n := w.max - len(w.b); n > 0 {
		w.b = append(w.b, p[:min(n, len(p))]...)
	}
	return len(p), nil
}

// diskType is the Content-Type of a file: from its extension, or for a name
// without one (an installer, SHA256SUMS) from its first bytes.
func diskType(name string, head []byte) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".gz", ".tgz":
		return "application/gzip"
	case "":
		return http.DetectContentType(head)
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}

// serve answers a GET or HEAD for f from disk, and reports false when f is no
// longer there, so the request can get the not-found answer instead.
func (f *diskFile) serve(w http.ResponseWriter, r *http.Request) bool {
	file, err := f.dir.Open(f.name)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	h := w.Header()
	h.Set("Vary", "Accept-Encoding")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if err == nil {
			file.Close()
		}
		h.Set("Allow", "GET, HEAD")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return true
	}
	if err != nil {
		http.Error(w, "500 the file cannot be read", http.StatusInternalServerError)
		return true
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() != f.size || !info.ModTime().Equal(f.modTime) {
		// Its ETag was computed from other bytes: never send it with these.
		http.Error(w, "500 the file changed since the server started; restart the server", http.StatusInternalServerError)
		return true
	}
	content, ok := file.(io.ReadSeeker)
	if !ok {
		b, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "500 the file cannot be read", http.StatusInternalServerError)
			return true
		}
		content = bytes.NewReader(b)
	}
	h.Set("Etag", f.etag)
	h.Set("Cache-Control", cacheNone)
	h.Set("Content-Type", f.typ)
	// No modification time: the ETag alone says whether the client has it.
	http.ServeContent(w, r, "", time.Time{}, content)
	return true
}
