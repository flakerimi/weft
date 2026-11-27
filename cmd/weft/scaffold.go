package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"math"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// weft scaffold <name> <field[:type[:modifier...]]>... [--plural <word>] [--api]
// writes one resource: app/<plural>/ (the model, its five routes through
// data.Resource, and a story test over HTTP), its line in app/modules.go
// and, unless --api, its table and record pages, then runs weft gen.

const scaffoldUsage = "usage is weft scaffold <name> <field[:type[:modifier...]]>... [--plural <word>] [--api]."

// scaffoldSpec is what the command line asks for.
type scaffoldSpec struct {
	Name, Plural string
	Fields       []scaffoldField
	API          bool
}

// scaffoldField is one field: its snake_case name, its type (string, text,
// email, url, int, float, bool, date, datetime or enum), whether it may be
// null (the modifier nullable), the enum's choices, and its modifiers. Min and
// Max are as written, "" when not given.
type scaffoldField struct {
	Name, Type       string
	Null             bool
	Choices          []string
	Required, Unique bool
	Min, Max         string
}

// scaffoldError is a refusal: one sentence without "weft scaffold: " in
// front, and the exit code.
type scaffoldError struct {
	code int
	msg  string
}

func refuse(code int, format string, args ...any) *scaffoldError {
	return &scaffoldError{code: code, msg: fmt.Sprintf(format, args...)}
}

var (
	snakeName   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	choiceRe    = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	textTypes   = map[string]bool{"string": true, "text": true, "email": true, "url": true}
	scalarTypes = []string{"string", "text", "email", "url", "int", "float", "bool", "date", "datetime"}
	// scaffoldReserved are the names of Weft's own packages and folders in an
	// app, which a resource's package or folder would collide with.
	scaffoldReserved = []string{"data", "auth", "warp", "app", "web", "test", "main"}
	// takenTypes are type names Weft's modules or the generated client have,
	// which a resource's type, or its Input twin, would collide with.
	takenTypes = map[string]bool{
		"User": true, "Login": true, "Setup": true, "Record": true, "Query": true, "List": true, "ID": true,
		"Message": true, "ApiError": true, "FieldRules": true, "Partial": true, "Promise": true, "Response": true,
	}
	// storyNames are the identifiers the story test uses, which the package's
	// name would hide.
	storyNames = []string{"http", "filepath", "slices", "strconv", "testing", "time", "authtest", "warptest", "sqlite",
		"start", "idsOf", "idOf", "c", "t", "l", "tc", "first", "second", "changed", "shown", "left", "anon", "name", "r", "e", "got"}
	futureTypes = map[string]string{
		"belongs_to":   "and relations land in Weft 0.3.0 (W3)",
		"has_many":     "and relations land in Weft 0.3.0 (W3)",
		"many_to_many": "and relations land in Weft 0.3.0 (W3)",
		"image":        "and file attachments land in Weft 0.4.0 (W6)",
		"file":         "and file attachments land in Weft 0.4.0 (W6)",
		"json":         "which lands in Weft 0.6.0 (W7)",
		"translated":   "which lands in Weft 0.6.0 (W7)",
	}
)

// irregular are the plurals the four rules do not make.
var irregular = map[string]string{
	"person": "people", "child": "children", "man": "men", "woman": "women", "quiz": "quizzes",
	"leaf": "leaves", "knife": "knives", "life": "lives", "wife": "wives", "wolf": "wolves",
	"series": "series", "news": "news", "equipment": "equipment", "species": "species",
	"information": "information", "staff": "staff",
}

// pluralOf is the plural of a snake_case name, made of its last word: the
// table above, then a consonant and y to ies, s, x, z, ch and sh to es, and
// any other with s. It is the same every time; --plural overrides it.
func pluralOf(name string) string {
	head, word := "", name
	if i := strings.LastIndexByte(name, '_'); i >= 0 {
		head, word = name[:i+1], name[i+1:]
	}
	if p, ok := irregular[word]; ok {
		return head + p
	}
	switch {
	case len(word) > 1 && strings.HasSuffix(word, "y") && !strings.ContainsRune("aeiou", rune(word[len(word)-2])):
		return head + word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s"), strings.HasSuffix(word, "x"), strings.HasSuffix(word, "z"),
		strings.HasSuffix(word, "ch"), strings.HasSuffix(word, "sh"):
		return head + word + "es"
	}
	return head + word + "s"
}

