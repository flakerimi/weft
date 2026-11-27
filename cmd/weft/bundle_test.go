package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"
)

func testBundle(files ...string) fstest.MapFS {
	b := fstest.MapFS{}
	for _, f := range files {
		b[f] = &fstest.MapFile{Data: []byte("x " + f)}
	}
	return b
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

func TestUnpackWritesTree(t *testing.T) {
	dir := t.TempDir()
	b := testBundle("weft/packages/ui/package.json", "weft/go/warp/serve.go", "weft/go/go.mod.bundle", "template/x")
	if err := unpack(b, dir, "0.0.6"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".weft/packages/ui/package.json", ".weft/go/warp/serve.go"} {
		if !exists(t, filepath.Join(dir, p)) {
			t.Errorf("%s is missing", p)
		}
	}
	// The bundle keeps the Go package's go.mod as go.mod.bundle, since
	// go:embed leaves out a folder with a go.mod; unpack names it back.
	if b, err := os.ReadFile(filepath.Join(dir, ".weft/go/go.mod")); err != nil || string(b) != "x weft/go/go.mod.bundle" {
		t.Errorf(".weft/go/go.mod = %q, %v", b, err)
	}
	if exists(t, filepath.Join(dir, ".weft/go/go.mod.bundle")) {
		t.Error(".weft/go/go.mod.bundle is left")
	}
	if exists(t, filepath.Join(dir, ".weft/template")) {
		t.Error("the template must not be unpacked")
	}
	if got := unpacked(dir); got != "0.0.6" {
		t.Errorf("unpacked = %q, want 0.0.6", got)
	}
}

func TestUnpackReplacesOlder(t *testing.T) {
	dir := t.TempDir()
	if err := unpack(testBundle("weft/a", "weft/b"), dir, "0.0.5"); err != nil {
		t.Fatal(err)
	}
	if err := unpack(testBundle("weft/a"), dir, "0.0.6"); err != nil {
		t.Fatal(err)
	}
	if exists(t, filepath.Join(dir, ".weft/b")) {
		t.Error("b survived the second unpack")
	}
	if !exists(t, filepath.Join(dir, ".weft/a")) {
		t.Error("a is missing")
	}
	if left := runTemps(t, dir); len(left) != 0 {
		t.Errorf("the run left %v", left)
	}
	if got := unpacked(dir); got != "0.0.6" {
		t.Errorf("unpacked = %q, want 0.0.6", got)
	}
}

// runTemps returns the folders an unpack makes for itself in dir
// (.weft-new-* and .weft-old-*) that are there.
func runTemps(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	for _, pattern := range []string{".weft-new-*", ".weft-old-*"} {
		m, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, m...)
	}
	return found
}

