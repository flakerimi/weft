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
// approved design (docs/design/site-2026-11-21/): the wordmark and its
// thread, the adaptive demo, the benchmark bars from bench/results/, and the
// agents section from the catalogue.

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
	want := []string{"Write a component once. It adapts.", "One binary, nothing left to do per request.", "Made to be built with an agent."}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("the sections are %q, want %q", got, want)
	}
	h1 := element(page, "<h1")
	if text(h1) != "One UI system for Go and Vue apps." {
		t.Errorf("the h1 is %q", text(h1))
	}
	// The wordmark is the owner's file, drawn as a mask, named for a screen reader; its thread is CSS.
	if !strings.Contains(page, `<span class="landing-word" role="img" aria-label="weft">`) || !strings.Contains(page, `<span class="landing-thread" aria-hidden="true">`) {
		t.Error("the hero has no wordmark with its thread")
	}
	if !strings.Contains(page, "url(/weft.svg)") {
		t.Error("the hero's wordmark is not drawn from /weft.svg")
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

// The landing page's budgets, brotli, all bytes. First view at most 12 KB:
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
// In the polish pass the date picker moved its popover and calendar to a
// chunk it loads on demand. Before, the page loaded 73,968 B before anyone
// touched it, the demo's island 28,237 B of it; after, 60,490 B, the demo's
// island 14,790 B, and 15,172 B on demand. Each limit below is the after
// figure plus 10 %. All of it together (75,662 B, a little more than before,
// as two chunks compress less well than one) stays under the 75 KB the
// controller ruled in the final fix wave, when the real DatePicker and the
// runtime were first counted. Islands load only what they need: the copy
// button's chunk imports nothing of the demo's, and the demo loads when it
// comes into view.
const (
	landingUntouchedLimit = 66539 // 60,490 B plus 10 %
	landingDemoLimit      = 16269 // 14,790 B plus 10 %
	landingOnDemandLimit  = 16689 // 15,172 B plus 10 %
	landingAllLimit       = 75 * 1024
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
		total := first + rest + onDemand
		t.Logf("landing, brotli: first view %d B; then the runtime and islands %d B (copy button %d B, demo %d B, runtime %d B); on demand %d B; total %d B",
			first, rest, copyChunks, demo, brotliSize(t, d, m, runtime), onDemand, total)
		if first > 12*1024 {
			t.Errorf("the landing page's first view is %d B brotli, over 12 KB", first)
		}
		if first+rest > landingUntouchedLimit {
			t.Errorf("the landing page with its runtime and islands is %d B brotli, over %d B", first+rest, landingUntouchedLimit)
		}
		if onDemand > landingOnDemandLimit {
			t.Errorf("the landing page's islands load %d B brotli on demand, over %d B", onDemand, landingOnDemandLimit)
		}
		if total > landingAllLimit {
			t.Errorf("the landing page with its runtime, islands and what they load on demand is %d B brotli, over %d B", total, landingAllLimit)
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
