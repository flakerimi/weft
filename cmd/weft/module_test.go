package main

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// templateBundle is a bundle of the app template as the repository has it;
// WEFT_BUNDLE names it. Its weft/ folder holds the Go package (a go.mod.bundle)
// only when withGo is set, as a development bundle does and a release's does
// not. The test is skipped without go, node and pnpm.
func templateBundle(t *testing.T, withGo bool) string {
	t.Helper()
	if err := prerequisites(); err != nil {
		t.Skipf("prerequisites: %v", err)
	}
	files := map[string]string{"weft/packages/ui/package.json": "{}\n"}
	if withGo {
		files["weft/go/go.mod.bundle"] = "module github.com/flakerimi/weft\n\ngo 1.26\n"
	}
	dir := folderBundle(t, files)
	if err := copyFS(os.DirFS("../../templates/app"), filepath.Join(dir, "template")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// templateGoMod is the go.mod weft create writes for myapp: Warp's module at
// version, beside the SQLite driver and the modules it needs, and with the
// replace into .weft/go when replaced. Since the template took the shop's
// frame, the driver is required too, and with it go 1.26.0, which its
// modules need.
func templateGoMod(version string, replaced bool) string {
	mod := "module myapp\n\ngo 1.26.0\n\nrequire (\n\tgithub.com/flakerimi/weft " + version + "\n\tmodernc.org/sqlite v1.63.0\n)\n\nrequire (\n" +
		"\tgithub.com/dustin/go-humanize v1.1.0 // indirect\n\tgithub.com/google/uuid v1.6.0 // indirect\n\tgithub.com/mattn/go-isatty v0.0.24 // indirect\n" +
		"\tgithub.com/ncruces/go-strftime v1.1.0 // indirect\n\tgithub.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect\n" +
		"\tgolang.org/x/sys v0.49.0 // indirect\n\tmodernc.org/libc v1.80.1 // indirect\n\tmodernc.org/mathutil v1.7.1 // indirect\n\tmodernc.org/memory v1.12.1 // indirect\n)\n"
	if replaced {
		mod += "\nreplace github.com/flakerimi/weft => ./.weft/go\n"
	}
	return mod
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A release's app requires Warp's module at the release, from GitHub through
// the module proxy, with no replace; the bundle carries no Go package.
func TestCreateRequiresTheReleaseModule(t *testing.T) {
	templateBundle(t, false)
	setVersion(t, "0.0.12")
	t.Chdir(t.TempDir())
	if code, out, errs := runCmd("create", "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	want := templateGoMod("v0.0.12", false)
	if got := readFile(t, "myapp/go.mod"); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
	if exists(t, "myapp/.weft/go") {
		t.Error(".weft/go is unpacked from a bundle without it")
	}
	// The app's own guide says so, for the agent working in it.
	if guide := readFile(t, "myapp/AGENTS.md"); !strings.Contains(guide, "\nGo gets Warp from GitHub: `go.mod` requires `github.com/flakerimi/weft` at the tag of this app's Weft release, and `go.sum` holds its checksums; commit both, and move the version with `weft upgrade`, not by hand.\n") {
		t.Errorf("AGENTS.md does not say how Go gets Warp:\n%s", guide)
	}
}

// A development build keeps the module in .weft/go, so working on Weft
// itself needs no tag and no network.
func TestCreateDevReplacesTheModule(t *testing.T) {
	templateBundle(t, true)
	t.Chdir(t.TempDir())
	if code, out, errs := runCmd("create", "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	want := templateGoMod("v0.0.0", true)
	if got := readFile(t, "myapp/go.mod"); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
	if guide := readFile(t, "myapp/AGENTS.md"); !strings.Contains(guide, "\nGo finds Warp in `.weft/go`, which `go.mod` replaces `github.com/flakerimi/weft` with, since a development build of `weft` made this app.\n") {
		t.Errorf("AGENTS.md does not say how Go gets Warp:\n%s", guide)
	}
}

// A development weft gives an app the module in .weft/go, so a bundle staged
// for a release, which has none, is refused before anything is written.
func TestCreateDevRefusesAReleaseBundle(t *testing.T) {
	dir := templateBundle(t, false)
	t.Chdir(t.TempDir())
	code, _, errs := runCmd("create", "myapp", "--no-install")
	want := "weft create: the bundle in " + dir + " was staged for a release, so it has no Go package (weft/go) for the replace a development weft writes; stage it with scripts/bundle.sh without WEFT_VERSION.\n"
	if code != 1 || errs != want {
		t.Fatalf("exit %d, stderr %q, want %q", code, errs, want)
	}
	if exists(t, "myapp") {
		t.Error("the refused create wrote myapp")
	}
}

// okPnpm puts a pnpm on PATH that says it is 10.0.0 and does nothing else.
func okPnpm(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in pnpm is a shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm"), []byte("#!/bin/sh\n[ \"$1\" != --version ] || echo 10.0.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// When Go cannot fetch the module, weft create says what finishes the app:
// weft dev or weft build in it, which fetch it again. Running weft create
// again would be refused, since the folder is there.
func TestCreateSaysHowToFinishAfterAFailedFetch(t *testing.T) {
	templateBundle(t, false)
	okPnpm(t)
	setVersion(t, "0.0.12")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GOFLAGS", "-modcacherw")
	t.Chdir(t.TempDir())
	code, _, errs := runCmd("create", "myapp")
	want := "weft create: Go could not fetch github.com/flakerimi/weft v0.0.12 for myapp (see its output above; the module proxy must be reachable); once it can, run cd myapp && weft dev (or weft build), which fetches it again.\n"
	if code != 1 || !strings.HasSuffix(errs, want) {
		t.Fatalf("exit %d, stderr %q, want it to end %q", code, errs, want)
	}
	if !exists(t, "myapp/go.mod") {
		t.Error("the app is not left for weft dev to finish")
	}
}

// pnpm install runs when node_modules is missing, or older than .weft: as
// after weft upgrade --no-install, which unpacked a new .weft over the old
// version's node_modules.
func TestStaleInstall(t *testing.T) {
	app := t.TempDir()
	os.MkdirAll(filepath.Join(app, ".weft"), 0o755)
	os.WriteFile(filepath.Join(app, ".weft", "VERSION"), []byte("0.0.12\n"), 0o644)
	if !staleInstall(app) {
		t.Error("no node_modules is not stale")
	}
	os.MkdirAll(filepath.Join(app, "node_modules"), 0o755)
	if !staleInstall(app) {
		t.Error("node_modules without pnpm's .modules.yaml is not stale")
	}
	modules := filepath.Join(app, "node_modules", ".modules.yaml")
	os.WriteFile(modules, []byte("x\n"), 0o644)
	now := time.Now()
	os.Chtimes(filepath.Join(app, ".weft", "VERSION"), now, now)
	os.Chtimes(modules, now.Add(-time.Minute), now.Add(-time.Minute))
	if !staleInstall(app) {
		t.Error("node_modules installed before .weft was unpacked is not stale")
	}
	os.Chtimes(modules, now.Add(time.Minute), now.Add(time.Minute))
	if staleInstall(app) {
		t.Error("node_modules installed after .weft was unpacked is stale")
	}
}

// madeAt is an app made by a development build (its go.mod replaces the
// module with ./.weft/go, as every app before the module came from GitHub)
// and recorded at version, as an app weft 0.0.11 made is; the test runs in it.
func madeAt(t *testing.T, version string) {
	t.Helper()
	templateBundle(t, true)
	t.Chdir(t.TempDir())
	if code, out, errs := runCmd("create", "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if err := writeAppVersion("myapp", version); err != nil {
		t.Fatal(err)
	}
	t.Chdir("myapp")
}

// weft upgrade moves the require to the release and drops the replace into
// .weft/go, which a release's bundle no longer fills.
func TestUpgradeRequiresTheReleaseModule(t *testing.T) {
	madeAt(t, "0.0.11")
	setVersion(t, "0.0.12")
	code, out, errs := runCmd("upgrade", "--no-install")
	if code != 0 || out != "Moved myapp from Weft 0.0.11 to 0.0.12\n" || errs != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	want := templateGoMod("v0.0.12", false)
	if got := readFile(t, "go.mod"); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
	// From a release to the next, only the require moves.
	setVersion(t, "0.0.13")
	if code, out, errs := runCmd("upgrade", "--no-install"); code != 0 || errs != "" {
		t.Fatalf("second upgrade: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if got, want := readFile(t, "go.mod"), strings.Replace(want, "v0.0.12", "v0.0.13", 1); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
}

// Only the replace into .weft/go is dropped, however it is written; one the
// author wrote to another folder is kept, and named.
func TestUpgradeKeepsTheAuthorsReplace(t *testing.T) {
	for _, c := range []struct{ name, replace, kept, say string }{
		{"into .weft/go with a version", "replace github.com/flakerimi/weft v0.0.0 => ./.weft/go/\n", "", ""},
		{"to the author's copy", "replace github.com/flakerimi/weft => ../weft\n", "replace github.com/flakerimi/weft => ../weft\n",
			"weft upgrade: go.mod replaces github.com/flakerimi/weft with ../weft, so the app builds against that, not Weft 0.0.12 from GitHub; remove that replace line from go.mod to build against the release.\n"},
		{"to another module", "replace github.com/flakerimi/weft => example.com/fork v1.2.3\n", "replace github.com/flakerimi/weft => example.com/fork v1.2.3\n",
			"weft upgrade: go.mod replaces github.com/flakerimi/weft with example.com/fork v1.2.3, so the app builds against that, not Weft 0.0.12 from GitHub; remove that replace line from go.mod to build against the release.\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			madeAt(t, "0.0.11")
			mod := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.0\n\n" + c.replace
			if err := os.WriteFile("go.mod", []byte(mod), 0o644); err != nil {
				t.Fatal(err)
			}
			setVersion(t, "0.0.12")
			code, _, errs := runCmd("upgrade", "--no-install")
			if code != 0 || errs != c.say {
				t.Fatalf("exit %d, stderr %q, want %q", code, errs, c.say)
			}
			want := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.12\n"
			if c.kept != "" {
				want += "\n" + c.kept
			}
			if got := readFile(t, "go.mod"); got != want {
				t.Errorf("go.mod = %q, want %q", got, want)
			}
		})
	}
}

// moduleZip is the zip the module proxy serves for module at version: every
// file named, under module@version/.
func moduleZip(t *testing.T, module, version string, files map[string][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for name, data := range files {
		w, err := zw.Create(module + "@" + version + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// moduleProxy serves github.com/flakerimi/weft at version, made of files,
// from a folder, as the module proxy would, and points Go at it with an
// empty module cache of the test's own and no checksum database for the
// module; it returns those variables.
func moduleProxy(t *testing.T, version string, files map[string][]byte) []string {
	t.Helper()
	proxy := t.TempDir()
	dir := filepath.Join(proxy, "github.com", "flakerimi", "weft", "@v")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"list":            []byte(version + "\n"),
		version + ".info": []byte(`{"Version":"` + version + `","Time":"2026-11-26T00:00:00Z"}`),
		version + ".mod":  files["go.mod"],
		version + ".zip":  moduleZip(t, weftModule, version, files),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return []string{
		"GOPROXY=" + fileProxyURL(filepath.ToSlash(proxy)),
		"GONOSUMDB=" + weftModule,
		"GOMODCACHE=" + t.TempDir(),
		// The cache's files are written read-only otherwise, and the test's
		// folder could not be removed.
		"GOFLAGS=-modcacherw",
	}
}

// repoModule is the files of the module at the repository's root as a tag
// would have them: every file git tracks or would add, outside a folder
// that holds a go.mod of its own (another module) and a vendor folder, as
// Go makes the module's zip from a tag.
func repoModule(t *testing.T) map[string][]byte {
	t.Helper()
	out, err := exec.Command("git", "-C", "../..", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}
	names := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	nested := map[string]bool{}
	for _, n := range names {
		if path.Base(n) == "go.mod" && n != "go.mod" {
			nested[path.Dir(n)] = true
		}
	}
	files := map[string][]byte{}
	var size int64
names:
	for _, n := range names {
		for d := path.Dir(n); d != "."; d = path.Dir(d) {
			if nested[d] || path.Base(d) == "vendor" {
				continue names
			}
		}
		data, err := os.ReadFile(filepath.Join("../..", filepath.FromSlash(n)))
		if err != nil {
			continue // tracked, but deleted from the working tree
		}
		files[n] = data
		size += int64(len(data))
	}
	t.Logf("the module at the root: %d files, %d B before compression", len(files), size)
	return files
}

func setEnv(t *testing.T, env []string) {
	t.Helper()
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
}

// tinyModule is a module github.com/flakerimi/weft with just enough Go for
// an app to build against.
var tinyModule = map[string][]byte{
	"go.mod":       []byte("module github.com/flakerimi/weft\n\ngo 1.26\n"),
	"version.go":   []byte("package weft\n\nvar Version = \"0.0.0-dev\"\n"),
	"warp/warp.go": []byte("package warp\n\nconst Name = \"warp\"\n"),
}

// An app that requires the module from the proxy gets its checksums in
// go.sum through go mod download, once: with them there, Go is not run.
func TestFetchModule(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go")
	}
	setEnv(t, moduleProxy(t, "v0.0.99", tinyModule))
	app := t.TempDir()
	mod := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.99\n"
	os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod), 0o644)
	os.WriteFile(filepath.Join(app, "main.go"), []byte("package main\n\nimport \"github.com/flakerimi/weft/warp\"\n\nfunc main() { println(warp.Name) }\n"), 0o644)
	var out, errb bytes.Buffer
	if err := fetchModule(app, &out, &errb); err != nil {
		t.Fatalf("fetchModule: %v\n%s", err, errb.String())
	}
	sum := readFile(t, filepath.Join(app, "go.sum"))
	for _, want := range []string{"github.com/flakerimi/weft v0.0.99 h1:", "github.com/flakerimi/weft v0.0.99/go.mod h1:"} {
		if !strings.Contains(sum, want) {
			t.Errorf("go.sum lacks %q: %q", want, sum)
		}
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "app"), ".")
	build.Dir = app
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, b)
	}
	// With the checksums there, nothing is fetched: with no proxy and an
	// empty cache, a download would fail.
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOMODCACHE", t.TempDir())
	if err := fetchModule(app, &out, &errb); err != nil {
		t.Errorf("fetchModule with go.sum complete: %v", err)
	}
	// Nor for an app that replaces the module with a folder.
	os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod+"\nreplace github.com/flakerimi/weft => ./.weft/go\n"), 0o644)
	os.WriteFile(filepath.Join(app, "go.sum"), nil, 0o644)
	if err := fetchModule(app, &out, &errb); err != nil {
		t.Errorf("fetchModule with a replace: %v", err)
	}
	// A download that fails says so in one sentence, naming the app.
	os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod), 0o644)
	err := fetchModule(app, &out, &errb)
	want := "go mod download github.com/flakerimi/weft did not finish in " + app + ", so the app cannot build against Weft 0.0.99; fix what its output above says (Go fetches the module from the module proxy, so the network must be up) and run the command again"
	if err == nil || err.Error() != want {
		t.Errorf("fetchModule with no proxy = %v, want %q", err, want)
	}
}

// The module an app downloads at a tag leaves out the top folders that hold
// no Go an app needs: each has a go.mod of its own, which makes it another
// module. None of them holds Go, so go vet ./... and go test ./... at the
// root still reach every Go package but those of the two modules that are
// apps of their own, dbtest and examples/shop.
func TestModuleLeavesOutFoldersWithoutGo(t *testing.T) {
	files := repoModule(t)
	var size int
	for name, data := range files {
		size += len(data)
		for _, dir := range []string{"docs/", "packages/", "catalog/"} {
			if strings.HasPrefix(name, dir) {
				t.Errorf("the module holds %s; %sgo.mod should leave the folder out", name, dir)
			}
		}
	}
	if size > 6<<20 {
		t.Errorf("the module is %d B before compression, over 6 MB", size)
	}
	out, err := exec.Command("git", "-C", "../..", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}
	names := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	modules := map[string]bool{}
	for _, n := range names {
		if path.Base(n) == "go.mod" && n != "go.mod" {
			modules[path.Dir(n)] = true
		}
	}
	withGo := map[string]bool{}
	for _, n := range names {
		if !strings.HasSuffix(n, ".go") {
			continue
		}
		for d := path.Dir(n); d != "."; d = path.Dir(d) {
			if modules[d] {
				withGo[d] = true
			}
		}
	}
	if len(withGo) != 2 || !withGo["dbtest"] || !withGo["examples/shop"] {
		t.Errorf("the nested modules that hold Go are %v, want only dbtest and examples/shop", withGo)
	}
}

// fileProxyURL is the file:// URL of a proxy folder given with slashes. A
// Windows path (C:/…) needs a slash before its drive, or Go reads the drive
// as the URL's host and refuses it ("invalid file:// proxy URL with non-path
// elements"), as it did on the Windows runner.
func fileProxyURL(dir string) string {
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	return "file://" + dir
}

func TestFileProxyURL(t *testing.T) {
	for in, want := range map[string]string{
		"/tmp/proxy":                        "file:///tmp/proxy",
		"C:/Users/RUNNER~1/AppData/Local/x": "file:///C:/Users/RUNNER~1/AppData/Local/x",
	} {
		if got := fileProxyURL(in); got != want {
			t.Errorf("fileProxyURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// An app made by 0.0.17 or before has no web/dist: weft upgrade puts the
// empty marker in, so go test ./... builds right after it.
func TestUpgradeWritesTheMarker(t *testing.T) {
	madeAt(t, "0.0.11")
	if err := os.RemoveAll(filepath.Join("web", "dist")); err != nil {
		t.Fatal(err)
	}
	setVersion(t, "0.0.12")
	if code, out, errs := runCmd("upgrade", "--no-install"); code != 0 {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if got := readFile(t, filepath.Join("web", "dist", ".weft-build")); got != unbuiltMarker {
		t.Errorf("web/dist/.weft-build = %q, want the empty marker", got)
	}
}
