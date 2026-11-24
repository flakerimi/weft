package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const errNoApp = "no Weft app here; run it in a folder made by weft create."

func buildCmd(args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft build [-o <file>]."
	fl := flag.NewFlagSet("build", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	out := fl.String("o", "", "the binary to write")
	if code, done := flagsFailed(fl.Parse(args), "build", usage, stdout, stderr); done {
		return code
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "weft build: %q is not a flag; %s\n", fl.Arg(0), usage)
		return 2
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintln(stderr, "weft build: "+errNoApp)
		return 1
	}
	cfg, err := ensure(appDir, stdout, stderr)
	if err != nil {
		return fail(stderr, "build", err)
	}

	pages := []string{"exec", "weft-build-pages", "--root", "web", "--out", filepath.Join("web", "dist")}
	if cfg.Theme != "" {
		pages = append(pages, "--theme", cfg.Theme)
	}
	if err := runIn(appDir, stdout, stderr, "pnpm", pages...); err != nil {
		fmt.Fprintln(stderr, "weft build: the page build (weft-build-pages) failed; fix what its output above says and run weft build again.")
		return 1
	}

	// -o is relative to where weft build runs; the default is the app's name
	// in the app's folder.
	shown, target := *out, *out
	if target == "" {
		target = filepath.Join(appDir, binaryName(cfg, appDir))
		shown = target
		if wd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(wd, target); err == nil {
				shown = rel
			}
		}
	} else if target, err = filepath.Abs(target); err != nil {
		return fail(stderr, "build", err)
	}
	if err := runIn(appDir, stdout, stderr, "go", "build", "-o", target, "."); err != nil {
		fmt.Fprintln(stderr, "weft build: go build failed; fix what its output above says and run weft build again.")
		return 1
	}
	fmt.Fprintf(stdout, "Built %s\n", shown)
	return 0
}

// binaryName is the app's package.json name when it makes a plain file name,
// else the app folder's name.
func binaryName(cfg appConfig, appDir string) string {
	if nameRe.MatchString(cfg.Name) {
		return cfg.Name
	}
	return filepath.Base(appDir)
}

// runIn runs name with args in dir, passing its output through. There is no
// shell, so no path needs quoting.
func runIn(dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

// fail prints err as one sentence after "weft <command>: " and returns 1.
func fail(stderr io.Writer, command string, err error) int {
	fmt.Fprintf(stderr, "weft %s: %s.\n", command, strings.TrimSuffix(err.Error(), "."))
	return 1
}
