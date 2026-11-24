package main

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The landing hero as a browser lays it out, at the widths where both frames
// show and in dark: the phone covers none of the desktop's stat values nor
// its first three table rows; the thread passes behind both frames and
// crosses no text; the desktop's open menu hangs from its trigger's cell,
// as the component places it. And on a phone the install command stays on
// one line, scrolling inside its box, its Copy button whole; and the prompt
// for agents, with its button, is in the first screen.
func TestLandingHeroLayout(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	for _, scheme := range []string{"light", "dark"} {
		b.call(b.session, "Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{
			{"name": "prefers-color-scheme", "value": scheme}, {"name": "prefers-reduced-motion", "value": "reduce"},
		}})
		for _, width := range []int{1024, 1280, 1440, 2560} {
			b.doing, b.deadline = fmt.Sprintf("the hero at %d px, %s", width, scheme), time.Now().Add(pageTimeout)
			b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": 1000, "deviceScaleFactor": 1, "mobile": false})
			b.open(srv.URL + "/")
			b.eval(`document.fonts.ready.then(() => true)`, new(bool))
			var got struct {
				Covered, OverThread, ThreadOnText, Menu []string
			}
			b.eval(`(() => {
				const box = (e) => e.getBoundingClientRect()
				const meets = (a, b) => a.left < b.right - 1 && b.left < a.right - 1 && a.top < b.bottom - 1 && b.top < a.bottom - 1
				const name = (e) => (e.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 30)
				const desk = document.querySelector('.hero-frame--desk'), phone = document.querySelector('.hero-frame--phone')
				const thread = document.querySelector('.landing-thread')
				const covered = [], overThread = [], threadOnText = [], menu = []
				// The stat values and the first three rows, under no part of the phone.
				const must = [...desk.querySelectorAll('.weft-stat-card')].concat([...desk.querySelectorAll('tbody tr')].slice(0, 3))
				for (const e of must) if (meets(box(e), box(phone))) covered.push(name(e))
				// Along the thread, where a frame is, the frame is on top. (An inert
				// element is not hit, so the frames are made hittable for this.)
				for (const f of [desk, phone]) f.inert = false
				const t = box(thread), y = t.top + t.height / 2
				for (let x = 2; x < innerWidth; x += 8) {
					const top = document.elementFromPoint(x, y)
					const inFrame = [desk, phone].some((f) => { const r = box(f); return x > r.left && x < r.right && y > r.top && y < r.bottom })
					if (inFrame && top === thread) overThread.push(x)
				}
				for (const f of [desk, phone]) f.inert = true
				// No text the page shows lies across the thread.
				const walker = document.createTreeWalker(document.querySelector('.landing-hero'), NodeFilter.SHOW_TEXT)
				for (let n; (n = walker.nextNode()); ) {
					if (!n.textContent.trim() || n.parentElement.closest('[inert], .weft-visually-hidden')) continue
					const r = document.createRange()
					r.selectNodeContents(n)
					for (const rect of r.getClientRects()) if (meets(rect, t)) threadOnText.push(n.textContent.trim().slice(0, 30))
				}
				// The open menu hangs from its trigger's cell, at its end edge, 6 px
				// below it as the component places it (less where the frames are drawn smaller).
				const panel = desk.querySelector('[role="menu"]'), cell = panel.closest('td')
				const p = box(panel), c = box(cell), zoom = parseFloat(getComputedStyle(desk.parentElement).zoom) || 1
				if (Math.abs(p.top - (c.bottom + 6 * zoom)) > 2 || Math.abs(p.right - c.right) > 2) menu.push('menu at ' + Math.round(p.left) + ',' + Math.round(p.top) + '; cell ends ' + Math.round(c.right) + ',' + Math.round(c.bottom))
				return { covered, overThread: overThread.slice(0, 3).map(String), threadOnText, menu }
			})()`, &got)
			if len(got.Covered) > 0 {
				t.Errorf("%s: the phone covers %s", b.doing, strings.Join(got.Covered, " | "))
			}
			if len(got.OverThread) > 0 {
				t.Errorf("%s: the thread lies over a frame at x = %s", b.doing, strings.Join(got.OverThread, ", "))
			}
			if len(got.ThreadOnText) > 0 {
				t.Errorf("%s: the thread crosses text: %s", b.doing, strings.Join(got.ThreadOnText, " | "))
			}
			if len(got.Menu) > 0 {
				t.Errorf("%s: the desktop's menu is not hung from its trigger's cell: %s", b.doing, strings.Join(got.Menu, "; "))
			}
		}
	}

	b.doing, b.deadline = "the install command at 390 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": true})
	b.open(srv.URL + "/")
	b.eval(`document.fonts.ready.then(() => true)`, new(bool))
	var cmd struct {
		Lines        float64
		Scrolls      bool
		Button, Text float64
	}
	b.eval(`(() => {
		const code = document.querySelector('.landing-install .copy-command__text'), button = document.querySelector('.landing-install .copy-button')
		const style = getComputedStyle(code)
		return { lines: (code.clientHeight - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom)) / parseFloat(style.lineHeight), scrolls: code.scrollWidth > code.clientWidth, button: button.getBoundingClientRect().width, text: button.scrollWidth }
	})()`, &cmd)
	if cmd.Lines > 1.2 {
		t.Errorf("at 390 px the install command wraps onto %.1f lines; it stays on one and scrolls", cmd.Lines)
	}
	if !cmd.Scrolls {
		t.Error("at 390 px the install command does not scroll inside its box; it is cut or squeezed")
	}
	if cmd.Button < 60 {
		t.Errorf("at 390 px the Copy button is %.0f px wide; it keeps its size", cmd.Button)
	}

	// The prompt for agents is the first thing to do: on a phone the headline,
	// the prompt's box and its button are in the first screen, the box shows
	// at most two lines of it (the rest is cut with an ellipsis, not lost:
	// the whole text is in the page), and the button is whole.
	for _, size := range [][2]int{{390, 844}, {320, 844}} {
		b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": size[0], "height": size[1], "deviceScaleFactor": 1, "mobile": true})
		b.open(srv.URL + "/")
		b.doing, b.deadline = fmt.Sprintf("the prompt for agents at %d × %d", size[0], size[1]), time.Now().Add(pageTimeout)
		b.eval(`document.fonts.ready.then(() => true)`, new(bool))
		var p struct {
			Below       []string
			Lines       float64
			Button      float64
			ButtonWhole bool
			Whole       string
		}
		b.eval(`(() => {
			const text = document.getElementById('agent-prompt'), button = document.querySelector('.landing-prompt .copy-button')
			const below = []
			for (const [what, e] of [['the headline', document.querySelector('.landing-hero h1')], ['the prompt', document.querySelector('.landing-prompt')], ['its button', button]]) {
				const r = e.getBoundingClientRect()
				if (r.top < 0 || r.bottom > innerHeight || r.width === 0) below.push(what + ' (' + Math.round(r.top) + '–' + Math.round(r.bottom) + ')')
			}
			const style = getComputedStyle(text)
			return {
				below, whole: text.textContent,
				lines: (text.clientHeight - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom)) / parseFloat(style.lineHeight),
				button: button.getBoundingClientRect().width, buttonWhole: button.scrollWidth <= button.clientWidth,
			}
		})()`, &p)
		if len(p.Below) > 0 {
			t.Errorf("%s: not in the first screen: %s", b.doing, strings.Join(p.Below, ", "))
		}
		if p.Lines > 2.2 {
			t.Errorf("%s: the prompt's box is %.1f lines tall; it shows at most two", b.doing, p.Lines)
		}
		if p.Whole != agentPrompt(t) {
			t.Errorf("%s: the prompt's text is %q, not the whole prompt", b.doing, p.Whole)
		}
		if p.Button < 60 || !p.ButtonWhole {
			t.Errorf("%s: the prompt's button is %.0f px wide, its label whole %v", b.doing, p.Button, p.ButtonWhole)
		}
	}
}

