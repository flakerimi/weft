package main

import (
	"encoding/json"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The landing page, /, as the build made it from pages/index.vue and the
// approved design's second pass (docs/design/site-2026-11-21-v2/): the night
// hero with one app drawn twice by the real components (a desktop and a
// phone), the bento of screens, the adaptive demo,
// the benchmark bars from bench/results/, and the agents section from the
// catalogue.

// benchResults is the parts of bench/results/<date>.json the page shows.
type benchResults struct {
	Date    string `json:"date"`
	Machine struct {
		Chip string `json:"chip"`
	} `json:"machine"`
	Targets []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Load  struct {
			RPS struct {
				Median float64 `json:"median"`
			} `json:"rps"`
		} `json:"load"`
		TTFB struct {
			MedianMS float64 `json:"median_ms"`
		} `json:"ttfb"`
	} `json:"targets"`
}

// newestBench is the results file with the latest date in its name.
func newestBench(t *testing.T) (string, benchResults) {
	t.Helper()
	files, err := filepath.Glob("../bench/results/*.json")
	if err != nil {
		t.Fatal(err)
	}
	dated := regexp.MustCompile(`/\d{4}-\d{2}-\d{2}\.json$`)
	var names []string
	for _, f := range files {
		if dated.MatchString(filepath.ToSlash(f)) {
			names = append(names, f)
		}
	}
	if len(names) == 0 {
		t.Fatal("bench/results has no results named <date>.json")
	}
	sort.Strings(names)
	b, err := os.ReadFile(names[len(names)-1])
	if err != nil {
		t.Fatal(err)
	}
	var r benchResults
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return names[len(names)-1], r
}

