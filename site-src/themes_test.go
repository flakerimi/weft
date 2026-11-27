package main

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The header's Theme menu: Weft's Button opening a popover, which the menu's
// module fills with the site's six themes when it is first opened, on every
// page but those for agents (/skills), which keep the site's own theme; its
// pre-paint snippet; and its module, which names the other five themes'
// stylesheets, each a file of the build, and brings the menu's CSS.

// menuThemes is the menu's order: the label, and the name each button picks.
var menuThemes = []struct{ label, name string }{
	{"Weft", "weft"}, {"Neutral", "neutral"}, {"Lisa", "lisa"}, {"Synthwave", "synthwave"}, {"Editor", "editor"}, {"Sandstorm", "sandstorm"},
}

var (
	themeTrigger  = regexp.MustCompile(`<button class="weft-button site-theme"[^>]*popovertarget="site-themes"[^>]*><span class="weft-button__content">Theme</span></button>`)
	prePaint      = regexp.MustCompile(`<script data-weft-prepaint>([\s\S]*?)</script>`)
	themeSheetURL = regexp.MustCompile(`/assets/([a-z]+)-[0-9a-f]{8}\.css`)
)

// themeSheets is each of the five themes besides weft by name, with the
// URL of its stylesheet, as the menu's module (/theme-menu.js) names them.
func themeSheets(t *testing.T, d fs.FS) map[string]string {
	t.Helper()
	b, err := fs.ReadFile(d, "theme-menu.js")
	if err != nil {
		t.Fatalf("the build has no /theme-menu.js, the Theme menu's module: %v", err)
	}
	m := regexp.MustCompile(`const U = (\{[^}]*\})`).FindSubmatch(b)
	if m == nil {
		t.Fatal("/theme-menu.js names no stylesheets")
	}
	sheets := map[string]string{}
	if err := json.Unmarshal(m[1], &sheets); err != nil {
		t.Fatal(err)
	}
	for name, url := range sheets {
		if !regexp.MustCompile(`^/assets/` + name + `-[0-9a-f]{8}\.css$`).MatchString(url) {
			t.Errorf("/theme-menu.js names %s for %s, not a stylesheet of the build named for it", url, name)
		}
	}
	return sheets
}

func TestThemeMenu(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	files := map[string]string{"/no-such-page": *r.NotFound}
	for _, p := range r.Pages {
		files[p.Path] = p.HTML
	}
	for path, file := range files {
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		if path == "/skills" || strings.HasPrefix(path, "/skills/") {
			if strings.Contains(html, "site-theme") {
				t.Errorf("%s has the Theme menu; a page for agents keeps the site's own theme", path)
			}
			continue
		}
		header := element(html, `<header class="weft-header`)
		if !themeTrigger.MatchString(header) {
			t.Errorf("%s: the header has no Theme button opening the popover site-themes", path)
		}
		// Empty and hidden: the menu's module fills it when it is first opened.
		if !strings.Contains(header, `<div id="site-themes" class="site-themes" popover hidden role="group" aria-label="Theme"></div>`) {
			t.Errorf("%s: the header has no empty popover group labelled Theme", path)
		}
		script := prePaint.FindStringSubmatch(html)
		if script == nil || !strings.Contains(script[1], `"weft:site-theme"`) || !strings.Contains(script[1], `import("/theme-menu.js")`) {
			t.Errorf("%s has no pre-paint script that applies the kept theme and fetches the menu's module", path)
		}
	}
}

