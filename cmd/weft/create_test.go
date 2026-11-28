package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
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
			// Through sh, which Windows does not find from the script's first
			// line; the folder with forward slashes, as Git Bash's sh reads it.
			cmd := exec.Command("sh", "../../scripts/bundle.sh", filepath.ToSlash(filepath.Join(dir, "bundle")))
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
	if got := strings.Split(read(".weft/VERSION"), "\n"); got[0] != "0.0.0-dev" || len(got[1]) != 64 {
		t.Errorf(".weft/VERSION = %q, want the version and the bundle's hash", got)
	}
	if !strings.Contains(read("package.json"), `"name": "`+filepath.Base(dir)+`"`) {
		t.Errorf("package.json has the wrong name: %q", read("package.json"))
	}
	ignore := strings.Split(read(".gitignore"), "\n")
	for _, want := range []string{".weft/", ".weft.lock", ".weft-new-*/", ".weft-old-*/", "node_modules/", "web/dist/", "web/.weft-types/", ".env", "/data/", "/" + filepath.Base(dir), "/" + filepath.Base(dir) + ".exe"} {
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
	for _, p := range []string{
		"web/pages/index.vue", "web/components/Hello.vue", "app/modules.go", "app/hello/routes.go", "app/hello/hello_test.go", ".env", ".env.sample", "dist_dev.go", "CLAUDE.md", "README.md", "AGENTS.md",
		// The shop's frame: the login, the owner's setup page, the pages to
		// sign up, ask for a reset and set a password from a mailed link, and
		// the check that the generated Me fits @weft/ui's SessionUser.
		"go.sum", "web/pages/login.vue", "web/pages/setup.vue", "web/components/LoginForm.vue", "web/components/SetupForm.vue", "web/answers.ts",
		"web/pages/register.vue", "web/pages/forgot.vue", "web/pages/password.vue", "web/components/RegisterForm.vue", "web/components/ForgotForm.vue", "web/components/PasswordForm.vue",
		"web/session-check.ts",
		// The account and users pages, and the front page's links to them.
		"web/pages/account.vue", "web/pages/users.vue", "web/components/AccountPanel.vue", "web/components/UsersPanel.vue", "web/components/AccountLinks.vue",
	} {
		if !exists(t, filepath.Join(dir, p)) {
			t.Errorf("%s is missing", p)
		}
	}
	// The session is @weft/ui's: the template keeps no session.ts and no
	// nextPath of its own.
	if exists(t, filepath.Join(dir, "web", "session.ts")) {
		t.Error("web/session.ts is written: the session is useSession from @weft/ui")
	}
	if strings.Contains(read("web/answers.ts"), "function nextPath") {
		t.Error("web/answers.ts has a nextPath of its own: it is @weft/ui's")
	}
	if got := read("web/dist/.weft-build"); got != `{"weft":1,"files":[]}`+"\n" {
		t.Errorf("web/dist/.weft-build = %q, want the empty marker the page build replaces", got)
	}
	if exists(t, filepath.Join(dir, "api", "api.go")) {
		t.Error("api/api.go is written: the template's API is in app/")
	}
	// .env holds what the frame needs to start: the database and a secret of
	// its own, 48 random bytes in base64url; .env.sample only a placeholder.
	env := envLines(read(".env"))
	for k, want := range map[string]string{"PORT": "8080", "DB_DRIVER": "sqlite", "DB_URL": "file:data/" + filepath.Base(dir) + ".db", "AUTH_OWNER_EMAIL": "owner@example.com"} {
		if env[k] != want {
			t.Errorf(".env has %s=%q, want %q: %q", k, env[k], want, read(".env"))
		}
	}
	if secret, err := base64.RawURLEncoding.DecodeString(env["AUTH_SECRET"]); err != nil || len(secret) != 48 {
		t.Errorf(".env has AUTH_SECRET=%q, want 48 random bytes in base64url (%v)", env["AUTH_SECRET"], err)
	}
	if len(env) != 5 {
		t.Errorf(".env = %q, want PORT, DB_DRIVER, DB_URL, AUTH_SECRET and AUTH_OWNER_EMAIL", read(".env"))
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(filepath.Join(dir, ".env")); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf(".env is %v, want only its owner to read it, as it holds the secret (%v)", info.Mode().Perm(), err)
		}
	}
	sample := read(".env.sample")
	if s := envLines(sample); s["AUTH_SECRET"] != "" || s["DB_DRIVER"] != "sqlite" || s["DB_URL"] != "file:data/"+filepath.Base(dir)+".db" || s["AUTH_OWNER_EMAIL"] != "owner@example.com" || s["PORT"] != "8080" {
		t.Errorf(".env.sample = %q, want the settings with an empty AUTH_SECRET", sample)
	}
	for _, want := range []string{"# HOST=127.0.0.1\n", "openssl rand -base64 48"} {
		if !strings.Contains(sample, want) {
			t.Errorf(".env.sample lacks %q: %q", want, sample)
		}
	}
	// Every setting of mail, the login and middleware is in .env.sample, as a
	// comment holding its default or an example.
	for _, key := range []string{"MAIL", "MAIL_HOST", "MAIL_PORT", "MAIL_USER", "MAIL_PASSWORD", "MAIL_FROM", "AUTH_REGISTRATION", "API_KEYS", "LOG", "RATE_LIMIT"} {
		if !strings.Contains(sample, "\n# "+key+"=") {
			t.Errorf(".env.sample has no line # %s=…: %q", key, sample)
		}
	}
	if !strings.Contains(read("go.mod"), "\tmodernc.org/sqlite v1.63.0\n") || !strings.Contains(read("go.sum"), "modernc.org/sqlite v1.63.0 h1:") {
		t.Errorf("go.mod and go.sum do not require the SQLite driver: %q", read("go.mod"))
	}
	if !strings.Contains(read("main.go"), "\t_ \"modernc.org/sqlite\"\n") {
		t.Errorf("main.go does not import the SQLite driver: %q", read("main.go"))
	}
	modules := read("app/modules.go")
	at := -1
	for _, m := range []string{"middleware", "data", "mail", "auth", "hello"} {
		i := strings.Index(modules, "\t\t"+m+".Module(),\n")
		if i <= at {
			t.Errorf("app/modules.go does not list middleware, data, mail, auth and hello in that order: %q", modules)
			break
		}
		at = i
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
	if out != "Set AUTH_OWNER_EMAIL in "+filepath.Join("myapp", ".env")+" to the owner's email address; it is owner@example.com until you do.\nCreated myapp. Next: cd myapp && weft dev\n" {
		t.Errorf("stdout %q", out)
	}
	assertCreated(t, "myapp")
}