// thousands writes 124503 as 124,503.
func thousands(n float64) string {
	s := strconv.FormatInt(int64(n+0.5), 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

var (
	tag        = regexp.MustCompile(`<[^>]+>`)
	landingH2  = regexp.MustCompile(`<h2 id="[a-z]+-title">(.*?)</h2>`)
	islandTags = regexp.MustCompile(`<weft-island data-id="([^"]+)" data-when="([^"]+)"`)
)

// text is what an HTML fragment reads as: no tags, no hydration markers, entities decoded.
func text(fragment string) string {
	return html.UnescapeString(tag.ReplaceAllString(htmlComment.ReplaceAllString(fragment, ""), ""))
}

func landing(t *testing.T) (fs.FS, string) {
	t.Helper()
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	for _, p := range r.Pages {
		if p.Path == "/" {
			b, err := fs.ReadFile(d, p.HTML)
			if err != nil {
				t.Fatal(err)
			}
			return d, string(b)
		}
	}
	t.Fatal("routes.json has no page at /")
	return nil, ""
}

// The sections in the design's order, the hero's copy, and the islands: the
// copy button when idle and the adaptive demo when it comes into view.
func TestLandingPage(t *testing.T) {
	_, page := landing(t)
	var got []string
	for _, m := range landingH2.FindAllStringSubmatch(page, -1) {
		got = append(got, text(m[1]))
	}
	want := []string{
		"Screens you’d ship, made from the parts on this site.",
		"Write a component once. It adapts.",
		"One binary, nothing left to do per request.",
		"Made to be built with an agent.",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("the sections are %q, want %q", got, want)
	}
	h1 := element(page, "<h1")
	if text(h1) != "One component tree, every screen." {
		t.Errorf("the h1 is %q", text(h1))
	}
	// The header's wordmark is the owner's file, drawn as a mask; the hero's thread is CSS.
	if !strings.Contains(page, `<span class="landing-thread" aria-hidden="true">`) {
		t.Error("the hero has no thread")
	}
	if !strings.Contains(page, "url(/weft.svg)") {
		t.Error("the wordmark is not drawn from /weft.svg")
	}
	if !strings.Contains(element(page, `class="landing-top"`), `data-weft-scheme="dark"`) {
		t.Error("the header is not drawn light on the night hero")
	}
	if !strings.Contains(text(element(page, `class="landing-install"`)), "curl -fsSL https://weft.base.al/install | sh") {
		t.Error("the hero has no install command")
	}
	var islands []string
	for _, m := range islandTags.FindAllStringSubmatch(page, -1) {
		islands = append(islands, m[1]+"@"+m[2])
	}
	if strings.Join(islands, " ") != "CopyCommand@idle AdaptiveDemo@visible" {
		t.Errorf("the islands are %v, want the copy button when idle and the adaptive demo when visible", islands)
	}
	// A keyboard reaches the content first; nothing comes from another site.
	if !strings.Contains(page, `<a class="site-skip" href="#main">`) || !regexp.MustCompile(`<main [^>]*id="main"`).MatchString(page) {
		t.Error("no skip link to the main content")
	}
	if regexp.MustCompile(`(?i)(src|href)="https?:|url\(https?:`).MatchString(page) {
		t.Error("the landing page links or loads something on another site")
	}
}

// The page's one automatic motion is the hero thread pulling across, and
// it stays still when motion is reduced.
func TestLandingMotion(t *testing.T) {
	css, err := os.ReadFile("styles/landing.css")
	if err != nil {
		t.Fatal(err)
	}
	s := string(css)
	if n := strings.Count(s, "@keyframes"); n != 1 {
		t.Errorf("landing.css has %d @keyframes, want 1 (the thread)", n)
	}
	if n := len(regexp.MustCompile(`\banimation(-name)?\s*:`).FindAllString(s, -1)); n != 2 {
		t.Errorf("landing.css sets animation %d times, want 2 (the thread, and none when motion is reduced)", n)
	}
	reduced := regexp.MustCompile(`(?s)@media \(prefers-reduced-motion: reduce\) \{\s*\.landing-thread \{ animation: none;`)
	if !reduced.MatchString(s) {
		t.Error("the thread still moves when motion is reduced")
	}
	_, page := landing(t)
	if !strings.Contains(page, "prefers-reduced-motion:reduce") && !strings.Contains(page, "prefers-reduced-motion: reduce") {
		t.Error("the page's CSS does not stop the thread when motion is reduced")
	}
}

// Without JavaScript the demo reads: the code, then both forms drawn by the
// server as labelled, inert previews (the real DatePicker and Calendar, in
// pointer and touch), and a note in the island's place.
func TestLandingWithoutJavaScript(t *testing.T) {
	_, page := landing(t)
	source, err := os.ReadFile("components/landing/VisitDay.vue")
	if err != nil {
		t.Fatal(err)
	}
	code := element(page, `class="landing-code landing-code--source"`)
	if strings.TrimSpace(text(element(code, "<pre"))) != strings.TrimSpace(string(source)) {
		t.Error("the demo's code is not components/landing/VisitDay.vue as it is")
	}
	for _, p := range []struct{ marker, title, input, calendar string }{
		{`class="landing-frame landing-frame--pointer"`, "Mouse and keyboard", "pointer", `data-form="field"`},
		{`class="landing-frame landing-frame--touch"`, "Touch", "touch", `data-form="button"`},
	} {
		frame := element(page, p.marker)
		if frame == "" {
			t.Errorf("no %s preview", p.input)
			continue
		}
		if !strings.Contains(text(element(frame, "<figcaption")), p.title) || !strings.Contains(text(element(frame, "<figcaption")), "Preview") {
			t.Errorf("the %s preview is not labelled %q, a preview", p.input, p.title)
		}
		preview := element(frame, `class="landing-preview"`)
		if !regexp.MustCompile(`^<div class="landing-preview" inert`).MatchString(preview) {
			t.Errorf("the %s preview is not inert", p.input)
		}
		if !strings.Contains(preview, `data-weft-input="`+p.input+`"`) || !strings.Contains(preview, p.calendar) || !strings.Contains(preview, "weft-calendar") {
			t.Errorf("the %s preview is not the DatePicker and its calendar drawn for %s", p.input, p.input)
		}
	}
	demo := element(page, `data-id="AdaptiveDemo"`)
	if !strings.Contains(text(demo), "JavaScript") {
		t.Error("before it starts, the demo's island does not say what it needs")
	}
	if strings.Contains(demo, "weft-date-picker") || strings.Contains(demo, "weft-toggle-group") {
		t.Error("the demo's island draws controls that cannot work before it starts")
	}
}

// The bars are the newest results file's, never typed into the page.
func TestLandingBenchmark(t *testing.T) {
	file, r := newestBench(t)
	_, page := landing(t)
	bench := element(page, `class="landing-bench"`)
	if bench == "" {
		t.Fatal("no benchmark bars")
	}
	rows := strings.Count(bench, `role="row"`)
	if rows != len(r.Targets) {
		t.Errorf("%d bars for %d targets in %s", rows, len(r.Targets), file)
	}
	for _, target := range r.Targets {
		n := thousands(target.Load.RPS.Median)
		if !strings.Contains(text(bench), n) {
			t.Errorf("%s: no bar reads %s requests per second", target.ID, n)
		}
		if !strings.Contains(text(bench), target.Label) {
			t.Errorf("%s: no bar is labelled %q", target.ID, target.Label)
		}
		if target.ID == "weft" {
			ttfb := strconv.FormatFloat(target.TTFB.MedianMS, 'f', 2, 64) + " ms"
			if !strings.Contains(text(page), ttfb+" to first byte") {
				t.Errorf("the facts do not say %s to first byte", ttfb)
			}
		}
	}
	note := text(element(page, `class="landing-bench-note"`))
	for _, must := range []string{r.Machine.Chip, "Part of the gap is the server", "less JavaScript"} {
		if !strings.Contains(note, must) {
			t.Errorf("the note under the bars does not say %q", must)
		}
	}
	// Typed into no source file of the page.
	for _, f := range []string{"pages/index.vue", "landing/model.ts", "styles/landing.css"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range r.Targets {
			for _, n := range []string{thousands(target.Load.RPS.Median), strconv.Itoa(int(target.Load.RPS.Median))} {
				if strings.Contains(string(b), n) {
					t.Errorf("%s has %s typed in; read it from bench/results", f, n)
				}
			}
		}
	}
}

// The agents section shows the DatePicker's catalogue entry, line for line,
// and links the skills and llms.txt.
func TestLandingAgents(t *testing.T) {
	_, _, plain := readCatalog(t)
	var entry map[string]any
	for _, c := range plain {
		if m := c.(map[string]any); m["name"] == "DatePicker" {
			entry = m
		}
	}
	if entry == nil {
		t.Fatal("the catalogue has no DatePicker")
	}
	_, page := landing(t)
	agents := element(page, `id="agents"`)
	code := text(element(agents, "<pre"))
	for _, key := range []string{"name", "purpose"} {
		want, _ := json.Marshal(entry[key])
		if !strings.Contains(code, `"`+key+`": `+string(want)) {
			t.Errorf("the agents section does not show the DatePicker's %s as the catalogue has it", key)
		}
	}
	input, _ := json.Marshal(entry["adapts"].(map[string]any)["input"])
	if !strings.Contains(code, `"input": `+strings.ReplaceAll(string(input), `>`, ">")) {
		t.Error("the agents section does not show how the DatePicker adapts to input, as the catalogue has it")
	}
	for _, href := range []string{`href="/skills"`, `href="/llms.txt"`, `href="/components/date-picker.json"`, `href="/catalog.json"`} {
		if !strings.Contains(agents, href) {
			t.Errorf("the agents section has no link %s", href)
		}
	}
}

// The landing page's budgets, brotli, all bytes. First view at most 22 KB:
// what a browser fetches before an idle island starts, which is the HTML
// with its CSS inlined, any stylesheet it links, the loader (boot-*.js) and
// every module the page or its Early Hints preload (fonts are cached and
// preloaded, so not counted). Neither island starts on load, so nothing of
// the runtime is in it. Then the runtime (client-*.js, Vue and the
// hydration, fetched once when the copy button starts at idle) and each
// island's chunk with the chunks it imports: with the first view, what the
// page loads before anyone touches it. Then what the islands load on
// demand: the date picker's popover and calendar, fetched when the pointer
// or focus first comes to it.
//
// The second design pass (docs/design/site-2026-11-21-v2) ruled the first
// view up from 12 KB to 22 KB and the landing page's total, runtime
// included, to 82 KB: the hero draws one app twice (a desktop and a phone)
// and the bento five screens, all with the real components, as HTML and CSS
// from the server, so the first view grows by their markup and their
// components' CSS, and nothing else does. The total is what a visitor loads
// without touching anything: the first view, the runtime and the islands.
// What the islands load on demand (the date picker's popover and calendar,
// when the pointer or focus first comes to it) keeps the limit the polish
// pass set, which moved them to a chunk of their own: 15,172 B then, plus
// 10 %; and the demo's island likewise 14,790 B plus 10 %. (Counting the
// chunks loaded on demand as well, as the 75 KB limit of the final fix wave
// did, could not hold both rulings: the islands and those chunks alone are
// about 63.6 KB, leaving 20.3 KB, not 22, for the first view.) Islands load
// only what they need: the copy button's chunk imports nothing of the
// demo's, and the demo loads when it comes into view.
const (
	landingFirstViewLimit = 22 * 1024
	landingTotalLimit     = 82 * 1024
	landingDemoLimit      = 16269 // 14,790 B plus 10 %
	landingOnDemandLimit  = 16689 // 15,172 B plus 10 %
)

func TestLandingBudgets(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	for _, p := range r.Pages {
		if p.Path != "/" {
			continue
		}
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		page := string(b)
		first := firstView(t, d, m, p.Path, p.HTML, page)
		runtime := pageRuntime(t, d, page)
		if runtime == "" {
			t.Fatal("the landing page loads no runtime")
		}
		later := map[string]bool{}
		staticImports(t, d, runtime, later)
		for _, key := range islandChunks(t, d, runtime, p.Islands) {
			later[key] = true
		}
		// What first view already fetched is not fetched again.
		seen := map[string]bool{}
		staticImports(t, d, moduleScript.FindStringSubmatch(page)[1], seen)
		rest := 0
		for key := range later {
			if !seen[key] {
				rest += brotliSize(t, d, m, key)
			}
		}
		// What the islands load on demand, and what that imports, not fetched before.
		onDemand := 0
		for key := range dynamicImports(t, d, keys(later)) {
			if !later[key] && !seen[key] {
				onDemand += brotliSize(t, d, m, key)
			}
		}
		copyChunks := 0
		for _, key := range islandChunks(t, d, runtime, []string{"CopyCommand"}) {
			copyChunks += brotliSize(t, d, m, key)
		}
		demo := 0
		for _, key := range islandChunks(t, d, runtime, []string{"AdaptiveDemo"}) {
			demo += brotliSize(t, d, m, key)
		}
		t.Logf("landing, brotli: first view %d B; then the runtime and islands %d B (copy button %d B, demo %d B, runtime %d B); total %d B; then on demand %d B (%d B with it)",
			first, rest, copyChunks, demo, brotliSize(t, d, m, runtime), first+rest, onDemand, first+rest+onDemand)
		if first > landingFirstViewLimit {
			t.Errorf("the landing page's first view is %d B brotli, over %d B", first, landingFirstViewLimit)
		}
		if first+rest > landingTotalLimit {
			t.Errorf("the landing page with its runtime and islands is %d B brotli, over %d B", first+rest, landingTotalLimit)
		}
		if onDemand > landingOnDemandLimit {
			t.Errorf("the landing page's islands load %d B brotli on demand, over %d B", onDemand, landingOnDemandLimit)
		}
		if demo > landingDemoLimit {
			t.Errorf("the adaptive demo's island starts with %d B brotli, over %d B; what it draws only when open belongs in a chunk it loads on demand", demo, landingDemoLimit)
		}
		if onDemand == 0 {
			t.Error("the islands load nothing on demand; the date picker's popover and calendar are in the demo's chunks")
		}
		if copyChunks > 2*1024 {
			t.Errorf("the copy button loads %d B brotli; it imports more than it needs", copyChunks)
		}
		if pre := modulePreload.FindAllStringSubmatch(page, -1); len(pre) != 0 {
			t.Errorf("the landing page preloads %v; no island starts on load, so nothing is preloaded", pre)
		}
		if n := len(moduleScript.FindAllString(page, -1)); n != 1 {
			t.Errorf("the landing page has %d module scripts, want 1", n)
		}
		// The demo's Vue features stay out of the runtime the docs and skills
		// load for their copy buttons (package.json gives those sections their own).
		for _, q := range r.Pages {
			if (strings.HasPrefix(q.Path, "/docs") || strings.HasPrefix(q.Path, "/skills")) && len(q.Islands) > 0 {
				b, err := fs.ReadFile(d, q.HTML)
				if err != nil {
					t.Fatal(err)
				}
				if pageRuntime(t, d, string(b)) == runtime {
					t.Errorf("%s loads the landing page's runtime %s", q.Path, runtime)
				}
			}
		}
		return
	}
	t.Fatal("routes.json has no page at /")
}

var dynamicImport = regexp.MustCompile("import\\(\\s*[`\"']\\./([\\w.-]+\\.js)[`\"']")

// dynamicImports is every chunk the given chunks import on demand
// (`import("./x.js")`), with what those import, statically or on demand,
// the runtime and the loader aside.
func dynamicImports(t *testing.T, d fs.FS, from []string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	var walk func(key string)
	walk = func(key string) {
		b, err := fs.ReadFile(d, strings.TrimPrefix(key, "/"))
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range dynamicImport.FindAllStringSubmatch(string(b), -1) {
			lazy := "/assets/" + imp[1]
			if out[lazy] || strings.HasPrefix(lazy, "/assets/client-") || strings.HasPrefix(lazy, "/assets/boot-") {
				continue
			}
			static := map[string]bool{}
			staticImports(t, d, lazy, static)
			for k := range static {
				if !strings.HasPrefix(k, "/assets/client-") && !strings.HasPrefix(k, "/assets/boot-") && !out[k] {
					out[k] = true
					walk(k)
				}
			}
		}
	}
	for _, key := range from {
		if !strings.HasPrefix(key, "/assets/client-") && !strings.HasPrefix(key, "/assets/boot-") {
			walk(key)
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// The hero draws one app, Northline's device list, twice with the real
// components, each frame drawn by the server and inert, with no script and
// no prefetch: a desktop (pointer) with its sidebar, breadcrumb, actions,
// three stat cards, a table with badges and a menu drawn open; and a phone
// (touch) where the same table is its cards and the same menu a bottom
// sheet over its backdrop. A screen reader is told what the picture shows.
func TestLandingHeroFrames(t *testing.T) {
	_, page := landing(t)
	hero := element(page, `class="landing-hero"`)
	if hero == "" {
		t.Fatal("no hero")
	}
	for _, f := range []struct {
		marker, input string
		must          []string
	}{
		{`class="hero-frame hero-frame--desk"`, "pointer", []string{`class="weft-nav`, `class="weft-breadcrumb`, `data-layout="table"`, `class="weft-badge`, `role="menu"`, `data-form="floating"`, "Northline", "Prishtina"}},
		{`class="hero-frame hero-frame--phone"`, "touch", []string{`data-layout="cards"`, `class="weft-badge`, `role="menu"`, `data-form="sheet"`, "weft-overlay-backdrop"}},
	} {
		frame := element(hero, f.marker)
		if frame == "" {
			t.Errorf("the hero has no %s frame", f.input)
			continue
		}
		if start := frame[:strings.Index(frame, ">")]; !regexp.MustCompile(`\sinert[\s>=]|\sinert$`).MatchString(start) || !strings.Contains(start, "data-weft-no-prefetch") {
			t.Errorf("the %s frame is not inert and kept from prefetching", f.input)
		}
		if !strings.Contains(frame, `data-weft-input="`+f.input+`"`) {
			t.Errorf("the %s frame is not drawn for %s", f.input, f.input)
		}
		for _, m := range f.must {
			if !strings.Contains(frame, m) {
				t.Errorf("the %s frame has no %s", f.input, m)
			}
		}
		if strings.Contains(frame, "<script") || strings.Contains(frame, "<weft-island") {
			t.Errorf("the %s frame carries a script or an island", f.input)
		}
	}
	if n := strings.Count(element(hero, `class="hero-frame hero-frame--desk"`), `class="weft-stat-card"`); n != 3 {
		t.Errorf("the desktop frame has %d stat cards, want 3", n)
	}
	if text(element(hero, `class="landing-frames-note`)) == "" {
		t.Error("the frames say nothing to a screen reader")
	}
}

// "Screens you'd ship": five scenes on tiles, each drawn by the server from
// the real components, inert, with a caption that names the scene and links
// each component it uses to its page.
func TestLandingScreens(t *testing.T) {
	_, page := landing(t)
	screens := element(page, `id="screens"`)
	if screens == "" {
		t.Fatal("no screens section")
	}
	for _, s := range []struct {
		scene      string
		components []string
		must       string
	}{
		{"Settings", []string{"Form", "FormField", "Input", "Select", "RadioGroup", "Switch", "Button"}, `class="weft-switch`},
		{"Search anything", []string{"CommandPalette", "Kbd"}, "<mark>"},
		{"Maintenance window", []string{"DateRangePicker", "Calendar"}, `class="weft-calendar`},
		{"Activity", []string{"Timeline", "Avatar"}, `class="weft-avatar`},
		{"Restore", []string{"FileUpload", "Progress", "Toast"}, `class="weft-toast`},
	} {
		at := strings.Index(screens, `<strong>`+s.scene+`</strong>`)
		if at == -1 {
			t.Errorf("no tile is captioned %q", s.scene)
			continue
		}
		tile := screens[strings.LastIndex(screens[:at], `<figure class="landing-tile`):]
		tile = element(tile, `<figure`)
		picture := element(tile, `class="landing-tile__stage"`)
		if start := picture[:strings.Index(picture, ">")]; !regexp.MustCompile(`\sinert[\s>=]|\sinert$`).MatchString(start) || !strings.Contains(start, "data-weft-no-prefetch") {
			t.Errorf("%s: the picture is not inert and kept from prefetching", s.scene)
		}
		if !strings.Contains(picture, s.must) {
			t.Errorf("%s: the picture has no %s", s.scene, s.must)
		}
		caption := element(tile, `<figcaption`)
		for _, name := range s.components {
			if !strings.Contains(caption, `href="/components/`+slugOf(name)+`"`) {
				t.Errorf("%s: the caption does not link %s to /components/%s", s.scene, name, slugOf(name))
			}
		}
	}
}

