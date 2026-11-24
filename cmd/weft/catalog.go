package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// catalogEntry is one component of catalog/components.json, as
// scripts/catalog.ts writes it. Left out of the printed form: file and each
// example's name and file, which are paths in the Weft repository, and each
// slot's dynamic, which its name already shows (<key>).
type catalogEntry struct {
	Name    string `json:"name"`
	Group   string `json:"group"`
	File    string `json:"file"`
	Purpose string `json:"purpose"`
	Use     string `json:"use"`
	Avoid   string `json:"avoid"`
	Adapts  struct {
		Input   string `json:"input"`
		Surface string `json:"surface"`
		Width   string `json:"width"`
	} `json:"adapts"`
	Props []struct {
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Required    bool    `json:"required"`
		Default     *string `json:"default"`
		Description string  `json:"description"`
	} `json:"props"`
	Emits []struct {
		Name        string `json:"name"`
		Payload     string `json:"payload"`
		Description string `json:"description"`
	} `json:"emits"`
	Slots []struct {
		Name    string   `json:"name"`
		Props   []string `json:"props"`
		Dynamic bool     `json:"dynamic"`
	} `json:"slots"`
	Notes    string `json:"notes"`
	Examples []struct {
		Name   string `json:"name"`
		Title  string `json:"title"`
		File   string `json:"file"`
		Source string `json:"source"`
	} `json:"examples"`
}

// component is a catalogue entry decoded, with its JSON as the file has it.
type component struct {
	catalogEntry
	raw json.RawMessage
}

func catalogCmd(args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft catalog [<Name>...] [--examples] [--json]."
	fl := flag.NewFlagSet("catalog", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	asJSON := fl.Bool("json", false, "the entries as the catalogue has them")
	examples := fl.Bool("examples", false, "every example, not only the first")
	var names []string
	for {
		if err := fl.Parse(args); err != nil {
			fmt.Fprintf(stderr, "weft catalog: %v; %s\n", err, usage)
			return 2
		}
		if fl.NArg() == 0 {
			break
		}
		names = append(names, fl.Arg(0))
		args = fl.Args()[1:]
	}
	if *examples && len(names) == 0 && !*asJSON {
		fmt.Fprintf(stderr, "weft catalog: --examples needs a component name; %s\n", usage)
		return 2
	}
	all, err := loadCatalog()
	if err != nil {
		return fail(stderr, "catalog", err)
	}
	picked := all
	if len(names) > 0 {
		picked = nil
		seen := map[string]bool{}
		for _, n := range names {
			c := findComponent(all, n)
			if c == nil {
				msg := fmt.Sprintf("weft catalog: no component named %q; ", n)
				if near := closest(all, n); len(near) > 0 {
					msg += "did you mean " + strings.Join(near, ", ") + "?"
				} else {
					msg += "run weft catalog to list them."
				}
				fmt.Fprintln(stderr, msg)
				return 1
			}
			if !seen[c.Name] {
				seen[c.Name] = true
				picked = append(picked, *c)
			}
		}
	}
	switch {
	case *asJSON:
		raws := make([]json.RawMessage, len(picked))
		for i, c := range picked {
			raws[i] = c.raw
		}
		// As the file has them: no < or > turned into \u003c and \u003e.
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(raws); err != nil {
			return fail(stderr, "catalog", err)
		}
	case len(names) == 0:
		printList(stdout, all)
	default:
		for i, c := range picked {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			printEntry(stdout, c.catalogEntry, *examples)
		}
	}
	return 0
}

// loadCatalog reads the app's .weft/catalog/components.json inside an app,
// else, or when the app has none yet, the bundle's.
func loadCatalog() ([]component, error) {
	if appDir, err := findApp("."); err == nil {
		file := filepath.Join(appDir, ".weft", "catalog", "components.json")
		data, err := os.ReadFile(file)
		if err == nil {
			return parseCatalog(data, file)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("cannot read %s (%v); make it readable, or move .weft away and run weft build to unpack it again", file, err)
		}
	}
	bundle, err := openBundle()
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(bundle, "weft/catalog/components.json")
	if err != nil {
		return nil, fmt.Errorf("the bundle has no weft/catalog/components.json (%v); install a release from https://weft.base.al/install, or stage the bundle again with scripts/bundle.sh", err)
	}
	return parseCatalog(data, "the bundle's weft/catalog/components.json")
}

func parseCatalog(data []byte, file string) ([]component, error) {
	var c struct {
		Components []json.RawMessage `json:"components"`
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s is not a catalogue (%v); unpack it again with weft upgrade, or stage the bundle again", file, err)
	}
	out := make([]component, len(c.Components))
	for i, raw := range c.Components {
		out[i].raw = raw
		if err := json.Unmarshal(raw, &out[i].catalogEntry); err != nil {
			return nil, fmt.Errorf("%s has an entry that is not a component (%v); unpack it again with weft upgrade, or stage the bundle again", file, err)
		}
	}
	return out, nil
}

// fold is a name as names are matched: without case or hyphens, so button,
// datepicker and date-picker find Button and DatePicker.
func fold(s string) string {
	return strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(s))
}

