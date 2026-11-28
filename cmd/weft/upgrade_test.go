package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// app024Modules is app/modules.go as Weft 0.0.24's template wrote it for an
// app named myapp (testdata/app-0.0.24/app/modules.go).
const app024Modules = `// Package app lists the modules of the app.
package app

import (
	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"

	"myapp/app/hello"
)

// Modules is the one list of the app's modules, in the order they start: the
// database first, since the login keeps its users in it. weft scaffold adds
// each resource it makes at the end.
func Modules() []warp.Module {
	return []warp.Module{
		data.Module(),
		auth.Module(),
		hello.Module(),
	}
}
`

// From 0.2.0 the login needs the mail module: weft upgrade lists
// mail.Module() just before auth.Module(), with its import, formatted, and
// leaves the rest of the file as it was; a list that has it, or no auth, is
// left alone; and a file it cannot edit is an error that says the line to add.
func TestAddMailModule(t *testing.T) {
	added := strings.Replace(strings.Replace(app024Modules,
		"\t\"github.com/flakerimi/weft/warp/data\"\n", "\t\"github.com/flakerimi/weft/warp/data\"\n\t\"github.com/flakerimi/weft/warp/mail\"\n", 1),
		"\t\tauth.Module(),\n", "\t\tmail.Module(),\n\t\tauth.Module(),\n", 1)
	for _, c := range []struct {
		name, src, want string
		changed         bool
	}{
		{"0.0.24's list", app024Modules, added, true},
		{"mail listed already", added, added, false},
		{"no auth", strings.Replace(strings.Replace(app024Modules, "\t\"github.com/flakerimi/weft/warp/auth\"\n", "", 1), "\t\tauth.Module(),\n", "", 1), "", false},
		{"one line, auth with options", "package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/auth\"\n\t\"github.com/flakerimi/weft/warp/data\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{data.Module(), auth.Module(auth.Grant(auth.RoleViewer, \"reports:export\"))}\n}\n",
			"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/auth\"\n\t\"github.com/flakerimi/weft/warp/data\"\n\t\"github.com/flakerimi/weft/warp/mail\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{data.Module(), mail.Module(), auth.Module(auth.Grant(auth.RoleViewer, \"reports:export\"))}\n}\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, changed, err := addMailModule([]byte(c.src))
			if err != nil {
				t.Fatal(err)
			}
			want := c.want
			if want == "" {
				want = c.src
			}
			if changed != c.changed || string(got) != want {
				t.Errorf("changed %v, got\n%s\nwant\n%s", changed, got, want)
			}
		})
	}

	const cannot = "app/modules.go lists auth.Module() without mail.Module(), which the login needs from Weft 0.2.0, and weft upgrade could not add it; add the line mail.Module(), before auth.Module() in Modules(), and the import \"github.com/flakerimi/weft/warp/mail\", then run weft upgrade again."
	for _, src := range []string{
		// Built by hand, not as one list.
		"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/auth\"\n)\n\nfunc Modules() []warp.Module {\n\tvar m []warp.Module\n\tm = append(m, auth.Module())\n\treturn m\n}\n",
		// Another package already named mail.
		"package app\n\nimport (\n\t\"net/mail\"\n\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/auth\"\n)\n\nvar _ = mail.ParseAddress\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{auth.Module()}\n}\n",
		// Not Go.
		"package app\n\nimport \"github.com/flakerimi/weft/warp/auth\"\n\nfunc Modules() []warp.Module {\n",
	} {
		if _, _, err := addMailModule([]byte(src)); err == nil || err.Error() != cannot {
			t.Errorf("addMailModule(%q) = %v, want %q", src, err, cannot)
		}
	}
}

// accountFiles are what weft upgrade writes into an app with the login that
// lacks them.
var accountFilesWant = []string{
	"web/components/AccountLinks.vue", "web/components/AccountPanel.vue", "web/components/ForgotForm.vue",
	"web/components/PasswordForm.vue", "web/components/RegisterForm.vue", "web/components/UsersPanel.vue",
	"web/pages/account.vue", "web/pages/forgot.vue", "web/pages/password.vue", "web/pages/register.vue", "web/pages/users.vue",
	"web/session-check.ts",
}

