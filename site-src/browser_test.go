package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests that need a browser: headless Chrome, driven over the DevTools
// protocol on a pipe (--remote-debugging-pipe), so the standard library is
// enough. Chrome is $CHROME_PATH, or the first found where it is usually
// installed; without one these tests are skipped and say so, except in CI
// ($CI set), where they fail: a browser test that never runs proves nothing.

// chromePath is the Chrome to run, or "".
func chromePath() string {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		return p
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// browser is one headless Chrome with one page, which a test drives with call.
type browser struct {
	t       *testing.T
	in      io.WriteCloser
	msgs    chan cdpMessage
	failed  chan error
	cmd     *exec.Cmd
	next    int
	session string
	events  []cdpMessage
	// What is being waited for, and until when: a page that hangs fails with its path.
	doing    string
	deadline time.Time
}

// pageTimeout is how long one page may take, from navigating to its last answer.
const pageTimeout = 20 * time.Second

type cdpMessage struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func startBrowser(t *testing.T) *browser {
	t.Helper()
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome found, and CI is set: install Chrome or set CHROME_PATH, so the browser tests run")
		}
		t.Skip("no Chrome found; set CHROME_PATH to run the browser tests")
	}
	toChrome, in, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	out, fromChrome, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--headless=new", "--remote-debugging-pipe", "--no-first-run", "--no-default-browser-check",
		"--hide-scrollbars", "--disable-gpu", "--disable-extensions", "--user-data-dir=" + t.TempDir()}
	if os.Geteuid() == 0 {
		args = append(args, "--no-sandbox")
	}
	cmd := exec.Command(path, args...)
	// Chrome reads the protocol from fd 3 and writes it to fd 4.
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting %s: %v", path, err)
	}
	toChrome.Close()
	fromChrome.Close()
	b := &browser{t: t, in: in, msgs: make(chan cdpMessage), failed: make(chan error, 1), cmd: cmd,
		doing: "starting Chrome", deadline: time.Now().Add(pageTimeout)}
	// Chrome's messages, each ending with a NUL, read where a wait can give up on them.
	go func() {
		r := bufio.NewReader(out)
		for {
			line, err := r.ReadBytes(0)
			if err != nil {
				b.failed <- err
				return
			}
			var m cdpMessage
			if err := json.Unmarshal(line[:len(line)-1], &m); err != nil {
				b.failed <- fmt.Errorf("Chrome sent %q: %v", line, err)
				return
			}
			b.msgs <- m
		}
	}()
	t.Cleanup(func() {
		in.Close()
		done := make(chan struct{})
		go func() { cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			<-done
		}
		out.Close()
	})
	var target struct {
		TargetID string `json:"targetId"`
	}
	b.decode(b.call("", "Target.createTarget", map[string]any{"url": "about:blank"}), &target)
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	b.decode(b.call("", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}), &attached)
	b.session = attached.SessionID
	b.call(b.session, "Page.enable", nil)
	return b
}

func (b *browser) decode(raw json.RawMessage, v any) {
	b.t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		b.t.Fatalf("Chrome sent %s: %v", raw, err)
	}
}

// read is the next message from Chrome, or a failure naming what was being
// done once the deadline passes.
func (b *browser) read() cdpMessage {
	b.t.Helper()
	timer := time.NewTimer(time.Until(b.deadline))
	defer timer.Stop()
	select {
	case m := <-b.msgs:
		return m
	case err := <-b.failed:
		b.t.Fatalf("%s: reading from Chrome: %v", b.doing, err)
	case <-timer.C:
		b.t.Fatalf("%s: no answer from Chrome within %v", b.doing, pageTimeout)
	}
	return cdpMessage{}
}

// call sends one command and waits for its result, keeping the events that come first.
func (b *browser) call(session, method string, params any) json.RawMessage {
	b.t.Helper()
	b.next++
	msg := map[string]any{"id": b.next, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	data, _ := json.Marshal(msg)
	if _, err := b.in.Write(append(data, 0)); err != nil {
		b.t.Fatalf("writing to Chrome: %v", err)
	}
	for {
		m := b.read()
		if m.ID == b.next {
			if m.Error != nil {
				b.t.Fatalf("%s: %s", method, m.Error.Message)
			}
			return m.Result
		}
		if m.Method != "" {
			b.events = append(b.events, m)
		}
	}
}

// wait returns once the page has sent the event.
func (b *browser) wait(method string) {
	b.t.Helper()
	for {
		for i, e := range b.events {
			if e.Method == method && e.SessionID == b.session {
				b.events = append(b.events[:i:i], b.events[i+1:]...)
				return
			}
		}
		if m := b.read(); m.Method != "" {
			b.events = append(b.events, m)
		}
	}
}

// eval runs the expression in the page, awaiting a promise, and decodes its value into v.
func (b *browser) eval(expr string, v any) {
	b.t.Helper()
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	b.decode(b.call(b.session, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true}), &r)
	if r.ExceptionDetails != nil {
		b.t.Fatalf("%s threw: %s", expr, r.ExceptionDetails.Text)
	}
	b.decode(r.Result.Value, v)
}

