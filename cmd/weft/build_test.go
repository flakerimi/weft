package main

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft"
)

var (
	installedOnce sync.Once
	installedDir  string
	installedErr  string
)

// installedApp returns an app named myapp that weft create made and installed,
// once for this test binary; tests copy it with copyApp rather than change it.
func installedApp(t *testing.T) string {
	t.Helper()
	installedOnce.Do(func() {
		dir := filepath.Join(filepath.Dir(stagedBundle), "installed")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			installedErr = err.Error()
			return
		}
		wd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(wd)
		var out, errb bytes.Buffer
		if code := run([]string{"create", "myapp"}, &out, &errb); code != 0 {
			installedErr = out.String() + errb.String()
			return
		}
		installedDir = filepath.Join(dir, "myapp")
	})
	if installedDir == "" {
		t.Fatalf("weft create myapp failed: %s", installedErr)
	}
	return installedDir
}

// copyApp copies the installed app to dir/myapp, keeping the symlinks pnpm
// made, and returns the copy's absolute path.
func copyApp(t *testing.T, dir string) string {
	t.Helper()
	src := installedApp(t)
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cp", "-a", src, dir).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v: %s", err, out)
	}
	return filepath.Join(dir, "myapp")
}

// build runs weft build with args in the current folder.
func build(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(append([]string{"build"}, args...), &out, &errb)
	return code, out.String(), errb.String()
}

// serveBinary starts bin on a free port and returns its base URL; the process
// is stopped when the test ends.
func serveBinary(t *testing.T, bin string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(port))
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); cmd.Wait() })
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if res, err := http.Get(url + "/api/hello"); err == nil {
			res.Body.Close()
			return url
		}
	}
	t.Fatalf("%s did not answer within 10 s: %s", bin, logs.String())
	return ""
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(body)
}

var bootRe = regexp.MustCompile(`<script type="module" src="(/assets/boot-[^"]+\.js)"`)

// assertServes checks the built binary answers the page, the script that
// starts its island, the API and a 404.
func assertServes(t *testing.T, bin string) {
	t.Helper()
	url := serveBinary(t, bin)
	code, body := get(t, url+"/")
	if code != 200 || !strings.Contains(body, ">Welcome</h1>") || !strings.Contains(body, `<weft-island data-id="Hello"`) {
		t.Errorf("GET / = %d %q", code, body)
	}
	if m := bootRe.FindStringSubmatch(body); m == nil {
		t.Errorf("GET / has no island script: %q", body)
	} else if code, _ := get(t, url+m[1]); code != 200 {
		t.Errorf("GET %s = %d", m[1], code)
	}
	if code, body := get(t, url+"/api/hello"); code != 200 || strings.TrimSpace(body) != `{"message":"Hello from Go"}` {
		t.Errorf("GET /api/hello = %d %q", code, body)
	}
	if code, body := get(t, url+"/nope"); code != 404 || !strings.Contains(body, "Not found") {
		t.Errorf("GET /nope = %d %q", code, body)
	}
	if code, body := get(t, url+"/api/nope"); code != 404 || !strings.Contains(body, `"code":"not_found"`) {
		t.Errorf("GET /api/nope = %d %q", code, body)
	}
	// A PORT that is no number stops the binary, with a sentence.
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "PORT=abc")
	out, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Errorf("PORT=abc: %v, want exit 1", err)
	}
	if want := `PORT is "abc"; set it to a number from 1 to 65535 in .env or the environment.`; !strings.Contains(string(out), want) {
		t.Errorf("PORT=abc printed %q, want %q", out, want)
	}
}

func TestGoBuildArgs(t *testing.T) {
	want := []string{"build", "-trimpath", "-ldflags", "-s -w", "-o", "x", "."}
	if got := goBuildArgs("x"); !slices.Equal(got, want) {
		t.Errorf("goBuildArgs = %q, want %q", got, want)
	}
}

func TestGoBuildEnv(t *testing.T) {
	in := []string{"PATH=/bin", "GOOS=linux"}
	want := []string{"PATH=/bin", "GOOS=linux", "CGO_ENABLED=0"}
	if got := goBuildEnv(in); !slices.Equal(got, want) {
		t.Errorf("goBuildEnv(%q) = %q, want %q", in, got, want)
	}
	if len(in) != 2 {
		t.Error("goBuildEnv changed its argument")
	}
	set := []string{"PATH=/bin", "CGO_ENABLED=1"}
	if got := goBuildEnv(set); !slices.Equal(got, set) {
		t.Errorf("goBuildEnv(%q) = %q, want it unchanged", set, got)
	}
}

