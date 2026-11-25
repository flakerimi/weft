package warp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// The build's manifest.json: every served file by its URL path, and the
// Early Hints Link values per page path ("404" for the not-found page).
type buildManifest struct {
	Files map[string]manifestFile `json:"files"`
	Hints map[string][]string     `json:"hints"`
}

type manifestFile struct {
	ETag      string `json:"etag"`
	Type      string `json:"type"`
	Br        bool   `json:"br"`
	Gz        bool   `json:"gz"`
	Immutable bool   `json:"immutable"`
}

// The build's routes.json. Its file names are relative to dist.
type buildRoutes struct {
	Pages []struct {
		Path string `json:"path"`
		HTML string `json:"html"`
	} `json:"pages"`
	NotFound *string `json:"notFound"`
	App      *struct {
		Prefix string `json:"prefix"`
		Index  string `json:"index"`
	} `json:"app"`
}

// A site is a page build held in memory: nothing is read from dist once it is made.
type site struct {
	pages     map[string]*resource
	files     map[string]*resource
	notFound  *resource // nil: a plain 404
	appPrefix string    // "" when there is no app
	app       *resource
}

// A resource is what one route serves: a file in up to three encodings, and
// the Link values it is hinted with.
type resource struct {
	identity, br, gzip *variant // br and gzip are nil when the build made none
	links              []string
}

// A variant is one encoding of a file: its bytes, its ETag, and the values of
// its headers in variantHeaders order (Content-Encoding only when compressed).
type variant struct {
	body   []byte
	etag   string
	values []string
}

// The response headers a variant sets, in the order of variant.values. A 304
// sends the first three.
var variantHeaders = [...]string{"Vary", "Etag", "Cache-Control", "Content-Type", "Content-Length", "Content-Encoding"}

const notModifiedHeaders = 3

// The files a build holds for Weft itself, never served as they are; nor is
// anything under /pages/.
var buildOnly = map[string]bool{"/manifest.json": true, "/routes.json": true, "/.weft-build": true}

// pageManifest returns dist's manifest.json when dist is a page build: it has
// a routes.json file, and a manifest.json whose files member is a JSON object.
// A single-page app's web app manifest.json is not one.
func pageManifest(dist fs.FS) ([]byte, bool) {
	if fi, err := fs.Stat(dist, "routes.json"); err != nil || !fi.Mode().IsRegular() {
		return nil, false
	}
	raw, err := fs.ReadFile(dist, "manifest.json")
	if err != nil {
		return nil, false
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(raw, &top) != nil {
		return nil, false
	}
	files := top["files"]
	return raw, len(files) > 0 && files[0] == '{'
}

// loadSite reads the page build that raw, dist's manifest.json, describes:
// routes.json and every listed file with its variants. It panics, naming the
// file, if any of it is missing or malformed.
func loadSite(dist fs.FS, raw []byte) *site {
	var m buildManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		panic(fmt.Sprintf("warp.Serve: dist/manifest.json is not a Weft build manifest (%v).", err))
	}
	rawRoutes, err := fs.ReadFile(dist, "routes.json")
	if err != nil {
		panic(fmt.Sprintf("warp.Serve: dist/routes.json cannot be read (%v).", err))
	}
	var routes buildRoutes
	if err := json.Unmarshal(rawRoutes, &routes); err != nil {
		panic(fmt.Sprintf("warp.Serve: dist/routes.json is not a Weft route map (%v).", err))
	}

	s := &site{pages: map[string]*resource{}, files: map[string]*resource{}}
	for key, f := range m.Files {
		if name := strings.TrimPrefix(key, "/"); !strings.HasPrefix(key, "/") || !fs.ValidPath(name) || name == "." {
			panic(fmt.Sprintf(`warp.Serve: manifest.json lists %s, which is not a path like "/assets/app.js".`, key))
		}
		if !entityTag(f.ETag) {
			panic(fmt.Sprintf(`warp.Serve: manifest.json gives %s the ETag %s, which is not an entity tag like "abc" with no quote, comma, space or control character inside.`, key, f.ETag))
		}
		s.files[key] = &resource{
			identity: readVariant(dist, key, f, ""),
			br:       readVariant(dist, key, f, "br"),
			gzip:     readVariant(dist, key, f, "gzip"),
		}
	}
	// A route serves a listed file, with that route's hints.
	route := func(what, file string, links []string) *resource {
		res := s.files["/"+file]
		if res == nil {
			panic(fmt.Sprintf("warp.Serve: routes.json serves %s from %s, which manifest.json does not list.", what, file))
		}
		return &resource{identity: res.identity, br: res.br, gzip: res.gzip, links: links}
	}
	for _, p := range routes.Pages {
		if s.pages[p.Path] != nil {
			panic(fmt.Sprintf("warp.Serve: routes.json lists the page %s twice.", p.Path))
		}
		s.pages[p.Path] = route(p.Path, p.HTML, m.Hints[p.Path])
	}
	if routes.NotFound != nil {
		s.notFound = route("the not-found page", *routes.NotFound, m.Hints["404"])
	}
	if routes.App != nil {
		s.appPrefix = routes.App.Prefix
		s.app = route(routes.App.Prefix, routes.App.Index, nil)
	}
	return s
}

