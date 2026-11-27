package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// shopSource is the repository's examples/shop, found while the test binary
// still runs in cmd/weft.
var shopSource = func() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "examples", "shop")
}()

// shopIgnored reads examples/shop/.gitignore, so that a copy of the example
// never takes what a run there made, and the two cannot drift apart. An
// entry with a slash other than a trailing one names a path from the app's
// folder; one without names a file or folder at any depth, as git reads them.
// A pattern with a wildcard or a negation is refused, since the copy does not
// read those.
func shopIgnored() (paths, names map[string]bool, err error) {
	f, err := os.Open(filepath.Join(shopSource, ".gitignore"))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	paths, names = map[string]bool{}, map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsAny(line, "*?[!\\") {
			return nil, nil, fmt.Errorf("examples/shop/.gitignore has %q, a pattern copyShop cannot read; list plain paths there", line)
		}
		entry := strings.TrimSuffix(line, "/")
		if strings.Contains(entry, "/") {
			paths[filepath.FromSlash(strings.TrimPrefix(entry, "/"))] = true
		} else {
			names[entry] = true
		}
	}
	return paths, names, sc.Err()
}

// copyShop copies examples/shop into dir/shop without what it git-ignores,
// and returns the copy's path.
func copyShop(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(shopSource, "main.go")); err != nil {
		return "", fmt.Errorf("examples/shop has no main.go: %v", err)
	}
	paths, names, err := shopIgnored()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "shop")
	return dst, filepath.WalkDir(shopSource, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(shopSource, path)
		if rel != "." && (paths[rel] || names[d.Name()] || d.Name() == ".DS_Store") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
}

// The copy leaves out exactly what the example's .gitignore names, so a run
// of scripts/shop.sh in examples/shop never reaches a test.
func TestShopCopyFollowsItsGitignore(t *testing.T) {
	paths, names, err := shopIgnored()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"shop", filepath.Join("web", "dist"), filepath.Join("web", ".weft-types")} {
		if !paths[want] {
			t.Errorf("examples/shop/.gitignore does not name %s as a path", want)
		}
	}
	for _, want := range []string{".weft", ".weft.lock", "node_modules", "data", ".env", "pnpm-lock.yaml"} {
		if !names[want] {
			t.Errorf("examples/shop/.gitignore does not name %s", want)
		}
	}
}

// writeShopEnv writes the .env the shop needs, with a fresh secret, as
// scripts/shop.sh does.
func writeShopEnv(app string) error {
	secret := make([]byte, 48)
	rand.Read(secret)
	env := "PORT=8080\nDB_DRIVER=sqlite\nDB_URL=file:data/shop.db\nAUTH_SECRET=" + base64.RawURLEncoding.EncodeToString(secret) + "\nAUTH_OWNER_EMAIL=owner@example.com\n"
	return os.WriteFile(filepath.Join(app, ".env"), []byte(env), 0o600)
}

var (
	shopOnce               sync.Once
	shopDir                string
	shopCode               int
	shopStdout, shopStderr string
	shopCopyErr            string
)

// builtShop returns a copy of examples/shop that weft build has run in,
// made once for this test binary; tests copy it with cp -a rather than
// change it. It must be called after inTempDir, which sets WEFT_BUNDLE.
func builtShop(t *testing.T) string {
	t.Helper()
	shopOnce.Do(func() {
		dir := filepath.Join(filepath.Dir(stagedBundle), "shop-built")
		app, err := copyShop(dir)
		if err == nil {
			err = writeShopEnv(app)
		}
		if err != nil {
			shopCopyErr = err.Error()
			return
		}
		// weft build runs in this process and finds the app from its folder.
		wd, err := os.Getwd()
		if err == nil {
			err = os.Chdir(app)
		}
		if err != nil {
			shopCopyErr = err.Error()
			return
		}
		var out, errb bytes.Buffer
		shopCode = run([]string{"build"}, &out, &errb)
		shopStdout, shopStderr, shopDir = out.String(), errb.String(), app
		if err := os.Chdir(wd); err != nil {
			shopCopyErr, shopDir = "cannot go back to "+wd+": "+err.Error(), ""
		}
	})
	if shopDir == "" {
		t.Fatalf("examples/shop could not be copied and built: %s", shopCopyErr)
	}
	if shopCode != 0 {
		t.Fatalf("weft build in examples/shop: exit %d\n%s%s", shopCode, shopStdout, shopStderr)
	}
	return shopDir
}