func TestBuildMakesBinary(t *testing.T) {
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	code, out, errs := build(t)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if !strings.HasSuffix(out, "Built myapp\n") {
		t.Errorf("stdout ends %q", out[max(0, len(out)-80):])
	}
	if !strings.Contains(out, "weft: built 2 pages into web/dist") {
		t.Errorf("the page build's output was not passed through: %q", out)
	}
	assertServes(t, filepath.Join(app, "myapp"))

	// A build for another system is one command: GOOS and GOARCH pass through.
	t.Setenv("GOOS", "linux")
	t.Setenv("GOARCH", "arm64")
	if code, out, errs := build(t, "-o", "cross"); code != 0 {
		t.Fatalf("cross build: exit %d: %s%s", code, out, errs)
	}
	head, err := os.ReadFile(filepath.Join(app, "cross"))
	if err != nil || len(head) < 4 || string(head[:4]) != "\x7fELF" {
		t.Errorf("GOOS=linux GOARCH=arm64 weft build -o cross wrote no ELF file: %v", err)
	}
	t.Setenv("GOOS", runtime.GOOS)
	t.Setenv("GOARCH", runtime.GOARCH)

	// The pages typecheck once the build has written web/.weft-types.
	if out, err := exec.Command("pnpm", "typecheck").CombinedOutput(); err != nil {
		t.Errorf("pnpm typecheck: %v\n%s", err, out)
	}

	// A second build replaces the first.
	if code, out, errs := build(t, "-o", "bin/server"); code != 0 || !strings.HasSuffix(out, "Built bin/server\n") {
		t.Fatalf("second build: exit %d: %s%s", code, out, errs)
	}
	if !exists(t, filepath.Join(app, "bin", "server")) {
		t.Error("-o bin/server was not written")
	}
}

