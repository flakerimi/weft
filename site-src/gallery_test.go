package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The /components gallery, as the build made it from catalog/components.json
// (gallery/plugin.ts, gallery/catalog.ts, pages/components/).

type catalogEntry struct {
	Name     string `json:"name"`
	Group    string `json:"group"`
	Purpose  string `json:"purpose"`
	Examples []struct {
		Name  string `json:"name"`
		Title string `json:"title"`
		File  string `json:"file"`
	} `json:"examples"`
}

// readCatalog is catalog/components.json: its bytes, its entries, and each entry as plain JSON values.
func readCatalog(t *testing.T) ([]byte, []catalogEntry, []any) {
	t.Helper()
	raw, err := os.ReadFile("../catalog/components.json")
	if err != nil {
		t.Fatal(err)
	}
	var typed struct{ Components []catalogEntry }
	var plain struct{ Components []any }
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plain); err != nil {
		t.Fatal(err)
	}
	if len(typed.Components) == 0 {
		t.Fatal("the catalogue lists no component")
	}
	return raw, typed.Components, plain.Components
}

var (
	kebabLower = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	kebabUpper = regexp.MustCompile(`([A-Z])([A-Z][a-z])`)
)

// slugOf is a component's page name, as gallery/model.ts makes it: DatePicker → date-picker.
func slugOf(name string) string {
	return strings.ToLower(kebabUpper.ReplaceAllString(kebabLower.ReplaceAllString(name, "$1-$2"), "$1-$2"))
}

func readPage(t *testing.T, d fs.FS, path string) string {
	t.Helper()
	file := "pages" + path + "/index.html"
	b, err := fs.ReadFile(d, file)
	if err != nil {
		t.Fatalf("no page at %s: %v", path, err)
	}
	return string(b)
}

// element is the outer HTML of the first element whose start tag contains
// marker: from its "<" to its matching end tag, nested elements of the same
// name counted.
func element(html, marker string) string {
	at := strings.Index(html, marker)
	if at == -1 {
		return ""
	}
	start := strings.LastIndex(html[:at+1], "<")
	name := regexp.MustCompile(`^<([a-z][a-z0-9-]*)`).FindStringSubmatch(html[start:])
	if name == nil {
		return ""
	}
	open := regexp.MustCompile(`<` + name[1] + `[\s>]|</` + name[1] + `>`)
	depth := 0
	for _, m := range open.FindAllStringIndex(html[start:], -1) {
		if html[start+m[0]+1] == '/' {
			depth--
		} else {
			depth++
		}
		if depth == 0 {
			return html[start : start+m[1]]
		}
	}
	return ""
}

var (
	htmlComment = regexp.MustCompile(`<!--[\s\S]*?-->`)
	// Ids from useId: Vue's per-app counter (weft-v-3) on a page, an island's own (weft-i0-3)
	// in one, a thumbnail's (weft-t12-3) on the index.
	generatedID = regexp.MustCompile(`-(?:v|[it]\d+)-\d+(?:-\d+)*`)
)

// sameRender is the HTML as compared across renders: no hydration markers, and generated ids as one.
func sameRender(html string) string {
	return generatedID.ReplaceAllString(htmlComment.ReplaceAllString(html, ""), "-ID")
}

// inner is what an element holds, without its own tags.
func inner(outer string) string {
	return outer[strings.Index(outer, ">")+1 : strings.LastIndex(outer, "</")]
}

