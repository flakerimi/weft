package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	// The typed client first, so the pages are built and checked against it.
	// An app that makes no warp.App is not run: it gets the client of an
	// app with no routes, and nothing is said, unless the client it has has
	// routes, which is left as it is and said. One that makes one, builds and
	// runs, but does not describe itself (it never reaches its Serve, or
	// prints something else) is said, as weft dev says it, and the pages are
	// built with the file as it was; one that does not compile or exits with
	// an error stops the build.
	if _, err := genTypes(context.Background(), "build", appDir, stderr); err != nil {
		switch {
		case errors.Is(err, errKeptClient):
			fail(stderr, "build", err)
		case errors.Is(err, errNoWarpApp):
		case errors.Is(err, errDescribeTimeout):
			fmt.Fprintf(stderr, "weft build: the app did not describe itself within %d s, so web/.weft-types/api.ts is as it was; make sure main.go calls Serve before doing other work.\n", int(describeWithin.Seconds()))
		case errors.Is(err, errSilent), errors.As(err, new(outputError)):
			if errors.As(err, new(outputError)) {
				fail(stderr, "build", err)
			}
			fmt.Fprintln(stderr, "weft build: the app did not describe itself, so web/.weft-types/api.ts is as it was; fix what its output above says.")
		default:
			return genFailed(stderr, "build", err)
		}
	}

	// The app's TypeScript, whatever web/tsconfig.json includes (the pages,
	// and tests too), is type-checked against that client, so a page that
	// reads a Go field that was renamed stops the build, not the browser. An
	// app without vue-tsc or web/tsconfig.json is not checked.
	if isFile(filepath.Join(appDir, "web", "tsconfig.json")) && isFile(filepath.Join(appDir, "node_modules", ".bin", "vue-tsc")) {
		if err := runIn(appDir, stdout, stderr, "pnpm", "exec", "vue-tsc", "-p", filepath.Join("web", "tsconfig.json"), "--noEmit"); err != nil {
			fmt.Fprintln(stderr, "weft build: the app's TypeScript does not type-check; fix what vue-tsc says above and run weft build again.")
			return 1
		}
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
	goCmd := exec.Command("go", goBuildArgs(target)...)
	goCmd.Dir, goCmd.Env = appDir, goBuildEnv(os.Environ())
	goCmd.Stdout, goCmd.Stderr = stdout, stderr
	if err := goCmd.Run(); err != nil {
		fmt.Fprintln(stderr, "weft build: go build failed; fix what its output above says and run weft build again.")
		return 1
	}
	fmt.Fprintf(stdout, "Built %s\n", shown)
	return 0
}

// isFile reports whether p is there and is not a folder.
func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// goBuildArgs is the go command's arguments for the binary to ship: paths
// and symbols stripped, so it is small and the same wherever it is built.
func goBuildArgs(target string) []string {
	return []string{"build", "-trimpath", "-ldflags", "-s -w", "-o", target, "."}
}

// goBuildEnv is environ with CGO_ENABLED=0 added unless it sets CGO_ENABLED,
// so the binary has no C library to carry and runs wherever it is copied.
func goBuildEnv(environ []string) []string {
	for _, kv := range environ {
		if strings.HasPrefix(kv, "CGO_ENABLED=") {
			return environ
		}
	}
	return append(slices.Clone(environ), "CGO_ENABLED=0")
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