// A created app checks the props of Weft's components before its first
// build: weft create leaves web/.weft-types/components.d.ts, and ensure, so
// every command that runs the app, puts it back when it is gone.
func TestCreatedAppChecksComponentProps(t *testing.T) {
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	if exists(t, filepath.Join(app, "web", "dist")) {
		t.Fatal("the created app was built")
	}
	dts := filepath.Join(app, "web", ".weft-types", "components.d.ts")
	if !exists(t, dts) {
		t.Fatal("weft create left no web/.weft-types/components.d.ts")
	}
	page := filepath.Join(app, "web", "pages", "bad.vue")
	if err := os.WriteFile(page, []byte("<template>\n  <Button tone=\"nope\" />\n</template>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("pnpm", "typecheck").CombinedOutput()
	if err == nil || !strings.Contains(string(out), "bad.vue") || !strings.Contains(string(out), `"nope"`) {
		t.Errorf("pnpm typecheck passed a Button with tone=\"nope\": %v\n%s", err, out)
	}
	if err := os.Remove(dts); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if _, err := ensure(app, &stdout, &stderr); err != nil {
		t.Fatalf("ensure: %v: %s%s", err, &stdout, &stderr)
	}
	if !exists(t, dts) {
		t.Error("ensure did not write web/.weft-types/components.d.ts again")
	}
}

// withoutExecBits copies the staged bundle with every executable bit cleared,
// as files from the embedded bundle are, and points WEFT_BUNDLE at it.
func withoutExecBits(t *testing.T) {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "bundle")
	err := filepath.WalkDir(stagedBundle, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(stagedBundle, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEFT_BUNDLE", dst)
}

func TestBuildFreshClone(t *testing.T) {
	inTempDir(t, "work")
	withoutExecBits(t)
	app := copyApp(t, ".")
	aside := t.TempDir()
	for _, p := range []string{".weft", "node_modules", "web/dist"} {
		if _, err := os.Lstat(filepath.Join(app, p)); err == nil {
			if err := os.Rename(filepath.Join(app, p), filepath.Join(aside, filepath.Base(p))); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Chdir(app)
	code, out, errs := build(t)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if got := unpacked(app); got != "0.0.0-dev" {
		t.Errorf(".weft/VERSION = %q", got)
	}
	assertServes(t, filepath.Join(app, "myapp"))
}

func TestBuildUnpacksAnOlderWeft(t *testing.T) {
	inTempDir(t, "work")
	app := copyApp(t, ".")
	os.WriteFile(filepath.Join(app, ".weft", "VERSION"), []byte("0.0.1\n"), 0o644)
	os.WriteFile(filepath.Join(app, ".weft", "stale.txt"), []byte("x"), 0o644)
	t.Chdir(app)
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if got := unpacked(app); got != "0.0.0-dev" {
		t.Errorf(".weft/VERSION = %q", got)
	}
	if exists(t, filepath.Join(app, ".weft", "stale.txt")) {
		t.Error(".weft was not unpacked again")
	}
}

// otherVersionApp makes an app without installing it and records it at version.
func otherVersionApp(t *testing.T, version string) []byte {
	t.Helper()
	inTempDir(t, "work")
	if code, _, errs := create(t, "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if version != weft.Version {
		if err := writeAppVersion("myapp", version); err != nil {
			t.Fatal(err)
		}
	}
	pkg, _ := os.ReadFile("myapp/package.json")
	t.Chdir("myapp")
	return pkg
}

// setVersion makes this weft's version v until the test ends, as a release
// build's is.
func setVersion(t *testing.T, v string) {
	t.Helper()
	was := weft.Version
	weft.Version = v
	t.Cleanup(func() { weft.Version = was })
}

func TestBuildRefusesOtherVersion(t *testing.T) {
	otherVersionApp(t, "9.9.9")
	code, out, errs := build(t)
	if code != 1 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	want := "weft build: this app is on Weft 9.9.9 and this weft is 0.0.0-dev; run weft update --version 9.9.9 to match it.\n"
	if errs != want {
		t.Errorf("stderr %q, want %q", errs, want)
	}
	if got := unpacked("."); got != "0.0.0-dev" {
		t.Errorf(".weft/VERSION = %q", got)
	}
	if exists(t, "node_modules") || exists(t, "web/dist") {
		t.Error("the refused build installed or built something")
	}
}

func TestUpgradeMovesApp(t *testing.T) {
	before := otherVersionApp(t, "0.0.5")
	setVersion(t, "0.1.0")
	os.WriteFile(".weft/stale.txt", []byte("x"), 0o644)
	var out, errb bytes.Buffer
	code := run([]string{"upgrade", "--no-install"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if out.String() != "Moved myapp from Weft 0.0.5 to 0.1.0\n" {
		t.Errorf("stdout %q", out.String())
	}
	after, _ := os.ReadFile("package.json")
	if want := bytes.Replace(before, []byte(`"0.0.5"`), []byte(`"0.1.0"`), 1); !bytes.Equal(after, want) {
		t.Errorf("package.json = %q, want %q", after, want)
	}
	if got := unpacked("."); got != "0.1.0" || exists(t, ".weft/stale.txt") {
		t.Errorf(".weft was not unpacked again (VERSION %q)", got)
	}
	if exists(t, "node_modules") {
		t.Error("--no-install installed")
	}

	out.Reset()
	if code := run([]string{"upgrade", "--no-install"}, &out, &errb); code != 0 || out.String() != "myapp is already on Weft 0.1.0\n" {
		t.Errorf("second upgrade: exit %d, stdout %q", code, out.String())
	}
}

// An app made by 0.0.6, whose main.go calls weft.Serve, does not compile once
// moved: weft upgrade says so, after its normal output, and still exits 0.
func TestUpgradeNamesWeftServe(t *testing.T) {
	otherVersionApp(t, "0.0.5")
	setVersion(t, "0.1.0")
	old := "package main\n\nimport \"github.com/flakerimi/weft\"\n\nfunc main() { _ = weft.Serve(nil, nil) }\n"
	if err := os.WriteFile("main.go", []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	const tail = " still calls weft.Serve, which moved; see https://weft.base.al/docs/reference/cli#moving-an-app-to-warp for the two lines to change.\n"
	code, out, errs := runCmd("upgrade", "--no-install")
	if code != 0 || out != "Moved myapp from Weft 0.0.5 to 0.1.0\n" || errs != "weft upgrade: main.go"+tail {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	// Run again, on the same version, it still says so; and it names the file.
	if err := os.WriteFile("main.go", []byte("package main\n\nfunc main() { serve() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("serve.go", []byte(strings.Replace(old, "func main()", "func serve()", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs = runCmd("upgrade", "--no-install")
	if code != 0 || out != "myapp is already on Weft 0.1.0\n" || errs != "weft upgrade: serve.go"+tail {
		t.Fatalf("again: exit %d, stdout %q, stderr %q", code, out, errs)
	}
}

// An app made by a development build moves forward to a release.
func TestUpgradeMovesADevApp(t *testing.T) {
	otherVersionApp(t, "0.0.0-dev")
	setVersion(t, "0.0.6")
	code, out, errs := runCmd("upgrade", "--no-install")
	if code != 0 || out != "Moved myapp from Weft 0.0.0-dev to 0.0.6\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
}

// An app on a newer Weft is never moved backwards.
func TestUpgradeRefusesNewer(t *testing.T) {
	for _, c := range []struct{ app, weft string }{{"9.9.9", "0.0.0-dev"}, {"0.2.0", "0.1.0"}, {"0.1.0", "0.1.0-alpha.2"}} {
		t.Run(c.app+"_"+c.weft, func(t *testing.T) {
			before := otherVersionApp(t, c.app)
			setVersion(t, c.weft)
			code, out, errs := runCmd("upgrade", "--no-install")
			want := "weft upgrade: this app is on Weft " + c.app + ", newer than this weft " + c.weft + "; run weft update to get " + c.app + " or later.\n"
			if code != 1 || errs != want || out != "" {
				t.Errorf("exit %d, stdout %q, stderr %q, want %q", code, out, errs, want)
			}
			if after, _ := os.ReadFile("package.json"); !bytes.Equal(after, before) {
				t.Errorf("package.json changed: %q", after)
			}
		})
	}
}

func TestNewerVersion(t *testing.T) {
	for _, c := range []struct {
		a, b  string
		newer bool
	}{
		{"0.1.0", "0.0.5", true},
		{"0.0.5", "0.1.0", false},
		{"0.0.10", "0.0.9", true},
		{"1.0.0", "0.99.99", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.1.0-alpha.1", true},
		{"0.1.0-alpha.1", "0.1.0", false},
		{"0.1.0-alpha.10", "0.1.0-alpha.2", true},
		{"0.1.0-beta", "0.1.0-alpha.9", true},
		{"0.1.0-alpha.1", "0.1.0-alpha", true},
		{"0.0.1", "0.0.0-dev", true},
		{"0.0.0-alpha", "0.0.0-dev", true},
		{"0.0.0-dev", "0.0.1", false},
		{"0.0.0-dev", "0.0.0-dev", false},
		{"nonsense", "0.0.1", false},
		{"0.0.1", "nonsense", false},
		{"nonsense", "0.0.0-dev", false},
	} {
		if got := newerVersion(c.a, c.b); got != c.newer {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", c.a, c.b, got, c.newer)
		}
	}
}

func TestUpgradeUsage(t *testing.T) {
	code, _, errs := runCmd("upgrade", "--nope")
	if code != 2 || !strings.HasPrefix(errs, "weft upgrade: ") {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

func TestBuildOutsideApp(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, c := range []string{"build", "upgrade"} {
		code, out, errs := runCmd(c)
		if code != 1 {
			t.Errorf("%s: exit %d", c, code)
		}
		if want := "weft " + c + ": no Weft app here; run it in a folder made by weft create.\n"; errs != want || out != "" {
			t.Errorf("%s: stderr %q, want %q", c, errs, want)
		}
	}
}

func TestBuildUsage(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"build", "extra"}, {"build", "--nope"}, {"build", "-o"}} {
		code, _, errs := runCmd(args...)
		if code != 2 || !strings.HasPrefix(errs, "weft build: ") || strings.Count(strings.TrimSpace(errs), "\n") != 0 {
			t.Errorf("%v: exit %d, stderr %q", args, code, errs)
		}
	}
}

func TestBuildInPathWithSpace(t *testing.T) {
	inTempDir(t, filepath.Join("My Apps", "café"))
	if code, out, errs := create(t, "myapp"); code != 0 {
		t.Fatalf("create: exit %d: %s%s", code, out, errs)
	}
	t.Chdir("myapp")
	code, out, errs := build(t)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	wd, _ := os.Getwd()
	assertServes(t, filepath.Join(wd, "myapp"))
}

func TestBuildFromSubfolder(t *testing.T) {
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(filepath.Join(app, "web", "pages"))
	code, out, errs := build(t)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if want := "Built " + filepath.Join("..", "..", "myapp") + "\n"; !strings.HasSuffix(out, want) {
		t.Errorf("stdout ends %q, want %q", out[max(0, len(out)-80):], want)
	}
	if !exists(t, filepath.Join(app, "myapp")) || !exists(t, filepath.Join(app, "web", "dist")) {
		t.Error("the app was not built")
	}
	if exists(t, "myapp") || exists(t, "web") {
		t.Error("the build wrote into the subfolder")
	}
}
