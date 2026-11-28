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
	// What Coming next lists as not built: the rows of its tables and the
	// items of its lists. Its prose may name what works today.
	for _, line := range strings.Split(string(coming), "\n") {
		if !strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "- ") {
			continue
		}
		for _, built := range []string{"`weft gen`", "`weft routes`", "`warp/data`", "`warp/describe`", "`warp/auth`", "`auth.Module`", "`auth.Can`", "`auth.Per`", "`data.Resource`", "`warp.Resource`", "`data.Store`", "`Store`", "`runtime`"} {
			if strings.Contains(line, built) {
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

// Windows is a supported platform: install.md, cli.md and index-md.ts name
// the PowerShell installer (irm https://weft.base.al/install.ps1 | iex), and
// coming-next.md no longer says Windows is not yet supported.
func TestDocsCoverWindows(t *testing.T) {
	for _, f := range []string{
		"content/docs/getting-started/install.md",
		"content/docs/reference/cli.md",
		"agents/index-md.ts",
	} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "install.ps1") {
			t.Errorf("%s does not name install.ps1", f)
		}
	}
	coming, err := os.ReadFile("content/docs/coming-next.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(coming), "Windows is not yet a supported platform") {
		t.Error("content/docs/coming-next.md still says Windows is not yet a supported platform")
	}
}

// The accounts release is documented: accounts, mail and middleware each
// have a guide the docs index links; Coming next no longer lists as not built
// what 0.2.0 built; and every setting the template's .env.sample names is in
// a guide.
func TestDocsCoverAccounts(t *testing.T) {
	index, err := os.ReadFile("content/docs/index.md")
	if err != nil {
		t.Fatal(err)
	}
	guides := ""
	for _, guide := range []string{"login", "accounts", "mail", "middleware"} {
		b, err := os.ReadFile("content/docs/guides/" + guide + ".md")
		if err != nil {
			t.Errorf("content/docs/guides/%s.md is missing (%v); write that guide", guide, err)
		}
		guides += string(b)
		if !strings.Contains(string(index), "(guides/"+guide+".md)") {
			t.Errorf("content/docs/index.md does not link guides/%s.md; list it with the guides", guide)
		}
	}
	warp, _ := os.ReadFile("content/docs/guides/warp.md")
	guides += string(warp)
	coming, err := os.ReadFile("content/docs/coming-next.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(coming), "\n") {
		if !strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "- ") {
			continue
		}
		for _, built := range []string{"egistration", "nvitation", "eset", "roles", "Roles", "session table", "sessions in a table", "`useSession", "`perm`", "route guard", "rate limit", "Rate limit", "date field", "Date field", "`mail`", "`middleware`", "API key"} {
			if strings.Contains(line, built) {
				t.Errorf("content/docs/coming-next.md lists %q as not built: %q; 0.2.0 built it, so say so or take it out", built, line)
			}
		}
	}
	sample, err := os.ReadFile("../templates/app/env.sample.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(sample), "\n") {
		key, _, ok := strings.Cut(strings.TrimPrefix(line, "# "), "=")
		if !ok || key == "" || strings.ContainsAny(key, " .,") {
			continue
		}
		all := guides
		for _, other := range []string{"data", "routes", "go-serving"} {
			b, _ := os.ReadFile("content/docs/guides/" + other + ".md")
			all += string(b)
		}
		if !strings.Contains(all, "`"+key+"`") && !strings.Contains(all, key+"=") {
			t.Errorf("no guide names %s, which the template's .env.sample sets; say what it does in its guide", key)
		}
	}
}

// The docs' front page sends a reader to every doc, a guide or a reference
// moved or added included.
func TestDocsIndexListsEveryDoc(t *testing.T) {
	index, err := os.ReadFile("content/docs/index.md")
	if err != nil {
		t.Fatal(err)
	}
	for path := range filesUnder(t, "content/docs") {
		rel := strings.TrimPrefix(path, "content/docs/")
		if rel == "index.md" || !strings.HasSuffix(rel, ".md") {
			continue
		}
		if !strings.Contains(string(index), "("+rel+")") && !strings.Contains(string(index), "("+rel+"#") {
			t.Errorf("content/docs/index.md does not link %s", rel)
		}
	}
	if strings.Contains(string(index), "their input, validation and errors") {
		t.Error("content/docs/index.md says Typed routes covers validation and errors, which have a guide of their own")
	}
}

// Agents and apps are sent to the page that has weft scaffold's types, not
// to the CLI page's short section.
func TestScaffoldLinksGoToItsPage(t *testing.T) {
	for _, f := range []string{"../skills/weft-go/SKILL.md", "agents/index-md.ts", "../templates/app/AGENTS.md.tmpl", "../cmd/weft/scaffold.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "reference/cli#weft-scaffold") || !strings.Contains(string(b), "docs/reference/scaffold") {
			t.Errorf("%s does not send its reader to /docs/reference/scaffold", f)
		}
	}
}
