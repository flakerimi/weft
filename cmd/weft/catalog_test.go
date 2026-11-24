package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// The catalogue tests read the repository's own catalog/components.json, as
// the staged bundle carries it, so the printer is proven on every entry.

// repoCatalog is the repository's catalogue, found before any test changes
// folder.
var repoCatalog, _ = filepath.Abs("../../catalog/components.json")

// repoEntries is the repository's catalogue, each entry decoded generically.
func repoEntries(t *testing.T) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(repoCatalog)
	if err != nil {
		t.Fatal(err)
	}
	var c struct{ Components []map[string]any }
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Components) == 0 {
		t.Fatal("catalog/components.json has no components")
	}
	return c.Components
}

// rawEntry is one entry of the repository's catalogue as its JSON bytes.
func rawEntry(t *testing.T, name string) []byte {
	t.Helper()
	b, _ := os.ReadFile(repoCatalog)
	var c struct{ Components []json.RawMessage }
	json.Unmarshal(b, &c)
	for _, e := range c.Components {
		var n struct{ Name string }
		json.Unmarshal(e, &n)
		if n.Name == name {
			var compact bytes.Buffer
			json.Compact(&compact, e)
			return compact.Bytes()
		}
	}
	t.Fatalf("the catalogue has no %s", name)
	return nil
}