// In the dark scheme a modal's backdrop darkens what is under it (it is
// mixed from black, not the light text colour), and a component page's
// stage canvas, following the page's scheme, is the site's raised surface
// rather than the neutral theme's near-black.
func TestDarkSurfaces(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.doing, b.deadline = "a sheet's backdrop in dark", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": "dark"}}})
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	b.open(srv.URL + "/components/dialog")
	b.until(`document.querySelector('.gallery-toolbar[data-ready]')`)
	var canvas, page string
	b.eval(`getComputedStyle(document.querySelector('.gallery-canvas')).backgroundColor`, &canvas)
	b.eval(`getComputedStyle(document.querySelector('.docs-page')).backgroundColor`, &page)
	if canvas != page {
		t.Errorf("in dark the stage canvas is %s, not the page's raised surface %s", canvas, page)
	}
	b.eval(`(document.querySelector('.gallery-canvas button').click(), true)`, new(bool))
	b.until(`document.querySelector('#weft-overlays .weft-overlay-backdrop')`)
	var bright float64
	b.eval(`(() => {
		const c = getComputedStyle(document.querySelector('#weft-overlays .weft-overlay-backdrop')).backgroundColor
		const n = c.match(/[\d.]+/g).map(Number)
		// color(srgb r g b / a) in 0..1, or rgba(r, g, b, a) in 0..255.
		return c.startsWith('color(') ? Math.max(n[0], n[1], n[2]) : Math.max(n[0], n[1], n[2]) / 255
	})()`, &bright)
	if bright > 0.1 {
		t.Errorf("in dark the modal backdrop is a light colour (brightest channel %.2f); it is mixed from black, so it darkens", bright)
	}
}