// initialisms are written in capitals in a Go name, as Go writes them.
var initialisms = map[string]string{"id": "ID", "url": "URL", "api": "API", "http": "HTTP", "ip": "IP", "uuid": "UUID", "html": "HTML", "json": "JSON", "sql": "SQL"}

// pascal is a snake_case name as a Go type or field name: line_item is
// LineItem, site_url SiteURL.
func pascal(name string) string {
	var b strings.Builder
	for _, w := range strings.Split(name, "_") {
		if w == "" {
			continue
		}
		if up, ok := initialisms[w]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// human is a snake_case name as words: published_at is "published at".
func human(name string) string {
	return strings.Join(strings.FieldsFunc(name, func(r rune) bool { return r == '_' }), " ")
}

// capital is s with its first letter in capitals.
func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func scaffoldCmd(args []string, stdout, stderr io.Writer) int {
	spec, perr := parseScaffold(args)
	if perr != nil {
		if perr.code == 0 { // --help
			fmt.Fprintln(stdout, strings.ToUpper(scaffoldUsage[:1])+scaffoldUsage[1:])
			return 0
		}
		fmt.Fprintf(stderr, "weft scaffold: %s\n", perr.msg)
		return perr.code
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintln(stderr, "weft scaffold: "+errNoApp)
		return 1
	}
	plan, err := planScaffold(appDir, spec)
	if err != nil {
		fmt.Fprintf(stderr, "weft scaffold: %s\n", err)
		return 1
	}
	if _, err := ensure(appDir, stdout, stderr); err != nil {
		return fail(stderr, "scaffold", err)
	}
	if err := writePlan(appDir, plan); err != nil {
		fmt.Fprintf(stderr, "weft scaffold: %s\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Scaffolded %s as %s:\n", spec.Name, spec.Plural)
	for _, f := range plan.files {
		if f.note != "" {
			fmt.Fprintf(stdout, "  %s (%s)\n", f.path, f.note)
		} else {
			fmt.Fprintf(stdout, "  %s\n", f.path)
		}
	}
	if !plan.auth {
		fmt.Fprintf(stdout, "weft scaffold: app/modules.go lists no auth module, so the routes of %s are open to anyone; to guard them, add auth.Module() and give data.Resource data.PerAction(auth.Per(%q)).\n", spec.Plural, spec.Plural)
	}
	ctx, stop := signal.NotifyContext(context.Background(), stopSignals...)
	defer stop()
	if code := gen(ctx, nil, stdout, stderr); code != 0 {
		fmt.Fprintf(stderr, "weft scaffold: the files above are written, and weft gen did not finish; fix what it says and run weft gen.\n")
		return code
	}
	return 0
}

// parseScaffold reads the command line. A nil error with code 0 is never
// returned; --help is a scaffoldError with code 0.
func parseScaffold(args []string) (scaffoldSpec, *scaffoldError) {
	var spec scaffoldSpec
	fl := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	plural := fl.String("plural", "", "the plural, when the pluralizer's is not the one")
	api := fl.Bool("api", false, "write only the Go")
	var words []string
	for {
		if err := fl.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return spec, &scaffoldError{}
			}
			return spec, refuse(2, "%v; %s", err, scaffoldUsage)
		}
		if fl.NArg() == 0 {
			break
		}
		words = append(words, fl.Arg(0))
		args = fl.Args()[1:]
	}
	spec.API = *api
	if len(words) == 0 {
		return spec, refuse(2, "give a resource name and at least one field; %s", scaffoldUsage)
	}
	spec.Name = words[0]
	if !snakeName.MatchString(spec.Name) {
		return spec, refuse(2, "%q is not a resource name; use a lowercase letter, then lowercase letters, digits and _, such as line_item.", spec.Name)
	}
	if len(words) == 1 {
		return spec, refuse(2, "give at least one field after %s, such as name:string; %s", spec.Name, scaffoldUsage)
	}
	spec.Plural = pluralOf(spec.Name)
	if *plural != "" {
		if !snakeName.MatchString(*plural) {
			return spec, refuse(2, "--plural %q is not a name; use a lowercase letter, then lowercase letters, digits and _, such as line_items.", *plural)
		}
		spec.Plural = *plural
	}
	for _, n := range []string{spec.Name, spec.Plural} {
		if slices.Contains(scaffoldReserved, n) {
			return spec, refuse(2, "%q is a name Weft uses in an app (%s); choose another.", n, strings.Join(scaffoldReserved, ", "))
		}
	}
	if takenTypes[pascal(spec.Name)] {
		return spec, refuse(2, "%q would make the type %s, which Weft's own modules or the generated client use already; choose another name.", spec.Name, pascal(spec.Name))
	}
	if spec.Plural == "users" {
		return spec, refuse(2, "%q is the table auth keeps its users in; give another with --plural.", spec.Plural)
	}
	if token.IsKeyword(spec.Plural) || types.Universe.Lookup(spec.Plural) != nil {
		return spec, refuse(2, "%q is a word Go keeps for itself, so it cannot name the package app/%s; give another with --plural.", spec.Plural, spec.Plural)
	}
	if slices.Contains(storyNames, spec.Plural) {
		return spec, refuse(2, "%q is a name the test weft scaffold writes uses for something else; give another with --plural.", spec.Plural)
	}
	goNames := map[string]string{"ID": "id", "CreatedAt": "created_at", "UpdatedAt": "updated_at", "Record": "data.Record"}
	seen := map[string]bool{}
	for _, arg := range words[1:] {
		f, err := parseField(arg)
		if err != nil {
			return spec, err
		}
		if seen[f.Name] {
			return spec, refuse(2, "field %q is given twice; give each field once.", f.Name)
		}
		seen[f.Name] = true
		if other, ok := goNames[pascal(f.Name)]; ok {
			return spec, refuse(2, "fields %q and %q would both be the Go field %s; rename one.", other, f.Name, pascal(f.Name))
		}
		goNames[pascal(f.Name)] = f.Name
		spec.Fields = append(spec.Fields, f)
	}
	return spec, nil
}

// parseField reads one name[:type[:modifier...]].
func parseField(arg string) (scaffoldField, *scaffoldError) {
	var f scaffoldField
	if strings.ContainsAny(arg, "|(") {
		return f, refuse(2, "write enum choices separated by commas after enum=, as in status:enum=draft,live.")
	}
	// A ? is a pattern to zsh, so it only arrives quoted: one form, the modifier.
	if strings.Contains(arg, "?") {
		return f, refuse(2, "write a field that may be null with the modifier nullable, as in sale:float:nullable.")
	}
	parts := strings.Split(arg, ":")
	f.Name, f.Type = parts[0], "string"
	if !snakeName.MatchString(f.Name) {
		return f, refuse(2, "%q is not a field name; use a lowercase letter, then lowercase letters, digits and _, such as published_at.", f.Name)
	}
	switch f.Name {
	case "id", "created_at", "updated_at":
		return f, refuse(2, "field %q comes from data.Record, which every model embeds; leave it out.", f.Name)
	case "actions":
		return f, refuse(2, "field %q is the key DataTable gives its column of row actions; rename it.", f.Name)
	case "constructor":
		return f, refuse(2, "field %q has the name of a property every JavaScript object has, which the generated client cannot check; rename it.", f.Name)
	}
	mods := parts[1:]
	if len(parts) > 1 {
		f.Type, mods = parts[1], parts[2:]
		if rest, ok := strings.CutPrefix(f.Type, "enum="); ok {
			f.Type = "enum"
			if rest == "" {
				return f, refuse(2, "field %q has enum= with no choices; write them separated by commas, as in %s:enum=draft,live.", f.Name, f.Name)
			}
			for _, c := range strings.Split(rest, ",") {
				switch {
				case c == "":
					return f, refuse(2, "field %q has an empty choice in enum=%s; write them separated by single commas, as in %s:enum=draft,live.", f.Name, rest, f.Name)
				case !choiceRe.MatchString(c):
					return f, refuse(2, "field %q has the choice %q; write each choice with letters, digits, _, - and . only.", f.Name, c)
				case slices.Contains(f.Choices, c):
					return f, refuse(2, "field %q has the choice %q twice; give each choice once.", f.Name, c)
				}
				f.Choices = append(f.Choices, c)
			}
		} else if why, ok := futureTypes[f.Type]; ok {
			return f, refuse(1, "field %q has type %q, %s; use a scalar type for now.", f.Name, f.Type, why)
		} else if f.Type == "enum" || !slices.Contains(scalarTypes, f.Type) {
			return f, refuse(2, "field %q has the type %q, which weft scaffold does not know; use string, text, email, url, int, float, bool, date, datetime or enum=a,b.", f.Name, f.Type)
		}
	}
	a := "a " + f.Type
	if f.Type == "enum" || f.Type == "email" || f.Type == "int" {
		a = "an " + f.Type
	}
	given := map[string]bool{}
	for _, m := range mods {
		name, value, hasValue := strings.Cut(m, "=")
		if given[name] {
			return f, refuse(2, "field %q has the modifier %q twice; give it once.", f.Name, name)
		}
		given[name] = true
		switch {
		case m == "required":
			if f.Type == "bool" {
				return f, refuse(2, "field %q is a bool, which is always true or false, so it cannot be required; remove :required.", f.Name)
			}
			f.Required = true
		case m == "nullable":
			f.Null = true
		case m == "unique":
			if f.Type == "bool" || f.Type == "text" || f.Type == "date" || f.Type == "datetime" {
				return f, refuse(2, "field %q is %s, and unique is for string, email, url, enum, int and float; remove :unique.", f.Name, a)
			}
			f.Unique = true
		case (name == "min" || name == "max") && hasValue:
			if f.Type == "bool" || f.Type == "date" || f.Type == "datetime" || f.Type == "enum" {
				return f, refuse(2, "field %q is %s, and %s= is for string, text, email, url, int and float; remove :%s.", f.Name, a, name, m)
			}
			switch {
			case textTypes[f.Type]:
				if n, err := strconv.ParseUint(value, 10, 31); err != nil {
					return f, refuse(2, "field %q has %s; for text, write a whole number of characters, 0 or more, as in %s=120.", f.Name, m, name)
				} else if n == 0 && name == "max" {
					return f, refuse(2, "field %q has max=0, which holds no text; write max=1 or more.", f.Name)
				}
			case f.Type == "int":
				if _, err := strconv.ParseInt(value, 10, 53); err != nil {
					return f, refuse(2, "field %q has %s; for an int, write a whole number, as in %s=0.", f.Name, m, name)
				}
			default:
				if v, err := strconv.ParseFloat(value, 64); err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
					return f, refuse(2, "field %q has %s; for a float, write a number, as in %s=0.5.", f.Name, m, name)
				}
			}
			if name == "min" {
				f.Min = value
			} else {
				f.Max = value
			}
		default:
			return f, refuse(2, "field %q has the modifier %q; use required, nullable, unique, min=N or max=N.", f.Name, m)
		}
	}
	if f.Required && f.Null {
		return f, refuse(2, "field %q is nullable, which may be empty, so it cannot be required; remove :nullable or :required.", f.Name)
	}
	if f.Min != "" && f.Max != "" {
		lo, _ := strconv.ParseFloat(f.Min, 64)
		hi, _ := strconv.ParseFloat(f.Max, 64)
		if lo > hi {
			return f, refuse(2, "field %q has min=%s above max=%s; make min the smaller.", f.Name, f.Min, f.Max)
		}
	}
	// The story test makes records of the field: there must be values to.
	if _, ok := f.samples(); !ok {
		switch {
		case f.Type == "email":
			return f, refuse(2, "field %q is an email with max=%s, which no address fits; raise max.", f.Name, f.Max)
		case f.Type == "url":
			return f, refuse(2, "field %q is a url with max=%s, which no web address fits; raise max.", f.Name, f.Max)
		case f.Unique:
			return f, refuse(2, "field %q is unique between min=%s and max=%s, which leaves too few values for two records; widen them.", f.Name, f.Min, f.Max)
		default:
			return f, refuse(2, "field %q is required between min=%s and max=%s, where 0 counts as empty; widen them.", f.Name, f.Min, f.Max)
		}
	}
	return f, nil
}

