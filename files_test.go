package weft

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// releaseDir writes a folder like the one the site serves its releases from,
// and beside it a file that is not in it.
func releaseDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "data")
	for name, data := range map[string]string{
		"install":                                "#!/bin/sh\necho install\n",
		"releases/latest":                        "0.0.2\n",
		"releases/0.0.2/SHA256SUMS":              "abc  weft_linux_amd64.tar.gz\n",
		"releases/0.0.2/weft_linux_amd64.tar.gz": "\x1f\x8b\x08\x00tarball",
	} {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, "secret"), []byte("not served"), 0o644)
	return dir
}

func TestWithFilesServesTheFolder(t *testing.T) {
	dir := releaseDir(t)
	for name, h := range map[string]http.Handler{
		"page build":      Serve(pageBuild(t), nil, WithFiles("/", os.DirFS(dir))),
		"single-page app": Serve(dist, nil, WithFiles("/", os.DirFS(dir))),
	} {
		for _, c := range []struct{ path, body, typ string }{
			{"/install", "#!/bin/sh\necho install\n", "text/plain; charset=utf-8"},
			{"/releases/latest", "0.0.2\n", "text/plain; charset=utf-8"},
			{"/releases/0.0.2/SHA256SUMS", "abc  weft_linux_amd64.tar.gz\n", "text/plain; charset=utf-8"},
			{"/releases/0.0.2/weft_linux_amd64.tar.gz", "\x1f\x8b\x08\x00tarball", "application/gzip"},
		} {
			r := do(t, h, "GET", c.path, "Accept-Encoding", "br, gzip")
			if r.StatusCode != 200 || r.body != c.body {
				t.Errorf("%s %s: status %d body %q", name, c.path, r.StatusCode, r.body)
				continue
			}
			if got := r.Header.Get("Content-Type"); got != c.typ {
				t.Errorf("%s %s: Content-Type %q, want %q", name, c.path, got, c.typ)
			}
			if r.Header.Get("Cache-Control") != "no-cache" || r.Header.Get("Content-Encoding") != "" || r.Header.Get("Content-Length") != strconv.Itoa(len(c.body)) {
				t.Errorf("%s %s: Cache-Control %q, Content-Encoding %q, Content-Length %q", name, c.path,
					r.Header.Get("Cache-Control"), r.Header.Get("Content-Encoding"), r.Header.Get("Content-Length"))
			}
		}
	}
	// The site's own pages and files are still served.
	h := Serve(pageBuild(t), nil, WithFiles("/", os.DirFS(dir)))
	if r := do(t, h, "GET", "/about"); r.StatusCode != 200 || r.body != "<p>about us</p>" {
		t.Fatalf("/about: status %d body %q", r.StatusCode, r.body)
	}
}

func TestWithFilesUnderAPrefix(t *testing.T) {
	dir := releaseDir(t)
	sub := filepath.Join(dir, "releases")
	h := Serve(pageBuild(t), nil, WithFiles("/downloads/", os.DirFS(sub)))
	if r := do(t, h, "GET", "/downloads/0.0.2/SHA256SUMS"); r.StatusCode != 200 || r.body != "abc  weft_linux_amd64.tar.gz\n" {
		t.Fatalf("status %d body %q", r.StatusCode, r.body)
	}
	for _, path := range []string{"/0.0.2/SHA256SUMS", "/downloads", "/downloads/", "/downloads/0.0.2", "/downloads/0.0.2/", "/downloads/latest/"} {
		if r := do(t, h, "GET", path); r.StatusCode != 404 || r.body != "<p>not here</p>" {
			t.Errorf("%s: status %d body %q, want the not-found page", path, r.StatusCode, r.body)
		}
	}
}

// The ETag is the first 16 hex digits of the SHA-256 of the file, as the page
// build gives its files, computed once at start; a client that has it gets a 304.
func TestWithFilesETagAndNotModified(t *testing.T) {
	dir := releaseDir(t)
	h := Serve(pageBuild(t), nil, WithFiles("/", os.DirFS(dir)))
	r := do(t, h, "GET", "/releases/latest")
	// sha256("0.0.2\n")
	const etag = `"a5dd346546a38436"`
	if got := r.Header.Get("ETag"); got != etag {
		t.Fatalf("ETag %q, want %s", got, etag)
	}
	for _, match := range []string{etag, "W/" + etag, `"other", ` + etag, "*"} {
		for _, method := range []string{"GET", "HEAD"} {
			r := do(t, h, method, "/releases/latest", "If-None-Match", match)
			if r.StatusCode != http.StatusNotModified || r.body != "" || r.Header.Get("ETag") != etag {
				t.Errorf("%s If-None-Match %s: status %d body %q ETag %q", method, match, r.StatusCode, r.body, r.Header.Get("ETag"))
			}
		}
	}
	if r := do(t, h, "GET", "/releases/latest", "If-None-Match", `"a5dd346546a38437"`); r.StatusCode != 200 || r.body != "0.0.2\n" {
		t.Fatalf("another tag: status %d body %q", r.StatusCode, r.body)
	}
	// HEAD has GET's headers and no body.
	head := do(t, h, "HEAD", "/releases/latest")
	if head.StatusCode != 200 || head.body != "" || head.Header.Get("Content-Length") != "6" || head.Header.Get("ETag") != etag {
		t.Fatalf("HEAD: status %d body %q headers %v", head.StatusCode, head.body, head.Header)
	}
	if v := r.Header.Values("Vary"); len(v) != 1 || v[0] != "Accept-Encoding" {
		t.Fatalf("Vary %q", v)
	}
	if r := do(t, h, "POST", "/releases/latest"); r.StatusCode != http.StatusMethodNotAllowed || r.Header.Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST: status %d Allow %q", r.StatusCode, r.Header.Get("Allow"))
	}
}

