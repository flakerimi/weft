package main

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stagedBundle is a bundle scripts/bundle.sh staged once for this test binary;
// it is empty under -short.
var stagedBundle string

func TestMain(m *testing.M) {
	code := 0
	func() {
		short := false
		for _, a := range os.Args[1:] {
			if a == "-test.short" || a == "-test.short=true" {
				short = true
			}
		}
		if !short {
			dir, err := os.MkdirTemp("", "weft-test-bundle")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(dir)
			cmd := exec.Command("../../scripts/bundle.sh", filepath.Join(dir, "bundle"))
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				println("scripts/bundle.sh failed:", err.Error())
				code = 1
				return
			}
			stagedBundle = filepath.Join(dir, "bundle")
		}
		code = m.Run()
	}()
	os.Exit(code)
}

// inTempDir runs the test in a new folder named sub under a temp folder, with
// WEFT_BUNDLE set; it skips the test when there is no bundle or no toolchain.
func inTempDir(t *testing.T, sub string) {
	t.Helper()
	if stagedBundle == "" {
		t.Skip("no staged bundle under -short")
	}
	if err := prerequisites(); err != nil {
		t.Skipf("prerequisites: %v", err)
	}
	t.Setenv("WEFT_BUNDLE", stagedBundle)
	dir := filepath.Join(t.TempDir(), sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
}

func create(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(append([]string{"create"}, args...), &out, &errb)
	return code, out.String(), errb.String()
}

func assertCreated(t *testing.T, dir string) {
	t.Helper()
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if !strings.HasPrefix(read("go.mod"), "module myapp\n") {
		t.Errorf("go.mod = %q", read("go.mod"))
	}
	if !strings.Contains(read("package.json"), `"weft":{"version":"0.0.0-dev","theme":"neutral"}`) {
		t.Errorf("package.json = %q", read("package.json"))
	}
	if got := strings.TrimSpace(read(".weft/VERSION")); got != "0.0.0-dev" {
		t.Errorf(".weft/VERSION = %q", got)
	}
	if !strings.Contains(read("package.json"), `"name": "`+filepath.Base(dir)+`"`) {
		t.Errorf("package.json has the wrong name: %q", read("package.json"))
	}
	ignore := strings.Split(read(".gitignore"), "\n")
	for _, want := range []string{".weft/", "node_modules/", "web/dist/", "web/.weft-types/", "/" + filepath.Base(dir)} {
		found := false
		for _, l := range ignore {
			found = found || l == want
		}
		if !found {
			t.Errorf(".gitignore lacks the line %q: %q", want, ignore)
		}
	}
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && info.Name() == ".weft" {
			return filepath.SkipDir
		}
		if err == nil && strings.HasSuffix(p, ".tmpl") {
			t.Errorf("%s was not rendered", p)
		}
		return nil
	})
	for _, p := range []string{"web/pages/index.vue", "web/components/Hello.vue", "api/api.go", "dist_dev.go", "CLAUDE.md", "README.md", "AGENTS.md"} {
		if !exists(t, filepath.Join(dir, p)) {
			t.Errorf("%s is missing", p)
		}
	}
	// An agent starts from /index.md, as the app's AGENTS.md says too.
	for _, f := range []string{"README.md", "AGENTS.md"} {
		text := read(f)
		if i, j := strings.Index(text, "https://weft.base.al/index.md"), strings.Index(text, "https://weft.base.al/llms.txt"); i < 0 || j < 0 || i > j {
			t.Errorf("%s does not name /index.md before /llms.txt: %q", f, text)
		}
	}
	if strings.Contains(read("AGENTS.md")+read("README.md")+read("main.go"), "[[") {
		t.Error("an unrendered [[ is left")
	}
}

func TestCreateWritesApp(t *testing.T) {
	inTempDir(t, "work")
	code, out, errs := create(t, "myapp", "--no-install")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if out != "Created myapp. Next: cd myapp && weft dev\n" {
		t.Errorf("stdout %q", out)
	}
	assertCreated(t, "myapp")
}

// An app named like a folder inside web/ gets an anchored .gitignore line, so
// web/pages is not ignored.
func TestCreateNamedLikeAFolder(t *testing.T) {
	inTempDir(t, "work")
	code, _, errs := create(t, "pages", "--no-install")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	b, _ := os.ReadFile("pages/.gitignore")
	lines := strings.Split(string(b), "\n")
	hasRooted := false
	for _, l := range lines {
		if l == "pages" || l == "pages/" {
			t.Errorf(".gitignore line %q would ignore web/pages", l)
		}
		hasRooted = hasRooted || l == "/pages"
	}
	if !hasRooted {
		t.Errorf(".gitignore = %q, want a /pages line", b)
	}
}