// entityTag reports whether t is an entity tag (RFC 9110 §8.8.3), "abc" or
// W/"abc", whose opaque part has no comma: etagMatch splits on commas.
func entityTag(t string) bool {
	t = strings.TrimPrefix(t, "W/")
	if len(t) < 2 || t[0] != '"' || t[len(t)-1] != '"' {
		return false
	}
	for _, c := range []byte(t[1 : len(t)-1]) {
		if c <= ' ' || c == '"' || c == ',' || c == 0x7f {
			return false
		}
	}
	return true
}

// readVariant reads the listed file at URL path key in one encoding ("" for
// none), or returns nil when the manifest says the build made no such variant.
func readVariant(dist fs.FS, key string, f manifestFile, encoding string) *variant {
	name := key[1:]
	etag, ext := f.ETag, ""
	switch {
	case encoding == "br" && f.Br:
		etag, ext = f.ETag[:len(f.ETag)-1]+`-br"`, ".br"
	case encoding == "gzip" && f.Gz:
		etag, ext = f.ETag[:len(f.ETag)-1]+`-gz"`, ".gz"
	case encoding != "":
		return nil
	}
	body, err := fs.ReadFile(dist, name+ext)
	if errors.Is(err, fs.ErrNotExist) && ext != "" {
		panic(fmt.Sprintf("warp.Serve: manifest.json says %s has a %s variant, but dist has no %s.", key, ext, name+ext))
	} else if errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("warp.Serve: manifest.json lists %s, but dist has no %s.", key, name))
	} else if err != nil {
		panic(fmt.Sprintf("warp.Serve: dist/%s cannot be read (%v).", name+ext, err))
	}
	cache := cacheNone
	if f.Immutable {
		cache = cacheImmutable
	}
	values := []string{"Accept-Encoding", etag, cache, f.Type, strconv.Itoa(len(body))}
	if encoding != "" {
		values = append(values, encoding)
	}
	return &variant{body: body, etag: etag, values: values}
}

// serves says what the site serves at the URL path p: "a page", "a file", or
// "" for nothing of its own.
func (s *site) serves(p string) string {
	switch {
	case s.pages[p] != nil:
		return "a page"
	case s.files[p] != nil:
		return "a file"
	}
	return ""
}

// ServeHTTP serves a request that is not for the API: a page, a listed file,
// the app, or the not-found page.
func (s *site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h := w.Header()
		h.Set("Vary", "Accept-Encoding")
		h.Set("Allow", "GET, HEAD")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = path.Clean(p)
	if res := s.pages[p]; res != nil {
		res.serve(w, r, http.StatusOK)
		return
	}
	// A file is only its exact path; the build's own files are never served.
	if !strings.HasSuffix(r.URL.Path, "/") && !buildOnly[p] && !strings.HasPrefix(p, "/pages/") {
		if res := s.files[p]; res != nil {
			res.serve(w, r, http.StatusOK)
			return
		}
	}
	// As without a manifest, a missing asset is a 404, never the app's index.
	if s.app != nil && (p == s.appPrefix || strings.HasPrefix(p, s.appPrefix+"/")) && !strings.HasPrefix(p, s.appPrefix+"/assets/") {
		s.app.serve(w, r, http.StatusOK)
		return
	}
	if s.notFound != nil {
		s.notFound.serve(w, r, http.StatusNotFound)
		return
	}
	w.Header().Set("Vary", "Accept-Encoding")
	http.NotFound(w, r)
}