// The index lists every catalogue entry once, under its group, as a card
// with its first example drawn by the server, its name linking to its page,
// and its purpose; and the gallery serves the catalogue itself.
func TestGalleryListsEveryComponent(t *testing.T) {
	d := dist(t)
	raw, entries, _ := readCatalog(t)
	html := readPage(t, d, "/components")
	cards := regexp.MustCompile(`<li class="gallery-card"`).FindAllStringIndex(html, -1)
	if len(cards) != len(entries) {
		t.Errorf("the index has %d cards for %d catalogue entries", len(cards), len(entries))
	}
	for _, c := range entries {
		link := fmt.Sprintf(`<h3><a href="/components/%s">%s</a></h3>`, slugOf(c.Name), c.Name)
		if strings.Count(html, link) != 1 {
			t.Errorf("the index does not link %s once", c.Name)
			continue
		}
		thumb := element(html, fmt.Sprintf(`data-component="%s" data-example="%s"`, c.Name, c.Examples[0].File))
		if !strings.HasPrefix(thumb, `<div class="gallery-thumb" inert `) || strings.TrimSpace(sameRender(inner(thumb))) == "" {
			t.Errorf("%s: no inert thumbnail of %s drawn by the server", c.Name, c.Examples[0].File)
		}
		// The card sits in its group's section.
		section := element(html, `<section class="gallery-group" aria-labelledby="group-`+c.Group+`"`)
		if !strings.Contains(section, link) {
			t.Errorf("%s is not under its group, %s", c.Name, c.Group)
		}
	}
	// The thumbnails are the page's HTML: it has no island but the filter.
	if got := regexp.MustCompile(`<weft-island data-id="([^"]+)"`).FindAllStringSubmatch(html, -1); len(got) != 1 || got[0][1] != "GalleryFilter" {
		t.Errorf("the index's islands are %v, want the filter alone", got)
	}

	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	res, body := get(t, srv, "/catalog.json", "Accept-Encoding", "identity")
	if res.StatusCode != 200 || body != string(raw) || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("GET /catalog.json: %d %s, %d bytes; want catalog/components.json as it is", res.StatusCode, res.Header.Get("Content-Type"), len(body))
	}
	for _, name := range []string{"parity", "sizes"} {
		want, _ := os.ReadFile("../docs/" + name + ".md")
		res, body := get(t, srv, "/components/"+name+".md", "Accept-Encoding", "identity")
		if res.StatusCode != 200 || body != string(want) || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/markdown") {
			t.Errorf("GET /components/%s.md: %d %s; want docs/%s.md as it is", name, res.StatusCode, res.Header.Get("Content-Type"), name)
		}
		if !strings.Contains(html, `href="/components/`+name+`.md"`) {
			t.Errorf("the index does not link docs/%s.md", name)
		}
	}
}