// open loads the page at url and waits for it, and for its islands to settle.
func (b *browser) open(url string) {
	b.t.Helper()
	b.doing, b.deadline = "loading "+url, time.Now().Add(pageTimeout)
	b.events = nil
	b.call(b.session, "Page.navigate", map[string]any{"url": url})
	b.wait("Page.loadEventFired")
	// Bring the visible islands in, and let the idle ones start.
	var ok bool
	b.eval(`new Promise((done) => { scrollTo(0, document.body.scrollHeight); requestAnimationFrame(() => requestIdleCallback(() => { scrollTo(0, 0); setTimeout(() => done(true), 50) }, { timeout: 500 })) })`, &ok)
}

// allPages is every page the build wrote, by path, the not-found page among them.
func allPages(t *testing.T, d fs.FS) []string {
	t.Helper()
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	var paths []string
	for _, p := range r.Pages {
		paths = append(paths, p.Path)
	}
	return append(paths, "/no-such-page")
}

// No page scrolls sideways on a phone or a tablet: at 320, 390 and 768 px
// wide, the document is no wider than the viewport, and nothing on the root
// hides what would overflow (overflow-x: clip or hidden on html or body).
func TestNoPageScrollsSideways(t *testing.T) {
	d := dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	pages := allPages(t, d)
	for _, width := range []int{320, 390, 768} {
		b.doing, b.deadline = fmt.Sprintf("emulating %d px", width), time.Now().Add(pageTimeout)
		b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": 800, "deviceScaleFactor": 1, "mobile": width < 700})
		for _, path := range pages {
			b.open(srv.URL + path)
			var got struct {
				Scroll, Client int
				Root, Body     string
				Wide           []string
			}
			b.eval(`(() => {
				const root = document.documentElement, w = root.clientWidth
				const name = (e) => e.tagName.toLowerCase() + (typeof e.className == 'string' && e.className.trim() ? '.' + e.className.trim().split(/\s+/).join('.') : '')
				const contained = (e) => { for (let a = e.parentElement; a && a !== document.body; a = a.parentElement) if (getComputedStyle(a).overflowX !== 'visible') return true; return false }
				// The outermost elements that stick out past the viewport and are in no scroll container, each with its parent.
				const wide = [...document.body.querySelectorAll('*')].filter((e) => {
					const r = e.getBoundingClientRect()
					return r.width > 0 && r.right > w + 1 && e.parentElement.getBoundingClientRect().right <= w + 1 && !contained(e)
				}).slice(0, 3).map((e) => name(e.parentElement) + ' > ' + name(e) + ' (' + Math.round(e.getBoundingClientRect().right) + ' px)')
				return { scroll: root.scrollWidth, client: w, root: getComputedStyle(root).overflowX, body: getComputedStyle(document.body).overflowX, wide }
			})()`, &got)
			if got.Scroll > got.Client {
				t.Errorf("%d px, %s: the page is %d px wide (first too wide: %s)", width, path, got.Scroll, strings.Join(got.Wide, ", "))
			}
			if got.Root != "visible" || got.Body != "visible" {
				t.Errorf("%d px, %s: overflow-x is %s on html and %s on body; let nothing overflow rather than hide it", width, path, got.Root, got.Body)
			}
		}
	}
}

// The site's own CSS clips nothing on the root: overflow is prevented where it would start.
func TestSiteCSSDoesNotClipTheRoot(t *testing.T) {
	root := regexp.MustCompile(`(?m)^\s*(html|body|:root)(\s*,\s*(html|body|:root))*\s*\{[^}]*overflow(-x)?\s*:\s*(clip|hidden)`)
	entries, err := os.ReadDir("styles")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile("styles/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if root.Match(b) {
			t.Errorf("styles/%s clips overflow on the root, which hides it rather than prevents it", e.Name())
		}
	}
}