// goType is the field's Go type.
func (f scaffoldField) goType() string {
	t := map[string]string{"int": "int64", "float": "float64", "bool": "bool", "date": "time.Time", "datetime": "time.Time"}[f.Type]
	if t == "" {
		t = "string"
	}
	if f.Null {
		return "*" + t
	}
	return t
}

// rules is the field's warp tag: required, email or url, enum, min, max and
// unique, in that order.
func (f scaffoldField) rules() []string {
	var r []string
	if f.Required {
		r = append(r, "required")
	}
	if f.Type == "email" || f.Type == "url" {
		r = append(r, f.Type)
	}
	if f.Type == "enum" {
		r = append(r, "enum="+strings.Join(f.Choices, "|"))
	}
	if f.Min != "" {
		r = append(r, "min="+f.Min)
	}
	if f.Max != "" {
		r = append(r, "max="+f.Max)
	}
	if f.Unique {
		r = append(r, "unique")
	}
	return r
}

// listTags is the field's data tag, from its type alone: every scalar but
// text is sorted on, text is searched, enum and bool are filtered on.
func (f scaffoldField) listTags() string {
	switch f.Type {
	case "string", "email", "url":
		return "sort,search"
	case "text":
		return "search"
	case "enum", "bool":
		return "sort,filter"
	}
	return "sort"
}