// shopCopy copies the built shop into a temp folder of the test's own and
// returns the copy's path.
func shopCopy(t *testing.T) string {
	t.Helper()
	src := builtShop(t)
	dir := t.TempDir()
	if out, err := exec.Command("cp", "-a", src, dir).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v: %s", err, out)
	}
	return filepath.Join(dir, "shop")
}

// inShop runs name with args in app and returns its output.
func inShop(app, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = app
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// typecheck runs the pages' type check in app.
func typecheck(app string) (string, error) {
	return inShop(app, "pnpm", "exec", "vue-tsc", "--noEmit", "-p", "web")
}

// apiBlock is the object of one module in the generated api: the lines from
// "  name: {" to its closing "  },".
func apiBlock(api, module string) string {
	_, rest, ok := strings.Cut(api, "\n  "+module+": {\n")
	if !ok {
		return ""
	}
	block, _, _ := strings.Cut(rest, "\n  },\n")
	return block
}

// setupLink is the line auth logs while no user exists, with the token in the
// link's fragment.
var setupLink = regexp.MustCompile(`Set up the owner at http://\S+/setup#token=([A-Za-z0-9_-]+)`)

// logBuffer is the output of a running process, read while it writes.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// startShop runs the shop's binary in its folder, so it reads its .env and
// keeps its database in data/, and returns its base URL and the setup token
// it logged. The port is a free one found by listening; another process can
// take it before the shop does, so a start that says the port is taken is
// tried again on another. The process is stopped when the test ends.
func startShop(t *testing.T, app string) (string, string) {
	t.Helper()
	for try := 1; ; try++ {
		url, token, err := tryStartShop(t, app)
		if err == nil {
			return url, token
		}
		if !errors.Is(err, errPortTaken) || try == 3 {
			t.Fatal(err)
		}
	}
}

var errPortTaken = errors.New("the port was taken")

func tryStartShop(t *testing.T, app string) (string, string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, filepath.Join(app, "shop"))
	cmd.Dir = app
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(port))
	logs := &logBuffer{}
	cmd.Stdout, cmd.Stderr = logs, logs
	if err := cmd.Start(); err != nil {
		cancel()
		return "", "", err
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	t.Cleanup(func() { cancel(); <-exited })
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		select {
		case <-exited:
			if strings.Contains(logs.String(), "port "+strconv.Itoa(port)+" is taken") {
				return "", "", errPortTaken
			}
			return "", "", fmt.Errorf("the shop stopped: %s", logs.String())
		default:
		}
		res, err := http.Get(url + "/api/auth/me")
		if err != nil {
			continue
		}
		res.Body.Close()
		m := setupLink.FindStringSubmatch(logs.String())
		if m == nil {
			return "", "", fmt.Errorf("the shop answers but logged no setup link: %s", logs.String())
		}
		return url, m[1], nil
	}
	return "", "", fmt.Errorf("the shop did not answer within 15 s: %s", logs.String())
}

// send sends a JSON body with client and returns the status and the body.
func send(t *testing.T, client *http.Client, method, url string, body any) (int, string) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

// A password form as the server draws it, before its island is live: a POST
// form, and the page's inline guard that stops any submit of it.
var postForm = regexp.MustCompile(`<form class="weft-form"[^>]*\bmethod="post"`)

// guardsSubmit reports whether page has what keeps a password form from
// being sent before its island is live: inline scripts in <head>, before any
// module script, that, run in order as the head runs them, cancel a submit
// of a .weft-form and of no other form. It judges the guard by what it does,
// in node, so its wording is @weft/ui's to change.
func guardsSubmit(t *testing.T, page string) bool {
	t.Helper()
	_, head, _ := strings.Cut(page, "<head>")
	head, _, _ = strings.Cut(head, "</head>")
	var scripts []string
	for {
		_, rest, ok := strings.Cut(head, "<script")
		if !ok {
			break
		}
		attrs, rest, _ := strings.Cut(rest, ">")
		code, rest, _ := strings.Cut(rest, "</script>")
		head = rest
		if strings.Contains(attrs, "src=") || strings.Contains(attrs, "module") {
			break // what comes after a module script runs too late
		}
		scripts = append(scripts, code)
	}
	if len(scripts) == 0 {
		return false
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node to run the page's guard in")
	}
	in, _ := json.Marshal(scripts)
	cmd := exec.Command(node, "--input-type=module", "-e", submitHarness)
	cmd.Stdin = bytes.NewReader(in)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running the page's head scripts in node: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out)) == "guards"
}