// Each theme besides weft is a stylesheet of its own, cached for good: its
// tokens scoped to <html data-site-theme="<name>"> (and to an element inside
// with a scheme of its own; neutral's and lisa's to the gallery stage's
// canvas too, which can force them), and the fonts it names that every page
// does not already have, self-hosted. No page inlines it or preloads its
// fonts.
func TestThemeStylesheets(t *testing.T) {
	d := dist(t)
	var m struct {
		Files map[string]struct {
			Type      string `json:"type"`
			Immutable bool   `json:"immutable"`
			Br        bool   `json:"br"`
		} `json:"files"`
		Hints map[string][]string `json:"hints"`
	}
	readJSON(t, d, "manifest.json", &m)
	sheets := themeSheets(t, d)
	if len(sheets) != len(menuThemes)-1 {
		t.Errorf("/theme-menu.js names %v, want the five themes besides weft", sheets)
	}
	module, err := fs.ReadFile(d, "theme-menu.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(module), `.site-themes`) || !strings.Contains(string(module), `"label":"Sandstorm"`) {
		t.Error("/theme-menu.js does not bring the menu's CSS and its themes' names")
	}
	// The module is the build's own, revalidated, so a page always gets the one that names the build's stylesheets.
	if f := m.Files["/theme-menu.js"]; f.Type != "text/javascript; charset=utf-8" || f.Immutable || !f.Br {
		t.Errorf("/theme-menu.js is served as %+v; want JavaScript, compressed and revalidated", f)
	}
	faces := map[string]string{"synthwave": "Space Grotesk Variable", "sandstorm": "Quicksand Variable"}
	fontFiles := map[string]string{"synthwave": "space-grotesk-latin-wght-normal", "sandstorm": "quicksand-latin-wght-normal"}
	selector := regexp.MustCompile(`(?:^|[}{])([^{}@]+)\{`)
	fontURL := regexp.MustCompile(`url\(([^)]+)\)`)
	for _, theme := range menuThemes[1:] {
		url := sheets[theme.name]
		if f, ok := m.Files[url]; !ok || !f.Immutable || !f.Br || f.Type != "text/css; charset=utf-8" {
			t.Errorf("%s's stylesheet %s is not in the build as CSS, compressed and cached for good: %+v", theme.name, url, f)
			continue
		}
		b, err := fs.ReadFile(d, strings.TrimPrefix(url, "/"))
		if err != nil {
			t.Fatal(err)
		}
		css := string(b)
		if !strings.Contains(css, "--weft-color-bg:") || !strings.Contains(css, "--weft-font-sans:") {
			t.Errorf("%s: %s sets no colours or fonts", theme.name, url)
		}
		// Neutral and lisa are also the gallery stage's to force on its canvas.
		canvas := theme.name == "neutral" || theme.name == "lisa"
		for _, s := range selector.FindAllStringSubmatch(css, -1) {
			for _, one := range strings.Split(s[1], ",") {
				if !strings.Contains(one, `[data-site-theme=`+theme.name+`]`) && !(canvas && strings.HasPrefix(strings.TrimSpace(one), `.gallery-canvas[data-theme=`+theme.name+`]`)) {
					t.Errorf("%s: %s has a rule for %q, which is not scoped to the theme", theme.name, url, one)
				}
			}
		}
		urls := fontURL.FindAllStringSubmatch(css, -1)
		face, hasFace := faces[theme.name]
		if !hasFace {
			if len(urls) != 0 || strings.Contains(css, "@font-face") {
				t.Errorf("%s: %s brings fonts %v; its fonts are every page's already, or the system's", theme.name, url, urls)
			}
			continue
		}
		if !strings.Contains(css, "@font-face{font-family:"+face+";") || len(urls) != 1 {
			t.Errorf("%s: %s has no one @font-face for %s (urls %v)", theme.name, url, face, urls)
			continue
		}
		font := urls[0][1]
		if !regexp.MustCompile(`^/assets/`+fontFiles[theme.name]+`-[0-9a-f]{8}\.woff2$`).MatchString(font) || !m.Files[font].Immutable {
			t.Errorf("%s: its font is %s, not the Latin file of its @fontsource-variable package in the build, cached for good", theme.name, font)
		}
		// Linked only when picked: no page preloads it, in its head or its Early Hints, or inlines the theme.
		for path, hints := range m.Hints {
			for _, h := range hints {
				if strings.Contains(h, fontFiles[theme.name]) {
					t.Errorf("%s preloads %s", path, font)
				}
			}
		}
	}
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	for _, p := range r.Pages {
		b, err := fs.ReadFile(d, p.HTML)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		css := strings.Join(regexp.MustCompile(`<style>([\s\S]*?)</style>`).FindAllString(html, -1), "")
		for _, link := range stylesheetLink.FindAllStringSubmatch(html, -1) {
			linked, err := fs.ReadFile(d, strings.TrimPrefix(link[1], "/"))
			if err != nil {
				t.Fatal(err)
			}
			css += string(linked)
		}
		for _, leak := range []string{"data-site-theme", "Space Grotesk", "Quicksand", "space-grotesk", "quicksand"} {
			if strings.Contains(css, leak) {
				t.Errorf("%s: its CSS has %s; a theme's CSS and fonts load only when it is picked", p.Path, leak)
			}
		}
	}
}