// tag is the field's struct tag.
func (f scaffoldField) tag() string {
	tag := fmt.Sprintf("json:%q", f.Name)
	if r := f.rules(); len(r) > 0 {
		tag += fmt.Sprintf(" warp:%q", strings.Join(r, ","))
	}
	return tag + fmt.Sprintf(" data:%q", f.listTags())
}

// sample is one value of a field the story test sends: its Go literal in a
// body, as it is written in a query, and the Go condition, over expr, that is
// true when a record's field is not it.
type sample struct {
	lit, query string
	differs    func(expr string) string
}

var sampleWords = []string{"alpha", "bravo", "charlie"}

// samples are three values of the field for the story test: the first two
// records, and the change to the first. They keep its rules; a unique field's
// first two differ. ok is false when its rules leave too few.
func (f scaffoldField) samples() ([3]sample, bool) {
	var out [3]sample
	deref := func(expr string) string {
		if f.Null {
			return "*" + expr
		}
		return expr
	}
	isNil := func(expr string) string {
		if f.Null {
			return expr + " == nil || "
		}
		return ""
	}
	switch {
	case textTypes[f.Type] || f.Type == "enum":
		for i := range out {
			var s string
			if f.Type == "enum" {
				s = f.Choices[i%len(f.Choices)]
			} else {
				var ok bool
				if s, ok = f.text(i); !ok {
					return out, false
				}
			}
			lit := strconv.Quote(s)
			out[i] = sample{lit: lit, query: s, differs: func(expr string) string { return isNil(expr) + deref(expr) + " != " + lit }}
		}
	case f.Type == "bool":
		for i, v := range []bool{false, true, true} {
			lit := strconv.FormatBool(v)
			out[i] = sample{lit: lit, query: lit, differs: func(expr string) string { return isNil(expr) + deref(expr) + " != " + lit }}
		}
	case f.Type == "date" || f.Type == "datetime":
		for i := range out {
			h, m, s := 0, 0, 0
			if f.Type == "datetime" {
				h, m, s = 3+i, 4, 5
			}
			text := fmt.Sprintf("2026-01-%02dT%02d:%02d:%02dZ", 2+i, h, m, s)
			goTime := fmt.Sprintf("time.Date(2026, 1, %d, %d, %d, %d, 0, time.UTC)", 2+i, h, m, s)
			out[i] = sample{lit: strconv.Quote(text), query: text, differs: func(expr string) string { return isNil(expr) + "!" + expr + ".Equal(" + goTime + ")" }}
		}
	default:
		nums := f.numbers()
		if len(nums) == 0 || f.Unique && len(nums) < 2 {
			return out, false
		}
		for i := range out {
			lit := strconv.FormatFloat(nums[i%len(nums)], 'g', -1, 64)
			out[i] = sample{lit: lit, query: lit, differs: func(expr string) string { return isNil(expr) + deref(expr) + " != " + lit }}
		}
	}
	return out, true
}