// The weave lies in the hero's ground, behind the copy only where its mask
// has faded it out: no text of the copy (the headline, the lead, the install
// command and the note under it) lies on the part of the picture that shows,
// from a phone to a wide screen. It is on the screen, not moved off it; it
// loads at the width the CSS gives each layout; at 1440 × 900 at least a
// third of the hero's end half in the first screen is weave that shows (its
// mask at half or more) and that no frame covers; and on a phone (844 px
// tall) it is fully drawn within the first screen. Side by side (1280 px and
// up) the header's links lie clear of it too; and with forced colours it is
// not drawn.
func TestLandingWeaveLeavesTheCopyClear(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.call(b.session, "Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-reduced-motion", "value": "reduce"}}})
	for _, width := range []int{320, 390, 768, 1024, 1280, 1440, 2560} {
		at := fmt.Sprintf("the weave at %d px", width)
		height := 900
		if width < 700 {
			height = 844
		}
		b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 1, "mobile": width < 700})
		b.open(srv.URL + "/")
		b.doing, b.deadline = at, time.Now().Add(pageTimeout)
		b.eval(`document.fonts.ready.then(() => true)`, new(bool))
		var got struct {
			Image          string
			OnWeave        []string
			Mask           string
			OnScreen, Room float64
			Drawn          float64
			OnNav          []string
		}
		b.eval(`(() => {
			const ground = document.querySelector('.landing-hero__ground'), art = getComputedStyle(ground, '::after'), g = ground.getBoundingClientRect()
			const w = parseFloat(art.width), h = parseFloat(art.height)
			const left = g.right - parseFloat(art.right) - w, top = g.bottom - parseFloat(art.bottom) - h
			// Each gradient of the mask: where it starts to show, where it is half drawn, where it is whole.
			let x = [left, left, left], y = [top, top, top]
			for (const m of art.maskImage.matchAll(/linear-gradient\((90deg, )?([^()]*(?:\([^()]*\)[^()]*)*)\)/g)) {
				const stops = [...m[2].matchAll(/(?:^|[\s,])([\d.]+)(%|px)/g)].map((s) => parseFloat(s[1]) / (s[2] == '%' ? 100 : m[1] ? w : h))
				if (m[2].startsWith('rgba(0, 0, 0, 0),')) stops.unshift(0)
				const [a, z] = [stops[0], stops.at(-1)], half = stops.length > 2 ? stops[1] : (a + z) / 2
				if (m[1]) x = [a, half, z].map((p) => left + p * w); else y = [a, half, z].map((p) => top + p * h)
			}
			const onWeave = []
			const walker = document.createTreeWalker(document.querySelector('.landing-hero__copy'), NodeFilter.SHOW_TEXT)
			for (let n; (n = walker.nextNode()); ) {
				if (!n.textContent.trim()) continue
				const r = document.createRange()
				r.selectNodeContents(n)
				for (const rect of r.getClientRects()) if (rect.right > x[0] + 1 && rect.bottom > y[0] + 1) onWeave.push(n.textContent.trim().slice(0, 30))
			}
			// The share of the end half of the hero, in the first screen, where the weave shows and no frame lies.
			const frames = [...document.querySelectorAll('.hero-frame')].map((f) => f.getBoundingClientRect()).filter((r) => r.width > 0)
			const bottom = Math.min(innerHeight, g.bottom)
			let shown = 0, all = 0
			for (let px = innerWidth / 2; px < innerWidth; px += 8) for (let py = 0; py < bottom; py += 8) {
				all++
				if (px >= x[1] && py >= y[1] && px < left + w && !frames.some((r) => px >= r.left && px < r.right && py >= r.top && py < r.bottom)) shown++
			}
			const onNav = [...document.querySelectorAll('.landing-top a:not(.site-skip)')].map((a) => [a, a.getBoundingClientRect()])
				.filter(([, r]) => r.width > 0 && r.right > x[0] + 1 && r.bottom > y[0] + 1).map(([a]) => (a.textContent.trim() || a.getAttribute('aria-label') || a.href).slice(0, 30))
			const image = performance.getEntriesByType('resource').map((e) => e.name).find((n) => /weave-\d+-/.test(n)) || ''
			return {
				onNav,
				image: image.replace(/^.*\/(weave-\d+)-.*$/, '$1'), onWeave,
				mask: art.maskImage + ' ' + [x[0], y[0], left, top, w, h].map(Math.round),
				onScreen: (Math.min(innerWidth, left + w) - Math.max(0, left)) / Math.min(w, innerWidth),
				room: shown / all, drawn: y[2],
			}
		})()`, &got)
		if len(got.OnWeave) > 0 {
			t.Errorf("%s: the copy lies on the weave (%s): %s", at, got.Mask, strings.Join(got.OnWeave, " | "))
		}
		if width >= 1280 && len(got.OnNav) > 0 {
			t.Errorf("%s: the header's links lie on the weave (%s): %s", at, got.Mask, strings.Join(got.OnNav, " | "))
		}
		if got.OnScreen < 0.8 {
			t.Errorf("%s: only %.0f %% of the weave's width is on the screen (%s)", at, got.OnScreen*100, got.Mask)
		}
		if width == 1440 && got.Room < 1.0/3 {
			t.Errorf("%s: the weave shows, uncovered, in %.0f %% of the hero's end half in the first screen, want a third or more", at, got.Room*100)
		}
		if width < 700 && got.Drawn > float64(height)-100 {
			t.Errorf("%s: the weave is whole only from y = %.0f, not within the first %d px", at, got.Drawn, height)
		}
		want := "weave-1024"
		if width <= 900 {
			want = "weave-640"
		}
		if got.Image != want {
			t.Errorf("%s: the page loaded %q, want %s at 1×", at, got.Image, want)
		}
	}
}

