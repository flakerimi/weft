package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The Theme menu in a real browser. Picking a theme switches the page in
// view, and the choice holds on the next page, on a reload and in light and
// dark, applied before the first paint: the first frame a page draws is
// already in it. Weft takes it all off again. On a docs page, a component
// page and the landing page, with no error in the console and nothing
// fetched from another site.

// themeGrounds is each theme's ground (--weft-color-bg) in light and in dark,
// as packages/themes/src gives it.
var themeGrounds = map[string][2]string{
	"weft":      {"#E8ECF4", "#0F1530"},
	"neutral":   {"#FAFAFA", "#09090B"},
	"lisa":      {"#F1F5F9", "#0F172A"},
	"synthwave": {"#F3EEFB", "#170E2E"},
	"editor":    {"#F5F0E6", "#23201C"},
	"sandstorm": {"#F1E7D8", "#1F1711"},
}

// firstFrame is a script each new document runs first. When the parser
// reaches <body> it keeps the theme on <html> and whether its stylesheet is
// in the head; at the first animation frame (after render-blocking
// stylesheets are in, before the first paint), the page's ground and theme
// and whether the browser counted the theme's stylesheet as blocking the
// render.
const firstFrame = `new MutationObserver((_, o) => { if (document.body) { o.disconnect(); window.__atBody = { theme: document.documentElement.dataset.siteTheme || 'weft', linked: !!document.querySelector('head link[data-site-theme]') } } }).observe(document, { childList: true, subtree: true });
requestAnimationFrame(() => { const s = getComputedStyle(document.documentElement), t = document.documentElement.dataset.siteTheme || 'weft', e = performance.getEntriesByType('resource').find((e) => e.name.includes('/assets/' + t + '-') && e.name.endsWith('.css')); window.__firstFrame = { bg: s.getPropertyValue('--weft-color-bg').trim().toUpperCase(), theme: t, blocking: e ? e.renderBlockingStatus : '', atBody: window.__atBody } })`

type frame struct {
	BG, Theme, Blocking string
	AtBody              struct {
		Theme  string
		Linked bool
	}
}

func (b *browser) firstFrame() frame {
	b.t.Helper()
	b.until(`window.__firstFrame`)
	var f frame
	b.eval(`window.__firstFrame`, &f)
	return f
}

// before checks that the theme was on before the first paint: on <html>, its
// stylesheet in the head before the body, blocking the render, and the first
// frame in its ground.
func (f frame) before(theme, ground string) string {
	linked := theme != "weft"
	blocking := map[bool]string{true: "blocking", false: ""}[linked]
	if f.AtBody.Theme != theme || f.AtBody.Linked != linked || f.Blocking != blocking || f.Theme != theme || f.BG != ground {
		return fmt.Sprintf("at <body> %s (stylesheet linked: %v), first frame %s on %s, its stylesheet %q; want %s (linked: %v) on %s, %q", f.AtBody.Theme, f.AtBody.Linked, f.Theme, f.BG, f.Blocking, theme, linked, ground, blocking)
	}
	return ""
}

// pick opens the Theme menu and presses the theme's button, as a person does.
func (b *browser) pick(name string) {
	b.t.Helper()
	b.click(`.site-theme`)
	// The menu's module fills it the first time.
	b.until(`document.querySelector('#site-themes').matches(':popover-open') && document.querySelector('[data-site-theme-pick="` + name + `"]')`)
	b.click(`[data-site-theme-pick="` + name + `"]`)
	want := name
	if name == "weft" {
		want = ""
	}
	b.until(`(document.documentElement.dataset.siteTheme || '') === ` + jsString(want) + ` && !document.querySelector('#site-themes').matches(':popover-open')`)
}