// writeFile writes text to dir/name, making its folders.
func writeFile(t *testing.T, dir, name, text string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Folders beside .weft that the run did not make are left alone, whatever
// their name: they may be the author's, such as an old .weft moved aside.
func TestUnpackLeavesOtherFolders(t *testing.T) {
	dir := t.TempDir()
	foreign := []string{".weft.old/keep.txt", ".weft.new/keep.txt", ".weft-old-mine/keep.txt", ".weft-new-mine/keep.txt"}
	for _, f := range foreign {
		writeFile(t, dir, f, "mine")
	}
	for _, v := range []string{"0.0.5", "0.0.6"} {
		if err := unpack(testBundle("weft/a"), dir, v); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range foreign {
		if b, err := os.ReadFile(filepath.Join(dir, f)); err != nil || string(b) != "mine" {
			t.Errorf("%s: %q, %v", f, b, err)
		}
	}
	if left := runTemps(t, dir); len(left) != 2 {
		t.Errorf("want only the two foreign temp-like folders, found %v", left)
	}
	if got := unpacked(dir); got != "0.0.6" {
		t.Errorf("unpacked = %q, want 0.0.6", got)
	}
}

// When the new tree cannot be renamed into place, the old .weft is put back
// and nothing the run made is left.
func TestUnpackFailedRenameRestores(t *testing.T) {
	dir := t.TempDir()
	if err := unpack(testBundle("weft/a"), dir, "0.0.5"); err != nil {
		t.Fatal(err)
	}
	was := renameDir
	t.Cleanup(func() { renameDir = was })
	renameDir = func(from, to string) error {
		if strings.HasPrefix(filepath.Base(from), ".weft-new-") {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: os.ErrPermission}
		}
		return was(from, to)
	}
	if err := unpack(testBundle("weft/b"), dir, "0.0.6"); err == nil {
		t.Fatal("unpack succeeded with a failing rename")
	}
	if got := unpacked(dir); got != "0.0.5" || !exists(t, filepath.Join(dir, ".weft/a")) || exists(t, filepath.Join(dir, ".weft/b")) {
		t.Errorf("the old .weft is not back (VERSION %q)", got)
	}
	if left := runTemps(t, dir); len(left) != 0 {
		t.Errorf("the failed run left %v", left)
	}
}

// A rename or a removal Windows refuses while another process has a file
// in the folder open (another weft command reading .weft/VERSION, a virus
// scanner) is tried again until it goes through, for up to busyWait; any
// other error, and one still busy after busyWait, is returned.
func TestRetryBusy(t *testing.T) {
	errBusy := errors.New("busy")
	was, wait := busy, busyWait
	busy, busyWait = func(err error) bool { return errors.Is(err, errBusy) }, 300*time.Millisecond
	t.Cleanup(func() { busy, busyWait = was, wait })

	tries := 0
	if err := retryBusy(func() error {
		if tries++; tries < 3 {
			return errBusy
		}
		return nil
	}); err != nil || tries != 3 {
		t.Errorf("busy twice: %v after %d tries, want nil after 3", err, tries)
	}
	tries = 0
	if err := retryBusy(func() error { tries++; return os.ErrPermission }); !errors.Is(err, os.ErrPermission) || tries != 1 {
		t.Errorf("another error: %v after %d tries, want it after 1", err, tries)
	}
	start := time.Now()
	if err := retryBusy(func() error { return errBusy }); !errors.Is(err, errBusy) || time.Since(start) < busyWait {
		t.Errorf("always busy: %v after %v, want it after %v", err, time.Since(start), busyWait)
	}
}

// brokenFS is m with the file bad unreadable.
type brokenFS struct {
	m   fstest.MapFS
	bad string
}

func (b brokenFS) Open(name string) (fs.File, error) {
	if name == b.bad {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return b.m.Open(name)
}

// A copy that fails leaves the old .weft as it was and nothing of the run.
func TestUnpackFailedCopyLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	if err := unpack(testBundle("weft/a"), dir, "0.0.5"); err != nil {
		t.Fatal(err)
	}
	bad := brokenFS{m: testBundle("weft/a", "weft/b"), bad: "weft/b"}
	if err := unpack(bad, dir, "0.0.6"); err == nil {
		t.Fatal("unpack succeeded with a bundle it cannot copy")
	}
	if got := unpacked(dir); got != "0.0.5" {
		t.Errorf("unpacked = %q, want 0.0.5", got)
	}
	if left := runTemps(t, dir); len(left) != 0 {
		t.Errorf("the failed run left %v", left)
	}
}

func TestUnpackRefusesForeignFolder(t *testing.T) {
	dir := t.TempDir()
	foreign := filepath.Join(dir, ".weft", "mine.txt")
	if err := os.MkdirAll(filepath.Dir(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := unpack(testBundle("weft/a"), dir, "0.0.6")
	if err == nil || !strings.Contains(err.Error(), ".weft") || !strings.HasSuffix(err.Error(), "; move it out of the app folder and run the command again") {
		t.Fatalf("err = %v, want one naming .weft and saying to move it out of the app folder", err)
	}
	if !exists(t, foreign) {
		t.Error("the foreign file is gone")
	}
}

func TestUnpackedMissing(t *testing.T) {
	if got := unpacked(t.TempDir()); got != "" {
		t.Errorf("unpacked = %q, want empty", got)
	}
}

func TestOpenBundleWithoutPackages(t *testing.T) {
	t.Setenv("WEFT_BUNDLE", "")
	_, err := openBundle()
	want := "this weft was built without its packages; install a release from https://weft.base.al/install, or set WEFT_BUNDLE to a folder made by scripts/bundle.sh"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func TestOpenBundleFromEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "weft", "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEFT_BUNDLE", dir)
	b, err := openBundle()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(b, "weft/packages"); err != nil {
		t.Fatal(err)
	}
}

func TestBundleScript(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the packages")
	}
	out := filepath.Join(t.TempDir(), "bundle")
	cmd := exec.Command("sh", "../../scripts/bundle.sh", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundle.sh: %v\n%s", err, b)
	}
	for _, p := range []string{"weft/packages/ui/dist/weft.css", "weft/go/go.mod.bundle", "weft/go/warp/serve.go", "weft/catalog/components.json", ".weft-bundle"} {
		if !exists(t, filepath.Join(out, p)) {
			t.Errorf("%s is missing", p)
		}
	}
	pkg, err := os.ReadFile(filepath.Join(out, "weft/packages/vite/package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(pkg), "devDependencies") {
		t.Error("vite's package.json keeps devDependencies")
	}
	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(p, "node_modules") || strings.HasSuffix(p, "_test.go") ||
			strings.HasSuffix(p, ".docs.md") || d.Name() == "test" || d.Name() == "examples" {
			t.Errorf("the bundle holds %s", p)
		}
		if d.Name() == "go.mod" {
			t.Errorf("the bundle holds %s, whose folder go:embed would leave out", p)
		}
		return nil
	})
	if exists(t, "../../templates/app") && !exists(t, filepath.Join(out, "template/package.json.tmpl")) {
		t.Error("template/package.json.tmpl is missing")
	}
	// A folder that is not a bundle is refused, and a bundle is reusable. A
	// release's bundle carries no Go package: its apps fetch the module from
	// GitHub at the release's tag.
	release := exec.Command("sh", "../../scripts/bundle.sh", out)
	release.Env = append(os.Environ(), "WEFT_VERSION=0.0.12")
	if b, err := release.CombinedOutput(); err != nil {
		t.Errorf("rebuilding a bundle: %v\n%s", err, b)
	}
	if exists(t, filepath.Join(out, "weft/go")) {
		t.Error("a release's bundle holds weft/go")
	}
	if !exists(t, filepath.Join(out, "weft/packages/ui/dist/weft.css")) || !exists(t, filepath.Join(out, "weft/catalog/components.json")) {
		t.Error("a release's bundle lacks its packages or its catalogue")
	}
	foreign := t.TempDir()
	os.WriteFile(filepath.Join(foreign, "mine"), []byte("x"), 0o644)
	if err := exec.Command("sh", "../../scripts/bundle.sh", foreign).Run(); err == nil {
		t.Error("bundle.sh wrote into a folder that is not a bundle")
	}
}