// A name whose binary would land in a folder of the app is refused, and
// nothing is written.
func TestCreateRefusesReservedName(t *testing.T) {
	inTempDir(t, "work")
	for _, name := range []string{"web", "api", "app", "dist"} {
		code, out, errs := create(t, name, "--no-install")
		want := "weft create: " + name + " is the name of a folder Weft uses in an app, where its binary would be written; choose another name.\n"
		if code != 2 || errs != want || out != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q, want %q", name, code, out, errs, want)
		}
	}
	if code, _, errs := create(t, "node_modules", "--no-install"); code != 2 || !strings.HasPrefix(errs, "weft create: ") {
		t.Errorf("node_modules: exit %d, stderr %q", code, errs)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("wrote %v", entries)
	}
}

// Tailwind is not set up by weft create, so the flag is unknown.
func TestCreateRefusesTailwind(t *testing.T) {
	inTempDir(t, "work")
	code, _, errs := create(t, "shop", "--tailwind", "--no-install")
	want := "weft create: flag provided but not defined: -tailwind; usage is weft create <name> [--module <path>] [--theme neutral|lisa|weft] [--no-install].\n"
	if code != 2 || errs != want {
		t.Errorf("exit %d, stderr %q, want %q", code, errs, want)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("wrote %v", entries)
	}
}

func TestCreateOptions(t *testing.T) {
	inTempDir(t, "work")
	code, _, errs := create(t, "--no-install", "shop", "--module", "example.com/shop", "--theme", "lisa")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	gomod, _ := os.ReadFile("shop/go.mod")
	if !strings.HasPrefix(string(gomod), "module example.com/shop\n") {
		t.Errorf("go.mod = %q", gomod)
	}
	cfg, err := readApp("shop")
	if err != nil || cfg.Theme != "lisa" || cfg.Version != "0.0.0-dev" {
		t.Errorf("readApp = %+v, %v", cfg, err)
	}
}

func TestCreateRefusesExistingFolder(t *testing.T) {
	inTempDir(t, "work")
	os.Mkdir("myapp", 0o755)
	os.WriteFile("myapp/keep.txt", []byte("x"), 0o644)
	code, _, errs := create(t, "myapp", "--no-install")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	want := "weft create: myapp already exists and is not empty; choose another name or move it away.\n"
	if errs != want {
		t.Errorf("stderr %q, want %q", errs, want)
	}
	entries, _ := os.ReadDir("myapp")
	if len(entries) != 1 || entries[0].Name() != "keep.txt" {
		t.Errorf("folder holds %v", entries)
	}
}

func TestCreateRefusesBadName(t *testing.T) {
	inTempDir(t, "work")
	for _, name := range []string{"My App", "1app", "../x"} {
		code, _, errs := create(t, name, "--no-install")
		if code != 2 {
			t.Errorf("%q: exit %d, stderr %q", name, code, errs)
		}
	}
	entries, _ := os.ReadDir(".")
	if len(entries) != 0 {
		t.Errorf("wrote %v", entries)
	}
	if _, err := os.Stat("../x"); err == nil {
		t.Error("../x was written")
	}
}

func TestCreateInPathWithSpace(t *testing.T) {
	inTempDir(t, filepath.Join("My Apps", "café"))
	code, _, errs := create(t, "myapp", "--no-install")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	assertCreated(t, "myapp")
}