func TestThemeMenuSwitchesTheSite(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.call(b.session, "Runtime.enable", nil)
	b.call(b.session, "Log.enable", nil)
	b.call(b.session, "Network.enable", nil)
	// Every stylesheet from the server, never the cache: the first frame waits for the theme's because it blocks rendering, not because it is at hand.
	b.call(b.session, "Network.setCacheDisabled", map[string]any{"cacheDisabled": true})
	b.call(b.session, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": firstFrame})
	b.doing, b.deadline = "emulating 1280 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	scheme := func(s string) {
		b.call(b.session, "Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": s}}})
	}
	ground := func() string {
		var bg string
		b.eval(`getComputedStyle(document.documentElement).getPropertyValue('--weft-color-bg').trim().toUpperCase()`, &bg)
		return bg
	}
	// pressed opens the menu, as a person does, and says which theme it marks
	// as on; the menu is Weft's Buttons, one per theme in the menu's order.
	pressed := func() string {
		b.click(`.site-theme`)
		b.until(`document.querySelector('#site-themes').matches(':popover-open') && document.querySelectorAll('#site-themes [data-site-theme-pick]').length === 6`)
		var got struct {
			Labels, Picks []string
			Pressed       string
			Weft          bool
		}
		b.eval(`(() => {
			const bs = [...document.querySelectorAll('#site-themes [data-site-theme-pick]')]
			return {
				labels: bs.map((e) => e.textContent.trim()),
				picks: bs.map((e) => e.dataset.siteThemePick),
				pressed: bs.filter((e) => e.getAttribute('aria-pressed') === 'true').map((e) => e.dataset.siteThemePick).join(),
				weft: bs.every((e) => e.matches('button.weft-button[type="button"][data-variant="ghost"]') && !e.hasAttribute('popovertarget')),
			}
		})()`, &got)
		for i, want := range menuThemes {
			if i >= len(got.Labels) || got.Labels[i] != want.label || got.Picks[i] != want.name {
				t.Errorf("the menu is %v (%v), want %v", got.Labels, got.Picks, menuThemes)
				break
			}
		}
		if !got.Weft {
			t.Error("the menu's themes are not Weft's Buttons")
		}
		b.eval(`(document.querySelector('#site-themes').hidePopover(), true)`, new(bool))
		return got.Pressed
	}

	// What the console and the network saw on the page in view, before the next one clears it.
	check := func() {
		for _, e := range b.events {
			switch e.Method {
			case "Runtime.exceptionThrown":
				t.Errorf("an error was thrown: %s", e.Params)
			case "Log.entryAdded":
				var l struct {
					Entry struct {
						Level string `json:"level"`
					} `json:"entry"`
				}
				json.Unmarshal(e.Params, &l)
				if l.Entry.Level == "error" || l.Entry.Level == "warning" {
					t.Errorf("the browser logged a %s: %s", l.Entry.Level, e.Params)
				}
			case "Runtime.consoleAPICalled":
				var c struct {
					Type string `json:"type"`
				}
				json.Unmarshal(e.Params, &c)
				if c.Type == "error" || c.Type == "warning" {
					t.Errorf("the console has a %s: %s", c.Type, e.Params)
				}
			case "Network.requestWillBeSent":
				var req struct {
					Request struct {
						URL string `json:"url"`
					} `json:"request"`
				}
				json.Unmarshal(e.Params, &req)
				if u := req.Request.URL; !strings.HasPrefix(u, srv.URL) && !strings.HasPrefix(u, "data:") {
					t.Errorf("a page fetched %s, from another site", u)
				}
			}
		}
	}
	open := func(url string) {
		check()
		b.open(url)
	}

	pages := []string{"/docs/guides/themes", "/components/button", "/"}
	for _, s := range []string{"light", "dark"} {
		i := 0
		if s == "dark" {
			i = 1
		}
		scheme(s)
		for _, theme := range menuThemes {
			for n, path := range pages {
				open(srv.URL + path)
				b.doing, b.deadline = fmt.Sprintf("%s, %s on %s", s, theme.name, path), time.Now().Add(pageTimeout)
				if n == 0 {
					// Picked on the first page: it switches in place.
					b.pick(theme.name)
				} else if why := b.firstFrame().before(theme.name, themeGrounds[theme.name][i]); why != "" {
					// The next pages: already in it before their first paint.
					t.Errorf("%s, %s on %s: %s", s, theme.name, path, why)
				}
				if got := ground(); got != themeGrounds[theme.name][i] {
					t.Errorf("%s, %s: %s's ground is %s, want %s", s, theme.name, path, got, themeGrounds[theme.name][i])
				}
				if got := pressed(); got != theme.name {
					t.Errorf("%s, %s: %s's menu marks %q as on", s, theme.name, path, got)
				}
			}
			// A reload keeps it, from the first frame.
			open(srv.URL + pages[0])
			if why := b.firstFrame().before(theme.name, themeGrounds[theme.name][i]); why != "" {
				t.Errorf("%s, %s, reloaded: %s", s, theme.name, why)
			}
		}
	}

	// Each theme's type: Synthwave's Space Grotesk and Sandstorm's Quicksand load with it, from the site.
	scheme("light")
	for theme, face := range map[string]string{"synthwave": "Space Grotesk Variable", "sandstorm": "Quicksand Variable", "editor": "JetBrains Mono Variable"} {
		open(srv.URL + "/docs")
		b.pick(theme)
		var loaded bool
		b.eval(`document.fonts.ready.then(() => document.fonts.check('16px "`+face+`"') && getComputedStyle(document.body).fontFamily.startsWith('"`+face+`"'))`, &loaded)
		if !loaded {
			t.Errorf("%s: the page is not set in %s", theme, face)
		}
	}
	// The landing page's night band (data-weft-scheme="dark") is the theme's dark, in light too.
	open(srv.URL + "/")
	b.pick("sandstorm")
	var band string
	b.eval(`getComputedStyle(document.querySelector('.landing-top')).getPropertyValue('--weft-color-bg').trim().toUpperCase()`, &band)
	if band != themeGrounds["sandstorm"][1] {
		t.Errorf("sandstorm: the landing page's night band is %s, want its dark ground %s", band, themeGrounds["sandstorm"][1])
	}
	// The gallery's stage shows the site's theme, unless its toolbar forces lisa (here by its link), whose stylesheet it links then.
	open(srv.URL + "/components/button")
	b.doing, b.deadline = "the stage in sandstorm", time.Now().Add(pageTimeout)
	var canvas string
	b.eval(`getComputedStyle(document.querySelector('.gallery-canvas')).getPropertyValue('--weft-color-bg').trim().toUpperCase()`, &canvas)
	if canvas != themeGrounds["sandstorm"][0] {
		t.Errorf("sandstorm: the stage's canvas is %s, want the site's theme's ground %s", canvas, themeGrounds["sandstorm"][0])
	}
	open(srv.URL + "/components/button?theme=lisa")
	b.doing, b.deadline = "the stage forced to lisa", time.Now().Add(pageTimeout)
	b.until(`document.querySelector('.gallery-canvas[data-theme="lisa"]') && getComputedStyle(document.querySelector('.gallery-canvas')).getPropertyValue('--weft-color-bg').trim().toUpperCase() === ` + jsString(themeGrounds["lisa"][0]))
	// A kept stylesheet a later build renamed: the page mends it, linking the build's and keeping that.
	var current string
	b.eval(`localStorage.getItem('weft:site-theme')`, &current)
	b.eval(`(localStorage.setItem('weft:site-theme', 'sandstorm /assets/sandstorm-00000000.css'), true)`, new(bool))
	open(srv.URL + "/docs")
	b.doing, b.deadline = "mending a renamed stylesheet", time.Now().Add(pageTimeout)
	b.until(`localStorage.getItem('weft:site-theme') === ` + jsString(current) + ` && document.querySelectorAll('link[data-site-theme]').length === 1 && getComputedStyle(document.documentElement).getPropertyValue('--weft-color-bg').trim().toUpperCase() === ` + jsString(themeGrounds["sandstorm"][0]))
	// The failed fetch of the old name is the browser's to report, and expected here.
	b.events = nil
	// Weft takes the attribute, the stylesheet and the kept choice off.
	b.pick("weft")
	var left struct {
		Links int
		Kept  *string
	}
	b.eval(`({ links: document.querySelectorAll('link[data-site-theme]').length, kept: localStorage.getItem('weft:site-theme') })`, &left)
	if left.Links != 0 || left.Kept != nil || ground() != themeGrounds["weft"][0] {
		t.Errorf("after Weft: %d theme stylesheets linked, %v kept, ground %s", left.Links, left.Kept, ground())
	}

	check()
}
