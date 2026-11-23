package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	if !exists(t, filepath.Join(dir, ".gitignore")) {
		t.Error(".gitignore is missing")
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

func TestCreateOptions(t *testing.T) {
	inTempDir(t, "work")
	code, _, errs := create(t, "--no-install", "shop", "--module", "example.com/shop", "--theme", "lisa", "--tailwind")
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