// text is the i-th text value of a text field within its min and max: a
// word, or an email or web address made from it, each starting with its own
// letter, so the three differ and none holds another's.
func (f scaffoldField) text(i int) (string, bool) {
	w := sampleWords[i]
	var forms []string
	switch f.Type {
	case "email":
		forms = []string{w + "@example.com", w[:1] + "@x.io", w[:1] + "@x"}
	case "url":
		forms = []string{"https://example.com/" + w, "http://x.io/" + w[:1], "http://" + w[:1]}
	default:
		forms = []string{w}
		if f.Max != "" {
			if n, _ := strconv.Atoi(f.Max); n < len(w) {
				forms = []string{w[:n]}
			}
		}
	}
	for _, s := range forms {
		if f.Max != "" {
			if n, _ := strconv.Atoi(f.Max); len(s) > n {
				continue
			}
		}
		if f.Min != "" {
			if n, _ := strconv.Atoi(f.Min); len(s) < n {
				pad := strings.Repeat("x", n-len(s))
				if at := strings.IndexByte(s, '@'); at >= 0 {
					s = s[:at] + pad + s[at:]
				} else {
					s += pad
				}
			}
		}
		return s, true
	}
	return "", false
}

// numbers are the values a number field's samples take, in order: from its
// min, small whole ones, and down from its max, kept within both, without 0
// when it is required (0 is empty), and once each.
func (f scaffoldField) numbers() []float64 {
	lo, hi := math.Inf(-1), math.Inf(1)
	var cands []float64
	if f.Min != "" {
		lo, _ = strconv.ParseFloat(f.Min, 64)
		cands = append(cands, lo, lo+1, lo+2)
	}
	cands = append(cands, 1, 2, 3)
	if f.Max != "" {
		hi, _ = strconv.ParseFloat(f.Max, 64)
		cands = append(cands, hi, hi-1, hi-2)
	}
	if f.Type == "float" && f.Min != "" && f.Max != "" {
		cands = append(cands, (lo+hi)/2, lo+(hi-lo)/4)
	}
	var out []float64
	for _, v := range cands {
		if v < lo || v > hi || f.Required && v == 0 || slices.Contains(out, v) {
			continue
		}
		if f.Type == "int" && v != math.Trunc(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

// plannedFile is one file weft scaffold writes, with what is said of it.
type plannedFile struct {
	path string // slash-separated, from the app's folder
	data []byte
	note string
}

// scaffoldPlan is everything weft scaffold writes, made and checked before
// any of it is.
type scaffoldPlan struct {
	spec  scaffoldSpec
	files []plannedFile
	auth  bool // app/modules.go lists auth.Module()
}

// planScaffold checks the app and makes every file: nothing is written. It
// refuses a resource whose package, pages or islands are there already, and
// an app/modules.go it cannot add the module to.
func planScaffold(appDir string, s scaffoldSpec) (*scaffoldPlan, error) {
	p := &scaffoldPlan{spec: s}
	pkg := "app/" + s.Plural
	if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(pkg))); err == nil {
		return nil, fmt.Errorf("%s already exists; remove it by hand or choose another resource name.", pkg)
	}
	r := newResource(s)
	if !s.API {
		for _, rel := range []string{"web/pages/" + s.Plural, "web/components/" + r.Table + ".vue", "web/components/" + r.View + ".vue"} {
			if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err == nil {
				return nil, fmt.Errorf("%s already exists; move it away, choose another resource name, or give --api to write only the Go.", rel)
			}
		}
	}
	module, err := appModule(appDir)
	if err != nil {
		return nil, err
	}
	r.Module = module
	src, err := os.ReadFile(filepath.Join(appDir, "app", "modules.go"))
	if err != nil {
		return nil, errors.New("app/modules.go is missing; make it as weft create does, with a func Modules() []warp.Module, then run weft scaffold again.")
	}
	modules, hasAuth, err := addModule(src, module+"/"+pkg, s.Plural)
	if err != nil {
		return nil, err
	}
	p.auth, r.Auth = hasAuth, hasAuth
	_, serr := os.Stat(filepath.Join(appDir, "web", "session.ts"))
	r.Session = serr == nil

	for _, f := range []struct {
		name string
		make func() ([]byte, error)
	}{
		{"model.go", r.model},
		{"routes.go", r.routes},
		{s.Plural + "_test.go", r.story},
	} {
		b, err := f.make()
		if err != nil {
			return nil, fmt.Errorf("the %s it would write does not format (%v); this is a bug in weft, so report it with the command you ran.", f.name, err)
		}
		p.files = append(p.files, plannedFile{path: pkg + "/" + f.name, data: b})
	}
	p.files = append(p.files, plannedFile{path: "app/modules.go", data: modules, note: s.Plural + ".Module() added"})
	if !s.API {
		p.files = append(p.files,
			plannedFile{path: "web/components/" + r.Table + ".vue", data: []byte(r.tableVue())},
			plannedFile{path: "web/components/" + r.View + ".vue", data: []byte(r.viewVue())},
			plannedFile{path: "web/pages/" + s.Plural + "/index.vue", data: []byte(r.indexVue())},
			plannedFile{path: "web/pages/" + s.Plural + "/[id].vue", data: []byte(r.recordVue())},
		)
	}
	return p, nil
}

// writePlan writes the plan's files: the new ones first, app/modules.go
// last, so an app is never left listing a module that is not there.
func writePlan(appDir string, p *scaffoldPlan) error {
	var modules *plannedFile
	for i, f := range p.files {
		if f.path == "app/modules.go" {
			modules = &p.files[i]
			continue
		}
		target := filepath.Join(appDir, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("cannot make the folder of %s (%v); make the app's folder writable and run weft scaffold again.", f.path, err)
		}
		if err := os.WriteFile(target, f.data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s (%v); make the app's folder writable and run weft scaffold again.", f.path, err)
		}
	}
	if modules != nil {
		file := filepath.Join(appDir, "app", "modules.go")
		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("app/modules.go went missing while weft scaffold ran (%v); put it back and add %s.Module() to it by hand.", err, p.spec.Plural)
		}
		if err := os.WriteFile(file, modules.data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("cannot write app/modules.go (%v); add %s.Module() to it by hand.", err, p.spec.Plural)
		}
	}
	return nil
}