func TestBundleScriptRelativeOut(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the packages")
	}
	script, err := filepath.Abs("../../scripts/bundle.sh")
	if err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	cmd := exec.Command("sh", script, "rel/out")
	cmd.Dir = cwd
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundle.sh: %v\n%s", err, b)
	}
	if !exists(t, filepath.Join(cwd, "rel/out/weft/packages/ui/package.json")) {
		t.Error("a relative OUT was not resolved against the working folder")
	}
}

func TestBundleScriptMarksBeforeBuilding(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err == nil {
		// Run with a PATH that cannot find pnpm so the build step fails.
		if p, _ := exec.LookPath("pnpm"); strings.HasPrefix(p, "/usr/bin") || strings.HasPrefix(p, "/bin") {
			t.Skip("pnpm is on the minimal PATH")
		}
	}
	out := filepath.Join(t.TempDir(), "bundle")
	cmd := exec.Command("sh", "../../scripts/bundle.sh", out)
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin")
	if err := cmd.Run(); err == nil {
		t.Fatal("bundle.sh succeeded without pnpm")
	}
	if !exists(t, filepath.Join(out, ".weft-bundle")) {
		t.Error("a failed run left no .weft-bundle, so the next run would refuse the folder")
	}
}

// folderBundle writes files (path to content) into a new folder and points
// WEFT_BUNDLE at it.
func folderBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("WEFT_BUNDLE", dir)
	return dir
}

