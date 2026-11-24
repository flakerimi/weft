package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"image/png"
	"io"
	"io/fs"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The site's icon (docs/design/favicon-2026-11-24): favicon.svg, favicon.ico
// and apple-touch-icon.png in public/, which the page build names in every
// page's <head> by itself.

// iconTags are the tags the build writes for the three files, in order.
var iconTags = []string{
	`<link rel="icon" href="/favicon.ico" sizes="32x32">`,
	`<link rel="icon" href="/favicon.svg" type="image/svg+xml">`,
	`<link rel="apple-touch-icon" href="/apple-touch-icon.png">`,
}

var anyIconTag = regexp.MustCompile(`<link rel="(?:icon|apple-touch-icon|shortcut icon)"[^>]*>`)

// Every page, the not-found page included, names the three icons in its
// <head>, in order, and no other.
func TestEveryPageHasIconTags(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	if r.NotFound == nil {
		t.Fatal("routes.json has no not-found page")
	}
	files := []string{*r.NotFound}
	for _, p := range r.Pages {
		files = append(files, p.HTML)
	}
	for _, file := range files {
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		head, _, ok := strings.Cut(string(b), "</head>")
		if !ok {
			t.Fatalf("%s has no </head>", file)
		}
		if got := anyIconTag.FindAllString(head, -1); strings.Join(got, "\n") != strings.Join(iconTags, "\n") {
			t.Errorf("%s has the icon tags %q; want %q", file, got, iconTags)
		}
	}
}

// The site serves the three files with their types, revalidated: their
// names carry no hash, so they are not cached for a year.
func TestSiteServesIcons(t *testing.T) {
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	for path, typ := range map[string]string{
		"/favicon.svg":          "image/svg+xml",
		"/favicon.ico":          "image/x-icon",
		"/apple-touch-icon.png": "image/png",
	} {
		res, body := get(t, srv, path, "Accept-Encoding", "identity")
		if res.StatusCode != 200 {
			t.Errorf("GET %s: %d", path, res.StatusCode)
			continue
		}
		if got := res.Header.Get("Content-Type"); got != typ {
			t.Errorf("GET %s: Content-Type %q, want %q", path, got, typ)
		}
		if got := res.Header.Get("Cache-Control"); got != "no-cache" {
			t.Errorf("GET %s: Cache-Control %q, want no-cache", path, got)
		}
		want, err := os.ReadFile("public" + path)
		if err != nil {
			t.Fatal(err)
		}
		if body != string(want) {
			t.Errorf("GET %s is not public%s", path, path)
		}
	}
}

// The SVG is drawn by hand: well-formed, under 2 KB, plain shapes with no
// picture, script, filter or link to anything.
func TestFaviconSVG(t *testing.T) {
	b, err := os.ReadFile("public/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= 2048 {
		t.Errorf("public/favicon.svg is %d B, want under 2 KB", len(b))
	}
	dec := xml.NewDecoder(bytes.NewReader(b))
	root := ""
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("public/favicon.svg is not well-formed XML: %v", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if root == "" {
			root = el.Name.Local
		}
		switch el.Name.Local {
		case "svg", "rect", "path":
		default:
			t.Errorf("public/favicon.svg has a <%s>; it is drawn with rect and path alone", el.Name.Local)
		}
		for _, a := range el.Attr {
			if a.Name.Local == "href" || a.Name.Local == "filter" || strings.Contains(a.Value, "url(") {
				t.Errorf("public/favicon.svg: <%s %s=%q> points at something", el.Name.Local, a.Name.Local, a.Value)
			}
		}
	}
	if root != "svg" {
		t.Errorf("public/favicon.svg's root is <%s>, want <svg>", root)
	}
	for _, bad := range []string{"<image", "<script", "filter", "href"} {
		if bytes.Contains(b, []byte(bad)) {
			t.Errorf("public/favicon.svg holds %q", bad)
		}
	}
}

// favicon.ico holds a 32 and a 16 px image, each PNG data of the size its
// entry says; apple-touch-icon.png is 180 px square and opaque.
func TestFaviconImageSizes(t *testing.T) {
	ico, err := os.ReadFile("public/favicon.ico")
	if err != nil {
		t.Fatal(err)
	}
	// ICONDIR: reserved 0, type 1 (icon), count; then 16 B per ICONDIRENTRY.
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[0:]) != 0 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		t.Fatal("public/favicon.ico has no icon header")
	}
	count := int(binary.LittleEndian.Uint16(ico[4:]))
	if count != 2 || len(ico) < 6+16*count {
		t.Fatalf("public/favicon.ico holds %d images, want 2", count)
	}
	var sizes []int
	for i := range count {
		e := ico[6+16*i:]
		w, h := int(e[0]), int(e[1])
		size, offset := binary.LittleEndian.Uint32(e[8:]), binary.LittleEndian.Uint32(e[12:])
		if int(offset)+int(size) > len(ico) {
			t.Fatalf("public/favicon.ico: image %d runs past the file's end", i)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(ico[offset : offset+size]))
		if err != nil {
			t.Fatalf("public/favicon.ico: image %d is not PNG data: %v", i, err)
		}
		if w != h || cfg.Width != w || cfg.Height != h {
			t.Errorf("public/favicon.ico: image %d is listed %d×%d and is %d×%d", i, w, h, cfg.Width, cfg.Height)
		}
		sizes = append(sizes, w)
	}
	if len(sizes) != 2 || sizes[0] != 32 || sizes[1] != 16 {
		t.Errorf("public/favicon.ico holds %v px images, want [32 16]", sizes)
	}

	f, err := os.Open("public/apple-touch-icon.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 180 || b.Dy() != 180 {
		t.Errorf("public/apple-touch-icon.png is %d×%d, want 180×180", b.Dx(), b.Dy())
	}
	if o, ok := img.(interface{ Opaque() bool }); !ok || !o.Opaque() {
		t.Error("public/apple-touch-icon.png is not opaque; iOS draws a transparent corner black")
	}
}