// submitHarness runs the scripts on its standard input, a JSON list, in one
// context with a window and a document that keep their listeners, as a
// page's head runs them (a script that throws, as one reading what the stub
// lacks, adds what it added before); then it sends a submit from a .weft-form and one
// from another form to each submit listener, and prints "guards" when the
// first was cancelled and the second was not.
const submitHarness = `
import vm from 'node:vm'
let text = ''
for await (const chunk of process.stdin) text += chunk
const listeners = []
const target = { addEventListener(type, fn) { if (type === 'submit') listeners.push(fn) } }
const quiet = new Proxy(function () {}, { get: (_, k) => k === Symbol.toPrimitive ? () => '' : quiet, apply: () => quiet, set: () => true })
const document = { ...target, documentElement: quiet, head: quiet, body: quiet }
const window = { ...target, document, localStorage: { getItem: () => null, setItem() {} }, matchMedia: () => ({ matches: false, addEventListener() {} }), console }
window.window = window.self = window
const context = vm.createContext(window)
for (const code of JSON.parse(text)) {
  try { vm.runInContext(code, context, { timeout: 1000 }) } catch {}
}
function form(cls) {
  const f = { tagName: 'FORM', nodeName: 'FORM', className: cls, classList: { contains: c => c === cls } }
  f.matches = sel => sel.split(',').some(s => (s = s.trim()) === 'form' || s === '.' + cls || s === 'form.' + cls)
  f.closest = sel => f.matches(sel) ? f : null
  return f
}
function submit(cls) {
  const e = { type: 'submit', target: form(cls), currentTarget: window, defaultPrevented: false, preventDefault() { this.defaultPrevented = true }, stopPropagation() {}, stopImmediatePropagation() {} }
  for (const fn of listeners) {
    try { typeof fn === 'function' ? fn.call(window, e) : fn.handleEvent(e) } catch {}
  }
  return e.defaultPrevented
}
console.log(submit('weft-form') && !submit('other') ? 'guards' : 'no')
`

// The shop builds with weft build (which writes its typed client), its own Go
// and web tests pass, its pages type-check against the client, and the binary
// serves the pages, the run-time page's shell and the API behind its login.
func TestShop(t *testing.T) {
	inTempDir(t, "work")
	app := shopCopy(t)

	src, err := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	api := string(src)
	if !regexp.MustCompile(`(?m)^export interface Product \{$`).MatchString(api) || !regexp.MustCompile(`(?m)^  ProductInput: \{$`).MatchString(api) {
		t.Errorf("web/.weft-types/api.ts has no interface Product or schema ProductInput:\n%s", api)
	}
	if block := apiBlock(api, "products"); !regexp.MustCompile(`\blist\(query\?`).MatchString(block) {
		t.Errorf("api.products has no list(query?): %q", block)
	}

	if out, err := inShop(app, "go", "test", "-count=1", "./..."); err != nil || !strings.Contains(out, "ok  \tshop/app/products") {
		t.Errorf("go test ./... in the shop: %v\n%s", err, out)
	}
	if out, err := inShop(app, "pnpm", "test"); err != nil || !strings.Contains(out, "# fail 0") && !strings.Contains(out, "ℹ fail 0") {
		t.Errorf("pnpm test (the web tests) in the shop: %v\n%s", err, out)
	}
	if out, err := typecheck(app); err != nil {
		t.Errorf("the pages do not type-check against the generated client: %v\n%s", err, out)
	}

	url, token := startShop(t, app)
	for path, want := range map[string]string{
		"/":            ">Shop</h1>",
		"/login":       `<weft-island data-id="LoginForm"`,
		"/setup":       `<weft-island data-id="SetupForm"`,
		"/products":    `<weft-island data-id="ProductsTable"`,
		"/products/42": `data-weft-route="/products/:id"`,
	} {
		code, body := get(t, url+path)
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d, want 200 with %q: %s", path, code, want, body)
		}
		if (path == "/login" || path == "/setup") && (!postForm.MatchString(body) || !guardsSubmit(t, body)) {
			t.Errorf("GET %s: the password form is not a POST form behind the inline submit guard: %s", path, body)
		}
	}
	if code, body := get(t, url+"/nope"); code != 404 || !strings.Contains(body, "Not found") {
		t.Errorf("GET /nope = %d %q", code, body)
	}
	if code, body := get(t, url+"/api/products"); code != 401 || !strings.Contains(body, `"code":"unauthorized"`) {
		t.Errorf("GET /api/products without a session = %d %q", code, body)
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if code, body := send(t, client, "POST", url+"/api/auth/setup", map[string]string{"token": token, "password": "a password for the test"}); code != 201 {
		t.Fatalf("setup = %d %s", code, body)
	}
	if code, body := send(t, client, "POST", url+"/api/products", map[string]any{"name": "Lamp", "price": 30, "status": "live"}); code != 201 {
		t.Fatalf("create = %d %s", code, body)
	}
	code, body := send(t, client, "GET", url+"/api/products", nil)
	var list struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if code != 200 || json.Unmarshal([]byte(body), &list) != nil || list.Total != 1 || len(list.Items) != 1 || list.Items[0].Name != "Lamp" {
		t.Errorf("list after create = %d %s", code, body)
	}
}