// A development weft is always 0.0.0-dev, so .weft/VERSION also records a
// hash of the bundle, and the bundle changing unpacks it again; the version
// stays the first line.
func TestRefreshUnpacksADevelopmentBundleThatChanged(t *testing.T) {
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "one", "weft/b": "two"})
	if did, err := refreshWeft(app, devVersion); err != nil || !did {
		t.Fatalf("first refresh = %v, %v; want it to unpack", did, err)
	}
	if got := unpacked(app); got != devVersion {
		t.Errorf("unpacked = %q, want %q", got, devVersion)
	}
	b, _ := os.ReadFile(filepath.Join(app, ".weft", "VERSION"))
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || lines[0] != devVersion || len(lines[1]) != 64 {
		t.Errorf("VERSION = %q, want the version, then a 64-digit hash", b)
	}
	os.WriteFile(filepath.Join(app, ".weft", "marker"), []byte("x"), 0o644)
	if did, err := refreshWeft(app, devVersion); err != nil || did {
		t.Fatalf("same bundle: refresh = %v, %v; want no unpack", did, err)
	}
	if !exists(t, filepath.Join(app, ".weft", "marker")) {
		t.Error("the same bundle was unpacked again")
	}
	// Changed bytes, then a file gone: each unpacks again, and the gone file
	// is gone from .weft.
	os.WriteFile(filepath.Join(bundle, "weft", "a"), []byte("one!"), 0o644)
	if did, err := refreshWeft(app, devVersion); err != nil || !did {
		t.Fatalf("changed bytes: refresh = %v, %v; want an unpack", did, err)
	}
	if got, _ := os.ReadFile(filepath.Join(app, ".weft", "a")); string(got) != "one!" || exists(t, filepath.Join(app, ".weft", "marker")) {
		t.Errorf(".weft/a = %q, or the marker is left", got)
	}
	if err := os.Rename(filepath.Join(bundle, "weft", "b"), filepath.Join(t.TempDir(), "b")); err != nil {
		t.Fatal(err)
	}
	if did, err := refreshWeft(app, devVersion); err != nil || !did {
		t.Fatalf("file gone: refresh = %v, %v; want an unpack", did, err)
	}
	if exists(t, filepath.Join(app, ".weft", "b")) {
		t.Error(".weft/b is still there after the bundle lost it")
	}
}

// Two bundles with the same bytes in other files, or one name moved, differ.
func TestBundleHashSeesNamesAndBytes(t *testing.T) {
	h := func(files ...string) string {
		t.Helper()
		got, err := bundleHash(testBundle(files...))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if h("weft/ab") == h("weft/a", "weft/b") || h("weft/a") == h("weft/b") || h("weft/a") != h("weft/a", "template/x") {
		t.Error("the hash does not tell the bundles apart by file names, or looks outside weft/")
	}
}

// A release is the version it says: other contents under the same version
// string are not unpacked again, and a .weft with no VERSION file stays.
func TestRefreshLeavesAReleaseAlone(t *testing.T) {
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "one"})
	if did, err := refreshWeft(app, "0.1.0"); err != nil || !did {
		t.Fatalf("first refresh = %v, %v", did, err)
	}
	if b, _ := os.ReadFile(filepath.Join(app, ".weft", "VERSION")); string(b) != "0.1.0\n" {
		t.Errorf("VERSION = %q, want the version alone", b)
	}
	os.WriteFile(filepath.Join(bundle, "weft", "a"), []byte("other"), 0o644)
	os.WriteFile(filepath.Join(bundle, "weft", "c"), []byte("new"), 0o644)
	if did, err := refreshWeft(app, "0.1.0"); err != nil || did {
		t.Fatalf("refresh = %v, %v; want no unpack", did, err)
	}
	if got, _ := os.ReadFile(filepath.Join(app, ".weft", "a")); string(got) != "one" {
		t.Errorf(".weft/a = %q, want the first bundle's", got)
	}
	foreign := t.TempDir()
	os.MkdirAll(filepath.Join(foreign, ".weft"), 0o755)
	if _, err := refreshWeft(foreign, devVersion); err == nil || !strings.Contains(err.Error(), "has no VERSION file") {
		t.Errorf("a .weft with no VERSION: err = %v", err)
	}
}

// A development weft with no bundle of its own keeps the .weft that is there.
func TestRefreshWithoutABundleKeepsTheTree(t *testing.T) {
	app := t.TempDir()
	folderBundle(t, map[string]string{"weft/a": "one"})
	if _, err := refreshWeft(app, devVersion); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEFT_BUNDLE", "")
	if did, err := refreshWeft(app, devVersion); err != nil || did {
		t.Errorf("refresh = %v, %v; want it to leave .weft as it is", did, err)
	}
}

