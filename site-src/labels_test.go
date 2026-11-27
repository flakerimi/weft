package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Every <label> on every built page labels a form control, and every form
// field has an id or a name (and autocomplete when its id or name is an
// autocomplete token), as Chrome checks in its DevTools Issues ("Incorrect use
// of <label for=FORM_ELEMENT>", "A <label> isn't associated with a form
// field", "A form field element should have an id or name attribute", "An
// element doesn't have an autocomplete attribute"): a <label>'s `for` names
// the id of a labelable element on the same page (button, input but a hidden
// one, meter, output, progress, select, textarea), or it has no `for` and
// holds one. A FormField around a control that is no labelable element (a
// radio group's fieldset, a slider's thumb, a listbox) names it by id, with no
// <label>.

var (
	tagPattern    = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9-]*)([^>]*)>`)
	attrPattern   = regexp.MustCompile(`([^\s=/]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+)))?`)
	rawText       = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)\s*>`)
	labelableTags = map[string]bool{"button": true, "input": true, "meter": true, "output": true, "progress": true, "select": true, "textarea": true}
)

type startTag struct {
	name  string
	attrs map[string]string
	has   map[string]bool
}

func (s startTag) labelable() bool {
	return labelableTags[s.name] && !(s.name == "input" && strings.EqualFold(s.attrs["type"], "hidden"))
}

func (s startTag) String() string {
	if id := s.attrs["id"]; id != "" {
		return fmt.Sprintf("<%s id=%q>", s.name, id)
	}
	return "<" + s.name + ">"
}

func parseAttrs(raw string) (map[string]string, map[string]bool) {
	attrs, has := map[string]string{}, map[string]bool{}
	for _, m := range attrPattern.FindAllStringSubmatch(raw, -1) {
		name := strings.ToLower(m[1])
		has[name] = true
		attrs[name] = html.UnescapeString(m[2] + m[3] + m[4])
	}
	return attrs, has
}

// labelProblems lists each <label> in the page that labels no form control, in a sentence.
func labelProblems(page string) []string {
	page = rawText.ReplaceAllString(page, "")
	byID := map[string]startTag{}
	type label struct {
		tag    startTag
		nested []startTag
	}
	var labels []label
	var open *label
	for _, m := range tagPattern.FindAllStringSubmatch(page, -1) {
		closing, name := m[1] == "/", strings.ToLower(m[2])
		if closing {
			if name == "label" && open != nil {
				labels = append(labels, *open)
				open = nil
			}
			continue
		}
		attrs, has := parseAttrs(m[3])
		tag := startTag{name, attrs, has}
		if id, ok := attrs["id"]; ok && has["id"] {
			if _, seen := byID[id]; !seen {
				byID[id] = tag
			}
		}
		if open != nil {
			open.nested = append(open.nested, tag)
		}
		if name == "label" {
			open = &label{tag: tag}
		}
	}
	var problems []string
	for _, l := range labels {
		if l.tag.has["for"] {
			target, ok := byID[l.tag.attrs["for"]]
			switch {
			case !ok:
				problems = append(problems, fmt.Sprintf("<label for=%q> names no element on the page", l.tag.attrs["for"]))
			case !target.labelable():
				problems = append(problems, fmt.Sprintf("<label for=%q> names %s, which is not labelable", l.tag.attrs["for"], target))
			}
			continue
		}
		holds := false
		for _, n := range l.nested {
			holds = holds || n.labelable()
		}
		if !holds {
			what := l.tag.String()
			if id := l.tag.attrs["id"]; id != "" {
				what = fmt.Sprintf("<label id=%q>", id)
			}
			problems = append(problems, what+" has no for and holds no form control")
		}
	}
	return problems
}

func TestLabelProblems(t *testing.T) {
	for _, c := range []struct {
		page string
		want int
	}{
		{`<label for="a">A</label><input id="a">`, 0},
		{`<label for="a">A</label><textarea id="a"></textarea>`, 0},
		{`<label for="a">A</label><button id="a"></button>`, 0},
		{`<label><input type="checkbox"> A</label>`, 0},
		{`<span id="l">Plan</span><fieldset aria-labelledby="l"></fieldset>`, 0},
		{`<label for="a">A</label><fieldset id="a"></fieldset>`, 1},
		{`<label for="a">A</label><span id="a" role="slider"></span>`, 1},
		{`<label for="a">A</label><div id="a" role="listbox"></div>`, 1},
		{`<label for="a">A</label><input id="a" type="hidden">`, 1},
		{`<label for="a">A</label>`, 1},
		{`<label id="l">Plan</label><fieldset aria-labelledby="l"></fieldset>`, 1},
		{`<script>const s = '<label for="x">'</script><label for="a">A</label><input id="a">`, 0},
	} {
		if got := labelProblems(c.page); len(got) != c.want {
			t.Errorf("%s: %d problems, want %d: %v", c.page, len(got), c.want, got)
		}
	}
}

