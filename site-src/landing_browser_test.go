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
// one line, scrolling inside its box, its Copy button whole.
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
				// The open menu hangs from its trigger's cell, at its end edge.
				const panel = desk.querySelector('[role="menu"]'), cell = panel.closest('td')
				const p = box(panel), c = box(cell)
				if (Math.abs(p.top - (c.bottom + 6)) > 2 || Math.abs(p.right - c.right) > 2) menu.push('menu at ' + Math.round(p.left) + ',' + Math.round(p.top) + '; cell ends ' + Math.round(c.right) + ',' + Math.round(c.bottom))
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