// weft unpack makes .weft/ current and does nothing more: no pnpm install, no
// check of node or pnpm.
func TestUnpackCommand(t *testing.T) {
	app := t.TempDir()
	os.WriteFile(filepath.Join(app, "package.json"), []byte(`{"name":"a","weft":{"version":"0.0.0-dev"}}`), 0o644)
	bundle := folderBundle(t, map[string]string{"weft/go/go.mod.bundle": "module x\n", "weft/packages/a": "one"})
	t.Setenv("PATH", t.TempDir())
	t.Chdir(app)
	code, out, errs := runCmd("unpack")
	if code != 0 || out != "Unpacked .weft (Weft 0.0.0-dev)\n" || errs != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if !exists(t, ".weft/go/go.mod") || exists(t, "node_modules") {
		t.Error(".weft/go/go.mod is missing, or something was installed")
	}
	// No dist.go, so no web/dist to embed: none is made.
	if exists(t, "web/dist") {
		t.Error("weft unpack made web/dist in an app with no dist.go")
	}
	// A fresh clone of an app has no web/dist, which git ignores: unpack
	// puts back the empty marker, so go test ./... works before a build.
	os.WriteFile("dist.go", []byte("package main\n"), 0o644)
	if code, _, errs := runCmd("unpack"); code != 0 || errs != "" {
		t.Fatalf("with dist.go: exit %d, stderr %q", code, errs)
	}
	if b, err := os.ReadFile("web/dist/.weft-build"); err != nil || string(b) != unbuiltMarker {
		t.Errorf("web/dist/.weft-build = %q, %v; want the empty marker", b, err)
	}
	// An empty web/dist embeds nothing either: the marker goes in.
	os.Remove("web/dist/.weft-build")
	if code, _, errs := runCmd("unpack"); code != 0 || errs != "" {
		t.Fatalf("empty web/dist: exit %d, stderr %q", code, errs)
	}
	if b, err := os.ReadFile("web/dist/.weft-build"); err != nil || string(b) != unbuiltMarker {
		t.Errorf("an empty web/dist: .weft-build = %q, %v; want the empty marker", b, err)
	}
	// A web/dist that is there, a build's, is left as it is.
	os.WriteFile("web/dist/.weft-build", []byte("built\n"), 0o644)
	if code, out, _ := runCmd("unpack"); code != 0 || out != ".weft is current (Weft 0.0.0-dev)\n" {
		t.Errorf("again: exit %d, stdout %q", code, out)
	}
	if b, _ := os.ReadFile("web/dist/.weft-build"); string(b) != "built\n" {
		t.Errorf("weft unpack rewrote a web/dist that was there: %q", b)
	}
	if code, out, _ := runCmd("unpack"); code != 0 || out != ".weft is current (Weft 0.0.0-dev)\n" {
		t.Errorf("again: exit %d, stdout %q", code, out)
	}
	os.WriteFile(filepath.Join(bundle, "weft", "packages", "a"), []byte("two"), 0o644)
	if code, out, _ := runCmd("unpack"); code != 0 || out != "Unpacked .weft (Weft 0.0.0-dev)\n" {
		t.Errorf("changed bundle: exit %d, stdout %q", code, out)
	}
	os.WriteFile("package.json", []byte(`{"name":"a","weft":{"version":"9.9.9"}}`), 0o644)
	code, _, errs = runCmd("unpack")
	if want := "weft unpack: this app is on Weft 9.9.9 and this weft is 0.0.0-dev; run weft update --version 9.9.9 to match it.\n"; code != 1 || errs != want {
		t.Errorf("other version: exit %d, stderr %q", code, errs)
	}
	if code, _, errs := runCmd("unpack", "x"); code != 2 || errs != "weft unpack: \"x\" is not a flag; usage is weft unpack.\n" {
		t.Errorf("argument: exit %d, stderr %q", code, errs)
	}
}

// countHashes counts the bundle hashes taken for the rest of the test.
func countHashes(t *testing.T) *int {
	t.Helper()
	n := new(int)
	was := hashBundle
	hashBundle = func(b fs.FS) (string, error) { *n++; return was(b) }
	t.Cleanup(func() { hashBundle = was })
	return n
}

