package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/flakerimi/weft"
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// templateData is what a .tmpl file of the app template can use.
type templateData struct {
	Name, Module, Version, Theme string
}

// reserved are names an app may not take: its binary, named like the app,
// would be written into a folder of the app or of its build.
var reserved = map[string]bool{"web": true, "api": true, "app": true, "node_modules": true, "dist": true}

func createCmd(args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft create <name> [--module <path>] [--theme neutral|lisa|weft] [--no-install]."
	fl := flag.NewFlagSet("create", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	module := fl.String("module", "", "the Go module path")
	theme := fl.String("theme", "neutral", "neutral, lisa or weft")
	noInstall := fl.Bool("no-install", false, "do not run pnpm install")
	var names []string
	for {
		if code, done := flagsFailed(fl.Parse(args), "create", usage, stdout, stderr); done {
			return code
		}
		if fl.NArg() == 0 {
			break
		}
		names = append(names, fl.Arg(0))
		args = fl.Args()[1:]
	}
	if len(names) != 1 {
		fmt.Fprintln(stderr, "weft create: give exactly one name; "+usage)
		return 2
	}
	name := names[0]
	if !nameRe.MatchString(name) {
		fmt.Fprintf(stderr, "weft create: %q is not a name; use a lowercase letter then lowercase letters, digits and hyphens, at most 64 in all.\n", name)
		return 2
	}
	if reserved[name] {
		fmt.Fprintf(stderr, "weft create: %s is the name of a folder Weft uses in an app, where its binary would be written; choose another name.\n", name)
		return 2
	}
	if *module == "" {
		*module = name
	}
	if strings.ContainsAny(*module, " \t\r\n\"`\\") {
		fmt.Fprintf(stderr, "weft create: %q is not a module path; use a path without spaces or quotes.\n", *module)
		return 2
	}
	switch *theme {
	case "neutral", "lisa", "weft":
	default:
		fmt.Fprintf(stderr, "weft create: %q is not a theme; use neutral, lisa or weft.\n", *theme)
		return 2
	}

	if err := prerequisites(); err != nil {
		fmt.Fprintf(stderr, "weft create: %v.\n", strings.TrimSuffix(err.Error(), "."))
		return 1
	}
	if entries, err := os.ReadDir(name); err == nil && len(entries) > 0 {
		fmt.Fprintf(stderr, "weft create: %s already exists and is not empty; choose another name or move it away.\n", name)
		return 1
	} else if _, statErr := os.Stat(name); err != nil && statErr == nil {
		fmt.Fprintf(stderr, "weft create: %s already exists and is not a folder; choose another name or move it away.\n", name)
		return 1
	}
	bundle, err := openBundle()
	if err != nil {
		fmt.Fprintf(stderr, "weft create: %v.\n", strings.TrimSuffix(err.Error(), "."))
		return 1
	}
	data := templateData{Name: name, Module: *module, Version: weft.Version, Theme: *theme}
	if err := writeTemplate(bundle, name, data); err != nil {
		fmt.Fprintf(stderr, "weft create: cannot write %s (%v); fix that and run the command again.\n", name, err)
		return 1
	}
	if err := unpack(bundle, name, weft.Version); err != nil {
		fmt.Fprintf(stderr, "weft create: %v.\n", strings.TrimSuffix(err.Error(), "."))
		return 1
	}
	if !*noInstall {
		cmd := exec.Command("pnpm", "install")
		cmd.Dir = name
		cmd.Stdout, cmd.Stderr = stdout, stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(stderr, "weft create: pnpm install did not finish; run pnpm install in %s to finish.\n", name)
			return 1
		}
		if err := writeTypes(name, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "weft create: %v.\n", strings.TrimSuffix(err.Error(), "."))
			return 1
		}
	}
	fmt.Fprintf(stdout, "Created %s. Next: cd %s && weft dev\n", name, name)
	return 0
}

// writeTemplate writes the bundle's template/ folder into dir. A file ending
// .tmpl is rendered with the delimiters [[ and ]] (Vue uses {{ }}) and loses
// the suffix; gitignore.tmpl, env.tmpl and env.sample.tmpl become .gitignore,
// .env and .env.sample.
func writeTemplate(bundle fs.FS, dir string, data templateData) error {
	src, err := fs.Sub(bundle, "template")
	if err != nil {
		return err
	}
	if _, err := fs.Stat(src, "."); err != nil {
		return fmt.Errorf("the bundle has no template folder: %w", err)
	}
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".tmpl") {
			t, err := template.New(p).Delims("[[", "]]").Option("missingkey=error").Parse(string(content))
			if err != nil {
				return err
			}
			var b strings.Builder
			if err := t.Execute(&b, data); err != nil {
				return err
			}
			content = []byte(b.String())
			p = strings.TrimSuffix(p, ".tmpl")
			// The template's dotfiles are kept without the dot, as a tool
			// that reads a folder may treat .gitignore and .env as its own.
			switch base := path.Base(p); base {
			case "gitignore", "env", "env.sample":
				p = path.Join(path.Dir(p), "."+base)
			}
			target = filepath.Join(dir, filepath.FromSlash(p))
		}
		mode := os.FileMode(0o644)
		if info, err := d.Info(); err == nil && info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		return os.WriteFile(target, content, mode)
	})
}
