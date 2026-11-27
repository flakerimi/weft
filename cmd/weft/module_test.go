package main

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// templateBundle is a bundle of the app template as the repository has it,
// with a weft/ folder that holds no Go package, as a release's bundle has
// none; WEFT_BUNDLE names it. The test is skipped without go, node and pnpm.
func templateBundle(t *testing.T) string {
	t.Helper()
	if err := prerequisites(); err != nil {
		t.Skipf("prerequisites: %v", err)
	}
	dir := folderBundle(t, map[string]string{"weft/packages/ui/package.json": "{}\n"})
	if err := copyFS(os.DirFS("../../templates/app"), filepath.Join(dir, "template")); err != nil {
		t.Fatal(err)
	}
	return dir
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
	templateBundle(t)
	setVersion(t, "0.0.12")
	t.Chdir(t.TempDir())
	if code, out, errs := runCmd("create", "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	want := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.12\n"
	if got := readFile(t, "myapp/go.mod"); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
	if exists(t, "myapp/.weft/go") {
		t.Error(".weft/go is unpacked from a bundle without it")
	}
}

// A development build keeps the module in .weft/go, so working on Weft
// itself needs no tag and no network.
func TestCreateDevReplacesTheModule(t *testing.T) {
	templateBundle(t)
	t.Chdir(t.TempDir())
	if code, out, errs := runCmd("create", "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	want := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.0\n\nreplace github.com/flakerimi/weft => ./.weft/go\n"
	if got := readFile(t, "myapp/go.mod"); got != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
}

// madeAt is an app made by a development build (its go.mod replaces the
// module with ./.weft/go, as every app before the module came from GitHub)
// and recorded at version, as an app weft 0.0.11 made is; the test runs in it.
func madeAt(t *testing.T, version string) {
	t.Helper()
	templateBundle(t)
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
	want := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.12\n"
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
		"GOPROXY=file://" + filepath.ToSlash(proxy),
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