// Each component's page has, in order, its title and purpose, when to use
// it, the preview stage (an island holding its first example as the server
// drew it), the example's source, what adapts, the API, every other example
// and the raw entry for agents, which /components/<slug>.json serves; and
// links to the components before and after it in its group.
func TestComponentPages(t *testing.T) {
	d := dist(t)
	_, entries, plain := readCatalog(t)
	index := readPage(t, d, "/components")
	byGroup := map[string][]string{}
	for _, c := range entries {
		byGroup[c.Group] = append(byGroup[c.Group], c.Name)
	}
	for _, names := range byGroup {
		sort.Strings(names)
	}
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()

	for i, c := range entries {
		slug := slugOf(c.Name)
		html := readPage(t, d, "/components/"+slug)
		order := []string{`<h1 class="docs-title">` + c.Name + `</h1>`, `<dt>Use it</dt>`, `<weft-island data-id="Stage_` + c.Name + `"`, `<h2 id="code">`, `<h2 id="adapts">`, `<h2 id="api">`}
		if len(c.Examples) > 1 {
			order = append(order, `<h2 id="examples">`)
		}
		order = append(order, `<h2 id="agents">`)
		last := -1
		for _, s := range order {
			at := strings.Index(html, s)
			if at <= last {
				t.Errorf("%s: %q is missing or out of order", c.Name, s)
				break
			}
			last = at
		}

		// The stage holds the first example as the index's thumbnail does: the
		// same server render, each inside its inset WeftProvider.
		stage := element(html, `data-id="Stage_`+c.Name+`"`)
		drawn := sameRender(inner(element(stage, `class="gallery-example"`)))
		thumb := sameRender(inner(element(element(index, `data-component="`+c.Name+`"`), `<div class="weft-provider"`)))
		if strings.TrimSpace(drawn) == "" || drawn != thumb {
			t.Errorf("%s: the stage does not hold %s as the server draws it\nstage: %.300s\nthumb: %.300s", c.Name, c.Examples[0].File, drawn, thumb)
		}
		// The source comes from the file, highlighted.
		if !strings.Contains(html, `<div id="code-`+c.Examples[0].Name+`"><pre class="shiki weft"`) {
			t.Errorf("%s: no highlighted source of %s", c.Name, c.Examples[0].File)
		}
		// Every other example, each with its preview and source.
		for _, e := range c.Examples[1:] {
			preview := element(html, `data-example="`+e.File+`"`)
			if strings.TrimSpace(sameRender(inner(preview))) == "" {
				t.Errorf("%s: %s is not drawn", c.Name, e.File)
			}
			if strings.Contains(preview, "<weft-island") && !strings.Contains(preview, `data-when="visible"`) {
				t.Errorf("%s: %s hydrates before it is visible", c.Name, e.File)
			}
			if !strings.Contains(html, `<div id="code-`+e.Name+`"><pre class="shiki weft"`) {
				t.Errorf("%s: no highlighted source of %s", c.Name, e.File)
			}
		}

		// Previous and next within the group.
		names := byGroup[c.Group]
		at := sort.SearchStrings(names, c.Name)
		if at > 0 && !strings.Contains(html, `href="/components/`+slugOf(names[at-1])+`" rel="prev"`) {
			t.Errorf("%s: no link back to %s", c.Name, names[at-1])
		}
		if at+1 < len(names) && !strings.Contains(html, `href="/components/`+slugOf(names[at+1])+`" rel="next"`) {
			t.Errorf("%s: no link on to %s", c.Name, names[at+1])
		}
		if at == 0 && strings.Contains(html, `rel="prev"`) || at+1 == len(names) && strings.Contains(html, `rel="next"`) {
			t.Errorf("%s links out of its group, %s", c.Name, c.Group)
		}
		// The sidebar marks it current, and the header marks Components.
		if !regexp.MustCompile(`<a class="weft-link weft-nav__item" href="/components/` + slug + `" aria-current="page"`).MatchString(html) {
			t.Errorf("%s: the sidebar does not mark it current", c.Name)
		}

		// For agents: its catalogue entry, served raw.
		res, body := get(t, srv, "/components/"+slug+".json", "Accept-Encoding", "identity")
		var served any
		if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/json" || json.Unmarshal([]byte(body), &served) != nil {
			t.Errorf("GET /components/%s.json: %d %s", slug, res.StatusCode, res.Header.Get("Content-Type"))
		} else if !reflect.DeepEqual(served, plain[i]) {
			t.Errorf("/components/%s.json is not %s's catalogue entry", slug, c.Name)
		}
		if !strings.Contains(html, `href="/components/`+slug+`.json"`) || !strings.Contains(html, `<iframe class="gallery-raw" src="/components/`+slug+`.json" loading="lazy"`) {
			t.Errorf("%s: the For agents section does not link and show /components/%s.json", c.Name, slug)
		}
	}
}

// The copy button's rule, as the build minifies it (the / escaped).
var copyHidden = regexp.MustCompile(`weft-island:has\(>script\[type=application\\/json\]\) \.copy-button\{visibility:hidden\}`)

// Without JavaScript the gallery reads: every example is drawn by the
// server and shown, the controls are inert (the stage's toolbar, the
// filter) or hidden until their island starts (the copy buttons), and the
// thumbnails, being pictures of examples, are inert.
func TestGalleryWithoutJavaScript(t *testing.T) {
	d := dist(t)
	_, entries, _ := readCatalog(t)
	index := readPage(t, d, "/components")
	if !strings.Contains(element(index, `data-id="GalleryFilter"`), `<div class="gallery-filter" inert>`) {
		t.Error("the filter is not inert before its island starts")
	}
	for _, c := range entries {
		html := readPage(t, d, "/components/"+slugOf(c.Name))
		stage := element(html, `data-id="Stage_`+c.Name+`"`)
		if !strings.Contains(stage, `<div class="gallery-toolbar" inert>`) {
			t.Errorf("%s: the stage's toolbar is not inert before the island starts", c.Name)
		}
		// Nothing hides an example: the page draws them all, shown.
		for _, e := range c.Examples {
			marker := `data-example="` + e.File + `"`
			if e.File == c.Examples[0].File {
				marker = `class="gallery-example"`
			}
			if el := element(html, marker); el == "" || regexp.MustCompile(`^<[^>]*\shidden[\s>=]`).MatchString(el) {
				t.Errorf("%s: %s is not shown", c.Name, e.File)
			}
		}
		// Each copy button is hidden until its island takes the props script out.
		if n := strings.Count(html, `data-id="CopyCommand"`); n != len(c.Examples) {
			t.Errorf("%s: %d copy buttons for %d examples", c.Name, n, len(c.Examples))
		}
		if !copyHidden.MatchString(html) {
			t.Errorf("%s: the copy buttons are offered before their island starts", c.Name)
		}
	}
}