// fakePnpm puts a pnpm first on PATH that prints a version for --version and
// fails every other command.
func fakePnpm(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 10.0.0; exit 0; fi\necho \"fake pnpm: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "pnpm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCreateInstallFails(t *testing.T) {
	inTempDir(t, "work")
	fakePnpm(t)
	code, _, errs := create(t, "myapp")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasSuffix(errs, "run pnpm install in myapp to finish.\n") {
		t.Errorf("stderr %q", errs)
	}
	if !strings.Contains(errs, "fake pnpm: install") {
		t.Errorf("pnpm's output was not passed through: %q", errs)
	}
	if !exists(t, "myapp/package.json") {
		t.Error("the app was not left in place")
	}
}

func TestCreatedAppGoTests(t *testing.T) {
	inTempDir(t, "work")
	if code, _, errs := create(t, "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	for _, args := range [][]string{{"vet", "-tags", "dev", "./..."}, {"test", "-tags", "dev", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = "myapp"
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
	}
}

// The server built with the dev tag exits when its standard input ends only
// under WEFT_DEV_PARENT=1, as weft dev runs it; run by hand, here with
// /dev/null as its standard input, it keeps serving.
func TestDevServerReadsStdinOnlyUnderWeftDev(t *testing.T) {
	inTempDir(t, "work")
	if code, _, errs := create(t, "myapp", "--no-install"); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	bin := filepath.Join(t.TempDir(), "app")
	build := exec.Command("go", "build", "-tags", "dev", "-o", bin, ".")
	build.Dir = "myapp"
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, parent := range []bool{false, true} {
		port := strconv.Itoa(freeTestPort(t))
		cmd := exec.Command(bin)
		cmd.Env = append(os.Environ(), "PORT="+port)
		if parent {
			cmd.Env = append(cmd.Env, "WEFT_DEV_PARENT=1")
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()
		select {
		case err := <-exited:
			if !parent {
				t.Errorf("without WEFT_DEV_PARENT the server exited (%v) as its standard input ended", err)
			} else if err != nil {
				t.Errorf("under WEFT_DEV_PARENT=1 the server exited with %v, want 0", err)
			}
		case <-time.After(3 * time.Second):
			if parent {
				t.Error("under WEFT_DEV_PARENT=1 the server did not exit within 3 s of its standard input ending")
			} else if c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second); err != nil {
				t.Errorf("without WEFT_DEV_PARENT the server does not answer: %v", err)
			} else {
				c.Close()
			}
			cmd.Process.Kill()
			<-exited
		}
	}
}

func TestWriteAppVersionKeepsFile(t *testing.T) {
	dir := t.TempDir()
	before := "{\n  \"name\": \"a\",\n  \"dependencies\": {\"weft\": \"1\"},\n\t\"weft\" : { \"theme\":\"lisa\",  \"version\" : \"0.0.5\" },\n  \"z\": 1\n}\n"
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(before), 0o644)
	if err := writeAppVersion(dir, "0.0.6"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if want := strings.Replace(before, "0.0.5", "0.0.6", 1); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
	cfg, err := readApp(dir)
	if err != nil || cfg.Version != "0.0.6" || cfg.Theme != "lisa" {
		t.Errorf("readApp = %+v, %v", cfg, err)
	}
}

func TestFindApp(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "web", "pages"), 0o755)
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"weft":{"version":"1"}}`), 0o644)
	os.WriteFile(filepath.Join(root, "web", "package.json"), []byte(`{"name":"x"}`), 0o644)
	got, err := findApp(filepath.Join(root, "web", "pages"))
	if err != nil || got != root {
		t.Errorf("findApp = %q, %v", got, err)
	}
	if _, err := findApp(t.TempDir()); err == nil {
		t.Error("findApp found an app in an empty folder")
	}
}

func TestFindAppNeedsAnObject(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"weft":{"version":"1"}}`), 0o644)
	for _, weftValue := range []string{`null`, `"0.0.6"`, `[]`, `1`} {
		os.WriteFile(filepath.Join(sub, "package.json"), []byte(`{"weft": `+weftValue+`}`), 0o644)
		if got, err := findApp(sub); err != nil || got != root {
			t.Errorf(`"weft": %s: findApp = %q, %v; want the folder above`, weftValue, got, err)
		}
	}
}

