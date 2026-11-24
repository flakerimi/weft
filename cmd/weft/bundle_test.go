package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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
	b := testBundle("weft/packages/ui/package.json", "weft/go/serve.go", "weft/go/go.mod.bundle", "template/x")
	if err := unpack(b, dir, "0.0.6"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".weft/packages/ui/package.json", ".weft/go/serve.go"} {
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
	for _, leftover := range []string{".weft.new", ".weft.old"} {
		if exists(t, filepath.Join(dir, leftover)) {
			t.Errorf("%s remains", leftover)
		}
	}
	if got := unpacked(dir); got != "0.0.6" {
		t.Errorf("unpacked = %q, want 0.0.6", got)
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
	if err == nil || !strings.Contains(err.Error(), ".weft") || !strings.Contains(err.Error(), "move it away") {
		t.Fatalf("err = %v, want one naming .weft and \"move it away\"", err)
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
	for _, p := range []string{"weft/packages/ui/dist/weft.css", "weft/go/go.mod.bundle", "weft/go/serve.go", "weft/catalog/components.json", ".weft-bundle"} {
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
	// A folder that is not a bundle is refused, and a bundle is reusable.
	if b, err := exec.Command("sh", "../../scripts/bundle.sh", out).CombinedOutput(); err != nil {
		t.Errorf("rebuilding a bundle: %v\n%s", err, b)
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