// No built page has a <label> that labels no form control.
func TestEveryLabelLabelsAFormControl(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	files := []string{}
	for _, p := range r.Pages {
		files = append(files, p.HTML)
	}
	if r.NotFound != nil {
		files = append(files, *r.NotFound)
	}
	labels := 0
	for _, file := range files {
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		labels += strings.Count(string(b), "<label")
		for _, p := range labelProblems(string(b)) {
			t.Errorf("%s: %s", file, p)
		}
	}
	if labels == 0 {
		t.Fatal("no built page has a <label>; the check would prove nothing")
	}
}

// formFieldProblems lists each <input> (not hidden), <select> or <textarea> in
// the page that has neither id nor name, in a sentence.
func formFieldProblems(page string) []string {
	page = rawText.ReplaceAllString(page, "")
	var problems []string
	for _, m := range tagPattern.FindAllStringSubmatch(page, -1) {
		closing, name := m[1] == "/", strings.ToLower(m[2])
		if closing || (name != "input" && name != "select" && name != "textarea") {
			continue
		}
		attrs, _ := parseAttrs(m[3])
		if name == "input" && strings.EqualFold(attrs["type"], "hidden") {
			continue
		}
		if strings.TrimSpace(attrs["id"]) == "" && strings.TrimSpace(attrs["name"]) == "" {
			what := "<" + name
			for _, k := range []string{"class", "type", "role", "aria-label", "placeholder"} {
				if v := attrs[k]; v != "" {
					what += fmt.Sprintf(" %s=%q", k, v)
				}
			}
			problems = append(problems, what+"> has neither id nor name")
		}
	}
	return problems
}

func TestFormFieldProblems(t *testing.T) {
	for _, c := range []struct {
		page string
		want int
	}{
		{`<input id="a"><input name="b"><textarea id="c"></textarea><select name="d"></select><input type="hidden">`, 0},
		{`<input type="text">`, 1},
		{`<input id="" name=" ">`, 1},
		{`<textarea></textarea>`, 1},
		{`<select></select>`, 1},
		{`<script>const s = '<input type="text">'</script><input id="a">`, 0},
	} {
		if got := formFieldProblems(c.page); len(got) != c.want {
			t.Errorf("%s: %d problems, want %d: %v", c.page, len(got), c.want, got)
		}
	}
}

// No built page has an <input>, <select> or <textarea> with neither id nor name.
func TestEveryFormFieldHasIdOrName(t *testing.T) {
	d := dist(t)
	var r builtRoutes
	readJSON(t, d, "routes.json", &r)
	files := []string{}
	for _, p := range r.Pages {
		files = append(files, p.HTML)
	}
	if r.NotFound != nil {
		files = append(files, *r.NotFound)
	}
	inputs := 0
	for _, file := range files {
		b, err := fs.ReadFile(d, file)
		if err != nil {
			t.Fatal(err)
		}
		inputs += strings.Count(string(b), "<input")
		for _, p := range formFieldProblems(string(b)) {
			t.Errorf("%s: %s", file, p)
		}
	}
	if inputs == 0 {
		t.Fatal("no built page has an <input>; the check would prove nothing")
	}
}