// envLines reads the KEY=value lines of a .env, leaving comments out.
func envLines(text string) map[string]string {
	out := map[string]string{}
	for _, l := range strings.Split(text, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok && !strings.HasPrefix(l, "#") {
			out[k] = v
		}
	}
	return out
}

// Every app gets a secret of its own: two apps never share one.
func TestCreateSecretsDiffer(t *testing.T) {
	inTempDir(t, "work")
	for _, name := range []string{"one", "two"} {
		if code, _, errs := create(t, name, "--no-install"); code != 0 {
			t.Fatalf("%s: exit %d: %s", name, code, errs)
		}
	}
	one, _ := os.ReadFile(filepath.Join("one", ".env"))
	two, _ := os.ReadFile(filepath.Join("two", ".env"))
	a, b := envLines(string(one))["AUTH_SECRET"], envLines(string(two))["AUTH_SECRET"]
	if len(a) != 64 || len(b) != 64 || a == b {
		t.Errorf("AUTH_SECRET is %q in one and %q in two, want 64 characters each, and different", a, b)
	}
}

// newSecret is 48 bytes of crypto/rand in base64url, without padding.
func TestNewSecret(t *testing.T) {
	seen := map[string]bool{}
	for range 20 {
		s, err := newSecret()
		if err != nil {
			t.Fatal(err)
		}
		if b, err := base64.RawURLEncoding.DecodeString(s); err != nil || len(b) != 48 || seen[s] {
			t.Fatalf("newSecret() = %q (%v), want a fresh 48 bytes in base64url", s, err)
		}
		seen[s] = true
	}
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
	name, script := "pnpm", "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 10.0.0; exit 0; fi\necho \"fake pnpm: $*\" >&2\nexit 1\n"
	if runtime.GOOS == "windows" {
		// Windows runs a program found on PATH by its extension, not its first line.
		name, script = "pnpm.cmd", "@echo off\r\nif \"%~1\"==\"--version\" (\r\n  echo 10.0.0\r\n  exit /b 0\r\n)\r\necho fake pnpm: %* 1>&2\r\nexit /b 1\r\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
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
	// Before any build, with or without the dev tag: web/dist holds the
	// empty marker weft create writes, which dist.go's embed matches.
	for _, args := range [][]string{{"vet", "-tags", "dev", "./..."}, {"test", "-tags", "dev", "./..."}, {"vet", "./..."}, {"test", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = "myapp"
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
	}
	// A binary built so, without its pages, refuses to start and says why.
	bin := filepath.Join(t.TempDir(), "myapp"+exeSuffix(runtime.GOOS))
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = "myapp"
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	run.Dir = "myapp"
	run.Env = append(os.Environ(), "PORT="+strconv.Itoa(freeTestPort(t)))
	out, err := run.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 || !strings.Contains(string(out), "warp: the pages are not built (dist holds only the empty marker weft create writes); run weft build, which builds them into the binary.") {
		t.Errorf("the unbuilt binary: %v\n%s", err, out)
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
	bin := filepath.Join(t.TempDir(), "app"+exeSuffix(runtime.GOOS))
	build := exec.Command("go", "build", "-tags", "dev", "-o", bin, ".")
	build.Dir = "myapp"
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, parent := range []bool{false, true} {
		port := strconv.Itoa(freeTestPort(t))
		cmd := exec.Command(bin)
		// In the app's folder, where its .env has the settings auth needs.
		cmd.Dir = "myapp"
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
	// From the open file, not os.Stat: on Windows a FileInfo from os.Stat
	// reads the file's identity only when os.SameFile asks, from the path,
	// which by then names the new file.
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.Stat()
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
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
	// Windows keeps no permission bits but read-only, so 0600 reads as 0666.
	if runtime.GOOS != "windows" && after.Mode().Perm() != 0o600 {
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

// A folder of the template with no file in it is not written: a release
// bundle is embedded, which has no empty folders, so a bundle read from a
// folder must make the same app. A file's folders are made.
func TestWriteTemplateWritesNoEmptyFolders(t *testing.T) {
	bundle := fstest.MapFS{
		"template/api":                 {Mode: fs.ModeDir | 0o755},
		"template/app":                 {Mode: fs.ModeDir | 0o755},
		"template/app/hello":           {Mode: fs.ModeDir | 0o755},
		"template/app/hello/x.go.tmpl": {Data: []byte("package [[.Name]]\n")},
		"template/README.md":           {Data: []byte("x")},
	}
	dir := filepath.Join(t.TempDir(), "demo")
	if err := writeTemplate(bundle, dir, templateData{Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "app", "hello", "x.go")); err != nil || string(got) != "package demo\n" {
		t.Errorf("app/hello/x.go = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "api")); err == nil {
		t.Error("the empty template folder api was written")
	}
}

// startApp runs bin in dir, so it reads the .env there and keeps its
// database in dir/data, and returns its base URL, the setup link's token it
// logged and its log. The process is stopped when the test ends.
func startApp(t *testing.T, bin, dir string) (string, string, *logBuffer) {
	t.Helper()
	port := freeTestPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(port))
	logs := &logBuffer{}
	cmd.Stdout, cmd.Stderr = logs, logs
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	t.Cleanup(func() { cancel(); <-exited })
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		select {
		case <-exited:
			t.Fatalf("%s stopped: %s", bin, logs.String())
		default:
		}
		res, err := http.Get(url + "/api/hello")
		if err != nil {
			continue
		}
		res.Body.Close()
		m := setupLink.FindStringSubmatch(logs.String())
		if m == nil {
			t.Fatalf("%s answers but logged no setup link: %s", bin, logs.String())
		}
		return url, m[1], logs
	}
	t.Fatalf("%s did not answer within 15 s: %s", bin, logs.String())
	return "", "", nil
}

// postJSON sends body as JSON to url and returns the status, the body and
// the session cookie the answer set, if any.
func postJSON(t *testing.T, url string, body any, cookie string) (int, string, string) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got, _ := io.ReadAll(res.Body)
	set := ""
	for _, c := range res.Cookies() {
		if c.Name == "session" {
			set = c.Name + "=" + c.Value
		}
	}
	return res.StatusCode, string(got), set
}

var mailedToken = regexp.MustCompile(`/password#token=([A-Za-z0-9_-]{43})`)

// mailedTo waits up to 5 s for the app to log a mail to address whose
// subject starts with subject, and returns the token of the link in the
// newest.
func mailedTo(t *testing.T, logs *logBuffer, address, subject string) string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		text := logs.String()
		if i := strings.LastIndex(text, "mail to "+address+": "+subject); i >= 0 {
			if m := mailedToken.FindStringSubmatch(text[i:]); m != nil {
				return m[1]
			}
		}
	}
	t.Fatalf("the app logged no mail to %s, %q, with a link: %s", address, subject, logs.String())
	return ""
}

// A created app is the shop's frame, ready to use: weft create wrote its
// typed client, so the login and setup pages type-check before any build; its
// Go tests pass over SQLite; and built, it answers /api/hello to anyone,
// keeps /api/auth/me for the signed in, logs the owner's setup link, and
// serves /login and /setup with their password forms guarded until their
// islands are live. With AUTH_REGISTRATION=open it signs a stranger up and
// resets a password through the links it prints, as the plan's
// TestCreatedAppSignsUpAndResets asks, on the same build; and every answer
// has the middleware's headers.
func TestCreatedAppIsTheFrame(t *testing.T) {
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	if src, err := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts")); err != nil || !strings.Contains(string(src), "async login(") || !strings.Contains(string(src), "async hello(") {
		t.Fatalf("weft create left no client with api.auth.login and api.hello.hello: %v\n%s", err, src)
	}
	// The client was written from the app's folder: the Go build left
	// nothing in a myapp folder inside it, where the binary goes.
	if exists(t, filepath.Join(app, "myapp")) {
		t.Error("weft create left a folder myapp inside the app")
	}
	if out, err := exec.Command("pnpm", "typecheck").CombinedOutput(); err != nil {
		t.Errorf("pnpm typecheck in a created app: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "test", "-count=1", "-tags", "dev", "./...").CombinedOutput(); err != nil || !strings.Contains(string(out), "ok  \tmyapp/app/hello") {
		t.Errorf("go test -tags dev ./... in a created app: %v\n%s", err, out)
	}
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("weft build: exit %d: %s%s", code, out, errs)
	}
	// The page build took the marker's place.
	if !exists(t, filepath.Join(app, "web", "dist", "routes.json")) {
		t.Error("weft build did not replace weft create's web/dist")
	}
	env, err := os.OpenFile(filepath.Join(app, ".env"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.WriteString("AUTH_REGISTRATION=open\n"); err != nil {
		t.Fatal(err)
	}
	env.Close()
	url, token, logs := startApp(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)), app)
	if token == "" {
		t.Error("no setup token was logged")
	}
	if code, body := get(t, url+"/api/hello"); code != 200 || strings.TrimSpace(body) != `{"message":"Hello from Go"}` {
		t.Errorf("GET /api/hello without a session = %d %q", code, body)
	}
	if code, body := get(t, url+"/api/auth/me"); code != 401 || !strings.Contains(body, `"code":"unauthorized"`) {
		t.Errorf("GET /api/auth/me without a session = %d %q", code, body)
	}
	for path, want := range map[string]string{
		"/":         `href="/login"`,
		"/login":    `<weft-island data-id="LoginForm"`,
		"/setup":    `<weft-island data-id="SetupForm"`,
		"/register": `<weft-island data-id="RegisterForm"`,
		"/forgot":   `<weft-island data-id="ForgotForm"`,
		"/password": `<weft-island data-id="PasswordForm"`,
		"/account":  `<weft-island data-id="AccountPanel"`,
		"/users":    `<weft-island data-id="UsersPanel"`,
	} {
		code, body := get(t, url+path)
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d, want 200 with %q: %s", path, code, want, body)
		}
		// /password draws its form only once the browser has read the link,
		// and /account and /users theirs once the session is known.
		if path != "/" && path != "/password" && path != "/account" && path != "/users" && (!postForm.MatchString(body) || !guardsSubmit(t, body)) {
			t.Errorf("GET %s: the form is not a POST form behind the inline submit guard: %s", path, body)
		}
	}
	if !exists(t, filepath.Join(app, "data", "myapp.db")) {
		t.Error("the app kept no database in data/myapp.db, which DB_URL in .env names")
	}

	// middleware is listed: its headers are on a page and an API answer.
	for _, path := range []string{"/", "/api/hello"} {
		res, err := http.Get(url + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("X-Frame-Options") != "DENY" {
			t.Errorf("GET %s has no security headers: %v", path, res.Header)
		}
	}
	// Signing up and resetting, through the mail the log driver prints.
	if code, body := get(t, url+"/api/auth/settings"); code != 200 || strings.TrimSpace(body) != `{"registration":"open","api_keys":false}` {
		t.Errorf("GET /api/auth/settings = %d %q", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/setup", map[string]string{"token": token, "password": "the owner's password"}, ""); code != 201 {
		t.Fatalf("setup = %d %s", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/register", map[string]string{"email": "new@example.com"}, ""); code != 204 {
		t.Fatalf("register = %d %s", code, body)
	}
	code, body, cookie := postJSON(t, url+"/api/auth/password", map[string]string{"token": mailedTo(t, logs, "new@example.com", "Finish signing up"), "password": "a newcomer's password"}, "")
	if code != 200 || !strings.Contains(body, `"role":"viewer"`) || cookie == "" {
		t.Fatalf("password from the sign-up link = %d %s, cookie %q", code, body, cookie)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/forgot", map[string]string{"email": "new@example.com"}, ""); code != 204 {
		t.Fatalf("forgot = %d %s", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/password", map[string]string{"token": mailedTo(t, logs, "new@example.com", "Reset your password"), "password": "a newer password"}, ""); code != 200 {
		t.Fatalf("password from the reset link = %d %s", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/login", map[string]string{"email": "new@example.com", "password": "a newer password"}, ""); code != 200 {
		t.Errorf("login with the new password = %d %s", code, body)
	}
}

// With the module proxy out of reach and an empty module cache, weft create
// makes the app, says the owner's line, and ends on one sentence saying Go
// could not download the modules and what to do.
func TestCreateWithoutTheModuleProxy(t *testing.T) {
	inTempDir(t, "work")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GOFLAGS", "-modcacherw")
	code, out, errs := create(t, "myapp")
	want := "weft create: Go could not download the modules myapp/go.mod requires, such as the SQLite driver (see its output above), so myapp is made but does not build yet; check the network or GOPROXY, then run cd myapp && weft gen.\n"
	if code != 1 || !strings.HasSuffix(errs, want) {
		t.Errorf("exit %d, stderr %q, want it to end %q", code, errs, want)
	}
	if !strings.Contains(out, ownerLine("myapp")) {
		t.Errorf("stdout %q lacks the owner line", out)
	}
	if !exists(t, filepath.Join("myapp", "package.json")) {
		t.Error("the app was not left in place")
	}
}