// The gallery's budgets, brotli, counted as every page's first view is (the
// HTML, its CSS inlined, any stylesheet it links, the loader and whatever the
// page or its Early Hints preload; firstView): a component page at most 17 KB,
// which is the 16 KB its HTML and CSS had before the loader (boot-*.js,
// about 0.55 KB) was counted, and 1 KB for that;
// the index at most 14 KB plus 300 B for each component it
// shows, since every card draws a live example with its component's CSS
// (see the Task 4 report: 14 KB alone cannot hold 76 live thumbnails).
// Island chunks are reported, not budgeted. The gallery's islands load a
// runtime of their own, so they do not grow the one the rest of the site
// loads.
func TestGalleryBudgets(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	_, entries, _ := readCatalog(t)
	// The first view, as for the docs and the landing page: HTML, CSS, the
	// loader and whatever the page or its Early Hints preload.
	static := func(path, html, file string) int { return firstView(t, d, m, path, file, html) }
	runtimeOf := func(html string) string { return pageRuntime(t, d, html) }
	var home, gallery string
	largest, largestPath := 0, ""
	for _, p := range r.Pages {
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		switch {
		case p.Path == "/":
			home = runtimeOf(html)
		case p.Path == "/components":
			size := static(p.Path, html, p.HTML)
			limit := 14*1024 + 300*len(entries)
			t.Logf("/components: first view %d B brotli (limit %d B); island chunks %d B", size, limit, chunkBytes(t, d, m, html, p.Islands))
			if size > limit {
				t.Errorf("/components's first view is %d B brotli, over %d B", size, limit)
			}
		case strings.HasPrefix(p.Path, "/components/"):
			size := static(p.Path, html, p.HTML)
			if size > largest {
				largest, largestPath = size, p.Path
			}
			if size > 17*1024 {
				t.Errorf("%s's first view is %d B brotli, over 17 KB", p.Path, size)
			}
			if rt := runtimeOf(html); gallery == "" {
				gallery = rt
			} else if rt != gallery {
				t.Errorf("%s loads the runtime %s, not the gallery's %s", p.Path, rt, gallery)
			}
		}
	}
	t.Logf("largest component page: %s, first view %d B brotli", largestPath, largest)
	if home == "" || gallery == "" || home == gallery {
		t.Errorf("the gallery's runtime %q is not its own (the home page loads %q)", gallery, home)
	} else {
		t.Logf("runtimes, brotli: the site's %d B, the gallery's %d B", brotliSize(t, d, m, home), brotliSize(t, d, m, gallery))
	}
}

// chunkBytes is the brotli bytes of the islands' chunks, as the page's runtime registers them, and the chunks they import, the runtime aside.
func chunkBytes(t *testing.T, d fs.FS, m builtFiles, html string, islands []string) int {
	t.Helper()
	n := 0
	for _, c := range islandChunks(t, d, pageRuntime(t, d, html), islands) {
		n += brotliSize(t, d, m, c)
	}
	return n
}