// A package.json that does not parse, as after a trailing comma, is the app
// to report: findApp returns the nearest such folder, so readApp says what
// to fix, rather than walking past it to "no Weft app here". One that parses
// with no "weft" object is still passed over.
func TestFindAppStopsAtBadJSON(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "app", "web")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"weft":{"version":"1"}}`), 0o644)
	os.WriteFile(filepath.Join(root, "app", "package.json"), []byte(`{"weft":{"version":"1",}}`), 0o644)
	os.WriteFile(filepath.Join(sub, "package.json"), []byte(`{"name":"x"}`), 0o644)
	if got, err := findApp(sub); err != nil || got != filepath.Join(root, "app") {
		t.Errorf("findApp = %q, %v; want the folder whose package.json does not parse", got, err)
	}
}

// Each command that runs an app says the file does not parse, in its own
// words, and weft catalog still answers there.
func TestCommandsReportBadJSON(t *testing.T) {
	outsideApp(t)
	file, _ := filepath.Abs("package.json")
	os.WriteFile(file, []byte("{\n  \"name\": \"myapp\",\n  \"weft\": {\"version\": \"0.0.0-dev\", \"theme\": \"lisa\",}\n}\n"), 0o644)
	port := strconv.Itoa(freeTestPort(t))
	for _, args := range [][]string{{"build"}, {"dev", "--port", port}, {"upgrade"}} {
		code, out, errs := runCmd(args...)
		prefix := "weft " + args[0] + ": " + file + " is not valid JSON ("
		if code != 1 || out != "" || !strings.HasPrefix(errs, prefix) || !strings.HasSuffix(errs, "); fix it and run the command again.\n") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q, want %q…", args, code, out, errs, prefix)
		}
	}
	if code, out, errs := runCmd("catalog", "Button"); code != 0 || !strings.HasPrefix(out, "# Button\n") {
		t.Errorf("catalog: exit %d, stdout %.40q, stderr %q", code, out, errs)
	}
}

// freeTestPort is a port nothing listens on at 127.0.0.1.
func freeTestPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestWriteAppVersionRenames(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "package.json")
	os.WriteFile(file, []byte(`{"weft":{"version":"0.0.5"}}`), 0o600)
	before, _ := os.Stat(file)
	if err := writeAppVersion(dir, "0.0.6"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Error("package.json was rewritten in place, not replaced by a renamed temp file")
	}
	if after.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", after.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("left %v in the folder", entries)
	}
}

// A "weft" object without a version is refused in a sentence that says what
// to add, by readApp and so by every command.
func TestReadAppNeedsAVersion(t *testing.T) {
	for _, pkg := range []string{`{"name":"a","weft":{}}`, `{"weft":{"theme":"lisa"}}`, `{"weft":{"version":""}}`} {
		dir := t.TempDir()
		file := filepath.Join(dir, "package.json")
		if err := os.WriteFile(file, []byte(pkg), 0o644); err != nil {
			t.Fatal(err)
		}
		sentence := ` has no weft.version; add "version": "0.0.0-dev", this weft's version, to its "weft" object`
		if _, err := readApp(dir); err == nil || err.Error() != file+sentence {
			t.Errorf("%s: readApp error %v, want %q", pkg, err, file+sentence)
		}
		t.Chdir(dir)
		for _, command := range []string{"build", "upgrade"} {
			code, _, errs := runCmd(command)
			if code != 1 || !strings.HasPrefix(errs, "weft "+command+": ") || !strings.HasSuffix(errs, "package.json"+sentence+".\n") {
				t.Errorf("%s: weft %s: exit %d, stderr %q", pkg, command, code, errs)
			}
		}
	}
}

// An app on another Weft is refused before anything is checked or written. An
// older app is pointed at weft upgrade, a newer one only at weft update, since
// weft upgrade never moves an app backwards.
func TestEnsureRefusesOtherVersion(t *testing.T) {
	for _, c := range []struct{ app, weft, want string }{
		{"0.0.5", "0.1.0", "this app is on Weft 0.0.5 and this weft is 0.1.0; run weft upgrade to move the app, or weft update --version 0.0.5 to match it"},
		{"0.0.0-dev", "0.0.6", "this app is on Weft 0.0.0-dev and this weft is 0.0.6; run weft upgrade to move the app, or weft update --version 0.0.0-dev to match it"},
		{"0.2.0", "0.1.0", "this app is on Weft 0.2.0 and this weft is 0.1.0; run weft update --version 0.2.0 to match it"},
		{"9.9.9", "0.0.0-dev", "this app is on Weft 9.9.9 and this weft is 0.0.0-dev; run weft update --version 9.9.9 to match it"},
		{"0.1.0", "0.1.0-alpha.2", "this app is on Weft 0.1.0 and this weft is 0.1.0-alpha.2; run weft update --version 0.1.0 to match it"},
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"a","weft":{"version":"`+c.app+`"}}`), 0o644)
		setVersion(t, c.weft)
		var out, errb bytes.Buffer
		if _, err := ensure(dir, &out, &errb); err == nil || err.Error() != c.want {
			t.Errorf("app %s, weft %s: %v, want %q", c.app, c.weft, err, c.want)
		}
		if exists(t, filepath.Join(dir, ".weft")) || out.Len()+errb.Len() != 0 {
			t.Errorf("app %s, weft %s: the refusal wrote something", c.app, c.weft)
		}
	}
}