// With forced colours the band is plain: system colours over a busy picture read badly.
func TestLandingWeaveNotInForcedColours(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.call(b.session, "Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "forced-colors", "value": "active"}}})
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1440, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	b.open(srv.URL + "/")
	b.doing, b.deadline = "the weave in forced colours", time.Now().Add(pageTimeout)
	var content string
	b.eval(`getComputedStyle(document.querySelector('.landing-hero__ground'), '::after').content`, &content)
	if content != "none" {
		t.Errorf("with forced colours the weave is drawn (content %s); the band stays plain", content)
	}
}

// The loom waits below the fold: at 1440 × 900 it is not fetched on load,
// and when it is scrolled into view the hero, with its weave, is out of it,
// so the two pictures are never on screen together.
func TestLandingLoomBelowTheFold(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1440, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	b.doing, b.deadline = "the loom at 1440 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Page.navigate", map[string]any{"url": srv.URL + "/"})
	b.wait("Page.loadEventFired")
	var early bool
	b.eval(`performance.getEntriesByType('resource').some((e) => e.name.includes('/loom-'))`, &early)
	if early {
		t.Error("at 1440 × 900 the loom is fetched on load; it is below the fold, loaded lazily")
	}
	b.eval(`(document.querySelector('#start img').scrollIntoView({ block: 'end' }), true)`, new(bool))
	b.until(`document.querySelector('#start img').complete && document.querySelector('#start img').naturalWidth > 0`)
	var heroBottom float64
	b.eval(`document.querySelector('.landing-hero').getBoundingClientRect().bottom`, &heroBottom)
	if heroBottom > 0 {
		t.Errorf("with the loom in view the hero is still %.0f px on screen", heroBottom)
	}
}