// The gallery's examples keep nothing between visits that could hide them:
// a Banner closed in a stage or a thumbnail would otherwise stay closed
// (localStorage, and a pre-paint snippet that hides it), leaving the stage
// and the index's thumbnail blank. So no example's Banner has an id.
func TestGalleryBannersAreNotRemembered(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	banner := regexp.MustCompile(`<div class="weft-banner"[^>]*>`)
	seen := 0
	for _, p := range r.Pages {
		if p.Path != "/components" && !strings.HasPrefix(p.Path, "/components/") {
			continue
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		if strings.Contains(html, "weft-banner:") {
			t.Errorf("%s remembers a closed Banner (weft-banner: in localStorage)", p.Path)
		}
		for _, tag := range banner.FindAllString(html, -1) {
			seen++
			if strings.Contains(tag, "data-id=") {
				t.Errorf("%s draws a Banner with an id, which a close would keep closed: %s", p.Path, tag)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no gallery page draws a Banner")
	}
}

// The gallery's live examples link to pages that are not the site's (/inbox,
// /settings): the speculation rules leave every link inside a stage, a
// preview or a thumbnail alone, marked data-weft-no-prefetch, so hovering
// one prefetches nothing.
func TestGalleryExamplesAreNotPrefetched(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	rules := regexp.MustCompile(`<script type="speculationrules">(.*?)</script>`)
	marked := regexp.MustCompile(`<[^>]*\bclass="(gallery-thumb|gallery-canvas|gallery-preview)\b[^"]*"[^>]*>`)
	checked := 0
	for _, p := range r.Pages {
		if p.Path != "/components" && !strings.HasPrefix(p.Path, "/components/") {
			continue
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		m := rules.FindStringSubmatch(html)
		if m == nil {
			t.Fatalf("%s has no speculation rules", p.Path)
		}
		var got struct {
			Prefetch []struct {
				Where struct {
					And []map[string]json.RawMessage `json:"and"`
				} `json:"where"`
			} `json:"prefetch"`
		}
		if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
			t.Fatalf("%s: %v", p.Path, err)
		}
		excluded := false
		for _, rule := range got.Prefetch {
			for _, cond := range rule.Where.And {
				excluded = excluded || strings.Contains(string(cond["not"]), `"selector_matches":["[data-weft-no-prefetch]","[data-weft-no-prefetch] *"]`)
			}
		}
		if !excluded {
			t.Errorf("%s: the speculation rules do not leave links in [data-weft-no-prefetch] alone: %s", p.Path, m[1])
		}
		for _, tag := range marked.FindAllString(html, -1) {
			checked++
			if !strings.Contains(tag, "data-weft-no-prefetch") {
				t.Errorf("%s: %s is not marked data-weft-no-prefetch", p.Path, tag)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no gallery page has a stage, a preview or a thumbnail")
	}
}

// Each gallery page keeps one main landmark and one h1, whatever its
// examples draw: a Page or an AppShell in a stage, a preview or a thumbnail
// is a preview of a page (WeftProvider inset), with no main and no h1.
func TestGalleryPagesHaveOneMainAndOneH1(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	main := regexp.MustCompile(`<main\b`)
	h1 := regexp.MustCompile(`<h1\b`)
	for _, p := range r.Pages {
		if p.Path != "/components" && !strings.HasPrefix(p.Path, "/components/") {
			continue
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		if n := len(main.FindAllString(html, -1)); n != 1 {
			t.Errorf("%s has %d main landmarks, want 1", p.Path, n)
		}
		if n := len(h1.FindAllString(html, -1)); n != 1 {
			t.Errorf("%s has %d h1, want 1", p.Path, n)
		}
	}
}

// Nor does an AppShell in an example remember its collapsed sidebar: in a
// preview (WeftProvider inset) it keeps nothing in localStorage, and no
// gallery page has the pre-paint that would collapse every AppShell drawn
// after one was collapsed in a stage.
func TestGalleryAppShellsAreNotRemembered(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	shells := 0
	for _, p := range r.Pages {
		if p.Path != "/components" && !strings.HasPrefix(p.Path, "/components/") {
			continue
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		shells += strings.Count(html, `class="weft-app-shell"`)
		// The pre-paint, not the props table, which names the default key.
		if pre := regexp.MustCompile(`<script data-weft-prepaint>[^<]*`).FindString(html); strings.Contains(pre, "weft:app-shell:collapsed") {
			t.Errorf("%s remembers a collapsed AppShell (weft:app-shell:collapsed in its pre-paint)", p.Path)
		}
	}
	if shells == 0 {
		t.Fatal("no gallery page draws an AppShell")
	}
}