// The gallery's own headings (a group's on the index, a section's on a
// component's page) have a saffron rule above them; a heading inside an
// example (a thumbnail, the stage, a preview) keeps its component's look,
// with no rule of the gallery's.
func TestGalleryExampleHeadingsKeepTheirOwnLook(t *testing.T) {
	d := dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.doing, b.deadline = "emulating 1280 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	examples := 0
	for _, path := range allPages(t, d) {
		if path != "/components" && !strings.HasPrefix(path, "/components/") {
			continue
		}
		b.open(srv.URL + path)
		var got struct {
			Own, Examples int
			Ruled          []string
		}
		b.eval(`(() => {
			const ruled = (h) => getComputedStyle(h).borderTopStyle !== 'none' && parseFloat(getComputedStyle(h).borderTopWidth) > 0
			const inExample = (h) => !!h.closest('.gallery-thumb, .gallery-canvas, .gallery-preview')
			const all = [...document.querySelectorAll('main h2, main h3')]
			const own = all.filter((h) => !inExample(h) && h.tagName === 'H2')
			const examples = all.filter(inExample)
			return {
				own: own.filter(ruled).length - own.length,
				examples: examples.length,
				ruled: examples.filter(ruled).slice(0, 3).map((h) => h.tagName.toLowerCase() + ' "' + h.textContent.trim().slice(0, 30) + '"'),
			}
		})()`, &got)
		examples += got.Examples
		if got.Own != 0 {
			t.Errorf("%s: %d of the page's own h2 have no rule above them", path, -got.Own)
		}
		if len(got.Ruled) > 0 {
			t.Errorf("%s: headings inside examples take the gallery's rule: %s", path, strings.Join(got.Ruled, ", "))
		}
	}
	if examples == 0 {
		t.Fatal("no example on a gallery page has a heading")
	}
}

// hover moves the mouse onto the middle of the first element matching
// selector, which must be on screen.
func (b *browser) hover(selector string) {
	b.t.Helper()
	var at struct{ X, Y float64 }
	b.eval(`(() => {
		const r = document.querySelector(`+jsString(selector)+`).getBoundingClientRect()
		return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
	})()`, &at)
	b.call(b.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": at.X, "y": at.Y})
}

// until waits, in the page, for the expression to be true, failing after a few seconds.
func (b *browser) until(expr string) {
	b.t.Helper()
	var ok bool
	b.eval(`new Promise((done) => {
		const end = Date.now() + 5000
		const check = () => (`+expr+`) ? done(true) : Date.now() > end ? done(false) : setTimeout(check, 20)
		check()
	})`, &ok)
	if !ok {
		b.t.Fatalf("%s: never became true: %s", b.doing, expr)
	}
}

func jsString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

// A link in an overlay opened from a gallery example is no more prefetched
// than one in the example itself: the Sheet teleports to the overlay root
// under <body>, outside the stage's [data-weft-no-prefetch], so its layer
// must carry the mark. A link of the site's own, hovered first, is
// prefetched, so the browser is shown to prefetch at all.
func TestOverlayLinksFromAnExampleAreNotPrefetched(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	var mu sync.Mutex
	var prefetched []string
	site := siteHandler(t, stagedReleases(t))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Purpose") != "" {
			mu.Lock()
			prefetched = append(prefetched, r.URL.Path)
			mu.Unlock()
		}
		site.ServeHTTP(w, r)
	}))
	defer srv.Close()
	got := func(path string) bool {
		mu.Lock()
		defer mu.Unlock()
		return slices.Contains(prefetched, path)
	}
	// Wait long enough for a moderate prefetch (200 ms of hover) to be sent, and answered.
	settle := func() {
		var ok bool
		b.eval(`new Promise((done) => setTimeout(() => done(true), 1000))`, &ok)
	}

	b.doing, b.deadline = "emulating 1280 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	b.open(srv.URL + "/components/sheet")
	b.doing, b.deadline = "hovering on /components/sheet", time.Now().Add(pageTimeout)
	b.until(`document.querySelector('.gallery-toolbar[data-ready]')`)

	b.hover(`.site-nav a[href="/skills"]`)
	settle()
	if !got("/skills") {
		t.Fatalf("hovering the header's /skills link prefetched nothing (prefetched: %v): the browser does not prefetch, so this test proves nothing", prefetched)
	}

	b.eval(`(document.querySelector('.gallery-canvas button[aria-label="Open menu"]').click(), true)`, new(bool))
	b.until(`document.querySelector('#weft-overlays a[href="/settings"]')`)
	b.until(`document.querySelector('#weft-overlays .weft-sheet').getAnimations().length === 0`)
	b.hover(`#weft-overlays a[href="/settings"]`)
	settle()
	if got("/settings") {
		t.Errorf("hovering the Settings link in the stage's Sheet prefetched /settings, a page of the example's app that is not here")
	}
}
