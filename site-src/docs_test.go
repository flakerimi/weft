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

// movingToWarp is the heading of the one section of cli.md that names
// weft.Serve: how to move an app made before warp.Serve.
const movingToWarp = "#### Moving an app to Warp\n"

// The root package holds only Version now: Go code serves with warp.Serve,
// so no doc, agent file or skill names weft.Serve or weft.WithFiles, and the
// Warp guide exists. cli.md's section on moving an app to Warp is the one
// exception.
func TestDocsNameWarpServe(t *testing.T) {
	cli := "content/docs/reference/cli.md"
	for p, text := range filesUnder(t, "content/docs", "agents", "../skills") {
		if p == cli {
			before, after, ok := strings.Cut(text, movingToWarp)
			if !ok {
				t.Errorf("%s has no %q section; keep how to move an app to Warp under that heading", p, strings.TrimSpace(movingToWarp))
			}
			// The section ends at the next heading of its level or higher.
			end := len(after)
			for _, next := range []string{"\n## ", "\n### ", "\n#### "} {
				if i := strings.Index(after, next); i >= 0 && i < end {
					end = i
				}
			}
			text = before + after[end:]
		}
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

// The Warp slice is documented: data, login and the types from Go each have
// a guide the docs index links, Coming next no longer lists as not built what
// the slice built, and no line of a doc, agent file or skill ties records to
// a client router without naming the run-time page that shows one.
func TestDocsCoverTheSlice(t *testing.T) {
	index, err := os.ReadFile("content/docs/index.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, guide := range []string{"data", "login", "types-from-go"} {
		if _, err := os.Stat("content/docs/guides/" + guide + ".md"); err != nil {
			t.Errorf("content/docs/guides/%s.md is missing (%v); write that guide", guide, err)
		}
		if !strings.Contains(string(index), "(guides/"+guide+".md)") {
			t.Errorf("content/docs/index.md does not link guides/%s.md; list it with the guides", guide)
		}
	}
	coming, err := os.ReadFile("content/docs/coming-next.md")
	if err != nil {
		t.Fatal(err)
	}
	// What Coming next says is not built: its Commands table, and the lines
	// of each section that are not a pointer to what works today.
	for _, line := range strings.Split(string(coming), "\n") {
		for _, built := range []string{"`weft gen`", "`weft routes`", "`warp/data`", "`warp/describe`", "`auth.Can`", "`warp.Resource`"} {
			if strings.Contains(line, built) && !strings.Contains(line, "work today") && !strings.Contains(line, "works today") {
				t.Errorf("content/docs/coming-next.md lists %s as not built: %q; it is built, so say so or take it out", built, line)
			}
		}
	}
	for p, text := range filesUnder(t, "content/docs", "agents", "../skills") {
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "router") && strings.Contains(line, "record") && !strings.Contains(line, "run-time page") {
				t.Errorf("%s says a client router is needed for a record: %q; a run-time page shows one", p, line)
			}
		}
	}
}
