package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
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
		// An app moved part-way, or by an older weft, has no /password, the
		// page every mailed link opens: it catches up on the account files,
		// written over nothing. With /password there, a page the author
		// removed stays removed.
		_, noPassword := os.Stat(filepath.Join(appDir, "web", "pages", "password.vue"))
		if src, err := os.ReadFile(filepath.Join(appDir, "app", "modules.go")); err == nil && errors.Is(noPassword, fs.ErrNotExist) && listsModule(src, authPackage) {
			bundle, err := openBundle()
			if err != nil {
				return fail(stderr, "upgrade", err)
			}
			added, err := addAccountFiles(bundle, appDir)
			for _, f := range added {
				fmt.Fprintln(stdout, "Added "+f)
			}
			if err != nil {
				return fail(stderr, "upgrade", err)
			}
		}
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
	// An app made before 0.2.0 lists the login without mail. What refuses
	// does so here, before anything is written, so a refused upgrade leaves
	// the app as it was.
	modulesGo := filepath.Join(appDir, "app", "modules.go")
	modules, err := os.ReadFile(modulesGo)
	moduleList, addedMail := modules, false
	if err == nil {
		if moduleList, addedMail, err = addMailModule(modules); err != nil {
			fmt.Fprintln(stderr, "weft upgrade: "+err.Error())
			return 1
		}
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
	// An app made by 0.0.17 or before has no web/dist marker.
	if err := keepDist(appDir); err != nil {
		return fail(stderr, "upgrade", err)
	}
	// mail.Module(), checked above, and the pages for the links it mails.
	if addedMail {
		if err := os.WriteFile(modulesGo, moduleList, 0o644); err != nil {
			return fail(stderr, "upgrade", err)
		}
		fmt.Fprintln(stdout, "weft upgrade: listed mail.Module() before auth.Module() in app/modules.go, which the login needs from Weft 0.2.0.")
	}
	login, middleware := listsModule(moduleList, authPackage), listsModule(moduleList, middlewarePackage)
	if login {
		added, err := addAccountFiles(bundle, appDir)
		for _, f := range added {
			fmt.Fprintln(stdout, "Added "+f)
		}
		if err != nil {
			return fail(stderr, "upgrade", err)
		}
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
	if login && !middleware {
		fmt.Fprintln(stdout, `Next: list middleware.Module() in app/modules.go, first, so the lines the other modules log are in its format, with the import "github.com/flakerimi/weft/warp/middleware"; it adds security headers, a request log and a rate limit on /api.`)
	}
	if _, err := os.Stat(filepath.Join(appDir, "web", "session.ts")); err == nil {
		fmt.Fprintln(stdout, "web/session.ts is kept: pages may now import useSession, requireSession and nextPath from @weft/ui, and once none imports ../session.ts, the file can go.")
	}
	sayWeftServe(appDir, stderr)
	return 0
}

// accountFiles are the pages and forms of the login's links and accounts,
// which weft create writes from 0.2.0 and an app made before has not.
var accountFiles = []string{
	"web/components/AccountLinks.vue", "web/components/AccountPanel.vue", "web/components/ForgotForm.vue",
	"web/components/PasswordForm.vue", "web/components/RegisterForm.vue", "web/components/UsersPanel.vue",
	"web/pages/account.vue", "web/pages/forgot.vue", "web/pages/password.vue", "web/pages/register.vue", "web/pages/users.vue",
	"web/session-check.ts",
}

// addAccountFiles writes each of accountFiles from the bundle's template that
// the app at appDir does not have, and returns those it wrote. A file the app
// has is the author's and is never written over; nor are other template
// files, so a page or module the author removed does not come back.
func addAccountFiles(bundle fs.FS, appDir string) ([]string, error) {
	var added []string
	for _, f := range accountFiles {
		target := filepath.Join(appDir, filepath.FromSlash(f))
		if _, err := os.Lstat(target); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return added, err
		}
		content, err := fs.ReadFile(bundle, "template/"+f)
		if err != nil {
			return added, fmt.Errorf("the bundle's template has no %s (%v); reinstall weft", f, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return added, err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return added, err
		}
		added = append(added, f)
	}
	return added, nil
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