// outsideApp runs the test in a temporary folder that is in no app, with
// WEFT_BUNDLE set to the staged bundle; under -short, which stages none, to a
// bundle holding only the repository's catalogue.
func outsideApp(t *testing.T) {
	t.Helper()
	bundle := stagedBundle
	if bundle == "" {
		bundle = t.TempDir()
		b, err := os.ReadFile(repoCatalog)
		if err != nil {
			t.Fatal(err)
		}
		os.MkdirAll(filepath.Join(bundle, "weft", "catalog"), 0o755)
		if err := os.WriteFile(filepath.Join(bundle, "weft", "catalog", "components.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("WEFT_BUNDLE", bundle)
	t.Chdir(t.TempDir())
}

func catalog(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	return runCmd(append([]string{"catalog"}, args...)...)
}

func str(v any) string { s, _ := v.(string); return s }

func list(v any) []map[string]any {
	var out []map[string]any
	for _, x := range v.([]any) {
		out = append(out, x.(map[string]any))
	}
	return out
}

// The bundle's catalogue is the repository's, and every field of every entry
// is one the printer knows: a new field fails here until it is printed or
// named as left out.
func TestCatalogKnowsEveryField(t *testing.T) {
	b, err := os.ReadFile(repoCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if stagedBundle != "" {
		staged, err := os.ReadFile(filepath.Join(stagedBundle, "weft", "catalog", "components.json"))
		if err != nil || !bytes.Equal(staged, b) {
			t.Fatalf("the staged bundle's catalogue is not catalog/components.json (%v)", err)
		}
	}
	var c struct {
		Components []json.RawMessage
	}
	json.Unmarshal(b, &c)
	for _, raw := range c.Components {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var e catalogEntry
		if err := dec.Decode(&e); err != nil {
			t.Errorf("%.40s: %v", raw, err)
		}
	}
}

func TestCatalogList(t *testing.T) {
	outsideApp(t)
	entries := repoEntries(t)
	code, out, errs := catalog(t)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if want := fmt.Sprintf("%d components. Run weft catalog <Name> for one.", len(entries)); lines[0] != want {
		t.Errorf("first line %q, want %q", lines[0], want)
	}
	// One line per component: its name padded, then its purpose, under its
	// group, the groups in the catalogue's order.
	var groups []string
	for _, e := range entries {
		if g := str(e["group"]); !contains(groups, g) {
			groups = append(groups, g)
		}
	}
	var want []string
	for _, g := range groups {
		want = append(want, "", g)
		for _, e := range entries {
			if str(e["group"]) == g {
				want = append(want, fmt.Sprintf("  %-15s %s", str(e["name"]), str(e["purpose"])))
			}
		}
	}
	if got := lines[1:]; !reflect.DeepEqual(got, want) {
		t.Errorf("the list is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if n := len(out); n > 8*1024 {
		t.Errorf("the list is %d B, over 8 KB", n)
	}
	t.Logf("the list of %d components: %d B", len(entries), len(out))
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Every entry prints every section it has, and no section it has not, with
// every prop, event and slot by name and no Go zero value or formatting slip.
func TestCatalogPrintsEveryEntry(t *testing.T) {
	outsideApp(t)
	// In what the printer writes itself (the notes and examples are the
	// catalogue's text), an empty string shows as a trailing or a double space.
	slip := regexp.MustCompile(`<nil>|map\[|%!|(?m) $|  `)
	for _, e := range repoEntries(t) {
		name := str(e["name"])
		code, out, errs := catalog(t, name)
		if code != 0 || errs != "" {
			t.Errorf("%s: exit %d, stderr %q", name, code, errs)
			continue
		}
		own, _, _ := strings.Cut(out, "\nNotes:\n")
		own, _, _ = strings.Cut(own, "\nExample: ")
		if m := slip.FindString(own); m != "" || strings.Contains(out, "<nil>") || strings.Contains(out, "%!") {
			t.Errorf("%s: the output has %q:\n%s", name, m, out)
		}
		if !strings.HasPrefix(out, "# "+name+"\n\n"+str(e["purpose"])+"\n\nUse: "+str(e["use"])+"\nAvoid: "+str(e["avoid"])+"\n") {
			t.Errorf("%s: no heading, purpose, use and avoid:\n%.300s", name, out)
		}
		adapts := e["adapts"].(map[string]any)
		for _, axis := range []string{"input", "surface", "width"} {
			if !strings.Contains(out, "\n- "+axis+": "+str(adapts[axis])+"\n") {
				t.Errorf("%s: Adapts has no %s", name, axis)
			}
		}
		section := func(title string, has bool) {
			if strings.Contains(out, "\n"+title+":") != has {
				t.Errorf("%s: the %s section is there %v, want %v", name, title, !has, has)
			}
		}
		props, emits, slots := list(e["props"]), list(e["emits"]), list(e["slots"])
		section("Props", len(props) > 0)
		section("Emits", len(emits) > 0)
		section("Slots", len(slots) > 0)
		section("Notes", str(e["notes"]) != "")
		for _, p := range props {
			line := "\n- " + str(p["name"]) + ": " + str(p["type"])
			if d, ok := p["default"].(string); ok {
				line += " = " + d
			}
			if p["required"] == true {
				line += ", required"
			}
			line += " — " + str(p["description"]) + "\n"
			if !strings.Contains(out, line) {
				t.Errorf("%s: no line %q", name, line)
			}
		}
		for _, ev := range emits {
			payload := strings.TrimSuffix(strings.TrimPrefix(str(ev["payload"]), "["), "]")
			line := "\n- " + str(ev["name"]) + "(" + payload + ") — " + str(ev["description"]) + "\n"
			if !strings.Contains(out, line) {
				t.Errorf("%s: no line %q", name, line)
			}
		}
		for _, s := range slots {
			text := str(s["name"])
			if ps := s["props"].([]any); len(ps) > 0 {
				var names []string
				for _, p := range ps {
					names = append(names, p.(string))
				}
				text += " { " + strings.Join(names, ", ") + " }"
			}
			if !strings.Contains(out, text) {
				t.Errorf("%s: no slot %q", name, text)
			}
		}
		if notes := str(e["notes"]); notes != "" && !strings.Contains(out, "\nNotes:\n"+notes+"\n") {
			t.Errorf("%s: the notes are not as the catalogue has them", name)
		}
		examples := list(e["examples"])
		first := "\nExample: " + str(examples[0]["title"]) + "\n\n```vue\n" + str(examples[0]["source"]) + "```\n"
		if !strings.HasSuffix(out, first) {
			t.Errorf("%s: does not end with its first example", name)
		}
		if len(examples) > 1 && strings.Contains(out, str(examples[1]["source"])) {
			t.Errorf("%s: prints more than the first example without --examples", name)
		}
	}
}

func TestCatalogButton(t *testing.T) {
	outsideApp(t)
	_, out, _ := catalog(t, "Button")
	for _, want := range []string{
		"# Button\n\nA button that runs an action, or a link drawn as a button when it has `to`.\n\nUse: ",
		"\n\nAdapts:\n- input: On touch an md button",
		"\n\nProps:\n- label: string — The button's text; the default slot replaces it.\n- variant: 'solid' | 'soft' | 'outline' | 'ghost' = 'solid' — ",
		"\n\nEmits:\n- click(event: MouseEvent) — Pressed by mouse, touch or keyboard; never while loading or disabled.\n",
		"\n\nSlots: default, leading, trailing\n",
		"\n\nNotes:\n`variant` is `solid`",
		"\n\nExample: Basic\n\n```vue\n<script setup vapor lang=\"ts\">\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Button lacks %q:\n%s", want, out)
		}
	}
	// A required prop, and a slot with props.
	_, out, _ = catalog(t, "Table")
	if !strings.Contains(out, ", required — ") || !strings.Contains(out, "<key>-cell { ") {
		t.Errorf("Table has no required prop or no slot props:\n%s", out)
	}
	// An event with no payload.
	_, out, _ = catalog(t, "Banner")
	if !strings.Contains(out, "\n- close() — ") {
		t.Errorf("Banner has no event without a payload:\n%s", out)
	}
}

func TestCatalogMatchesNames(t *testing.T) {
	outsideApp(t)
	_, want, _ := catalog(t, "DatePicker")
	for _, name := range []string{"datepicker", "date-picker", "DATEPICKER", "Date-Picker"} {
		if code, out, errs := catalog(t, name); code != 0 || out != want {
			t.Errorf("%s: exit %d, stderr %q, output %.40q", name, code, errs, out)
		}
	}
	_, button, _ := catalog(t, "button")
	_, both, _ := catalog(t, "button", "date-picker")
	if both != button+"\n"+want {
		t.Errorf("two names do not print both entries, a blank line between:\n%s", both)
	}
	// A name given twice is printed once.
	if _, again, _ := catalog(t, "Button", "button"); again != button {
		t.Errorf("Button twice:\n%s", again)
	}
}

func TestCatalogExamples(t *testing.T) {
	outsideApp(t)
	var button map[string]any
	for _, e := range repoEntries(t) {
		if e["name"] == "Button" {
			button = e
		}
	}
	code, out, errs := catalog(t, "Button", "--examples")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	examples := list(button["examples"])
	if len(examples) < 2 {
		t.Fatal("Button has fewer than two examples")
	}
	var tail string
	for _, ex := range examples {
		tail += "\nExample: " + str(ex["title"]) + "\n\n```vue\n" + str(ex["source"]) + "```\n"
	}
	if !strings.HasSuffix(out, tail) {
		t.Errorf("--examples does not end with every example:\n%s", out)
	}
	if code, _, errs := catalog(t, "--examples"); code != 2 || !strings.HasPrefix(errs, "weft catalog: --examples needs a component name; usage is ") {
		t.Errorf("--examples alone: exit %d, stderr %q", code, errs)
	}
}

func TestCatalogJSON(t *testing.T) {
	outsideApp(t)
	b, _ := os.ReadFile(repoCatalog)
	var file struct{ Components []any }
	json.Unmarshal(b, &file)

	code, out, errs := catalog(t, "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	var all []any
	if err := json.Unmarshal([]byte(out), &all); err != nil {
		t.Fatalf("--json is not JSON: %v", err)
	}
	if !reflect.DeepEqual(all, file.Components) {
		t.Error("--json does not give the catalogue's entries unchanged")
	}
	_, out, _ = catalog(t, "date-picker", "--json", "Button")
	var two []map[string]any
	json.Unmarshal([]byte(out), &two)
	if len(two) != 2 || two[0]["name"] != "DatePicker" || two[1]["name"] != "Button" {
		t.Errorf("--json with names gives %d entries: %.80s", len(two), out)
	}
	var compact bytes.Buffer
	json.Compact(&compact, []byte(out))
	var want bytes.Buffer
	want.WriteString("[")
	want.Write(rawEntry(t, "DatePicker"))
	want.WriteString(",")
	want.Write(rawEntry(t, "Button"))
	want.WriteString("]")
	if !bytes.Equal(compact.Bytes(), want.Bytes()) {
		t.Error("--json changes the entries' bytes beyond their spacing")
	}
}

func TestCatalogUnknownName(t *testing.T) {
	outsideApp(t)
	for _, c := range []struct{ name, want string }{
		{"Buton", `weft catalog: no component named "Buton"; did you mean Button?` + "\n"},
		{"date", `weft catalog: no component named "date"; did you mean DatePicker, DateRangePicker?` + "\n"},
		{"Datepikker", `weft catalog: no component named "Datepikker"; did you mean DatePicker?` + "\n"},
		// A name it starts comes before a closer one it is only part of.
		{"Field", `weft catalog: no component named "Field"; did you mean FieldGroup, FormField?` + "\n"},
		{"Picker", `weft catalog: no component named "Picker"; did you mean DatePicker, DateRangePicker?` + "\n"},
		{"Inpt", `weft catalog: no component named "Inpt"; did you mean Input?` + "\n"},
		{"Spreadsheet", `weft catalog: no component named "Spreadsheet"; did you mean Sheet?` + "\n"},
		{"qqqqqq", `weft catalog: no component named "qqqqqq"; run weft catalog to list them.` + "\n"},
	} {
		code, out, errs := catalog(t, "Button", c.name)
		if code != 1 || out != "" || errs != c.want {
			t.Errorf("%s: exit %d, stdout %.30q, stderr %q, want %q", c.name, code, out, errs, c.want)
		}
	}
}

func TestCatalogUsage(t *testing.T) {
	outsideApp(t)
	code, out, errs := catalog(t, "--nope")
	if code != 2 || out != "" || errs != "weft catalog: flag provided but not defined: -nope; usage is weft catalog [<Name>...] [--examples] [--json].\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
}

// writeApp makes a Weft app in dir, with a catalogue in .weft/ when catalogue
// is not nil.
func writeApp(t *testing.T, dir string, catalogue []byte) {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, "web"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"myapp","weft":{"version":"0.0.0-dev"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if catalogue != nil {
		os.MkdirAll(filepath.Join(dir, ".weft", "catalog"), 0o755)
		if err := os.WriteFile(filepath.Join(dir, ".weft", "catalog", "components.json"), catalogue, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Inside an app, from any folder of it, the catalogue is the app's.
func TestCatalogReadsTheApp(t *testing.T) {
	outsideApp(t)
	b, _ := os.ReadFile(repoCatalog)
	b = bytes.Replace(b, []byte(`"purpose": "A button that runs an action, or a link drawn as a button when it has `+"`to`"+`."`), []byte(`"purpose": "The app's own button."`), 1)
	if !bytes.Contains(b, []byte("The app's own button.")) {
		t.Fatal("the test did not change Button's purpose")
	}
	writeApp(t, "myapp", b)
	t.Chdir("myapp/web")
	_, out, errs := catalog(t, "Button")
	if !strings.HasPrefix(out, "# Button\n\nThe app's own button.\n") {
		t.Errorf("Button in the app: %.80q, stderr %q", out, errs)
	}
	if _, out, _ := catalog(t); !strings.Contains(out, "The app's own button.") {
		t.Error("the list in the app is not the app's")
	}
}

// An app whose .weft/ has no catalogue yet, as in a fresh clone, reads the
// bundle's.
func TestCatalogAppWithoutWeftFolder(t *testing.T) {
	outsideApp(t)
	writeApp(t, "myapp", nil)
	t.Chdir("myapp")
	if code, out, errs := catalog(t, "Button"); code != 0 || !strings.HasPrefix(out, "# Button\n\nA button that runs an action") {
		t.Errorf("exit %d, %.60q, stderr %q", code, out, errs)
	}
}

func TestCatalogBadFile(t *testing.T) {
	outsideApp(t)
	writeApp(t, "myapp", []byte("{not json"))
	t.Chdir("myapp")
	code, _, errs := catalog(t)
	file := filepath.Join(".weft", "catalog", "components.json")
	if code != 1 || !strings.HasPrefix(errs, "weft catalog: ") || !strings.Contains(errs, file) || strings.Count(strings.TrimSpace(errs), "\n") != 0 {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

func TestCatalogWithoutBundle(t *testing.T) {
	t.Setenv("WEFT_BUNDLE", "")
	t.Chdir(t.TempDir())
	if _, err := openBundle(); err == nil {
		t.Skip("this weft carries its bundle")
	}
	code, _, errs := catalog(t)
	if code != 1 || errs != "weft catalog: "+errNoBundle+".\n" {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

// One entry is smaller than its JSON. Measured on 2026-11-24: Button 49.5 %,
// DatePicker 73.7 %. The notes, the docs' prose, are most of the rest
// (Button 29 %, DatePicker 49 % without them), and every prop, event and slot
// is kept, so the limits leave room for the docs to grow and catch a printer
// that writes the JSON's keys or every example again.
func TestCatalogIsSmall(t *testing.T) {
	outsideApp(t)
	for _, c := range []struct {
		name  string
		limit float64
	}{{"Button", 0.55}, {"DatePicker", 0.80}} {
		_, out, _ := catalog(t, c.name)
		raw := rawEntry(t, c.name)
		ratio := float64(len(out)) / float64(len(raw))
		t.Logf("%s: %d B printed, %d B of JSON: %.1f %%", c.name, len(out), len(raw), 100*ratio)
		if ratio >= c.limit {
			t.Errorf("%s prints %.1f %% of its JSON, at or over %.0f %%", c.name, 100*ratio, 100*c.limit)
		}
	}
}