// serve writes res with status: the 304 when it is a 200 the client already
// has, else a 103 with its Link values when it has some (and the client
// speaks HTTP/1.1 or later), then the variant the client accepts best. Its
// only allocation is the one slice that holds the header values, so a handler
// after it may change them without changing res.
func (res *resource) serve(w http.ResponseWriter, r *http.Request, status int) {
	v := res.pick(r.Header["Accept-Encoding"])
	n := len(v.values)
	values := make([]string, n+len(res.links))
	copy(values, v.values)
	copy(values[n:], res.links)
	h := w.Header()
	if status == http.StatusOK && etagMatch(r.Header["If-None-Match"], v.etag) {
		for i := range notModifiedHeaders {
			h[variantHeaders[i]] = values[i : i+1 : i+1]
		}
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if len(res.links) > 0 {
		h["Link"] = values[n:len(values):len(values)]
		// RFC 9110 §15.2: no 1xx response to an HTTP/1.0 client.
		if r.ProtoAtLeast(1, 1) {
			w.WriteHeader(http.StatusEarlyHints)
		}
	}
	for i := range n {
		h[variantHeaders[i]] = values[i : i+1 : i+1]
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(v.body)
	}
}

// pick returns the variant to send for these Accept-Encoding values: the
// accepted encoding with the highest q-value, br when br and gzip tie, unless
// the client names identity with a higher q-value still; else the file as it
// is, which is also what a client that accepts nothing gets (RFC 9110 §12.5.3).
func (res *resource) pick(accept []string) *variant {
	brQ, gzQ, idQ := acceptQ(accept)
	br := brQ > 0 && res.br != nil
	gz := gzQ > 0 && res.gzip != nil
	v, q := res.identity, 0.0
	switch {
	case br && (!gz || brQ >= gzQ):
		v, q = res.br, brQ
	case gz:
		v, q = res.gzip, gzQ
	}
	if idQ > q {
		return res.identity
	}
	return v
}

// acceptQ returns the q-values Accept-Encoding gives br and gzip (from their
// own entries, else from "*") and identity (from its own entry only), or -1
// for none. An entry without a q-value has 1, and one whose q-value is not a
// number has 0.
func acceptQ(accept []string) (br, gz, identity float64) {
	br, gz, identity = -1.0, -1.0, -1.0
	star := -1.0
	for _, line := range accept {
		for line != "" {
			var item string
			item, line, _ = strings.Cut(line, ",")
			name, params, _ := strings.Cut(item, ";")
			name = strings.TrimSpace(name)
			switch {
			case strings.EqualFold(name, "br"):
				br = qValue(params)
			case strings.EqualFold(name, "gzip"), strings.EqualFold(name, "x-gzip"):
				gz = qValue(params)
			case strings.EqualFold(name, "identity"):
				identity = qValue(params)
			case name == "*":
				star = qValue(params)
			}
		}
	}
	if br < 0 {
		br = star
	}
	if gz < 0 {
		gz = star
	}
	return br, gz, identity
}

func qValue(params string) float64 {
	for params != "" {
		var p string
		p, params, _ = strings.Cut(params, ";")
		p = strings.TrimSpace(p)
		if len(p) >= 2 && (p[0] == 'q' || p[0] == 'Q') && p[1] == '=' {
			q, err := strconv.ParseFloat(strings.TrimSpace(p[2:]), 64)
			if err != nil || q < 0 || q > 1 {
				return 0
			}
			return q
		}
	}
	return 1
}

// etagMatch reports whether If-None-Match lists etag, compared weakly
// (a W/ prefix is ignored), or is "*". The list is split on commas, which an
// entity tag may contain in general but never does here: the build's tags are
// hex digits, and loadSite refuses any tag with a comma.
func etagMatch(ifNoneMatch []string, etag string) bool {
	etag = strings.TrimPrefix(etag, "W/")
	for _, line := range ifNoneMatch {
		for line != "" {
			var tag string
			tag, line, _ = strings.Cut(line, ",")
			tag = strings.TrimSpace(tag)
			if tag == "*" || strings.TrimPrefix(tag, "W/") == etag {
				return true
			}
		}
	}
	return false
}
