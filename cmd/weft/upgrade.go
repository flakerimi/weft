package main

import (
	"cmp"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flakerimi/weft"
)

func upgradeCmd(args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft upgrade [--no-install]."
	fl := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	noInstall := fl.Bool("no-install", false, "do not run pnpm install or go mod download")
	if code, done := flagsFailed(fl.Parse(args), "upgrade", usage, stdout, stderr); done {
		return code
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "weft upgrade: %q is not a flag; %s\n", fl.Arg(0), usage)
		return 2
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintln(stderr, "weft upgrade: "+errNoApp)
		return 1
	}
	cfg, err := readApp(appDir)
	if err != nil {
		return fail(stderr, "upgrade", err)
	}
	name := cfg.Name
	if name == "" {
		name = filepath.Base(appDir)
	}
	if cfg.Version == weft.Version {
		fmt.Fprintf(stdout, "%s is already on Weft %s\n", name, weft.Version)
		sayWeftServe(appDir, stderr)
		return 0
	}
	if newerVersion(cfg.Version, weft.Version) {
		fmt.Fprintf(stderr, "weft upgrade: this app is on Weft %s, newer than this weft %s; run weft update to get %s or later.\n", cfg.Version, weft.Version, cfg.Version)
		return 1
	}
	if err := prerequisites(); err != nil {
		return fail(stderr, "upgrade", err)
	}
	bundle, err := openBundle()
	if err != nil {
		return fail(stderr, "upgrade", err)
	}
	// go.mod first: it is the author's file, the likeliest to be refused.
	if !localModule(weft.Version) {
		if err := moveGoMod(appDir, weft.Version, stderr); err != nil {
			return fail(stderr, "upgrade", err)
		}
	}
	if err := writeAppVersion(appDir, weft.Version); err != nil {
		return fail(stderr, "upgrade", err)
	}
	if err := unpack(bundle, appDir, weft.Version); err != nil {
		return fail(stderr, "upgrade", err)
	}
	if !*noInstall {
		if err := pnpmInstall(appDir, stdout, stderr); err != nil {
			return fail(stderr, "upgrade", err)
		}
		if err := fetchModule(appDir, stdout, stderr); err != nil {
			return fail(stderr, "upgrade", err)
		}
		if err := writeTypes(appDir, stdout, stderr); err != nil {
			return fail(stderr, "upgrade", err)
		}
	}
	fmt.Fprintf(stdout, "Moved %s from Weft %s to %s\n", name, cfg.Version, weft.Version)
	sayWeftServe(appDir, stderr)
	return 0
}

// sayWeftServe says how to move an app whose Go, in its folder's .go files,
// still uses weft.Serve, which moved to warp: an app made by Weft 0.0.6 or
// before does not compile once upgraded. main.go is looked at first.
func sayWeftServe(appDir string, stderr io.Writer) {
	entries, _ := os.ReadDir(appDir)
	names := []string{"main.go"}
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".go") && n != "main.go" && !e.IsDir() {
			names = append(names, n)
		}
	}
	for _, n := range names {
		if usesWeftServe(filepath.Join(appDir, n)) {
			fmt.Fprintf(stderr, "weft upgrade: %s still calls weft.Serve, which moved; see https://weft.base.al/docs/reference/cli#moving-an-app-to-warp for the two lines to change.\n", n)
			return
		}
	}
}

// usesWeftServe reports whether the Go file at path uses Serve from the root
// package, github.com/flakerimi/weft, under whatever name it is imported:
// the code is parsed, so a comment naming weft.Serve is not a use. A file
// that does not parse is left to the Go compiler.
func usesWeftServe(path string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	name := ""
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "github.com/flakerimi/weft" {
			name = "weft"
			if imp.Name != nil {
				name = imp.Name.Name
			}
		}
	}
	if name == "" || name == "_" {
		return false
	}
	found := false
	var look func(n ast.Node) bool
	look = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && x.Name == name && n.Sel.Name == "Serve" {
				found = true
			}
			// What follows the dot is another package's or a value's own name.
			ast.Inspect(n.X, look)
			return false
		case *ast.Ident:
			// Imported with a dot, Serve is named alone.
			if name == "." && n.Name == "Serve" {
				found = true
			}
		}
		return !found
	}
	ast.Inspect(f, look)
	return found
}

// newerVersion says whether version a is newer than b, by semantic
// versioning's precedence, for versions shaped as scripts/version.sh accepts
// (1.2.3, 1.2.3-alpha.1). A development build's version (one ending -dev) is
// older than any other. A version of another shape is never newer, nor older.
func newerVersion(a, b string) bool {
	if !versionShape.MatchString(a) || !versionShape.MatchString(b) {
		return false
	}
	if devBuild(a) != devBuild(b) {
		return devBuild(b)
	}
	return compareVersions(a, b) > 0
}

func devBuild(v string) bool { return strings.HasSuffix(v, "-dev") }

// compareVersions returns -1, 0 or 1 as a is older than, as old as, or newer
// than b, and 0 when either is not a version.
func compareVersions(a, b string) int {
	ma, mb := versionShape.FindStringSubmatch(a), versionShape.FindStringSubmatch(b)
	if ma == nil || mb == nil {
		return 0
	}
	for i := 1; i <= 3; i++ {
		if c := compareNumbers(ma[i], mb[i]); c != 0 {
			return c
		}
	}
	pa, pb := ma[4], mb[4]
	switch {
	case pa == pb:
		return 0
	case pa == "":
		return 1 // a release is newer than its pre-releases
	case pb == "":
		return -1
	}
	fa, fb := strings.Split(pa, "."), strings.Split(pb, ".")
	for i := 0; i < len(fa) && i < len(fb); i++ {
		na, nb := numeric.MatchString(fa[i]), numeric.MatchString(fb[i])
		var c int
		switch {
		case na && nb:
			c = compareNumbers(fa[i], fb[i])
		case na:
			c = -1 // a number is older than a word
		case nb:
			c = 1
		default:
			c = strings.Compare(fa[i], fb[i])
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(fa), len(fb))
}

var (
	versionShape = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-z0-9.]+))?$`)
	numeric      = regexp.MustCompile(`^\d+$`)
)

// compareNumbers compares two strings of digits by their value, however long.
func compareNumbers(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