// renamedPrice is a type error in a page that names price.
var renamedPrice = regexp.MustCompile(`\.vue\(\d+,\d+\): error TS\d+:[^\n]*\bprice\b`)

// A Go field renamed is a TypeScript error in the pages after weft gen: the
// design's promise, on the shop's own pages.
func TestShopRenamedFieldBreaksTypes(t *testing.T) {
	inTempDir(t, "work")
	app := shopCopy(t)
	model := filepath.Join(app, "app", "products", "model.go")
	src, err := os.ReadFile(model)
	if err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(string(src), "`json:\"price\"", "`json:\"cost\"", 1)
	if renamed == string(src) {
		t.Fatalf("app/products/model.go has no json:\"price\": %s", src)
	}
	if err := os.WriteFile(model, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(app)
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("weft gen after the rename: exit %d: %s%s", code, out, errs)
	}
	out, err := typecheck(app)
	if err == nil || !renamedPrice.MatchString(out) {
		t.Errorf("the pages type-check, or fail elsewhere, though Product has no price any more: %v\n%s", err, out)
	}
}

// browserPath is a Chrome or Chromium to drive the shop with: WEFT_BROWSER,
// else one installed in the usual places, else "".
func browserPath() string {
	if p := os.Getenv("WEFT_BROWSER"); p != "" {
		return p
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// The shop in a real browser, as a person uses it (examples/shop/test/browser.mjs):
// the setup link, whose token then leaves the address; next links that lead
// off the site; create, a taken name, the browser's own checks, edit, search
// and filter; the product page, and addresses that only look like an id;
// delete; sign out and the way back; a submit on /login before its island is
// live, which sends nothing; and a wrong password. Skipped when no Chrome or
// Chromium is installed (set WEFT_BROWSER to name one).
func TestShopInBrowser(t *testing.T) {
	inTempDir(t, "work")
	browser := browserPath()
	if browser == "" {
		t.Skip("no Chrome or Chromium is installed; set WEFT_BROWSER to drive the shop in one")
	}
	app := shopCopy(t)
	url, token := startShop(t, app)
	cmd := exec.Command("node", "test/browser.mjs", browser, url, url+"/setup#token="+token, t.TempDir())
	cmd.Dir = app
	t.Logf("drove the shop in %s:\n%s", browser, driveBrowser(t, cmd, "the shop in "+browser))
}

// driveBrowser runs cmd, a node script that drives a browser, with
// runInGroup for the time the test has left (browserWaitFor), and fails the
// test if it fails or takes too long. It returns what the script printed.
func driveBrowser(t *testing.T, cmd *exec.Cmd, what string) string {
	t.Helper()
	deadline, set := t.Deadline()
	wait := browserWaitFor(time.Now(), deadline, set)
	out, err := runInGroup(cmd, wait)
	if errors.Is(err, errBrowserTimedOut) {
		t.Fatalf("%s did not finish within %s, so its browser was stopped:\n%s", what, wait, out)
	}
	if err != nil {
		t.Fatalf("%s: %v\n%s", what, err, out)
	}
	return out
}

// errBrowserTimedOut is runInGroup's error when cmd did not finish in time.
var errBrowserTimedOut = errors.New("did not finish in time")

// runInGroup runs cmd in a process group of its own, which the browser it
// starts joins, and returns what it printed. However it ends, the group is
// stopped and gone before runInGroup returns (stopGroup waits for that): a
// browser still closing its helpers after the script exits writes nothing
// into the test's folders while go test removes them, and no other browser
// is touched. After wait it stops the group and returns errBrowserTimedOut.
func runInGroup(cmd *exec.Cmd, wait time.Duration) (string, error) {
	var got lockedBuffer
	cmd.Stdout, cmd.Stderr = &got, &got
	// Wait returns at most 5 s after the script exits even if something it
	// started still holds its output open, so the group is still stopped.
	cmd.WaitDelay = 5 * time.Second
	if err := startGroup(cmd); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		stopGroup(cmd) // what is left of it, such as a browser still closing
		return got.String(), err
	case <-time.After(wait):
		stopGroup(cmd)
		<-done
		return got.String(), errBrowserTimedOut
	}
}

// browserWait is how long the shop's browser test may run, and browserMargin
// how long before go test's own time is up it is stopped, so the test stops
// the browser it started before the runner ends the process.
var browserWait, browserMargin = 4 * time.Minute, 30 * time.Second

// browserWaitFor is how long the browser test waits, started at now in a test
// whose time ends at deadline when set: browserWait, or less, so it ends
// browserMargin before the deadline; at least a second.
func browserWaitFor(now, deadline time.Time, set bool) time.Duration {
	if !set {
		return browserWait
	}
	return max(min(browserWait, deadline.Sub(now)-browserMargin), time.Second)
}

// lockedBuffer is a bytes.Buffer two writers may share.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// guardsSubmit finds the guard by what it does, in any wording: run as the
// page's head runs it, it cancels a submit of a .weft-form and of no other
// form; and only in <head> before a module script.
func TestGuardsSubmit(t *testing.T) {
	guard := `<script data-weft-prepaint>window.addEventListener("submit",function(e){var t=e.target;if(t&&t.classList&&t.classList.contains("weft-form"))e.preventDefault()},true)</script>`
	reworded := `<script>addEventListener( 'submit', (ev) => { if (ev.target.matches?.('.weft-form')) { ev.preventDefault ( ) } }, true )</script>`
	byHandle := `<script>document.addEventListener("submit", { handleEvent(e) { if (e.target.closest('form.weft-form')) e.preventDefault() } }, { capture: true })</script>`
	theme := `<script>document.documentElement.dataset.theme = localStorage.getItem('theme') || 'light'</script>`
	module := `<script type="module" src="/assets/a.js"></script>`
	for page, want := range map[string]bool{
		"<html><head>" + guard + module + "</head><body></body></html>":            true,
		"<html><head>" + reworded + module + "</head><body></body></html>":         true,
		"<html><head>" + theme + byHandle + module + "</head><body></body></html>": true,
		"<html><head>" + module + guard + "</head><body></body></html>":            false,
		"<html><head>" + module + "</head><body>" + guard + "</body></html>":       false,
		"<html><head><script>console.log(1)</script></head></html>":                false,
		// It names a submit, weft-form and preventDefault(), and stops nothing.
		`<html><head><script>// "submit" .weft-form preventDefault()` + "\n" + `</script></head></html>`: false,
		// It stops every form's submit, not the password form's alone.
		`<html><head><script>addEventListener("submit", e => e.preventDefault(), true)</script></head></html>`: false,
	} {
		if got := guardsSubmit(t, page); got != want {
			t.Errorf("guardsSubmit(%q) = %v, want %v", page, got, want)
		}
	}
}

// The browser test stops what it started before go test's own time is up:
// its wait is browserWait, or what is left of the test's time less a margin
// for stopping the browser.
func TestBrowserWaitEndsBeforeTheTest(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		deadline time.Time
		set      bool
		want     time.Duration
	}{
		{time.Time{}, false, browserWait},
		{now.Add(time.Hour), true, browserWait},
		{now.Add(2 * time.Minute), true, 2*time.Minute - browserMargin},
		{now.Add(10 * time.Second), true, time.Second},
	} {
		got := browserWaitFor(now, c.deadline, c.set)
		if got != c.want {
			t.Errorf("deadline %v (set %v): wait %v, want %v", c.deadline.Sub(now), c.set, got, c.want)
		}
	}
}