// A file changed since the server started would be sent with a stale ETag,
// so it is refused rather than served; one removed is not found.
func TestWithFilesChangedOnDisk(t *testing.T) {
	dir := releaseDir(t)
	h := Serve(pageBuild(t), nil, WithFiles("/", os.DirFS(dir)))
	os.WriteFile(filepath.Join(dir, "releases/latest"), []byte("0.0.10\n"), 0o644)
	if r := do(t, h, "GET", "/releases/latest"); r.StatusCode != http.StatusInternalServerError || strings.Contains(r.body, "0.0.") {
		t.Fatalf("changed file: status %d body %q", r.StatusCode, r.body)
	}
	os.Rename(filepath.Join(dir, "install"), filepath.Join(t.TempDir(), "install"))
	if r := do(t, h, "GET", "/install"); r.StatusCode != 404 || r.body != "<p>not here</p>" {
		t.Fatalf("removed file: status %d body %q", r.StatusCode, r.body)
	}
	// A file added after the start is not served: the files are those found at start.
	os.WriteFile(filepath.Join(dir, "releases/new"), []byte("new"), 0o644)
	if r := do(t, h, "GET", "/releases/new"); r.StatusCode != 404 {
		t.Fatalf("added file: status %d", r.StatusCode)
	}
}

func TestWithFilesPathTraversal(t *testing.T) {
	dir := releaseDir(t)
	h := Serve(pageBuild(t), nil, WithFiles("/releases/", os.DirFS(filepath.Join(dir, "releases"))))
	for _, path := range []string{
		"/releases/../secret",
		"/releases/../../secret",
		"/releases/%2e%2e/%2e%2e/secret",
		"/releases/..%2f..%2fsecret",
		"/releases/0.0.2/../../../secret",
		"/releases/../install",
		"/releases//..//..//secret",
	} {
		r := do(t, h, "GET", path)
		if r.StatusCode != 404 || strings.Contains(r.body, "not served") || strings.Contains(r.body, "echo install") {
			t.Errorf("%s: status %d body %q", path, r.StatusCode, r.body)
		}
	}
}

// A symlink in the folder could lead anywhere, so the server does not start.
func TestWithFilesRefusesASymlink(t *testing.T) {
	dir := releaseDir(t)
	if err := os.Symlink(filepath.Join(dir, "..", "secret"), filepath.Join(dir, "releases", "leak")); err != nil {
		t.Skip(err)
	}
	wantPanic(t, "weft.WithFiles: releases/leak in the folder served at / is a symlink; put the file itself there.", func() {
		Serve(pageBuild(t), nil, WithFiles("/", os.DirFS(dir)))
	})
}

// A file the build or another folder serves at the same path, or a path the
// server never reaches, stops the server from starting, naming it.
func TestWithFilesConflictsPanic(t *testing.T) {
	files := fstest.MapFS{"robots.txt": {Data: []byte("x")}}
	wantPanic(t, "weft.WithFiles: /robots.txt is a file of the build too; serve it from one place only.", func() {
		Serve(pageBuild(t), nil, WithFiles("/", files))
	})
	wantPanic(t, "weft.WithFiles: /favicon.svg is a file of the build too; serve it from one place only.", func() {
		Serve(dist, nil, WithFiles("/", fstest.MapFS{"favicon.svg": {Data: []byte("x")}}))
	})
	wantPanic(t, "weft.WithFiles: /about is a page of the build too; serve it from one place only.", func() {
		Serve(pageBuild(t), nil, WithFiles("/", fstest.MapFS{"about": {Data: []byte("x")}}))
	})
	wantPanic(t, "weft.WithFiles: /x/a is served from two folders; serve it from one place only.", func() {
		Serve(pageBuild(t), nil, WithFiles("/x/", fstest.MapFS{"a": {}}), WithFiles("/", fstest.MapFS{"x/a": {}}))
	})
	wantPanic(t, "weft.WithFiles: /api/a is under /api/, which goes to the API; serve the folder at another prefix.", func() {
		Serve(pageBuild(t), nil, WithFiles("/api/", fstest.MapFS{"a": {}}))
	})
	for _, prefix := range []string{"", "releases/", "/releases", "/a/../b/", "/a//b/"} {
		wantPanic(t, `weft.WithFiles: the prefix "`+prefix+`" is not a path that starts and ends with "/", like "/releases/".`, func() {
			Serve(pageBuild(t), nil, WithFiles(prefix, files))
		})
	}
}