// A development refresh hashes the bundle once, whether .weft is missing or
// its hash is old.
func TestRefreshHashesOnce(t *testing.T) {
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "one"})
	n := countHashes(t)
	if did, err := refreshWeft(app, devVersion); err != nil || !did || *n != 1 {
		t.Fatalf("first refresh = %v, %v, %d hashes; want an unpack and one hash", did, err, *n)
	}
	os.WriteFile(filepath.Join(bundle, "weft", "a"), []byte("two"), 0o644)
	*n = 0
	if did, err := refreshWeft(app, devVersion); err != nil || !did || *n != 1 {
		t.Fatalf("changed: refresh = %v, %v, %d hashes; want an unpack and one hash", did, err, *n)
	}
}

// A bundle that cannot be read for its hash is named, with what to do.
func TestRefreshNamesAnUnreadableBundle(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads every file")
	}
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "one"})
	makeUnreadable(t, filepath.Join(bundle, "weft", "a"))
	_, err := refreshWeft(app, devVersion)
	if err == nil || !strings.HasPrefix(err.Error(), "cannot read the bundle in "+bundle+" (") || !strings.HasSuffix(err.Error(), "); make its files readable, or point WEFT_BUNDLE at a folder made by scripts/bundle.sh") {
		t.Errorf("refresh = %v", err)
	}
}

// A VERSION an older development build wrote, one line with no hash, is
// unpacked again; one written with CRLF line endings is read as written.
func TestRefreshReadsOlderAndCRLFVersions(t *testing.T) {
	app := t.TempDir()
	folderBundle(t, map[string]string{"weft/a": "one"})
	if _, err := refreshWeft(app, devVersion); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(app, ".weft", "VERSION"))
	hash := strings.Split(strings.TrimSpace(string(b)), "\n")[1]
	for record, want := range map[string]bool{
		devVersion + "\n":                   true,
		devVersion + "\r\n":                 true,
		devVersion + "\r\n" + hash + "\r\n": false,
		devVersion + "\n" + hash + "\n":     false,
		"0.0.1\r\n":                         true,
	} {
		os.WriteFile(filepath.Join(app, ".weft", "VERSION"), []byte(record), 0o644)
		if did, err := refreshWeft(app, devVersion); err != nil || did != want {
			t.Errorf("VERSION %q: refresh = %v, %v; want %v", record, did, err, want)
		}
	}
	os.WriteFile(filepath.Join(app, ".weft", "VERSION"), []byte("0.1.0\r\n"), 0o644)
	if did, err := refreshWeft(app, "0.1.0"); err != nil || did {
		t.Errorf("a release VERSION with CRLF: refresh = %v, %v; want none", did, err)
	}
}

// Two weft commands unpacking in one app at once both succeed, ten times
// over, and leave one whole .weft and nothing of theirs beside it.
func TestTwoUnpacksAtOnce(t *testing.T) {
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "zero", "weft/sub/b": "b"})
	for i := range 10 {
		os.WriteFile(filepath.Join(bundle, "weft", "a"), []byte(strconv.Itoa(i)), 0o644)
		errs := make(chan error, 2)
		for range 2 {
			go func() {
				_, err := refreshWeft(app, devVersion)
				errs <- err
			}()
		}
		for range 2 {
			if err := <-errs; err != nil {
				t.Fatalf("round %d: %v", i, err)
			}
		}
		if got, _ := os.ReadFile(filepath.Join(app, ".weft", "a")); string(got) != strconv.Itoa(i) || !exists(t, filepath.Join(app, ".weft", "sub", "b")) {
			t.Fatalf("round %d: .weft/a = %q", i, got)
		}
		entries, _ := os.ReadDir(app)
		if len(entries) != 1 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("round %d: the app folder holds %q", i, names)
		}
	}
}