// In Chrome, the pages with FormFields over every kind of control raise no
// DevTools issue, each field's control is named by its label's
// text in the accessibility tree, whether through a <label for> or
// aria-labelledby, and a press on a <label> focuses its input.
func TestFieldLabelsInChrome(t *testing.T) {
	dist(t)
	b := startBrowser(t)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	b.doing, b.deadline = "emulating 1280 px", time.Now().Add(pageTimeout)
	b.call(b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	// Chrome checks a page's form fields and lazy images once it has loaded,
	// and reports what it finds while Audits is on.
	b.call(b.session, "Audits.enable", nil)
	b.call(b.session, "DOM.enable", nil)
	b.call(b.session, "Accessibility.enable", nil)
	// First check the pages without scrolling to the bottom, and then with
	// b.open (which scrolls to wake visible islands and lets idle ones start).
	for _, path := range []string{"/", "/components", "/components/command-palette", "/components/date-range-picker",
		"/components/input", "/components/pin-input", "/docs"} {
		b.doing, b.deadline = "loading "+srv.URL+path, time.Now().Add(pageTimeout)
		b.events = nil
		b.call(b.session, "Page.navigate", map[string]any{"url": srv.URL + path})
		b.wait("Page.loadEventFired")
		b.eval(`new Promise((done) => requestAnimationFrame(() => requestIdleCallback(() => setTimeout(() => done(true), 200), { timeout: 500 })))`, new(bool))
		for _, e := range b.events {
			if e.Method == "Audits.issueAdded" {
				t.Errorf("%s (plain load): Chrome reports issue %s", path, string(e.Params))
			}
		}
		b.open(srv.URL + path)
		for _, e := range b.events {
			if e.Method == "Audits.issueAdded" {
				t.Errorf("%s: Chrome reports issue %s", path, string(e.Params))
			}
		}
	}
	var checked []string
	for _, path := range []string{"/", "/components", "/components/form-field", "/components/input", "/components/radio-group",
		"/components/checkbox-group", "/components/slider", "/components/date-range-picker", "/components/input-time",
		"/components/listbox", "/components/toggle-group"} {
		b.open(srv.URL + path)
		for _, e := range b.events {
			if e.Method == "Audits.issueAdded" {
				t.Errorf("%s: Chrome reports issue %s", path, string(e.Params))
			}
		}
		// Each field label's controls, marked for the accessibility tree below;
		// a picture of a page (the landing's screens, a gallery thumbnail) is
		// inert, out of that tree.
		var named []struct {
			Text, Tag, Label string
		}
		b.eval(`(() => {
			const out = []
			for (const label of document.querySelectorAll('.weft-form-field__label')) {
				if (label.closest('[inert]')) continue
				const text = label.textContent.replace(/\*\s*$/, '').trim()
				const controls = label.tagName === 'LABEL'
					? [label.control].filter(Boolean)
					: [...document.querySelectorAll('[aria-labelledby]')].filter((e) => e.getAttribute('aria-labelledby').split(' ').includes(label.id))
				if (controls.length === 0) out.push({ text, tag: '', label: label.outerHTML.slice(0, 120) })
				for (const c of controls) {
					c.setAttribute('data-ax-check', String(out.length))
					out.push({ text, tag: c.tagName.toLowerCase(), label: label.tagName.toLowerCase() })
				}
			}
			return out
		})()`, &named)
		var doc struct {
			Root struct {
				NodeID int `json:"nodeId"`
			} `json:"root"`
		}
		b.decode(b.call(b.session, "DOM.getDocument", map[string]any{"depth": 0}), &doc)
		for i, n := range named {
			if n.Tag == "" {
				t.Errorf("%s: %s names no control", path, n.Label)
				continue
			}
			checked = append(checked, fmt.Sprintf("%s %q <%s> by <%s>", path, n.Text, n.Tag, n.Label))
			var found struct {
				NodeIDs []int `json:"nodeIds"`
			}
			b.decode(b.call(b.session, "DOM.querySelectorAll", map[string]any{"nodeId": doc.Root.NodeID, "selector": fmt.Sprintf(`[data-ax-check="%d"]`, i)}), &found)
			if len(found.NodeIDs) == 0 {
				continue // named by two labels; the later one checks it
			}
			var ax struct {
				Nodes []struct {
					Name *struct {
						Value json.RawMessage `json:"value"`
					} `json:"name"`
				} `json:"nodes"`
			}
			b.decode(b.call(b.session, "Accessibility.getPartialAXTree", map[string]any{"nodeId": found.NodeIDs[0], "fetchRelatives": false}), &ax)
			name := ""
			if len(ax.Nodes) > 0 && ax.Nodes[0].Name != nil {
				json.Unmarshal(ax.Nodes[0].Name.Value, &name)
			}
			if !strings.Contains(name, n.Text) {
				t.Errorf("%s: the <%s> under the field label %q (a <%s>) is named %q", path, n.Tag, n.Text, n.Label, name)
			}
		}
	}
	if len(checked) == 0 {
		t.Fatal("no page had a field label naming a control; the check would prove nothing")
	}
	t.Logf("named by their field's label in the accessibility tree:\n%s", strings.Join(checked, "\n"))

	// A press on a field's <label> focuses its input.
	b.open(srv.URL + "/components/input")
	var marked bool
	b.eval(`(() => {
		const label = [...document.querySelectorAll('label.weft-form-field__label')].find((l) => l.control && l.control.tagName === 'INPUT' && l.getBoundingClientRect().width > 0)
		if (!label) return false
		label.setAttribute('data-press', '')
		return true
	})()`, &marked)
	if !marked {
		t.Fatal("/components/input: no field <label> for an input to press")
	}
	b.click(`[data-press]`)
	var focused bool
	b.eval(`document.activeElement === document.querySelector('[data-press]').control`, &focused)
	if !focused {
		t.Error("/components/input: a press on a field's <label> leaves its input unfocused")
	}
}