var moduleLine = regexp.MustCompile(`(?m)^module\s+("[^"]+"|\S+)`)

// appModule is the module path go.mod names.
func appModule(appDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(appDir, "go.mod"))
	if err != nil {
		return "", errors.New("go.mod is missing; run weft scaffold in an app weft create made.")
	}
	m := moduleLine.FindSubmatch(b)
	if m == nil {
		return "", errors.New("go.mod has no module line; write one, as in module myapp, then run weft scaffold again.")
	}
	return strings.Trim(string(m[1]), `"`), nil
}

const (
	authPackage = "github.com/flakerimi/weft/warp/auth"
	dataPackage = "github.com/flakerimi/weft/warp/data"
)

// addModule returns app/modules.go with the import of importPath and
// name.Module() at the end of the list Modules returns, the rest of the file
// as it was, and whether the list has auth.Module(). The edit is made in the
// text, at the places the syntax tree gives, and formatted, so every comment
// stays where it was.
func addModule(src []byte, importPath, name string) ([]byte, bool, error) {
	const noList = "app/modules.go has no func Modules() []warp.Module returning a []warp.Module{...} list; write the modules as weft create does, then run weft scaffold again."
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app/modules.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, false, fmt.Errorf("app/modules.go does not parse (%v); fix it, then run weft scaffold again.", err)
	}
	local := map[string]string{} // import path to the name the file uses
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		n := path.Base(p)
		if imp.Name != nil {
			n = imp.Name.Name
		}
		if n == name {
			return nil, false, fmt.Errorf("app/modules.go imports a package named %s already; choose another resource name, or give another with --plural.", name)
		}
		local[p] = n
	}
	var list *ast.CompositeLit
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "Modules" || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				return list == nil
			}
			if lit, ok := ret.Results[0].(*ast.CompositeLit); ok && isModuleList(lit, local[warpPackage]) {
				list = lit
			}
			return list == nil
		})
	}
	if list == nil {
		return nil, false, errors.New(noList)
	}
	lists := func(pkg string) bool {
		return slices.ContainsFunc(list.Elts, func(e ast.Expr) bool {
			call, ok := e.(*ast.CallExpr)
			if !ok {
				return false
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return false
			}
			x, ok := sel.X.(*ast.Ident)
			return ok && local[pkg] != "" && x.Name == local[pkg] && sel.Sel.Name == "Module"
		})
	}
	if !lists(dataPackage) {
		return nil, false, errors.New("app/modules.go does not list data.Module(), which keeps the records; add it first in Modules(), import a database driver in main.go as weft create's template does, then run weft scaffold again.")
	}

	// The edits, by byte offset, made from the last to the first.
	type edit struct {
		at   int
		text string
	}
	var edits []edit
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	line := func(p token.Pos) int { return fset.Position(p).Line }
	call := name + ".Module()"
	rbrace := off(list.Rbrace)
	switch {
	case len(list.Elts) == 0:
		edits = append(edits, edit{rbrace, call})
	case line(list.Elts[len(list.Elts)-1].End()) == line(list.Rbrace):
		end := off(list.Elts[len(list.Elts)-1].End())
		if i := bytes.IndexByte(src[end:rbrace], ','); i >= 0 {
			edits = append(edits, edit{end + i + 1, " " + call})
		} else {
			edits = append(edits, edit{end, ", " + call})
		}
	default:
		start := bytes.LastIndexByte(src[:rbrace], '\n') + 1
		edits = append(edits, edit{start, "\t\t" + call + ",\n"})
	}
	quoted := strconv.Quote(importPath)
	var last *ast.GenDecl
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			last = g
		}
	}
	switch {
	case last == nil:
		edits = append(edits, edit{off(f.Name.End()), "\n\nimport " + quoted})
	case last.Lparen.IsValid():
		start := bytes.LastIndexByte(src[:off(last.Rparen)], '\n') + 1
		if line(last.Rparen) == line(last.Lparen) {
			edits = append(edits, edit{off(last.Rparen), "; " + quoted})
		} else {
			edits = append(edits, edit{start, "\t" + quoted + "\n"})
		}
	default:
		// import "x" // a comment: into a group, the comment kept on its line.
		end := off(last.End())
		eol := bytes.IndexByte(src[end:], '\n')
		if eol < 0 {
			eol = len(src) - end
		}
		edits = append(edits, edit{end + eol, "\n\t" + quoted + "\n)"}, edit{off(last.Specs[0].Pos()), "(\n\t"})
	}
	slices.SortFunc(edits, func(a, b edit) int { return b.at - a.at })
	out := slices.Clone(src)
	for _, e := range edits {
		out = slices.Insert(out, e.at, []byte(e.text)...)
	}
	formatted, err := format.Source(out)
	if err != nil {
		return nil, false, fmt.Errorf("app/modules.go could not take %s (%v); add it to the list by hand, then run weft gen.", call, err)
	}
	return formatted, lists(authPackage), nil
}

// isModuleList says whether lit is []warp.Module{...}, warp under the name
// the file imports it by.
func isModuleList(lit *ast.CompositeLit, warpName string) bool {
	arr, ok := lit.Type.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return false
	}
	sel, ok := arr.Elt.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && warpName != "" && x.Name == warpName && sel.Sel.Name == "Module"
}
