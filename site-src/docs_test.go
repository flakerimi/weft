package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The docs, the files for agents and the skills describe the Go side as it
// is: an app is a warp.App, built with weft build. These tests read the
// sources, so they need no build.

// filesUnder returns the path and text of every file under each of dirs.
func filesUnder(t *testing.T, dirs ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(p)] = string(b)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(out) == 0 {
		t.Fatalf("no file under %v", dirs)
	}
	return out
}

// The root package holds only Version now: Go code serves with warp.Serve,
// so no doc, agent file or skill names weft.Serve or weft.WithFiles, and the
// Warp guide exists.
func TestDocsNameWarpServe(t *testing.T) {
	for p, text := range filesUnder(t, "content/docs", "agents", "../skills") {
		for _, old := range []string{"weft.Serve", "weft.WithFiles"} {
			if strings.Contains(text, old) {
				t.Errorf("%s names %s; write warp.%s, as the warp package has it", p, old, strings.TrimPrefix(old, "weft."))
			}
		}
	}
	if _, err := os.Stat("content/docs/guides/warp.md"); err != nil {
		t.Errorf("content/docs/guides/warp.md is missing (%v); write the Warp guide there", err)
	}
}

// weftItself is the heading of the one section of install.md that builds
// inside the Weft repository: the section for working on Weft itself.
const weftItself = "## Working on Weft itself\n"

// An app is run with weft dev and built with weft build, so the
// getting-started pages, the guides and the skills never build by hand with
// the go command or build a site inside the repository. install.md's section
// on working on Weft itself is the one exception.
func TestDocsBuildWithWeft(t *testing.T) {
	install := "content/docs/getting-started/install.md"
	for p, text := range filesUnder(t, "content/docs/getting-started", "content/docs/guides", "../skills") {
		if p == install {
			before, after, ok := strings.Cut(text, weftItself)
			if !ok {
				t.Errorf("%s has no %q section; keep the work on Weft itself under that heading", p, strings.TrimSpace(weftItself))
			}
			text = before
			if _, next, more := strings.Cut(after, "\n## "); more {
				text += next
			}
		}
		for _, by := range []string{"go build", "go run ./", "pnpm --filter hello"} {
			if strings.Contains(text, by) {
				t.Errorf("%s says %q; an app is run with weft dev and built with weft build", p, by)
			}
		}
	}
}