func findComponent(all []component, name string) *component {
	for i := range all {
		if fold(all[i].Name) == fold(name) {
			return &all[i]
		}
	}
	return nil
}

// closest returns up to three names near name: those it starts or that start
// it, then those it is part of or that are part of it; only when there are
// none, those within a few edits of it. Each kind is by fewest edits, then in
// the catalogue's order.
func closest(all []component, name string) []string {
	q := fold(name)
	limit := max(2, len(q)/3)
	type near struct {
		name        string
		kind, edits int
	}
	var found []near
	for _, c := range all {
		n := fold(c.Name)
		d := editDistance(q, n)
		switch {
		case strings.HasPrefix(n, q) || strings.HasPrefix(q, n):
			found = append(found, near{c.Name, 0, d})
		case strings.Contains(n, q) || strings.Contains(q, n):
			found = append(found, near{c.Name, 1, d})
		case d <= limit:
			found = append(found, near{c.Name, 2, d})
		}
	}
	if len(found) > 0 && found[0].kind < 2 {
		kept := found[:0]
		for _, f := range found {
			if f.kind < 2 {
				kept = append(kept, f)
			}
		}
		found = kept
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].kind != found[j].kind {
			return found[i].kind < found[j].kind
		}
		return found[i].edits < found[j].edits
	})
	var out []string
	for i := 0; i < len(found) && i < 3; i++ {
		out = append(out, found[i].name)
	}
	return out
}

// editDistance is the Levenshtein distance between a and b, by bytes.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// printList writes one line per component, its name and its purpose, under
// its group, the groups in the order they first appear.
func printList(w io.Writer, all []component) {
	fmt.Fprintf(w, "%d components. Run weft catalog <Name> for one.\n", len(all))
	var groups []string
	byGroup := map[string][]component{}
	for _, c := range all {
		if _, ok := byGroup[c.Group]; !ok {
			groups = append(groups, c.Group)
		}
		byGroup[c.Group] = append(byGroup[c.Group], c)
	}
	width := 0
	for _, c := range all {
		width = max(width, len(c.Name))
	}
	for _, g := range groups {
		fmt.Fprintf(w, "\n%s\n", g)
		for _, c := range byGroup[g] {
			fmt.Fprintf(w, "  %-*s %s\n", width, c.Name, c.Purpose)
		}
	}
}

// printEntry writes one entry as compact Markdown, leaving out the sections
// it has nothing for, with its first example, or every one.
func printEntry(w io.Writer, e catalogEntry, allExamples bool) {
	fmt.Fprintf(w, "# %s\n\n%s\n\nUse: %s\nAvoid: %s\n", e.Name, e.Purpose, e.Use, e.Avoid)
	fmt.Fprintf(w, "\nAdapts:\n- input: %s\n- surface: %s\n- width: %s\n", e.Adapts.Input, e.Adapts.Surface, e.Adapts.Width)
	if len(e.Props) > 0 {
		fmt.Fprintln(w, "\nProps:")
		for _, p := range e.Props {
			line := "- " + p.Name + ": " + p.Type
			if p.Default != nil {
				line += " = " + *p.Default
			}
			if p.Required {
				line += ", required"
			}
			fmt.Fprintf(w, "%s — %s\n", line, p.Description)
		}
	}
	if len(e.Emits) > 0 {
		fmt.Fprintln(w, "\nEmits:")
		for _, ev := range e.Emits {
			// The payload is a tuple, [value: T]; written as the handler's
			// parameters it reads as what the handler gets.
			payload := ev.Payload
			if strings.HasPrefix(payload, "[") && strings.HasSuffix(payload, "]") {
				payload = "(" + payload[1:len(payload)-1] + ")"
			} else {
				payload = " " + payload
			}
			fmt.Fprintf(w, "- %s%s — %s\n", ev.Name, payload, ev.Description)
		}
	}
	if len(e.Slots) > 0 {
		slots := make([]string, len(e.Slots))
		for i, s := range e.Slots {
			slots[i] = s.Name
			if len(s.Props) > 0 {
				slots[i] += " { " + strings.Join(s.Props, ", ") + " }"
			}
		}
		fmt.Fprintf(w, "\nSlots: %s\n", strings.Join(slots, ", "))
	}
	if e.Notes != "" {
		fmt.Fprintf(w, "\nNotes:\n%s\n", strings.TrimRight(e.Notes, "\n"))
	}
	for i, ex := range e.Examples {
		if i > 0 && !allExamples {
			break
		}
		fmt.Fprintf(w, "\nExample: %s\n\n```vue\n%s", ex.Title, ex.Source)
		if !strings.HasSuffix(ex.Source, "\n") {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "```")
	}
}