// as024 turns the app weft create made in app into one as 0.0.24 made it: its
// files from testdata/app-0.0.24, without the ones 0.2.0 added, recorded at
// 0.0.24. It removes only files of the test's own copy.
func as024(t *testing.T, app string) {
	t.Helper()
	old := filepath.Join(repoCmdWeft, "testdata", "app-0.0.24")
	err := filepath.WalkDir(old, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(old, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(app, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range accountFilesWant {
		if err := os.Remove(filepath.Join(app, filepath.FromSlash(f))); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeAppVersion(app, "0.0.24"); err != nil {
		t.Fatal(err)
	}
}

// I1 and I2 of the final review of 0.2.0: an app made by 0.0.24, upgraded,
// builds and works. weft upgrade lists mail.Module() before auth.Module(),
// writes the account pages and forms the app lacks, touching none it has,
// and says what is left to the author: listing middleware.Module() first,
// and the old web/session.ts, which it keeps. Then weft gen and weft build
// pass, the owner signs in, and an invitation's link opens /password.
// installed024 is a copy of the installed app as 0.0.24 made it, the test
// run in it, whose go.mod replaces Warp's module with this repository, as an
// author's own replace, which upgrade keeps, so the app builds against the
// code under test. It returns the app's folder and the repository's.
func installed024(t *testing.T) (string, string) {
	t.Helper()
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	as024(t, app)
	root, _ := filepath.Abs(filepath.Join(repoCmdWeft, "..", ".."))
	mod := readFile(t, "go.mod")
	mod = mod[:strings.Index(mod, "replace ")] + "replace github.com/flakerimi/weft => " + filepath.ToSlash(root) + "\n"
	if err := os.WriteFile("go.mod", []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	return app, root
}

func TestUpgradeFrom0024(t *testing.T) {
	app, root := installed024(t)
	before := map[string]string{}
	for _, f := range []string{"web/answers.ts", "web/session.ts", "web/components/LoginForm.vue", "web/pages/index.vue", "web/tsconfig.json"} {
		before[f] = readFile(t, f)
	}
	setVersion(t, "0.2.0")
	code, out, errs := runCmd("upgrade", "--no-install")
	if code != 0 {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	var want strings.Builder
	want.WriteString("weft upgrade: listed mail.Module() before auth.Module() in app/modules.go, which the login needs from Weft 0.2.0.\n")
	for _, f := range accountFilesWant {
		want.WriteString("Added " + f + "\n")
	}
	want.WriteString("Moved myapp from Weft 0.0.24 to 0.2.0\n")
	want.WriteString("Next: list middleware.Module() in app/modules.go, first, so the lines the other modules log are in its format, with the import \"github.com/flakerimi/weft/warp/middleware\"; it adds security headers, a request log and a rate limit on /api.\n")
	want.WriteString("web/session.ts is kept: pages may now import useSession, requireSession and nextPath from @weft/ui, and once none imports ../session.ts, the file can go.\n")
	// The one line on stderr is about the test's own replace.
	replaced := "weft upgrade: go.mod replaces github.com/flakerimi/weft with " + filepath.ToSlash(root) + ", so the app builds against that, not Weft 0.2.0 from GitHub; remove that replace line from go.mod to build against the release.\n"
	if out != want.String() || errs != replaced {
		t.Errorf("stdout\n%s\nwant\n%s\nstderr %q, want %q", out, want.String(), errs, replaced)
	}
	if got := readFile(t, "app/modules.go"); !strings.Contains(got, "\t\tmail.Module(),\n\t\tauth.Module(),\n") || !strings.Contains(got, "\t\"github.com/flakerimi/weft/warp/mail\"\n") {
		t.Errorf("app/modules.go:\n%s", got)
	}
	for f, b := range before {
		if readFile(t, f) != b {
			t.Errorf("%s was changed; upgrade writes only what an app lacks", f)
		}
	}

	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("weft gen: exit %d: %s%s", code, out, errs)
	}
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("weft build: exit %d: %s%s", code, out, errs)
	}
	url, token, logs := startApp(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)), app)
	code, body, cookie := postJSON(t, url+"/api/auth/setup", map[string]string{"token": token, "password": "the owner's password"}, "")
	if code != 201 {
		t.Fatalf("setup = %d %s", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/login", map[string]string{"email": "owner@example.com", "password": "the owner's password"}, ""); code != 200 {
		t.Fatalf("the owner's login = %d %s", code, body)
	}
	if code, body, _ := postJSON(t, url+"/api/auth/invitations", map[string]string{"email": "new@example.com", "role": "member"}, cookie); code != 201 {
		t.Fatalf("invite = %d %s", code, body)
	}
	link := mailedTo(t, logs, "new@example.com", "You are invited")
	if code, body := get(t, url+"/password"); code != 200 || !strings.Contains(body, `<weft-island data-id="PasswordForm"`) {
		t.Errorf("GET /password, where the invitation's link goes, = %d: %s", code, body)
	}
	code, body, _ = postJSON(t, url+"/api/auth/token", map[string]string{"token": link}, "")
	var info struct{ Purpose, Role string }
	if code != 200 || json.Unmarshal([]byte(body), &info) != nil || info.Purpose != "invite" || info.Role != "member" {
		t.Errorf("the invitation's token = %d %s", code, body)
	}
	for _, path := range []string{"/forgot", "/register", "/account", "/users"} {
		if code, _ := get(t, url+path); code != http.StatusOK {
			t.Errorf("GET %s = %d after the upgrade", path, code)
		}
	}
}

// A modules.go weft upgrade cannot edit stops it before it writes anything
// (final review R2-I1): go.mod, the recorded version and .weft/ are as they
// were, and the sentence says to run weft upgrade again. Run again once the
// line is added by hand, it moves the app and adds the account pages, and
// the app builds.
func TestUpgradeCannotAddMail(t *testing.T) {
	_, root := installed024(t)
	hand := "package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/auth\"\n\t\"github.com/flakerimi/weft/warp/data\"\n\n\t\"myapp/app/hello\"\n)\n\nfunc Modules() []warp.Module {\n\tm := []warp.Module{data.Module(), auth.Module()}\n\treturn append(m, hello.Module())\n}\n"
	if err := os.WriteFile(filepath.Join("app", "modules.go"), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, f := range []string{"go.mod", "go.sum", "package.json", ".weft/VERSION", "app/modules.go"} {
		before[f] = readFile(t, f)
	}
	setVersion(t, "0.2.0")
	code, out, errs := runCmd("upgrade", "--no-install")
	want := "weft upgrade: app/modules.go lists auth.Module() without mail.Module(), which the login needs from Weft 0.2.0, and weft upgrade could not add it; add the line mail.Module(), before auth.Module() in Modules(), and the import \"github.com/flakerimi/weft/warp/mail\", then run weft upgrade again.\n"
	if code != 1 || errs != want || out != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, want exit 1 and %q", code, out, errs, want)
	}
	for f, b := range before {
		if readFile(t, f) != b {
			t.Errorf("%s was changed by the refused upgrade", f)
		}
	}
	if exists(t, "web/pages/password.vue") {
		t.Error("the refused upgrade wrote pages")
	}

	// As the sentence says: the line by hand, then weft upgrade again.
	fixed := strings.Replace(strings.Replace(hand, "\t\"github.com/flakerimi/weft/warp/data\"\n", "\t\"github.com/flakerimi/weft/warp/data\"\n\t\"github.com/flakerimi/weft/warp/mail\"\n", 1), "data.Module(), auth.Module()", "data.Module(), mail.Module(), auth.Module()", 1)
	if err := os.WriteFile(filepath.Join("app", "modules.go"), []byte(fixed), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs = runCmd("upgrade", "--no-install")
	if code != 0 || !strings.Contains(out, "Added web/pages/password.vue\n") || !strings.Contains(out, "Moved myapp from Weft 0.0.24 to 0.2.0\n") {
		t.Fatalf("again: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if !strings.HasPrefix(errs, "weft upgrade: go.mod replaces github.com/flakerimi/weft with "+filepath.ToSlash(root)) {
		t.Errorf("again: stderr %q", errs)
	}
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("weft gen: exit %d: %s%s", code, out, errs)
	}
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("weft build: exit %d: %s%s", code, out, errs)
	}
}

// An app already on this Weft catches up only when /password, the page every
// mailed link opens, is missing, as after an upgrade made part-way or by an
// older candidate: then the account files it lacks come back, writing over
// nothing. Otherwise a page the author removed on purpose stays removed
// (final review R2-I1 and R3-M1).
func TestUpgradeAlreadyOnAddsMissingPages(t *testing.T) {
	otherVersionApp(t, "0.2.0")
	setVersion(t, "0.2.0")
	register := filepath.Join("web", "pages", "register.vue")
	if err := os.Remove(register); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd("upgrade", "--no-install")
	if code != 0 || out != "myapp is already on Weft 0.2.0\n" || errs != "" || exists(t, register) {
		t.Fatalf("register.vue removed: exit %d, stdout %q, stderr %q; it must stay removed", code, out, errs)
	}

	if err := os.Remove(filepath.Join("web", "pages", "password.vue")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("web", "pages", "users.vue"), []byte("<template><p>mine</p></template>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs = runCmd("upgrade", "--no-install")
	if code != 0 || out != "myapp is already on Weft 0.2.0\nAdded web/pages/password.vue\nAdded web/pages/register.vue\n" || errs != "" {
		t.Fatalf("password.vue removed: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	want, err := os.ReadFile(filepath.Join(repoCmdWeft, "..", "..", "templates", "app", "web", "pages", "password.vue"))
	if err != nil || readFile(t, filepath.Join("web", "pages", "password.vue")) != string(want) {
		t.Errorf("web/pages/password.vue is not the template's (%v)", err)
	}
	if got := readFile(t, filepath.Join("web", "pages", "users.vue")); got != "<template><p>mine</p></template>\n" {
		t.Errorf("web/pages/users.vue was written over: %q", got)
	}
}