// A lock no running weft command holds is taken over at once, whatever it
// says: one naming a process that is gone, one naming none, and one naming a
// running process that is not a weft command (this test's parent), as a
// pid reused after a weft command was killed would. One a running command
// holds is waited for, which is said once.
func TestLockLeftBehind(t *testing.T) {
	app := t.TempDir()
	lock := filepath.Join(app, ".weft.lock")
	var said bytes.Buffer
	was, wait := lockSays, lockWait
	lockSays.command, lockSays.to, lockWait = "routes", &said, 5*time.Second
	t.Cleanup(func() { lockSays, lockWait = was, wait })

	gone := exec.Command("true")
	if err := gone.Run(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		text string
	}{
		{"a gone process", fmt.Sprintf("%d\n", gone.Process.Pid)},
		{"no process", ""},
		{"garbage", "hello world"},
		{"a running process that is not weft", fmt.Sprintf("%d\n", os.Getppid())},
		{"this very process", fmt.Sprintf("%d\n", os.Getpid())},
	} {
		os.WriteFile(lock, []byte(c.text), 0o644)
		start := time.Now()
		unlock, err := lockWeft(app)
		if took := time.Since(start); err != nil || took > 500*time.Millisecond {
			t.Fatalf("%s: %v after %v", c.name, err, took)
		}
		if runtime.GOOS == "windows" {
			// The lock is the file, opened with no sharing: while it is held
			// nobody else opens it, so what it says cannot be read.
			if b, err := os.ReadFile(lock); err == nil {
				t.Errorf("%s: the lock could be opened while held, and holds %q", c.name, b)
			}
		} else if b, _ := os.ReadFile(lock); string(b) != fmt.Sprintf("%d\n", os.Getpid()) {
			t.Errorf("%s: the lock holds %q", c.name, b)
		}
		unlock()
		if exists(t, lock) {
			t.Errorf("%s: the lock is left after unlock", c.name)
		}
	}
	if said.Len() != 0 {
		t.Errorf("taking over said %q", said.String())
	}

	// A holder that is running: another lockWeft, which lets go after 300 ms.
	held, err := lockWeft(app)
	if err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(300*time.Millisecond, held)
	start := time.Now()
	unlock, err := lockWeft(app)
	if took := time.Since(start); err != nil || took < 250*time.Millisecond {
		t.Fatalf("a running holder: %v after %v, want a wait", err, took)
	}
	unlock()
	if want := "weft routes: another weft command is unpacking in this app; waiting for it.\n"; said.String() != want {
		t.Errorf("waiting said %q, want %q once", said.String(), want)
	}

	// One that does not let go is waited for lockWait, then said.
	lockWait = 300 * time.Millisecond
	held, err = lockWeft(app)
	if err != nil {
		t.Fatal(err)
	}
	defer held()
	if _, err := lockWeft(app); err == nil || !strings.HasPrefix(err.Error(), "another weft command has been unpacking .weft in "+app) {
		t.Errorf("a holder that does not let go: %v", err)
	}
}

// Eight weft commands started at once on a lock left behind by a killed
// one, fifty rounds over: every one exits 0, one unpacks, and .weft is whole,
// with nothing of theirs left beside it.
func TestEightCommandsOnALeftLock(t *testing.T) {
	if os.Getenv("WEFT_LOCK_HELPER") != "" {
		return
	}
	app := t.TempDir()
	bundle := folderBundle(t, map[string]string{"weft/a": "zero", "weft/sub/b": "b"})
	rounds := 50
	if testing.Short() {
		rounds = 10
	}
	for i := range rounds {
		os.WriteFile(filepath.Join(bundle, "weft", "a"), []byte(strconv.Itoa(i)), 0o644)
		os.WriteFile(filepath.Join(app, ".weft.lock"), []byte(fmt.Sprintf("%d\n", os.Getppid())), 0o644)
		var cmds []*exec.Cmd
		var outs []*bytes.Buffer
		for range 8 {
			cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$", "-test.short")
			cmd.Env = append(os.Environ(), "WEFT_LOCK_HELPER="+app)
			out := &bytes.Buffer{}
			cmd.Stdout, cmd.Stderr = out, out
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			cmds, outs = append(cmds, cmd), append(outs, out)
		}
		for j, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				t.Fatalf("round %d, command %d: %v\n%s", i, j, err, outs[j])
			}
		}
		if got, _ := os.ReadFile(filepath.Join(app, ".weft", "a")); string(got) != strconv.Itoa(i) || !exists(t, filepath.Join(app, ".weft", "sub", "b")) {
			t.Fatalf("round %d: .weft/a = %q", i, got)
		}
		if entries, _ := os.ReadDir(app); len(entries) != 1 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("round %d: the app folder holds %q", i, names)
		}
	}
}

// TestLockHelperProcess is one weft command of TestEightCommandsOnALeftLock:
// it refreshes the .weft of the app WEFT_LOCK_HELPER names.
func TestLockHelperProcess(t *testing.T) {
	app := os.Getenv("WEFT_LOCK_HELPER")
	if app == "" {
		t.Skip("run by TestEightCommandsOnALeftLock")
	}
	if _, err := refreshWeft(app, devVersion); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// A rename whose folder is gone because another command finished the same
// unpack meanwhile is no error when .weft is then current; any other failure
// of the unpack is a sentence, not the system's words alone.
func TestUnpackWhoseFolderWentMeanwhile(t *testing.T) {
	app := t.TempDir()
	folderBundle(t, map[string]string{"weft/a": "one"})
	if _, err := refreshWeft(app, devVersion); err != nil {
		t.Fatal(err)
	}
	bundle := folderBundle(t, map[string]string{"weft/a": "two"})
	b, err := openBundle()
	if err != nil {
		t.Fatal(err)
	}
	_ = bundle
	hash, err := bundleHash(b)
	if err != nil {
		t.Fatal(err)
	}
	old := renameDir
	t.Cleanup(func() { renameDir = old })
	renameDir = func(from, to string) error {
		// Another command finished: .weft is current, and what this one was
		// about to move is gone.
		os.WriteFile(filepath.Join(app, ".weft", "VERSION"), []byte(devVersion+"\n"+hash+"\n"), 0o644)
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: fs.ErrNotExist}
	}
	if _, err := refreshWeft(app, devVersion); err != nil {
		t.Errorf("another command finished the unpack: %v", err)
	}

	folderBundle(t, map[string]string{"weft/a": "three"})
	renameDir = func(from, to string) error {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: fs.ErrNotExist}
	}
	_, err = refreshWeft(app, devVersion)
	if err == nil || strings.HasPrefix(err.Error(), "rename ") || !strings.HasSuffix(err.Error(), "run the command again") {
		t.Errorf("a rename that failed: %v, want a sentence", err)
	}
}

// The folders an unpack that was killed left beside .weft (.weft-new-* and
// .weft-old-*, which only weft makes, under its lock) are removed by the
// next unpack, under the lock; a folder of another name is left alone.
func TestUnpackRemovesWhatAKilledOneLeft(t *testing.T) {
	app := t.TempDir()
	folderBundle(t, map[string]string{"weft/a": "one"})
	for _, d := range []string{".weft-new-123", ".weft-old-456/.weft", ".weft-mine", ".weft-new-x"} {
		os.MkdirAll(filepath.Join(app, d), 0o755)
		os.WriteFile(filepath.Join(app, d, "f"), []byte("x"), 0o644)
	}
	if _, err := refreshWeft(app, devVersion); err != nil {
		t.Fatal(err)
	}
	for d, want := range map[string]bool{".weft-new-123": false, ".weft-old-456": false, ".weft-mine": true, ".weft-new-x": true, ".weft": true} {
		if got := exists(t, filepath.Join(app, d)); got != want {
			t.Errorf("%s there: %v, want %v", d, got, want)
		}
	}
}

// On Windows a rename or a removal is refused while another program has a
// file open; the sentence says so there, and only there.
func TestHeldOpenOnlyOnWindows(t *testing.T) {
	for _, c := range []struct {
		err  error
		goos string
		want bool
	}{
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(32)}, "windows", true},
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(5)}, "windows", true},
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(32)}, "linux", false},
		{&os.PathError{Op: "mkdir", Path: "a", Err: syscall.Errno(28)}, "windows", false},
	} {
		if got := heldOpen(c.err, c.goos); got != c.want {
			t.Errorf("heldOpen(%v, %s) = %v, want %v", c.err, c.goos, got, c.want)
		}
	}
	if runtime.GOOS != "windows" {
		err := unpackFailed(t.TempDir(), &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(32)})
		if !strings.Contains(err.Error(), "make the app folder writable") {
			t.Errorf("off Windows the sentence is %q", err)
		}
	}
}
